package main

import (
	"fmt"
	"image"
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func setupTestGamePathfinding(playerPos image.Point, blockers []image.Point) *Game {
	g := &Game{
		Player: &Player{
			Entity: Entity{X: playerPos.X, Y: playerPos.Y, Width: 1, Height: 1, HP: 10, MaxHP: 10, CurrentAlpha: 1.0},
		},
		Enemies:       make([]*Enemy, 0),
		FloatingTexts: make([]*FloatingText, 0),
		CombatLog:     make([]string, 0, combatLogLength),
	}
	for i, pos := range blockers {
		g.Enemies = append(g.Enemies, &Enemy{
			Entity: Entity{Name: fmt.Sprintf("Blocker%d", i), X: pos.X, Y: pos.Y, Width: 1, Height: 1, HP: 1, MaxHP: 1, CurrentAlpha: 1.0},
		})
	}
	return g
}

func createTestPlayerWithClass(level int, className string, initialResources map[string]int, knownSpells []string, enemiesToSpawn []EnemySpawnInfo, playerPos image.Point) *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)
	g.FloatingTexts = make([]*FloatingText, 0)
	g.WaveDefinitions = []WaveDefinition{}
	g.InputMode = InputModeMap

	classDef, exists := ClassDefinitions[className]
	if !exists {
		panic(fmt.Sprintf("Test setup error: Class %s not found", className))
	}

	playerStr, playerDex, playerCon := 8, 13, 14
	playerInt, playerWis, playerCha := 15, 12, 10
	if className == "Fighter" {
		playerStr, playerDex, playerCon = 15, 14, 13
		playerInt, playerWis, playerCha = 8, 12, 10
	}

	playerConMod := getModifier(playerCon)
	playerAC := 10 + getModifier(playerDex)

	playerMaxHP := classDef.HitDieSize + playerConMod
	for i := 2; i <= level; i++ {
		avgRoll := classDef.HitDieSize/2 + 1
		hpIncrease := max(1, avgRoll+playerConMod)
		playerMaxHP += hpIncrease
	}

	maxSlotsL1 := 0
	if className == "Mage" {
		if level >= 1 {
			maxSlotsL1 = 2
		}
		if level >= 2 {
			maxSlotsL1 = 3
		}
	}

	g.Player = &Player{
		Entity: Entity{
			X: playerPos.X, Y: playerPos.Y, Width: 1, Height: 1,
			HP: playerMaxHP, MaxHP: playerMaxHP, AC: playerAC,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Name: "TestPlayer", CurrentAlpha: 1.0,
		},
		Level:                level,
		Class:                className,
		ProficiencyBonus:     calculateProficiencyBonus(level),
		MaxMovementPoints:    playerBaseMovement,
		ClassResources:       make(map[string]int),
		MaxHitDice:           level,
		HitDice:              level,
		MaxSpellSlotsL1:      maxSlotsL1,
		SpellSlotsL1:         maxSlotsL1,
		KnownSpells:          knownSpells,
		UsedReaction:         false,
		UsedArcaneRecovery:   false,
		ACBonusUntilNextTurn: 0,
		LastSpellCastID:      "",
	}

	g.initializePlayerResources()
	g.Player.SpellSlotsL1 = g.Player.MaxSpellSlotsL1

	if initialResources != nil {
		for key, value := range initialResources {
			g.Player.ClassResources[key] = value
		}
	}

	spawnPoints := []image.Point{
		{X: g.Player.X + 1, Y: g.Player.Y},
		{X: g.Player.X - 1, Y: g.Player.Y},
		{X: g.Player.X, Y: g.Player.Y + 1},
		{X: g.Player.X, Y: g.Player.Y - 1},
	}
	for i, spawnInfo := range enemiesToSpawn {
		enemyDef, defExists := EnemyDefinitions[spawnInfo.TypeName]
		if !defExists {
			panic(fmt.Sprintf("Test setup error: Enemy type %s not found", spawnInfo.TypeName))
		}
		spawnIdx := i % len(spawnPoints)

		enemyX, enemyY := spawnPoints[spawnIdx].X, spawnPoints[spawnIdx].Y

		var sprite *ebiten.Image = nil

		g.spawnEnemyFromDef(enemyX, enemyY, enemyDef, sprite)
	}
	g.CurrentTurn = PlayerTurn
	g.startPlayerTurn()
	return g
}

type pathResult struct {
	finalX, finalY, finalDist int
	steps                     int
}

func simulateMultiStepPath(g *Game, startX, startY, targetX, targetY, entityW, entityH, movingIdx, maxSteps int) pathResult {
	currX, currY := startX, startY
	stepsUsed := 0

	for stepsUsed < maxSteps {
		nextX, nextY, found := g.findPathStep(currX, currY, targetX, targetY, entityW, entityH, movingIdx)
		if !found || (nextX == currX && nextY == currY) {
			break
		}

		currX, currY = nextX, nextY
		stepsUsed++

		if distance(currX, currY, targetX, targetY) <= 1 {
			break
		}
	}

	return pathResult{
		finalX:    currX,
		finalY:    currY,
		finalDist: distance(currX, currY, targetX, targetY),
		steps:     stepsUsed,
	}
}

func TestIsTileFullyBlocked(t *testing.T) {
	blockers := []image.Point{{X: 3, Y: 3}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)

	cases := []struct {
		x, y, w, h int
		expected   bool
		desc       string
	}{
		{2, 2, 1, 1, false, "Empty space"},
		{3, 3, 1, 1, true, "Directly on blocker"},
		{2, 2, 2, 2, true, "Overlaps blocker"},
		{-1, 3, 1, 1, true, "Off map left"},
		{mapWidth, 3, 1, 1, true, "Off map right"},
		{3, -1, 1, 1, true, "Off map top"},
		{3, mapHeight, 1, 1, true, "Off map bottom"},
		{mapWidth - 1, mapHeight - 1, 2, 2, true, "Partially off map bottom-right"},
		{5, 5, 1, 1, true, "On player position"},
	}

	for _, tc := range cases {
		result := g.isTileFullyBlocked(tc.x, tc.y, tc.w, tc.h, -1)
		if result != tc.expected {
			t.Errorf("%s: expected=%v, got=%v for pos=(%d,%d) size=%dx%d",
				tc.desc, tc.expected, result, tc.x, tc.y, tc.w, tc.h)
		}
	}
}

func TestFindPathStep_DirectPath(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 1, 1
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 2, 1
	expectedFound := true

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findPathStep DirectPath failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindPathStep_BlockedDirect_Sideways(t *testing.T) {
	blockers := []image.Point{{X: 4, Y: 5}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 3, 5
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 2, 5
	expectedFound := true

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findPathStep BlockedDirect failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindPathStep_TargetAdjacent(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 4, 5
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 4, 5
	expectedFound := false

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound {
		t.Errorf("findPathStep TargetAdjacent failed: expected found %t, got %t. Position: (%d,%d)",
			expectedFound, found, nextX, nextY)
	}
	if !found && (nextX != expectedX || nextY != expectedY) {
		t.Errorf("findPathStep TargetAdjacent failed: expected position (%d,%d) when found=false, got (%d,%d)",
			expectedX, expectedY, nextX, nextY)
	}
}

func TestFindPathStep_CompletelyBlocked(t *testing.T) {
	blockers := []image.Point{
		{X: 0, Y: 1}, {X: 1, Y: 0}, {X: 2, Y: 1}, {X: 1, Y: 2},
	}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 1, 1
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 1, 1
	expectedFound := false

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findPathStep CompletelyBlocked failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindPathStep_ComplexObstacleConfiguration(t *testing.T) {
	blockers := []image.Point{{X: 3, Y: 2}, {X: 3, Y: 3}, {X: 3, Y: 4}, {X: 4, Y: 4}, {X: 5, Y: 4}}
	g := setupTestGamePathfinding(image.Point{X: 7, Y: 7}, blockers)
	startX, startY := 1, 3
	targetX, targetY := 6, 3
	initialDist := distance(startX, startY, targetX, targetY)

	result := simulateMultiStepPath(g, startX, startY, targetX, targetY, 1, 1, -1, 15)

	if result.finalDist >= initialDist && result.finalDist > 1 {
		t.Errorf("Failed to navigate around obstacle wall: start dist=%d, final dist=%d after %d steps, ended at (%d,%d)",
			initialDist, result.finalDist, result.steps, result.finalX, result.finalY)
	}
	if result.finalDist > 1 && result.steps == 0 {
		t.Errorf("Failed to make any progress around obstacle wall: start dist=%d, final dist=%d", initialDist, result.finalDist)
	}
}

func TestFindPathStep_DiagonalCorner(t *testing.T) {
	blockers := []image.Point{{X: 2, Y: 1}, {X: 1, Y: 2}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 1, 1
	targetX, targetY := 3, 3
	entityW, entityH := 1, 1
	movingIdx := -1

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if !found {
		t.Errorf("Entity should find a way out of diagonal corner, but got stuck")
	}
	if found && (nextX == startX && nextY == startY) {
		t.Errorf("Entity found a path but didn't move out of diagonal corner, stayed at (%d,%d)", nextX, nextY)
	}
}

func TestFindPathStep_LargeEntityMovement(t *testing.T) {
	blockers := []image.Point{{X: 3, Y: 3}}
	g := setupTestGamePathfinding(image.Point{X: 7, Y: 7}, blockers)
	startX, startY := 1, 1
	targetX, targetY := 4, 4
	entityW, entityH := 2, 2
	movingIdx := -1

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found {
		if nextX < 0 || nextY < 0 || nextX+entityW > mapWidth || nextY+entityH > mapHeight {
			t.Errorf("Large entity moved partially out of bounds: (%d,%d)", nextX, nextY)
		}

		for w := 0; w < entityW; w++ {
			for h := 0; h < entityH; h++ {
				if g.isTileBlocked(nextX+w, nextY+h, movingIdx) {
					t.Errorf("Large entity moved into a blocked tile at (%d,%d)", nextX+w, nextY+h)
				}
			}
		}
	} else {
		t.Logf("Large entity movement test: No path found from (%d,%d) towards (%d,%d), which might be expected depending on blocking.", startX, startY, targetX, targetY)
	}
}

func TestFindRetreatStep_DirectRetreat(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 5, 6
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 5, 7
	expectedFound := true

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findRetreatStep DirectRetreat failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindRetreatStep_BlockedDirect_Sideways(t *testing.T) {
	blockers := []image.Point{{X: 5, Y: 7}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 5, 6
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX1, expectedY1 := 4, 6
	expectedX2, expectedY2 := 6, 6
	expectedFound := true

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound {
		t.Errorf("findRetreatStep BlockedDirect failed: expected found %t, got %t", expectedFound, found)
	}
	if !((nextX == expectedX1 && nextY == expectedY1) || (nextX == expectedX2 && nextY == expectedY2)) {
		t.Errorf("findRetreatStep BlockedDirect failed: expected (%d,%d) or (%d,%d), got (%d,%d)",
			expectedX1, expectedY1, expectedX2, expectedY2, nextX, nextY)
	}
}

func TestFindRetreatStep_Cornered(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 1, Y: 1}, nil)
	startX, startY := 0, 0
	targetX, targetY := 1, 1
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX1, expectedY1 := 1, 0
	expectedX2, expectedY2 := 0, 1
	expectedFound := true

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound {
		t.Errorf("findRetreatStep Cornered failed: expected found %t, got %t", expectedFound, found)
	}

	if found && !((nextX == expectedX1 && nextY == expectedY1) || (nextX == expectedX2 && nextY == expectedY2)) {
		t.Errorf("findRetreatStep Cornered failed: expected (%d,%d) or (%d,%d), got (%d,%d)",
			expectedX1, expectedY1, expectedX2, expectedY2, nextX, nextY)
	}
}

func TestFindRetreatStep_NoValidMoves(t *testing.T) {
	blockers := []image.Point{{X: 1, Y: 0}, {X: 0, Y: 1}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 0, 0
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 0, 0
	expectedFound := false

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findRetreatStep NoValidMoves failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindRetreatStep_NotAdjacent(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 3, 3
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if !found {
		t.Errorf("findRetreatStep NotAdjacent failed: Expected a move to be found (to maximize distance)")
	}

	initialDist := distance(startX, startY, targetX, targetY)
	newDist := distance(nextX, nextY, targetX, targetY)

	if found && newDist < initialDist {
		t.Errorf("findRetreatStep NotAdjacent failed: Moved closer (%d,%d -> dist %d) instead of further/same (start dist %d)",
			nextX, nextY, newDist, initialDist)
	}
}

func TestFighterResourceInitialization(t *testing.T) {
	initialPos := image.Point{X: 5, Y: 5}
	g1 := createTestPlayerWithClass(1, "Fighter", nil, nil, nil, initialPos)
	p1 := g1.Player
	if p1.HitDice != 1 || p1.MaxHitDice != 1 {
		t.Errorf("Lvl 1 Fighter Hit Dice incorrect: expected 1/1, got %d/%d", p1.HitDice, p1.MaxHitDice)
	}
	if _, ok := p1.ClassResources["action_surge"]; !ok {
		t.Errorf("Lvl 1 Fighter Action Surge uses incorrect: expected entry, but not found")
	}
	if _, ok := p1.ClassResources["second_wind"]; !ok {
		t.Errorf("Lvl 1 Fighter Second Wind uses incorrect: expected entry, but not found")
	}

	g1.buildAvailableActions()
	for _, action := range g1.availableActions {
		if action.ID == "action_surge" {
			t.Errorf("Lvl 1 Fighter should not have Action Surge available (requires Lvl 2)")
		}
		if action.ID == "second_wind" {
			t.Errorf("Lvl 1 Fighter should not have Second Wind available (requires Lvl 2)")
		}
	}

	g2 := createTestPlayerWithClass(2, "Fighter", nil, nil, nil, initialPos)
	p2 := g2.Player
	if p2.HitDice != 2 || p2.MaxHitDice != 2 {
		t.Errorf("Lvl 2 Fighter Hit Dice incorrect: expected 2/2, got %d/%d", p2.HitDice, p2.MaxHitDice)
	}
	if asUses, ok := p2.ClassResources["action_surge"]; !ok || asUses != 1 {
		t.Errorf("Lvl 2 Fighter Action Surge uses incorrect: expected 1, got %d (found: %t)", asUses, ok)
	}
	if swUses, ok := p2.ClassResources["second_wind"]; !ok || swUses != 1 {
		t.Errorf("Lvl 2 Fighter Second Wind uses incorrect: expected 1, got %d (found: %t)", swUses, ok)
	}

	g2.buildAvailableActions()
	asFound := false
	swFound := false
	for _, action := range g2.availableActions {
		if action.ID == "action_surge" {
			asFound = true
		}
		if action.ID == "second_wind" {
			swFound = true
		}
	}
	if !asFound {
		t.Errorf("Lvl 2 Fighter should have Action Surge available")
	}
	if !swFound {
		t.Errorf("Lvl 2 Fighter should have Second Wind available")
	}
}

func TestFighterResourceConsumption(t *testing.T) {
	initialPos := image.Point{X: 5, Y: 5}
	g := createTestPlayerWithClass(2, "Fighter", nil, nil, nil, initialPos)
	p := g.Player
	actionSurgeDef := ActionTable["action_surge"]
	secondWindDef := ActionTable["second_wind"]

	initialAS := p.ClassResources["action_surge"]
	if initialAS != 1 {
		t.Fatalf("Test setup error: Expected 1 Action Surge use, got %d", initialAS)
	}
	successAS1 := g.executeAction(actionSurgeDef, -1, -1)
	if !successAS1 {
		t.Errorf("First Action Surge execution failed unexpectedly")
	}
	if p.ClassResources["action_surge"] != 0 {
		t.Errorf("Action Surge uses after 1st execution: expected 0, got %d", p.ClassResources["action_surge"])
	}
	if p.ActionTaken {
		t.Errorf("Action Surge incorrectly marked ActionTaken as true")
	}
	p.ActionTaken = false

	successAS2 := g.executeAction(actionSurgeDef, -1, -1)
	if successAS2 {
		t.Errorf("Second Action Surge execution succeeded unexpectedly (should have 0 uses)")
	}
	if p.ClassResources["action_surge"] != 0 {
		t.Errorf("Action Surge uses after failed 2nd execution: expected 0, got %d", p.ClassResources["action_surge"])
	}

	p.HP = p.MaxHP / 2
	initialHP := p.HP
	initialSW := p.ClassResources["second_wind"]
	if initialSW != 1 {
		t.Fatalf("Test setup error: Expected 1 Second Wind use, got %d", initialSW)
	}
	successSW1 := g.executeAction(secondWindDef, -1, -1)
	if !successSW1 {
		t.Errorf("First Second Wind execution failed unexpectedly")
	}
	if p.ClassResources["second_wind"] != 0 {
		t.Errorf("Second Wind uses after 1st execution: expected 0, got %d", p.ClassResources["second_wind"])
	}
	if p.HP <= initialHP {
		t.Errorf("Second Wind did not heal: HP before=%d, HP after=%d", initialHP, p.HP)
	}
	if !p.ActionTaken {
		t.Errorf("Second Wind did not mark ActionTaken as true")
	}

	successSW2 := g.executeAction(secondWindDef, -1, -1)
	if successSW2 {
		t.Errorf("Second Second Wind execution succeeded unexpectedly (should have 0 uses)")
	}
	if p.ClassResources["second_wind"] != 0 {
		t.Errorf("Second Wind uses after failed 2nd execution: expected 0, got %d", p.ClassResources["second_wind"])
	}

	p.HP = 1
	p.HitDice = 1
	g.shortRest()
	if p.HitDice != 0 {
		t.Errorf("Hit Dice after short rest with 1 die: expected 0, got %d", p.HitDice)
	}
	if p.HP <= 1 {
		t.Errorf("Short rest did not heal with 1 Hit Die available")
	}
	hpAfterRest1 := p.HP
	g.shortRest()
	if p.HitDice != 0 {
		t.Errorf("Hit Dice after short rest with 0 dice: expected 0, got %d", p.HitDice)
	}
	if p.HP != hpAfterRest1 {
		t.Errorf("Short rest incorrectly healed with 0 Hit Dice available")
	}
}

func TestFighterResourceReset(t *testing.T) {
	initialPos := image.Point{X: 5, Y: 5}
	initialRes := map[string]int{
		"action_surge": 0,
		"second_wind":  0,
	}
	g := createTestPlayerWithClass(2, "Fighter", initialRes, nil, nil, initialPos)
	p := g.Player
	p.HitDice = 0

	g.longRest()

	if asUses, ok := p.ClassResources["action_surge"]; !ok || asUses != 0 {
		t.Errorf("Action Surge uses after Long Rest: expected 0 (RestTypeNever), got %d (exists: %t)", asUses, ok)
	}
	if swUses, ok := p.ClassResources["second_wind"]; !ok || swUses != 0 {
		t.Errorf("Second Wind uses after Long Rest: expected 0 (RestTypeNever), got %d (exists: %t)", swUses, ok)
	}

	expectedDiceRecovery := max(1, p.MaxHitDice/2)
	if p.HitDice != expectedDiceRecovery {
		t.Errorf("Hit Dice after Long Rest: expected %d, got %d", expectedDiceRecovery, p.HitDice)
	}
}

func TestLevelUpResourceReset(t *testing.T) {
	initialPos := image.Point{X: 5, Y: 5}
	initialRes := map[string]int{
		"action_surge": 0,
		"second_wind":  0,
	}
	g := createTestPlayerWithClass(2, "Fighter", initialRes, nil, nil, initialPos)
	p := g.Player

	g.levelUpPlayer()

	if p.Level != 3 {
		t.Fatalf("Player did not level up correctly: expected 3, got %d", p.Level)
	}

	if asUses := p.ClassResources["action_surge"]; asUses != 1 {
		t.Errorf("Action Surge uses after Level Up: expected 1, got %d", asUses)
	}
	if swUses := p.ClassResources["second_wind"]; swUses != 1 {
		t.Errorf("Second Wind uses after Level Up: expected 1, got %d", swUses)
	}
	if p.HitDice != p.MaxHitDice {
		t.Errorf("Hit Dice after Level Up: expected %d, got %d", p.MaxHitDice, p.HitDice)
	}
}

func TestIntegration_Fighter_ActionSurge_DoubleAttack(t *testing.T) {
	initialPos := image.Point{X: 5, Y: 5}
	enemies := []EnemySpawnInfo{{TypeName: "Melee Skeleton"}}
	g := createTestPlayerWithClass(2, "Fighter", nil, nil, enemies, initialPos)
	p := g.Player
	if len(g.Enemies) == 0 {
		t.Fatal("Test setup error: Enemy did not spawn")
	}
	enemy := g.Enemies[0]
	actionSurgeDef := ActionTable["action_surge"]
	meleeAttackDef := ActionTable["melee_attack"]

	if p.ClassResources["action_surge"] != 1 {
		t.Fatalf("Test setup error: Expected 1 Action Surge use, got %d", p.ClassResources["action_surge"])
	}

	g.Player.X = enemy.X - 1
	g.Player.Y = enemy.Y

	successAttack1 := g.executeAction(meleeAttackDef, enemy.X, enemy.Y)
	if !successAttack1 {
		t.Fatalf("First melee attack failed unexpectedly")
	}
	if !p.ActionTaken {
		t.Fatalf("First melee attack did not consume standard action")
	}

	successSurge := g.executeAction(actionSurgeDef, -1, -1)
	if !successSurge {
		t.Fatalf("Action Surge execution failed unexpectedly")
	}
	if p.ClassResources["action_surge"] != 0 {
		t.Fatalf("Action Surge resource not consumed")
	}
	if p.ActionTaken {
		t.Fatalf("Action Surge incorrectly set ActionTaken to true (should be false)")
	}

	successAttack2 := g.executeAction(meleeAttackDef, enemy.X, enemy.Y)
	if !successAttack2 {
		t.Errorf("Second melee attack (after surge) failed unexpectedly")
	}
	if !p.ActionTaken {
		t.Errorf("Second melee attack did not consume the surged standard action")
	}
}

func TestIntegration_Fighter_SecondWind_MidCombat(t *testing.T) {
	initialPos := image.Point{X: 5, Y: 5}
	enemies := []EnemySpawnInfo{{TypeName: "Small Slime"}}
	g := createTestPlayerWithClass(2, "Fighter", nil, nil, enemies, initialPos)
	p := g.Player
	secondWindDef := ActionTable["second_wind"]

	if p.ClassResources["second_wind"] != 1 {
		t.Fatalf("Test setup error: Expected 1 Second Wind use, got %d", p.ClassResources["second_wind"])
	}

	p.HP = 5
	initialHP := p.HP

	successSW := g.executeAction(secondWindDef, -1, -1)
	if !successSW {
		t.Fatalf("Second Wind execution failed unexpectedly")
	}
	if p.ClassResources["second_wind"] != 0 {
		t.Fatalf("Second Wind resource not consumed")
	}
	if !p.ActionTaken {
		t.Fatalf("Second Wind did not consume standard action")
	}
	if p.HP <= initialHP {
		t.Errorf("Second Wind did not heal player: HP before=%d, HP after=%d", initialHP, p.HP)
	}
	if p.HP > p.MaxHP {
		t.Errorf("Second Wind healed player above MaxHP: HP=%d, MaxHP=%d", p.HP, p.MaxHP)
	}
}

func TestMoveAoOReactionDeclined(t *testing.T) {
	rand.Seed(1)
	startPos := image.Point{X: 5, Y: 5}
	targetPos := image.Point{X: 5, Y: 6}
	enemyInfo := []EnemySpawnInfo{{TypeName: "Melee Skeleton"}}
	g := createTestPlayerWithClass(1, "Mage", nil, nil, enemyInfo, startPos)
	if len(g.Enemies) == 0 {
		t.Fatal("Test setup error: Enemy did not spawn")
	}
	enemy := g.Enemies[0]
	enemy.X = startPos.X + 1
	enemy.Y = startPos.Y
	enemy.Strength = 18
	g.Player.AC = 11
	if g.Player.SpellSlotsL1 < 1 {
		t.Fatal("Test setup error: Player needs spell slots")
	}

	initialHP := g.Player.HP
	initialSlots := g.Player.SpellSlotsL1
	g.Player.MovementPoints = 1

	killed, hit := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
	if killed {
		t.Fatal("Test setup error: Player killed by AoO unexpectedly")
	}
	if !hit {
		t.Logf("Warning: Seeded AoO roll missed base AC; cannot fully verify reaction trigger prevention logic. HP check might be inaccurate.")
	}

	if hit && !g.Player.UsedReaction && g.Player.SpellSlotsL1 > 0 {
		g.reactionPending = true
		g.InputMode = InputModeReactionPrompt
	} else if hit {
		t.Fatalf("AoO hit but reaction conditions not met in setup? UsedReaction=%t, Slots=%d", g.Player.UsedReaction, g.Player.SpellSlotsL1)
	}

	if !g.reactionPending || g.InputMode != InputModeReactionPrompt {
		if hit {
			t.Fatalf("Expected reactionPending=true and InputModeReactionPrompt after AoO trigger, got pending=%t, mode=%v", g.reactionPending, g.InputMode)
		}
	}

	g.addCombatLog("Declined Shield reaction.")
	g.reactionPending = false
	g.InputMode = InputModeMap
	g.addCombatLog("Attack resolved.")

	g.Player.X = targetPos.X
	g.Player.Y = targetPos.Y
	g.Player.MovementPoints--

	if g.InputMode != InputModeMap {
		t.Errorf("Expected InputModeMap after declining reaction, got %v", g.InputMode)
	}
	if hit && g.Player.HP >= initialHP {
		t.Errorf("Player HP should have decreased after declining Shield, HP: %d, Initial: %d", g.Player.HP, initialHP)
	}
	if g.Player.SpellSlotsL1 != initialSlots {
		t.Errorf("Player spell slots changed after declining Shield, Slots: %d, Initial: %d", g.Player.SpellSlotsL1, initialSlots)
	}
	if g.Player.UsedReaction {
		t.Error("Player UsedReaction should be false after declining Shield")
	}
	if g.Player.X != targetPos.X || g.Player.Y != targetPos.Y {
		t.Errorf("Player position incorrect after declining Shield, Pos: (%d,%d), Target: (%d,%d)", g.Player.X, g.Player.Y, targetPos.X, targetPos.Y)
	}
	if g.Player.MovementPoints != 0 {
		t.Errorf("Player movement points not decremented, got %d", g.Player.MovementPoints)
	}
}

func TestMoveAoOReactionAccepted(t *testing.T) {
	rand.Seed(1)
	startPos := image.Point{X: 5, Y: 5}
	targetPos := image.Point{X: 5, Y: 6}
	enemyInfo := []EnemySpawnInfo{{TypeName: "Melee Skeleton"}}
	g := createTestPlayerWithClass(1, "Mage", nil, nil, enemyInfo, startPos)
	if len(g.Enemies) == 0 {
		t.Fatal("Test setup error: Enemy did not spawn")
	}
	enemy := g.Enemies[0]
	enemy.X = startPos.X + 1
	enemy.Y = startPos.Y
	enemy.Strength = 18
	g.Player.AC = 11
	if g.Player.SpellSlotsL1 < 1 {
		t.Fatal("Test setup error: Player needs spell slots for Shield")
	}

	initialHP := g.Player.HP
	initialSlots := g.Player.SpellSlotsL1
	g.Player.MovementPoints = 1

	killed, hit := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
	if killed {
		t.Fatal("Test setup error: Player killed by AoO unexpectedly")
	}

	if !hit {
		t.Skipf("Skipping reaction accept test: Seeded AoO roll (vs AC %d) missed, cannot test reaction trigger.", g.Player.AC)
	}

	if !g.reactionPending || g.InputMode != InputModeReactionPrompt {
		t.Fatalf("Expected reactionPending=true and InputModeReactionPrompt after AoO hit, got pending=%t, mode=%v", g.reactionPending, g.InputMode)
	}

	shieldAction := ActionTable["shield"]
	simulatedSuccess := g.executeAction(shieldAction, -1, -1)
	if !simulatedSuccess {
		t.Fatalf("Executing Shield action failed unexpectedly during test simulation")
	}

	g.reactionPending = false
	g.InputMode = InputModeMap
	g.addCombatLog("Attack resolved (after Shield).")

	g.Player.X = targetPos.X
	g.Player.Y = targetPos.Y
	g.Player.MovementPoints--

	if g.InputMode != InputModeMap {
		t.Errorf("Expected InputModeMap after accepting reaction, got %v", g.InputMode)
	}

	if g.Player.HP != initialHP {
		attackMissedWithShield := false
		lastLog := g.CombatLog[len(g.CombatLog)-1]
		if strings.Contains(lastLog, " Miss!") || !strings.Contains(lastLog, " Hit! ") {
			attackMissedWithShield = true
		}
		if !attackMissedWithShield {
			t.Errorf("Player HP changed unexpectedly after accepting Shield (HP:%d, Initial:%d). Check if Shield correctly caused a miss vs AC %d.", g.Player.HP, initialHP, g.Player.AC+5)
		} else {
			t.Logf("Player HP changed (HP:%d, Initial:%d) despite Shield - potential issue or high damage roll?", g.Player.HP, initialHP)
		}
	}
	if g.Player.SpellSlotsL1 != initialSlots-1 {
		t.Errorf("Player spell slots incorrect after accepting Shield, Slots: %d, Expected: %d", g.Player.SpellSlotsL1, initialSlots-1)
	}
	if !g.Player.UsedReaction {
		t.Error("Player UsedReaction should be true after accepting Shield")
	}
	if g.Player.ACBonusUntilNextTurn != 5 {
		t.Errorf("Player AC Bonus not set correctly after Shield, got %d", g.Player.ACBonusUntilNextTurn)
	}
	if g.Player.X != targetPos.X || g.Player.Y != targetPos.Y {
		t.Errorf("Player position incorrect after accepting Shield, Pos: (%d,%d), Target: (%d,%d)", g.Player.X, g.Player.Y, targetPos.X, targetPos.Y)
	}
	if g.Player.MovementPoints != 0 {
		t.Errorf("Player movement points not decremented, got %d", g.Player.MovementPoints)
	}
}

func TestMoveAoOReactionNoSlots(t *testing.T) {
	rand.Seed(1)
	startPos := image.Point{X: 5, Y: 5}
	targetPos := image.Point{X: 5, Y: 6}
	enemyInfo := []EnemySpawnInfo{{TypeName: "Melee Skeleton"}}
	g := createTestPlayerWithClass(1, "Mage", nil, nil, enemyInfo, startPos)
	if len(g.Enemies) == 0 {
		t.Fatal("Test setup error: Enemy did not spawn")
	}
	enemy := g.Enemies[0]
	enemy.X = startPos.X + 1
	enemy.Y = startPos.Y
	enemy.Strength = 18
	g.Player.AC = 5
	g.Player.SpellSlotsL1 = 0

	initialHP := g.Player.HP
	g.Player.MovementPoints = 1

	killed, hit := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")

	if !killed {
		g.Player.X = targetPos.X
		g.Player.Y = targetPos.Y
		g.Player.MovementPoints--
	} else {
		g.Player.MovementPoints--
	}

	if g.InputMode == InputModeReactionPrompt {
		t.Errorf("InputMode should not be ReactionPrompt when player has no slots, got %v", g.InputMode)
	}
	if !hit {
		t.Logf("Warning: Seeded AoO roll missed AC; cannot fully verify reaction trigger prevention logic. HP check might be inaccurate.")
	}
	if hit && g.Player.HP >= initialHP {
		t.Errorf("Player HP should have decreased from AoO with no slots, HP: %d, Initial: %d", g.Player.HP, initialHP)
	}
	if g.Player.UsedReaction {
		t.Error("Player UsedReaction should be false with no slots")
	}
	if !killed && (g.Player.X != targetPos.X || g.Player.Y != targetPos.Y) {
		t.Errorf("Player position incorrect with no slots, Pos: (%d,%d), Target: (%d,%d)", g.Player.X, g.Player.Y, targetPos.X, targetPos.Y)
	}
	if g.Player.MovementPoints != 0 {
		t.Errorf("Player movement points not decremented, got %d", g.Player.MovementPoints)
	}
}

func TestMoveAoONonMage(t *testing.T) {
	rand.Seed(1)
	startPos := image.Point{X: 5, Y: 5}
	targetPos := image.Point{X: 5, Y: 6}
	enemyInfo := []EnemySpawnInfo{{TypeName: "Melee Skeleton"}}
	g := createTestPlayerWithClass(1, "Fighter", nil, nil, enemyInfo, startPos)
	if len(g.Enemies) == 0 {
		t.Fatal("Test setup error: Enemy did not spawn")
	}
	enemy := g.Enemies[0]
	enemy.X = startPos.X + 1
	enemy.Y = startPos.Y
	enemy.Strength = 18
	g.Player.AC = 5

	initialHP := g.Player.HP
	g.Player.MovementPoints = 1

	killed, hit := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")

	if !killed {
		g.Player.X = targetPos.X
		g.Player.Y = targetPos.Y
		g.Player.MovementPoints--
	} else {
		g.Player.MovementPoints--
	}

	if g.InputMode == InputModeReactionPrompt {
		t.Errorf("InputMode should not be ReactionPrompt for Fighter, got %v", g.InputMode)
	}
	if !hit {
		t.Logf("Warning: Seeded AoO roll missed AC; cannot fully verify reaction trigger prevention logic. HP check might be inaccurate.")
	}
	if hit && g.Player.HP >= initialHP {
		t.Errorf("Fighter HP should have decreased from AoO, HP: %d, Initial: %d", g.Player.HP, initialHP)
	}
	if g.Player.UsedReaction {
		t.Error("Fighter UsedReaction should be false")
	}
	if !killed && (g.Player.X != targetPos.X || g.Player.Y != targetPos.Y) {
		t.Errorf("Fighter position incorrect, Pos: (%d,%d), Target: (%d,%d)", g.Player.X, g.Player.Y, targetPos.X, targetPos.Y)
	}
	if g.Player.MovementPoints != 0 {
		t.Errorf("Player movement points not decremented, got %d", g.Player.MovementPoints)
	}
}

func TestPlayerDeathGameOverState(t *testing.T) {
	initialPos := image.Point{X: 5, Y: 5}
	enemies := []EnemySpawnInfo{{TypeName: "Goblin Scout"}}
	g := createTestPlayerWithClass(1, "Fighter", nil, nil, enemies, initialPos)
	if len(g.Enemies) == 0 {
		t.Fatal("Test setup error: Enemy did not spawn")
	}
	enemy := g.Enemies[0]
	enemy.X = g.Player.X + 1
	enemy.Y = g.Player.Y

	g.Player.HP = 1
	g.Player.AC = 0
	g.CurrentTurn = EnemyTurn
	enemy.ActionAvailable = true
	enemy.MovementPoints = enemy.MaxMovementPoints

	killed, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")

	if !killed {
		t.Fatalf("Enemy attack did not kill player with 1 HP and 0 AC. Check attack/damage logic.")
	}
	if !g.Player.IsDying {
		t.Errorf("Player.IsDying was not set to true after lethal damage.")
	}
	if g.CurrentGameState != StateGameOverScreen {
		t.Errorf("Game.CurrentGameState was not set to StateGameOverScreen immediately after lethal damage. Got: %v", g.CurrentGameState)
	}
	if g.CurrentTurn != GameOver {
		t.Errorf("Game.CurrentTurn was not set to GameOver immediately after lethal damage. Got: %v", g.CurrentTurn)
	}

	g.endEnemyTurn()

	if g.Player.IsDying == false {
		t.Errorf("Player.IsDying became false after endEnemyTurn.")
	}
	if g.CurrentGameState != StateGameOverScreen {
		t.Errorf("Game.CurrentGameState was not StateGameOverScreen after endEnemyTurn. Got: %v", g.CurrentGameState)
	}
	if g.CurrentTurn != GameOver {
		t.Errorf("Game.CurrentTurn was not GameOver after endEnemyTurn. Got: %v", g.CurrentTurn)
	}
}
