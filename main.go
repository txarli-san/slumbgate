package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // Import for loading PNG files
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

const (
	screenWidth         = 640
	screenHeight        = 480
	tileSize            = 32
	spriteSize          = 32
	spriteScale         = float64(tileSize) / float64(spriteSize)
	mapWidth            = 10
	mapHeight           = 10
	combatLogLength     = 7
	playerBaseMovement  = 5
	enemyBaseMovement   = 4
	playerRangedRange   = 5
	hpBarHeight         = 4
	hpBarOffsetY        = 2
	sheetWidthInSprites = 32
)

type TurnState int

const (
	PlayerTurn TurnState = iota
	EnemyTurn
	// RestPhase // Future state
	GameOver
)

func (ts TurnState) String() string {
	switch ts {
	case PlayerTurn:
		return "Player Turn"
	case EnemyTurn:
		return "Enemy Turn"
	// case RestPhase:
	// 	return "Rest Phase"
	case GameOver:
		return "Game Over"
	default:
		return "Unknown"
	}
}

type InputMode int

const (
	InputModeMap InputMode = iota
	InputModeActionSelect
	InputModeCharacterSheet
	// InputModeRest // Future mode
)

type Entity struct {
	X            int
	Y            int
	Width        int // Width in tiles
	Height       int // Height in tiles
	HP           int
	MaxHP        int
	AC           int
	Strength     int
	Dexterity    int
	Constitution int
	Intelligence int
	Wisdom       int
	Charisma     int
	Sprite       *ebiten.Image
	DrawOpts     ebiten.DrawImageOptions
	Name         string
}

type Player struct {
	Entity
	ProficiencyBonus  int
	MovementPoints    int
	MaxMovementPoints int
	ActionTaken       bool
	IsDisengaging     bool
}

type Enemy struct {
	Entity
	MovementPoints    int
	MaxMovementPoints int
	ActionAvailable   bool
	// TODO: Add AttackType string ("melee", "ranged") and MaxRange int later
}

type Game struct {
	Player               *Player
	Enemies              []*Enemy
	GameMap              [mapWidth][mapHeight]int
	TileImage            *ebiten.Image
	CurrentTurn          TurnState
	CurrentWave          int // Tracks wave number
	CombatLog            []string
	MapOffsetX           int
	MapOffsetY           int
	RangeOverlayTile     *ebiten.Image
	InputMode            InputMode
	rogueSheet           *ebiten.Image
	monsterSheet         *ebiten.Image
	availableActions     []*ActionDefinition
	selectedActionIndex  int
	primedActionID       string
	lastExecutedActionID string
}

type ActionExecuteFunc func(g *Game, targetX, targetY int) bool

type TargetType string

const (
	TargetSelf          TargetType = "self"
	TargetEnemyAdjacent TargetType = "enemy_adjacent"
	TargetEnemyRange    TargetType = "enemy_range"
	TargetEmptyTile     TargetType = "empty_tile"
	TargetNone          TargetType = "none"
)

type ActionDefinition struct {
	ID             string
	Name           string
	Targeting      TargetType
	Range          int
	RequiresTarget bool
	Execute        ActionExecuteFunc
}

// --- Action Table Definition ---
var ActionTable = map[string]*ActionDefinition{
	"melee_attack": {
		ID:             "melee_attack",
		Name:           "Melee Attack",
		Targeting:      TargetEnemyAdjacent,
		Range:          1,
		RequiresTarget: true,
		Execute:        executeMeleeAttack,
	},
	"ranged_attack": {
		ID:             "ranged_attack",
		Name:           "Ranged Attack",
		Targeting:      TargetEnemyRange,
		Range:          playerRangedRange,
		RequiresTarget: true,
		Execute:        executeRangedAttack,
	},
	"dash": {
		ID:             "dash",
		Name:           "Dash",
		Targeting:      TargetSelf,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeDash,
	},
	"disengage": {
		ID:             "disengage",
		Name:           "Disengage",
		Targeting:      TargetSelf,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeDisengage,
	},
	"wait": {
		ID:             "wait",
		Name:           "Wait (End Turn)",
		Targeting:      TargetNone,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeWait,
	},
}

// --- Initialization Check ---
func init() {
	if len(ActionTable) == 0 {
		fmt.Println("DEBUG [init]: WARNING - ActionTable is empty immediately after declaration!")
	}
}

// --- End Initialization Check ---

func loadImage(path string) (*ebiten.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image %s: %w", path, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", path, err)
	}
	return ebiten.NewImageFromImage(img), nil
}

// (sx=col, sy=row).
func getSpriteFromSheet(sheet *ebiten.Image, sx, sy int) *ebiten.Image {
	if sheet == nil {
		log.Println("Warning: Sprite sheet not loaded.")
		img := ebiten.NewImage(spriteSize, spriteSize)
		img.Fill(color.White)
		return img
	}
	x := sx * spriteSize
	y := sy * spriteSize
	rect := image.Rect(x, y, x+spriteSize, y+spriteSize)
	bounds := sheet.Bounds()
	if !rect.In(bounds) {
		log.Printf("Warning: Sprite index (%d, %d) out of sheet bounds (%v).", sx, sy, bounds)
		img := ebiten.NewImage(spriteSize, spriteSize)
		img.Fill(color.White)
		return img
	}
	return sheet.SubImage(rect).(*ebiten.Image)
}

func getModifier(score int) int { return (score - 10) / 2 }

func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)
	g.MapOffsetX = (screenWidth - (mapWidth * tileSize)) / 2
	g.MapOffsetY = (screenHeight - (mapHeight * tileSize)) / 2
	g.InputMode = InputModeMap
	g.availableActions = make([]*ActionDefinition, 0)
	g.lastExecutedActionID = ""
	g.CurrentWave = 0 // Start at 0, SpawnNextWave increments before spawning

	var err error
	rogueSheetPath := "assets/rogues.png"
	monsterSheetPath := "assets/monsters.png"
	tilePath := "assets/tiles.png"

	g.rogueSheet, err = loadImage(rogueSheetPath)
	if err != nil {
		log.Printf("Error loading rogue sheet: %v. Using fallback.", err)
		g.rogueSheet = ebiten.NewImage(spriteSize*sheetWidthInSprites, spriteSize*10)
		g.rogueSheet.Fill(color.Gray{Y: 50})
	}

	g.monsterSheet, err = loadImage(monsterSheetPath)
	if err != nil {
		log.Printf("Error loading monster sheet: %v. Using fallback.", err)
		g.monsterSheet = ebiten.NewImage(spriteSize*sheetWidthInSprites, spriteSize*15)
		g.monsterSheet.Fill(color.Gray{Y: 100})
	}

	tileSheet, err := loadImage(tilePath)
	if err != nil {
		log.Printf("Error loading tile sheet: %v. Creating fallback tile.", err)
		g.TileImage = ebiten.NewImage(tileSize, tileSize)
		vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)
		vector.StrokeRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), 1, color.RGBA{R: 80, G: 80, B: 80, A: 255}, false)
	} else {
		floorSprite := getSpriteFromSheet(tileSheet, 0, 0)
		g.TileImage = floorSprite // 32x32 tile sprite
	}

	g.RangeOverlayTile = ebiten.NewImage(tileSize, tileSize)
	overlayColor := color.NRGBA{R: 0, G: 100, B: 200, A: 80}
	vector.DrawFilledRect(g.RangeOverlayTile, 0, 0, float32(tileSize), float32(tileSize), overlayColor, false)

	playerSprite := getSpriteFromSheet(g.rogueSheet, 1, 1) // Male fighter
	playerStr, playerDex, playerCon := 15, 14, 13
	playerInt, playerWis, playerCha := 8, 12, 10
	playerConMod := getModifier(playerCon)
	playerMaxHP := 10 + playerConMod
	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2, Width: 1, Height: 1, // Player is 1x1
			HP: playerMaxHP, MaxHP: playerMaxHP, AC: 13,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Sprite: playerSprite, Name: "Player",
		},
		ProficiencyBonus:  2,
		MaxMovementPoints: playerBaseMovement,
	}

	g.CurrentTurn = PlayerTurn
	g.SpawnNextWave()
	g.startPlayerTurn()
	return g
}

func (g *Game) startPlayerTurn() {
	g.CurrentTurn = PlayerTurn
	g.Player.MovementPoints = g.Player.MaxMovementPoints
	g.Player.ActionTaken = false
	g.Player.IsDisengaging = false
	g.InputMode = InputModeMap
	g.primedActionID = ""
}

func (g *Game) endPlayerTurn() {
	g.Player.ActionTaken = true
	g.cleanupDeadEnemies()

	if len(g.Enemies) == 0 {
		g.SpawnNextWave()
		if len(g.Enemies) > 0 {
			g.startEnemyTurn()
		} else if g.CurrentTurn != GameOver {
			g.addCombatLog("All defined waves cleared! VICTORY!")
			g.CurrentTurn = GameOver
		}
	} else {
		g.startEnemyTurn()
	}

	g.primedActionID = ""
	if g.CurrentTurn != GameOver {
		g.InputMode = InputModeMap
	}
}

func (g *Game) startEnemyTurn() {
	g.CurrentTurn = EnemyTurn
}

func (g *Game) endEnemyTurn() {
	g.cleanupDeadEnemies()

	if g.Player.HP <= 0 {
		if g.CurrentTurn != GameOver {
			g.addCombatLog("Player has died! Game Over.")
			g.CurrentTurn = GameOver
		}
		return
	}

	if len(g.Enemies) == 0 {
		g.SpawnNextWave()
		if len(g.Enemies) > 0 {
			g.startPlayerTurn()
		} else if g.CurrentTurn != GameOver {
			g.addCombatLog("All defined waves cleared! VICTORY!")
			g.CurrentTurn = GameOver
		}
	} else {
		g.startPlayerTurn()
	}
}

// SpawnNextWave handles wave progression and enemy spawning.
func (g *Game) SpawnNextWave() {
	g.CurrentWave++
	g.addCombatLog(fmt.Sprintf("--- Starting Wave %d ---", g.CurrentWave))
	g.Enemies = make([]*Enemy, 0)

	spawnPoints := []image.Point{
		{X: 2, Y: 2},
		{X: mapWidth - 3, Y: mapHeight - 3},
		{X: 2, Y: mapHeight - 3},
		{X: mapWidth - 3, Y: 2},
		{X: mapWidth / 2, Y: 1},
	}
	spawnIndex := 0

	smallSlimeSprite := getSpriteFromSheet(g.monsterSheet, 0, 2) // 3.a
	bigSlimeSprite := getSpriteFromSheet(g.monsterSheet, 1, 2)   // 3.b

	switch g.CurrentWave {
	case 1:
		if spawnIndex < len(spawnPoints) {
			g.spawnEnemy(spawnPoints[spawnIndex].X, spawnPoints[spawnIndex].Y, "Slime A", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement, 1, 1, smallSlimeSprite)
			spawnIndex++
		}
		if spawnIndex < len(spawnPoints) {
			g.spawnEnemy(spawnPoints[spawnIndex].X, spawnPoints[spawnIndex].Y, "Slime B", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement, 1, 1, smallSlimeSprite)
			spawnIndex++
		}
	case 2:
		if spawnIndex < len(spawnPoints) {
			g.spawnEnemy(spawnPoints[spawnIndex].X, spawnPoints[spawnIndex].Y, "Slime C", 7, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement, 1, 1, smallSlimeSprite)
			spawnIndex++
		}
		if spawnIndex < len(spawnPoints) {
			g.spawnEnemy(spawnPoints[spawnIndex].X, spawnPoints[spawnIndex].Y, "Slime D", 7, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement, 1, 1, smallSlimeSprite)
			spawnIndex++
		}
	case 3:
		bossX, bossY := mapWidth/2-1, 1
		if bossX+1 >= mapWidth {
			bossX = mapWidth - 2
		}
		if bossY+1 >= mapHeight {
			bossY = mapHeight - 2
		}
		if bossX < 0 {
			bossX = 0
		}
		if bossY < 0 {
			bossY = 0
		}
		g.spawnEnemy(bossX, bossY, "Big Slime Boss", 25, 12, 14, 8, 15, 6, 6, 6, enemyBaseMovement-1, 2, 2, bigSlimeSprite)

	default:
		g.addCombatLog(fmt.Sprintf("No definition for wave %d. Ending.", g.CurrentWave))
	}
}

// spawnEnemy includes width and height parameters.
// TODO: Add attackType and maxRange parameters later.
func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move, w, h int, sprite *ebiten.Image) {
	conMod := getModifier(con)
	maxHp := baseHp + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, Width: w, Height: h, // Assign size
			HP: maxHp, MaxHP: maxHp, AC: ac,
			Strength: str, Dexterity: dex, Constitution: con,
			Intelligence: intel, Wisdom: wis, Charisma: cha,
			Sprite: sprite, Name: name,
		},
		MaxMovementPoints: move,
		// AttackType: attackType, // Add later
		// MaxRange: maxRange,     // Add later
	}
	g.Enemies = append(g.Enemies, enemy)
}

func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
}

// isTileBlocked checks all tiles occupied by entities.
func (g *Game) isTileBlocked(checkX, checkY, movingEnemyIndex int) bool {
	// Check player collision (player is 1x1)
	if g.Player.HP > 0 && g.Player.X == checkX && g.Player.Y == checkY {
		return true
	}
	// Check enemy collision
	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 { // Ignore dead enemies
			continue
		}
		if i == movingEnemyIndex { // Ignore the enemy that is currently moving
			continue
		}
		// Check if the target tile falls within the bounds of this enemy
		if checkX >= enemy.X && checkX < enemy.X+enemy.Width &&
			checkY >= enemy.Y && checkY < enemy.Y+enemy.Height {
			return true
		}
	}
	return false
}

// isTileFullyBlocked checks if the entire area an entity would occupy starting at (checkX, checkY) is blocked.
// Used for checking multi-tile entity movement validity.
func (g *Game) isTileFullyBlocked(checkX, checkY, entityWidth, entityHeight, movingEnemyIndex int) bool {
	for w := 0; w < entityWidth; w++ {
		for h := 0; h < entityHeight; h++ {
			tileX, tileY := checkX+w, checkY+h
			// Check map bounds first
			if tileX < 0 || tileX >= mapWidth || tileY < 0 || tileY >= mapHeight {
				return true // Part of the entity would be off map
			}
			// Check if this specific tile is blocked by player or another enemy
			if g.isTileBlocked(tileX, tileY, movingEnemyIndex) {
				return true // One of the tiles is blocked
			}
		}
	}
	return false // None of the tiles are blocked
}

// getEnemyAt returns the enemy occupying the tile (x, y).
func (g *Game) getEnemyAt(x, y int) *Enemy {
	for _, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if x >= enemy.X && x < enemy.X+enemy.Width &&
			y >= enemy.Y && y < enemy.Y+enemy.Height {
			return enemy
		}
	}
	return nil
}

func distance(x1, y1, x2, y2 int) int {
	return int(math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2)))
}

// isAdjacentSimple checks Manhattan distance == 1 between two points.
func isAdjacentSimple(x1, y1, x2, y2 int) bool {
	return distance(x1, y1, x2, y2) == 1
}

// isAdjacentToEntity checks if point (px, py) is adjacent to any tile of the entity.
func isAdjacentToEntity(px, py int, entity *Entity) bool {
	if entity == nil {
		return false
	}
	for ex := entity.X; ex < entity.X+entity.Width; ex++ {
		for ey := entity.Y; ey < entity.Y+entity.Height; ey++ {
			if isAdjacentSimple(px, py, ex, ey) {
				return true
			}
		}
	}
	return false
}

// isPlayerAdjacentToEnemy checks if player is adjacent to any part of any living enemy.
func isPlayerAdjacentToEnemy(g *Game) bool {
	for _, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity) {
			return true
		}
	}
	return false
}

func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int, attackType string) bool {
	if attacker.HP <= 0 || defender.HP <= 0 {
		return false
	}
	var attackAbilityMod int
	var abilityName string
	switch attackType {
	case "ranged":
		attackAbilityMod = getModifier(attacker.Dexterity)
		abilityName = "DEX"
	default: // Melee
		attackAbilityMod = getModifier(attacker.Strength)
		abilityName = "STR"
	}
	roll := rand.Intn(20) + 1
	attackRoll := roll + attackerProfBonus + attackAbilityMod
	hit := attackRoll >= defender.AC
	modString := fmt.Sprintf("%+d", attackAbilityMod)
	profString := ""
	if attackerProfBonus != 0 {
		profString = fmt.Sprintf("+%d", attackerProfBonus)
	}
	rollString := fmt.Sprintf("Roll: %d%s%s(%s) = %d", roll, profString, modString, abilityName, attackRoll)
	attackVerb := "attacks"
	if attackType == "ranged" {
		attackVerb = "shoots"
	}
	logMsg := fmt.Sprintf("%s %s %s (AC %d). %s.", attacker.Name, attackVerb, defender.Name, defender.AC, rollString)
	if hit {
		// TODO: Implement weapon damage dice (e.g., 1d6 + mod)
		damage := max(1, attackAbilityMod) // Simple damage: ability mod (min 1)
		defender.HP -= damage
		logMsg += fmt.Sprintf(" Hit! Deals %d damage.", damage)
		if defender.HP <= 0 {
			logMsg += fmt.Sprintf(" %s dies!", defender.Name)
			g.addCombatLog(logMsg)
			return true
		}
	} else {
		logMsg += " Miss!"
	}
	g.addCombatLog(logMsg)
	return false
}

func executeMeleeAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil && isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		killed := g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "melee")
		g.lastExecutedActionID = "melee_attack"
		if killed {
			g.cleanupDeadEnemies()
		}
		return true
	}
	g.addCombatLog("Invalid target for melee attack (or not adjacent).")
	return false
}

func executeRangedAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil {
		// Use top-left corner for distance check (simplification)
		dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
		if dist <= playerRangedRange {
			// TODO: Add check for line of sight?
			// TODO: Disadvantage if adjacent to an enemy?
			killed := g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "ranged")
			g.lastExecutedActionID = "ranged_attack"
			if killed {
				g.cleanupDeadEnemies()
			}
			return true
		}
		g.addCombatLog(fmt.Sprintf("Target %s out of range.", targetEnemy.Name))
		return false
	}
	g.addCombatLog("No target selected at cursor.")
	return false
}

func executeDash(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Dash!")
	g.Player.MovementPoints += g.Player.MaxMovementPoints
	g.lastExecutedActionID = "dash"
	return true
}

func executeDisengage(g *Game, targetX, targetY int) bool {
	if !isPlayerAdjacentToEnemy(g) {
		g.addCombatLog("Cannot Disengage when not adjacent.")
		return false
	}
	g.addCombatLog("Player uses Disengage!")
	g.Player.IsDisengaging = true
	g.lastExecutedActionID = "disengage"
	return true
}

func executeWait(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player ends turn (Wait).")
	g.lastExecutedActionID = "wait"
	g.endPlayerTurn()
	return true
}

func (g *Game) handlePlayerInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if g.InputMode == InputModeCharacterSheet {
			g.InputMode = InputModeMap
		} else if g.InputMode == InputModeMap || g.InputMode == InputModeActionSelect {
			g.InputMode = InputModeCharacterSheet
		}
		return
	}

	switch g.InputMode {
	case InputModeCharacterSheet:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyC) {
			g.InputMode = InputModeMap
		}
		return

	case InputModeActionSelect:
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.selectedActionIndex--
			if g.selectedActionIndex < 0 {
				g.selectedActionIndex = len(g.availableActions) - 1
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.selectedActionIndex++
			if g.selectedActionIndex >= len(g.availableActions) {
				g.selectedActionIndex = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if g.selectedActionIndex >= 0 && g.selectedActionIndex < len(g.availableActions) {
				selectedActionDef := g.availableActions[g.selectedActionIndex]
				if !selectedActionDef.RequiresTarget {
					success := selectedActionDef.Execute(g, -1, -1)
					if success && selectedActionDef.ID != "wait" {
						g.Player.ActionTaken = true
					}
					if selectedActionDef.ID != "wait" && g.CurrentTurn != GameOver {
						g.InputMode = InputModeMap
						g.primedActionID = ""
					}
				} else {
					g.primedActionID = selectedActionDef.ID
					g.InputMode = InputModeMap
				}
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.InputMode = InputModeMap
			g.primedActionID = ""
		}
		return

	case InputModeMap:
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			if !g.Player.ActionTaken {
				g.availableActions = []*ActionDefinition{}

				for _, actionDef := range ActionTable {
					isAvailable := true
					if actionDef.ID == "disengage" {
						if !isPlayerAdjacentToEnemy(g) {
							isAvailable = false
						}
					}
					// Add other availability checks here...

					if isAvailable {
						g.availableActions = append(g.availableActions, actionDef)
					}
				}

				if len(g.availableActions) == 0 {
					g.addCombatLog("No actions available!")
				} else {
					g.selectedActionIndex = 0
					if g.lastExecutedActionID != "" {
						for i, actionDef := range g.availableActions {
							if actionDef.ID == g.lastExecutedActionID {
								g.selectedActionIndex = i
								break
							}
						}
					}
					g.InputMode = InputModeActionSelect
					g.primedActionID = ""
					return
				}
			} else {
				g.addCombatLog("Action already taken this turn.")
			}
		}

		actionExecutedByClick := false
		if g.primedActionID != "" && !g.Player.ActionTaken {
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				cursorX, cursorY := ebiten.CursorPosition()
				gridX := (cursorX - g.MapOffsetX) / tileSize
				gridY := (cursorY - g.MapOffsetY) / tileSize

				if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight {
					actionDef, exists := ActionTable[g.primedActionID]
					if exists {
						success := actionDef.Execute(g, gridX, gridY)
						if success {
							g.Player.ActionTaken = true
							actionExecutedByClick = true
						}
					} else {
						g.addCombatLog(fmt.Sprintf("Error: Unknown primed action ID '%s'.", g.primedActionID))
					}
					g.primedActionID = ""
				} else {
					g.addCombatLog("Clicked outside map.")
					g.primedActionID = ""
				}
			} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				g.addCombatLog("Targeting cancelled.")
				g.primedActionID = ""
			}
		}

		if actionExecutedByClick {
			if g.CurrentTurn == GameOver {
				return
			}
			return
		}

		if g.primedActionID == "" && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			waitAction, exists := ActionTable["wait"]
			if exists {
				waitAction.Execute(g, -1, -1)
			} else {
				g.addCombatLog("Player ends turn (Fallback).")
				g.endPlayerTurn()
			}
			return
		}

		if g.Player.MovementPoints > 0 && g.primedActionID == "" {
			moved := false
			startX, startY := g.Player.X, g.Player.Y
			targetX, targetY := startX, startY

			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				targetY--
				moved = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				targetY++
				moved = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				targetX--
				moved = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				targetX++
				moved = true
			}

			if moved {
				if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
					// Player is 1x1, so use simpler isTileBlocked check
					if !g.isTileBlocked(targetX, targetY, -1) {
						performAoOCheck := !g.Player.IsDisengaging
						if performAoOCheck {
							for _, enemy := range g.Enemies {
								if enemy.HP <= 0 {
									continue
								}
								wasAdj := isAdjacentToEntity(startX, startY, &enemy.Entity)
								isStillAdj := isAdjacentToEntity(targetX, targetY, &enemy.Entity) // Check potential new position

								if wasAdj && !isStillAdj {
									if g.Player.HP > 0 {
										g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
										killedByAoO := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
										if killedByAoO {
											g.Player.MovementPoints-- // Consume move point even if killed
											return                    // Stop processing input if player died
										}
									}
								}
							}
						}

						// If player survived AoO (or none occurred)
						if g.Player.HP > 0 {
							g.Player.X = targetX
							g.Player.Y = targetY
							g.Player.MovementPoints--
							// Reset Disengage after moving? Or at start of turn?
							// g.Player.IsDisengaging = false
						}
					} else {
						g.addCombatLog("Movement blocked.")
					}
				} else {
					g.addCombatLog("Cannot move outside map.")
				}
			}
		}
	}
}

func (g *Game) handleEnemyTurns() {
	if g.Player.HP <= 0 {
		return // Skip enemy turns if player is dead
	}

	// Define potential move offsets (Up, Down, Left, Right)
	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}

	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue // Skip dead enemies
		}

		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true

		// --- AI Logic ---
		// TODO: Add logic for Ranged Kiting here later

		// --- Movement Phase ---
		isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
		if !isAdj {
			for enemy.MovementPoints > 0 {
				currentX, currentY := enemy.X, enemy.Y
				currentDist := distance(currentX, currentY, g.Player.X, g.Player.Y)
				bestMoveX, bestMoveY := currentX, currentY // Start assuming no move is best
				minDist := currentDist                     // Minimum distance found so far

				potentialMoves := []image.Point{} // Store all valid potential moves
				bestMoves := []image.Point{}      // Store moves that achieve the minimum distance

				// 1. Evaluate all potential moves and find the minimum possible distance
				for _, offset := range moveOffsets {
					nextX, nextY := currentX+offset.X, currentY+offset.Y

					// Check if the potential move is valid
					if !g.isTileFullyBlocked(nextX, nextY, enemy.Width, enemy.Height, i) {
						potentialMoves = append(potentialMoves, image.Point{X: nextX, Y: nextY}) // Add to list of valid moves
						distToPlayer := distance(nextX, nextY, g.Player.X, g.Player.Y)

						if distToPlayer < minDist {
							minDist = distToPlayer // Found a new closer distance
						}
					}
				} // End checking potential moves

				// 2. Collect all moves that achieve the minimum distance
				for _, move := range potentialMoves {
					if distance(move.X, move.Y, g.Player.X, g.Player.Y) == minDist {
						bestMoves = append(bestMoves, move)
					}
				}

				// 3. Choose the best move from the bestMoves list
				chosenMove := false
				if len(bestMoves) > 0 {
					if len(bestMoves) == 1 {
						// Only one best move, take it
						bestMoveX = bestMoves[0].X
						bestMoveY = bestMoves[0].Y
						chosenMove = true
					} else {
						// Tie-breaker: prioritize move along the axis with greater distance to player
						dx := g.Player.X - currentX
						dy := g.Player.Y - currentY
						preferredMoveFound := false

						if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
							// Prefer horizontal movement
							for _, move := range bestMoves {
								if move.X != currentX { // Check if it's a horizontal move
									bestMoveX = move.X
									bestMoveY = move.Y
									chosenMove = true
									preferredMoveFound = true
									break
								}
							}
						} else {
							// Prefer vertical movement (includes diagonal tie where abs(dx)==abs(dy))
							for _, move := range bestMoves {
								if move.Y != currentY { // Check if it's a vertical move
									bestMoveX = move.X
									bestMoveY = move.Y
									chosenMove = true
									preferredMoveFound = true
									break
								}
							}
						}

						// If no preferred move was found among the best, just take the first one
						if !preferredMoveFound {
							bestMoveX = bestMoves[0].X
							bestMoveY = bestMoves[0].Y
							chosenMove = true
						}
					}
				}

				// 4. Execute the chosen move (if any)
				if chosenMove {
					enemy.X = bestMoveX
					enemy.Y = bestMoveY
					enemy.MovementPoints--
					// Re-check adjacency after moving
					isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
					if isAdj {
						break // Stop moving if now adjacent
					}
				} else {
					// No valid move found (or all valid moves increase distance), stop trying to move
					break
				}

			} // End of movement loop (while movement points > 0)
		} // End of movement phase (if !isAdj)

		// --- Action Phase ---
		// TODO: Modify this later for Ranged attacks based on MaxRange
		if isAdj && enemy.ActionAvailable {
			if g.Player.HP > 0 {
				g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee") // Assuming melee for now
				enemy.ActionAvailable = false
			} else {
				enemy.ActionAvailable = false // Player already dead
			}
		} else if enemy.ActionAvailable {
			// If not adjacent or couldn't attack, action is still used up (effectively 'Wait')
			enemy.ActionAvailable = false
		}
		// --- End Action Phase ---

		// Check if player died during this enemy's turn
		if g.Player.HP <= 0 {
			return // Stop processing further enemies if player died
		}
	} // End of enemy loop
}

func (g *Game) cleanupDeadEnemies() {
	initialCount := len(g.Enemies)
	aliveEnemies := make([]*Enemy, 0, len(g.Enemies))
	for _, enemy := range g.Enemies {
		if enemy.HP > 0 {
			aliveEnemies = append(aliveEnemies, enemy)
		}
	}
	if len(aliveEnemies) != initialCount {
		g.Enemies = aliveEnemies
	}
}

func (g *Game) Update() error {
	if g.CurrentTurn == GameOver {
		// TODO: Handle restart input?
		return nil
	}

	turnBeforeInput := g.CurrentTurn

	if turnBeforeInput == PlayerTurn || g.InputMode == InputModeActionSelect || g.InputMode == InputModeCharacterSheet {
		g.handlePlayerInput()
	}

	// Only process enemy turn if the state *is* EnemyTurn after player input might have changed it.
	if g.CurrentTurn == EnemyTurn {
		g.handleEnemyTurns()
		g.endEnemyTurn() // Handles state transitions (Enemy->Player or Enemy->GameOver)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	mapOffsetX, mapOffsetY := g.MapOffsetX, g.MapOffsetY
	tileOpts := &ebiten.DrawImageOptions{}
	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			screenX := float64(mapOffsetX + x*tileSize)
			screenY := float64(mapOffsetY + y*tileSize)
			tileOpts.GeoM.Reset()
			tileOpts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(g.TileImage, tileOpts)
		}
	}

	if g.CurrentTurn == PlayerTurn && g.primedActionID != "" {
		actionDef, exists := ActionTable[g.primedActionID]
		if exists && actionDef.RequiresTarget && actionDef.Range > 0 {
			overlayOpts := &ebiten.DrawImageOptions{}
			for x := 0; x < mapWidth; x++ {
				for y := 0; y < mapHeight; y++ {
					dist := distance(g.Player.X, g.Player.Y, x, y)
					if dist > 0 && dist <= actionDef.Range {
						// TODO: Add line-of-sight check here?
						screenX := float64(mapOffsetX + x*tileSize)
						screenY := float64(mapOffsetY + y*tileSize)
						overlayOpts.GeoM.Reset()
						overlayOpts.GeoM.Translate(screenX, screenY)
						screen.DrawImage(g.RangeOverlayTile, overlayOpts)
					}
				}
			}
		}
	}

	entitiesToDraw := make([]*Entity, 0, len(g.Enemies)+1)
	for _, enemy := range g.Enemies {
		if enemy.HP > 0 {
			entitiesToDraw = append(entitiesToDraw, &enemy.Entity)
		}
	}
	if g.Player.HP > 0 {
		entitiesToDraw = append(entitiesToDraw, &g.Player.Entity)
	}

	// TODO: Sort entities by Y-coordinate for pseudo-3D layering?
	// sort.Slice(entitiesToDraw, func(i, j int) bool {
	//     return entitiesToDraw[i].Y < entitiesToDraw[j].Y
	// })

	entityOpts := &ebiten.DrawImageOptions{}

	for _, entity := range entitiesToDraw {
		if entity.Sprite == nil {
			continue
		}
		entityScreenX := float64(mapOffsetX + entity.X*tileSize)
		entityScreenY := float64(mapOffsetY + entity.Y*tileSize)

		entityOpts.GeoM.Reset()
		if entity.Width > 1 || entity.Height > 1 {
			// Scale sprite to cover the entity's tile area
			entityOpts.GeoM.Scale(float64(entity.Width), float64(entity.Height))
		}
		entityOpts.GeoM.Translate(entityScreenX, entityScreenY)
		screen.DrawImage(entity.Sprite, entityOpts)

		// Draw HP Bar
		hpBarBaseX := float64(mapOffsetX + entity.X*tileSize)
		hpBarBaseY := float64(mapOffsetY + entity.Y*tileSize)
		hpBarX := float32(hpBarBaseX)
		hpBarY := float32(hpBarBaseY + float64(entity.Height*tileSize) + hpBarOffsetY) // Position below entity
		hpBarWidth := float32(tileSize * entity.Width)                                 // Span entity width
		hpRatio := float32(entity.HP) / float32(entity.MaxHP)
		if hpRatio < 0 {
			hpRatio = 0
		}
		if hpRatio > 1 {
			hpRatio = 1
		}
		vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, color.RGBA{R: 80, G: 0, B: 0, A: 255}, false)          // Background
		vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth*hpRatio, hpBarHeight, color.RGBA{R: 0, G: 200, B: 0, A: 255}, false) // Foreground
		vector.StrokeRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, 1, color.Black, false)                                     // Border
	}

	// --- Draw UI ---
	uiStartY := 10
	uiLineHeight := 15
	statusStartY := 10

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Wave: %d", g.CurrentWave), screenWidth-100, statusStartY)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Turn: %s", g.CurrentTurn.String()), 10, uiStartY)
	playerHpVal := max(0, g.Player.HP)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP: %d/%d AC: %d", playerHpVal, g.Player.MaxHP, g.Player.AC), 10, uiStartY+uiLineHeight*1)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Move: %d/%d", g.Player.MovementPoints, g.Player.MaxMovementPoints), 10, uiStartY+uiLineHeight*2)

	actionStatusText := "Action: Available"
	if g.Player.ActionTaken {
		actionStatusText = "Action: Used"
	}
	ebitenutil.DebugPrintAt(screen, actionStatusText, 10, uiStartY+uiLineHeight*3)

	primedActionText := "Primed: None"
	if g.primedActionID != "" {
		if actionDef, exists := ActionTable[g.primedActionID]; exists {
			primedActionText = fmt.Sprintf("Primed: %s", actionDef.Name)
		} else {
			primedActionText = fmt.Sprintf("Primed: ??? (%s)", g.primedActionID)
		}
	}
	ebitenutil.DebugPrintAt(screen, primedActionText, 10, uiStartY+uiLineHeight*4)

	statusText := ""
	if g.Player.IsDisengaging {
		statusText = "Status: Disengaging"
	}
	// Add other statuses here
	if statusText != "" {
		ebitenutil.DebugPrintAt(screen, statusText, 10, uiStartY+uiLineHeight*5)
	}

	// Draw Action Selection Menu
	if g.InputMode == InputModeActionSelect {
		menuX, menuY := screenWidth/4, screenHeight/4
		menuW, menuH := screenWidth/2, screenHeight/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 20, B: 30, A: 220}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, color.White, false)

		title := "Select Action"
		titleX := menuX + 10
		titleY := menuY + 20
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, color.White)

		itemStartY := titleY + 25
		itemLineHeight := 18
		for i, actionDef := range g.availableActions {
			actionText := actionDef.Name
			var itemColor color.Color = color.Gray{Y: 180} // Default color

			if i == g.selectedActionIndex {
				actionText = "> " + actionText
				itemColor = color.White // Highlight selected
			}

			itemX := menuX + 15
			itemY := itemStartY + (i * itemLineHeight)
			text.Draw(screen, actionText, basicfont.Face7x13, itemX, itemY, itemColor)
		}

		closeMsg := "Press [Tab] or [Esc] to cancel"
		closeY := float32(menuY + menuH - 20)
		closeX := float32(menuX + (menuW-text.BoundString(basicfont.Face7x13, closeMsg).Dx())/2)
		text.Draw(screen, closeMsg, basicfont.Face7x13, int(closeX), int(closeY), color.Gray{Y: 150})
	}

	// Draw Character Sheet
	if g.InputMode == InputModeCharacterSheet {
		menuX, menuY := screenWidth/4, screenHeight/4
		menuW, menuH := screenWidth/2, screenHeight/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 30, G: 20, B: 20, A: 220}, false) // Slightly different color
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, color.White, false)

		title := fmt.Sprintf("%s - Character Sheet", g.Player.Name)
		titleX := menuX + 10
		titleY := menuY + 20
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, color.White)

		infoStartY := titleY + 25
		infoLineHeight := 15
		infoX := menuX + 15
		lineNum := 0

		// Stats
		healthStr := fmt.Sprintf("HP: %d / %d", max(0, g.Player.HP), g.Player.MaxHP)
		acStr := fmt.Sprintf("AC: %d", g.Player.AC)
		moveStr := fmt.Sprintf("Movement: %d", g.Player.MaxMovementPoints)
		profStr := fmt.Sprintf("Proficiency Bonus: +%d", g.Player.ProficiencyBonus)
		text.Draw(screen, healthStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, acStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, moveStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, profStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		lineNum++ // Gap

		// Attributes
		text.Draw(screen, "Attributes:", basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.Gray{Y: 200})
		lineNum++
		attrStr := fmt.Sprintf("  STR: %d (%+d)", g.Player.Strength, getModifier(g.Player.Strength))
		text.Draw(screen, attrStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		attrDex := fmt.Sprintf("  DEX: %d (%+d)", g.Player.Dexterity, getModifier(g.Player.Dexterity))
		text.Draw(screen, attrDex, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		attrCon := fmt.Sprintf("  CON: %d (%+d)", g.Player.Constitution, getModifier(g.Player.Constitution))
		text.Draw(screen, attrCon, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		attrInt := fmt.Sprintf("  INT: %d (%+d)", g.Player.Intelligence, getModifier(g.Player.Intelligence))
		text.Draw(screen, attrInt, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		attrWis := fmt.Sprintf("  WIS: %d (%+d)", g.Player.Wisdom, getModifier(g.Player.Wisdom))
		text.Draw(screen, attrWis, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		attrCha := fmt.Sprintf("  CHA: %d (%+d)", g.Player.Charisma, getModifier(g.Player.Charisma))
		text.Draw(screen, attrCha, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		lineNum++ // Gap

		// Close instruction
		closeMsg := "Press [C] or [Esc] to close"
		closeY := float32(menuY + menuH - 20)
		closeX := float32(menuX + (menuW-text.BoundString(basicfont.Face7x13, closeMsg).Dx())/2)
		text.Draw(screen, closeMsg, basicfont.Face7x13, int(closeX), int(closeY), color.Gray{Y: 150})
	}

	// Draw Combat Log
	logStartY := screenHeight - (combatLogLength * 15) - 10
	logX := 10
	for i, msg := range g.CombatLog {
		text.Draw(screen, msg, basicfont.Face7x13, logX, logStartY+(i*15), color.White)
	}

	// Draw Game Over / Victory Message
	if g.CurrentTurn == GameOver {
		gameOverMsg := "GAME OVER"
		if g.Player.HP > 0 && len(g.Enemies) == 0 {
			gameOverMsg = "VICTORY!"
		}

		msgFont := basicfont.Face7x13
		bounds := text.BoundString(msgFont, gameOverMsg)
		msgX := (screenWidth - bounds.Dx()) / 2
		msgY := (screenHeight - bounds.Dy()) / 2

		text.Draw(screen, gameOverMsg, msgFont, msgX+1, msgY+1, color.Black) // Shadow
		text.Draw(screen, gameOverMsg, msgFont, msgX, msgY, color.White)     // Text

		restartMsg := "Press [R] to Restart (Not Implemented)"
		restartBounds := text.BoundString(basicfont.Face7x13, restartMsg)
		restartX := (screenWidth - restartBounds.Dx()) / 2
		restartY := msgY + bounds.Dy() + 10
		text.Draw(screen, restartMsg, basicfont.Face7x13, restartX, restartY, color.Gray{Y: 150})
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	rand.Seed(time.Now().UnixNano())
	game := NewGame()
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Slumb Gate - Character Sheet")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
