package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
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
	maxLevel            = 20 // Define a max level
)

type TurnState int

const (
	PlayerTurn TurnState = iota
	EnemyTurn
	GameOver
)

func (ts TurnState) String() string {
	switch ts {
	case PlayerTurn:
		return "Player Turn"
	case EnemyTurn:
		return "Enemy Turn"
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
	InputModeRestPrompt
	InputModeLevelUp
)

type ActionType int

const (
	ActionTypeStandard ActionType = iota
	ActionTypeBonus
	ActionTypeFree
	ActionTypeReaction
)

func (at ActionType) String() string {
	switch at {
	case ActionTypeStandard:
		return "Action"
	case ActionTypeBonus:
		return "Bonus Action"
	case ActionTypeFree:
		return "Free Action"
	case ActionTypeReaction:
		return "Reaction"
	default:
		return "Unknown Action Type"
	}
}

type ResourceType string

const (
	ResourceNone         ResourceType = "none"
	ResourceHitDice      ResourceType = "hit_dice"
	ResourceClassFeature ResourceType = "class_feature"
	ResourceSpellSlot    ResourceType = "spell_slot"
)

type ResourceRestType int

const (
	RestTypeShort ResourceRestType = iota
	RestTypeLong
	RestTypeCombat
	RestTypeNever // Refreshes on Level Up
)

type Entity struct {
	X            int
	Y            int
	Width        int
	Height       int
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
	Level             int
	Class             string
	ProficiencyBonus  int
	MovementPoints    int
	MaxMovementPoints int
	ActionTaken       bool
	BonusActionTaken  bool
	IsDisengaging     bool
	ClassResources    map[string]int
	HitDice           int
	MaxHitDice        int
}

type Enemy struct {
	Entity
	MovementPoints    int
	MaxMovementPoints int
	ActionAvailable   bool
	AttackType        string
	MaxRange          int
}

type ClassAction struct {
	ActionID      string
	Name          string
	RequiredLevel int
	ActionType    ActionType
	ResourceType  ResourceType
	ResourceCost  int
	UsesPerRest   int
	RefreshesOn   ResourceRestType
	Description   string
}

type ClassDefinition struct {
	Name             string
	HitDieSize       int
	PrimaryAbility   string
	SavingThrowProfs []string
	ClassActions     map[string]*ClassAction
}

type Game struct {
	Player               *Player
	Enemies              []*Enemy
	GameMap              [mapWidth][mapHeight]int
	TileImage            *ebiten.Image
	CurrentTurn          TurnState
	CurrentWaveIndex     int
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
	WaveDefinitions      []WaveDefinition
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
	ActionType     ActionType
	Targeting      TargetType
	Range          int
	RequiresTarget bool
	Execute        ActionExecuteFunc
}

type EnemyDefinition struct {
	Name         string
	SpriteSheetX int
	SpriteSheetY int
	BaseHP       int
	AC           int
	Str          int
	Dex          int
	Con          int
	Int          int
	Wis          int
	Cha          int
	Move         int
	Width        int
	Height       int
	AttackType   string
	MaxRange     int
}

type EnemySpawnInfo struct {
	TypeName      string
	SpawnPointIdx int
}

type WaveDefinition struct {
	EnemiesToSpawn []EnemySpawnInfo
	IsBossWave     bool
}

var EnemyDefinitions = map[string]EnemyDefinition{
	"Small Slime": {
		Name: "Small Slime", SpriteSheetX: 0, SpriteSheetY: 2,
		BaseHP: 6, AC: 10, Str: 12, Dex: 10, Con: 11, Int: 8, Wis: 8, Cha: 8, Move: enemyBaseMovement,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
	},
	"Tough Slime": {
		Name: "Tough Slime", SpriteSheetX: 1, SpriteSheetY: 2,
		BaseHP: 7, AC: 10, Str: 12, Dex: 10, Con: 11, Int: 8, Wis: 8, Cha: 8, Move: enemyBaseMovement,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
	},
	"Goo Spitter": {
		Name: "Goo Spitter", SpriteSheetX: 2, SpriteSheetY: 2,
		BaseHP: 5, AC: 11, Str: 8, Dex: 14, Con: 10, Int: 8, Wis: 8, Cha: 8, Move: enemyBaseMovement,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 4,
	},
	"Big Slime Boss": {
		Name: "Big Slime Boss", SpriteSheetX: 1, SpriteSheetY: 2,
		BaseHP: 25, AC: 12, Str: 14, Dex: 8, Con: 15, Int: 6, Wis: 6, Cha: 6, Move: enemyBaseMovement - 1,
		Width: 2, Height: 2, AttackType: "melee", MaxRange: 1,
	},
	"Melee Skeleton": {
		Name: "Melee Skeleton", SpriteSheetX: 0, SpriteSheetY: 4, // Corrected Y coordinate
		BaseHP: 13, AC: 13, Str: 10, Dex: 14, Con: 15, Int: 6, Wis: 8, Cha: 5, Move: enemyBaseMovement,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
	},
	"Ranged Skeleton": {
		Name: "Ranged Skeleton", SpriteSheetX: 1, SpriteSheetY: 4, // Corrected Y coordinate
		BaseHP: 11, AC: 13, Str: 8, Dex: 16, Con: 13, Int: 6, Wis: 8, Cha: 5, Move: enemyBaseMovement,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 5,
	},
}

var ClassDefinitions = map[string]*ClassDefinition{
	"Fighter": {
		Name:             "Fighter",
		HitDieSize:       10,
		PrimaryAbility:   "STR",
		SavingThrowProfs: []string{"STR", "CON"},
		ClassActions: map[string]*ClassAction{
			"second_wind": {
				ActionID:      "second_wind",
				Name:          "Second Wind",
				RequiredLevel: 2,
				ActionType:    ActionTypeStandard,
				ResourceType:  ResourceClassFeature,
				ResourceCost:  1,
				UsesPerRest:   1,
				RefreshesOn:   RestTypeNever,
				Description:   "Use Action: Heal using 1 Hit Die (d10 + CON).",
			},
			"action_surge": {
				ActionID:      "action_surge",
				Name:          "Action Surge",
				RequiredLevel: 2,
				ActionType:    ActionTypeFree,
				ResourceType:  ResourceClassFeature,
				ResourceCost:  1,
				UsesPerRest:   1,
				RefreshesOn:   RestTypeNever,
				Description:   "Gain an additional standard action this turn.",
			},
		},
	},
}

var ActionTable = map[string]*ActionDefinition{
	"melee_attack": {
		ID:             "melee_attack",
		Name:           "Melee Attack",
		ActionType:     ActionTypeStandard,
		Targeting:      TargetEnemyAdjacent,
		Range:          1,
		RequiresTarget: true,
		Execute:        executeMeleeAttack,
	},
	"ranged_attack": {
		ID:             "ranged_attack",
		Name:           "Ranged Attack",
		ActionType:     ActionTypeStandard,
		Targeting:      TargetEnemyRange,
		Range:          playerRangedRange,
		RequiresTarget: true,
		Execute:        executeRangedAttack,
	},
	"dash": {
		ID:             "dash",
		Name:           "Dash",
		ActionType:     ActionTypeStandard,
		Targeting:      TargetSelf,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeDash,
	},
	"disengage": {
		ID:             "disengage",
		Name:           "Disengage",
		ActionType:     ActionTypeStandard,
		Targeting:      TargetSelf,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeDisengage,
	},
	"wait": {
		ID:             "wait",
		Name:           "Wait (End Turn)",
		ActionType:     ActionTypeFree,
		Targeting:      TargetNone,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeWait,
	},
	"second_wind": {
		ID:             "second_wind",
		Name:           "Second Wind",
		ActionType:     ActionTypeStandard,
		Targeting:      TargetSelf,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeSecondWind,
	},
	"action_surge": {
		ID:             "action_surge",
		Name:           "Action Surge",
		ActionType:     ActionTypeFree,
		Targeting:      TargetSelf,
		Range:          0,
		RequiresTarget: false,
		Execute:        executeActionSurge,
	},
}

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

func calculateProficiencyBonus(level int) int {
	if level < 5 {
		return 2
	}
	if level < 9 {
		return 3
	}
	if level < 13 {
		return 4
	}
	if level < 17 {
		return 5
	}
	return 6
}

func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)
	g.MapOffsetX = (screenWidth - (mapWidth * tileSize)) / 2
	g.MapOffsetY = (screenHeight - (mapHeight * tileSize)) / 2
	g.InputMode = InputModeMap
	g.availableActions = make([]*ActionDefinition, 0)
	g.lastExecutedActionID = ""
	g.CurrentWaveIndex = -1

	g.WaveDefinitions = []WaveDefinition{
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Small Slime", SpawnPointIdx: 0}, {TypeName: "Small Slime", SpawnPointIdx: 1}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Tough Slime", SpawnPointIdx: 2}, {TypeName: "Tough Slime", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Tough Slime", SpawnPointIdx: 0}, {TypeName: "Goo Spitter", SpawnPointIdx: 4}, {TypeName: "Tough Slime", SpawnPointIdx: 1}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Big Slime Boss", SpawnPointIdx: 4}}, IsBossWave: true},
		{
			EnemiesToSpawn: []EnemySpawnInfo{
				{TypeName: "Melee Skeleton", SpawnPointIdx: 0},
				{TypeName: "Melee Skeleton", SpawnPointIdx: 1},
				{TypeName: "Melee Skeleton", SpawnPointIdx: 2},
				{TypeName: "Ranged Skeleton", SpawnPointIdx: 4},
			},
			IsBossWave: false,
		},
	}

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
		g.TileImage = floorSprite
	}

	g.RangeOverlayTile = ebiten.NewImage(tileSize, tileSize)
	overlayColor := color.NRGBA{R: 0, G: 100, B: 200, A: 80}
	vector.DrawFilledRect(g.RangeOverlayTile, 0, 0, float32(tileSize), float32(tileSize), overlayColor, false)

	playerSprite := getSpriteFromSheet(g.rogueSheet, 1, 1)
	playerStr, playerDex, playerCon := 15, 14, 13
	playerInt, playerWis, playerCha := 8, 12, 10
	playerConMod := getModifier(playerCon)

	playerClass := "Fighter"
	classDef, classExists := ClassDefinitions[playerClass]
	if !classExists {
		log.Fatalf("FATAL: Player class '%s' not found in ClassDefinitions!", playerClass)
	}

	startLevel := 1
	playerMaxHP := classDef.HitDieSize + playerConMod
	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2, Width: 1, Height: 1,
			HP: playerMaxHP, MaxHP: playerMaxHP, AC: 13,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Sprite: playerSprite, Name: "Player",
		},
		Level:             startLevel,
		Class:             playerClass,
		ProficiencyBonus:  calculateProficiencyBonus(startLevel),
		MaxMovementPoints: playerBaseMovement,
		ClassResources:    make(map[string]int),
		MaxHitDice:        startLevel,
		HitDice:           startLevel,
	}

	g.initializePlayerResources()

	g.CurrentTurn = PlayerTurn
	g.SpawnNextWave()
	g.startPlayerTurn()
	return g
}

func (g *Game) initializePlayerResources() {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}
	for actionID, classAction := range classDef.ClassActions {
		if classAction.ResourceType == ResourceClassFeature {
			if classAction.RequiredLevel <= g.Player.Level {
				if classAction.UsesPerRest > 0 {
					g.Player.ClassResources[actionID] = classAction.UsesPerRest
				} else {
					g.Player.ClassResources[actionID] = 0
				}
			} else {
				g.Player.ClassResources[actionID] = 0
			}
		}
	}
}

func (g *Game) refreshLevelUpResources() {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}

	refreshedSomething := false
	logMsg := "Level Up! Resources refreshed: "

	for actionID, classAction := range classDef.ClassActions {
		if classAction.RefreshesOn == RestTypeNever {
			if g.Player.Level >= classAction.RequiredLevel {
				currentUses := g.Player.ClassResources[actionID]
				maxUses := classAction.UsesPerRest

				if currentUses < maxUses {
					g.Player.ClassResources[actionID] = maxUses
					logMsg += fmt.Sprintf("%s, ", classAction.Name)
					refreshedSomething = true
				} else if g.Player.Level == classAction.RequiredLevel {
					g.Player.ClassResources[actionID] = maxUses
					logMsg += fmt.Sprintf("%s (Unlocked!), ", classAction.Name)
					refreshedSomething = true
				}
			}
		}
	}
	if refreshedSomething {
		if len(logMsg) > 2 {
			logMsg = logMsg[:len(logMsg)-2] + "."
		}
		g.addCombatLog(logMsg)
	}
}

func (g *Game) resetPlayerResources(restType ResourceRestType) {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}
	logMsg := ""
	if restType == RestTypeShort {
		logMsg = "Short Rest complete. Resources refreshed: "
	} else if restType == RestTypeLong {
		logMsg = "Long Rest complete. Resources refreshed: "
	}

	refreshedSomething := false
	for actionID, classAction := range classDef.ClassActions {
		if classAction.RefreshesOn == RestTypeNever {
			continue
		}

		shouldReset := false
		if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
			if classAction.RefreshesOn == restType {
				shouldReset = true
			}
			if restType == RestTypeLong && classAction.RefreshesOn == RestTypeShort {
				shouldReset = true
			}
		}

		if shouldReset && g.Player.Level >= classAction.RequiredLevel {
			currentUses := g.Player.ClassResources[actionID]
			maxUses := classAction.UsesPerRest

			if currentUses < maxUses {
				g.Player.ClassResources[actionID] = maxUses
				logMsg += fmt.Sprintf("%s, ", classAction.Name)
				refreshedSomething = true
			}
		}
	}

	if refreshedSomething {
		if len(logMsg) > 2 {
			logMsg = logMsg[:len(logMsg)-2] + "."
		}
		g.addCombatLog(logMsg)
	}

	if restType == RestTypeLong {
		diceToRecover := max(1, g.Player.MaxHitDice/2)
		initialDice := g.Player.HitDice
		g.Player.HitDice = min(g.Player.MaxHitDice, g.Player.HitDice+diceToRecover)
		recoveredCount := g.Player.HitDice - initialDice
		if recoveredCount > 0 {
			g.addCombatLog(fmt.Sprintf("Recovered %d Hit Dice.", recoveredCount))
		}
	}
}

func (g *Game) shortRest() {
	g.addCombatLog("Player takes a Short Rest...")
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		g.addCombatLog("Cannot determine class for Hit Dice.")
		return
	}

	if g.Player.HitDice > 0 {
		conMod := getModifier(g.Player.Constitution)
		healRoll := rand.Intn(classDef.HitDieSize) + 1
		healAmount := max(1, healRoll+conMod)
		actualHeal := min(healAmount, g.Player.MaxHP-g.Player.HP)
		actualHeal = max(0, actualHeal)

		if actualHeal > 0 {
			g.Player.HP += actualHeal
			g.addCombatLog(fmt.Sprintf("Spent 1 Hit Die (d%d), recovered %d HP (Rolled %d%+d).", classDef.HitDieSize, actualHeal, healRoll, conMod))
		} else {
			g.addCombatLog("Spent 1 Hit Die, but already at full HP.")
		}
		g.Player.HitDice--
	} else {
		g.addCombatLog("No Hit Dice left to spend.")
	}
}

func (g *Game) longRest() {
	g.addCombatLog("Player takes a Long Rest...")
	g.Player.HP = g.Player.MaxHP
	g.resetPlayerResources(RestTypeLong)
	g.addCombatLog("HP fully restored.")
}

func (g *Game) levelUpPlayer() {
	if g.Player.Level >= maxLevel {
		g.addCombatLog("Already at max level!")
		return
	}

	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		log.Printf("Error: Cannot find class definition %s for level up.", g.Player.Class)
		return
	}

	g.Player.Level++
	g.addCombatLog(fmt.Sprintf("LEVEL UP! Reached Level %d!", g.Player.Level))

	conMod := getModifier(g.Player.Constitution)
	hpRoll := rand.Intn(classDef.HitDieSize) + 1
	hpIncrease := max(1, hpRoll+conMod)
	g.Player.MaxHP += hpIncrease
	g.Player.HP += hpIncrease
	g.addCombatLog(fmt.Sprintf("Max HP increased by %d (Rolled %d%+d).", hpIncrease, hpRoll, conMod))

	g.Player.MaxHitDice++
	g.Player.HitDice++
	g.addCombatLog("Gained 1 Hit Die.")

	newProfBonus := calculateProficiencyBonus(g.Player.Level)
	if newProfBonus > g.Player.ProficiencyBonus {
		g.Player.ProficiencyBonus = newProfBonus
		g.addCombatLog(fmt.Sprintf("Proficiency Bonus increased to +%d.", newProfBonus))
	}

	g.refreshLevelUpResources()
}

func (g *Game) startPlayerTurn() {
	g.CurrentTurn = PlayerTurn
	g.Player.MovementPoints = g.Player.MaxMovementPoints
	g.Player.ActionTaken = false
	g.Player.BonusActionTaken = false
	g.Player.IsDisengaging = false
	g.InputMode = InputModeMap
	g.primedActionID = ""
}

func (g *Game) handleWaveCompletion() {
	currentWaveDef := g.WaveDefinitions[g.CurrentWaveIndex]
	isFinalWave := g.CurrentWaveIndex+1 >= len(g.WaveDefinitions)

	if currentWaveDef.IsBossWave && g.Player.Level < maxLevel { // Check if boss and not max level
		g.addCombatLog("Final Boss Defeated!")
		g.levelUpPlayer()
		g.InputMode = InputModeLevelUp
	} else if !isFinalWave { // Not final wave, offer rest
		g.addCombatLog(fmt.Sprintf("Wave Cleared! Spend 1 Hit Die (of %d) to heal? [Y/N]", g.Player.HitDice))
		g.InputMode = InputModeRestPrompt
	} else { // Final wave cleared (or boss cleared at max level) -> Victory
		g.addCombatLog("All challenges overcome! VICTORY!")
		g.longRest() // Perform final long rest
		g.CurrentTurn = GameOver
	}
}

func (g *Game) startNextWave() {
	g.SpawnNextWave()
	if len(g.Enemies) > 0 {
		if g.CurrentTurn == PlayerTurn {
			g.startEnemyTurn()
		} else {
			g.startPlayerTurn()
		}
	} else {
		g.addCombatLog("Error spawning next wave or wave empty.")
		g.CurrentTurn = GameOver
	}
	g.InputMode = InputModeMap
}

func (g *Game) endPlayerTurn() {
	g.cleanupDeadEnemies()

	if len(g.Enemies) == 0 {
		g.handleWaveCompletion()
	} else {
		g.startEnemyTurn()
	}

	g.primedActionID = ""
	if g.CurrentTurn != GameOver && g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
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
		g.handleWaveCompletion()
	} else {
		g.startPlayerTurn()
	}
}

func (g *Game) SpawnNextWave() {
	g.CurrentWaveIndex++

	if g.CurrentWaveIndex >= len(g.WaveDefinitions) {
		log.Printf("Attempted to spawn wave index %d, but only %d waves are defined.", g.CurrentWaveIndex, len(g.WaveDefinitions))
		g.addCombatLog("No more waves defined.")
		if g.CurrentTurn != GameOver {
			g.CurrentTurn = GameOver
		}
		return
	}

	waveDef := g.WaveDefinitions[g.CurrentWaveIndex]
	g.addCombatLog(fmt.Sprintf("--- Starting Wave %d ---", g.CurrentWaveIndex+1))
	g.Enemies = make([]*Enemy, 0)

	spawnPoints := []image.Point{
		{X: 2, Y: 2},
		{X: mapWidth - 3, Y: mapHeight - 3},
		{X: 2, Y: mapHeight - 3},
		{X: mapWidth - 3, Y: 2},
		{X: mapWidth / 2, Y: 1},
	}

	for _, spawnInfo := range waveDef.EnemiesToSpawn {
		enemyDef, exists := EnemyDefinitions[spawnInfo.TypeName]
		if !exists {
			log.Printf("Warning: Enemy type '%s' not found in definitions.", spawnInfo.TypeName)
			continue
		}

		spawnX, spawnY := -1, -1
		if spawnInfo.SpawnPointIdx >= 0 && spawnInfo.SpawnPointIdx < len(spawnPoints) {
			spawnPoint := spawnPoints[spawnInfo.SpawnPointIdx]
			spawnX = spawnPoint.X
			spawnY = spawnPoint.Y
			if enemyDef.Width > 1 {
				spawnX -= (enemyDef.Width - 1) / 2
			}
			if enemyDef.Height > 1 {
				spawnY -= (enemyDef.Height - 1) / 2
			}
			if spawnX < 0 {
				spawnX = 0
			}
			if spawnY < 0 {
				spawnY = 0
			}
			if spawnX+enemyDef.Width > mapWidth {
				spawnX = mapWidth - enemyDef.Width
			}
			if spawnY+enemyDef.Height > mapHeight {
				spawnY = mapHeight - enemyDef.Height
			}

		} else {
			log.Printf("Warning: Invalid spawn point index %d for enemy '%s'. Skipping.", spawnInfo.SpawnPointIdx, enemyDef.Name)
			continue
		}

		if g.isTileFullyBlocked(spawnX, spawnY, enemyDef.Width, enemyDef.Height, -1) {
			log.Printf("Warning: Spawn point %d (%d,%d) for %s is blocked. Trying fallback.", spawnInfo.SpawnPointIdx, spawnX, spawnY, enemyDef.Name)
			foundAlt := false
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					altX, altY := spawnX+dx, spawnY+dy
					if altX >= 0 && altX+enemyDef.Width <= mapWidth && altY >= 0 && altY+enemyDef.Height <= mapHeight {
						if !g.isTileFullyBlocked(altX, altY, enemyDef.Width, enemyDef.Height, -1) {
							spawnX, spawnY = altX, altY
							foundAlt = true
							break
						}
					}
				}
				if foundAlt {
					break
				}
			}
			if !foundAlt {
				log.Printf("Error: Could not find alternative spawn location for %s near (%d,%d). Skipping.", enemyDef.Name, spawnX, spawnY)
				continue
			}
		}

		sprite := getSpriteFromSheet(g.monsterSheet, enemyDef.SpriteSheetX, enemyDef.SpriteSheetY)
		g.spawnEnemyFromDef(spawnX, spawnY, enemyDef, sprite)
	}
}

func (g *Game) spawnEnemyFromDef(x, y int, def EnemyDefinition, sprite *ebiten.Image) {
	conMod := getModifier(def.Con)
	maxHp := def.BaseHP + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, Width: def.Width, Height: def.Height,
			HP: maxHp, MaxHP: maxHp, AC: def.AC,
			Strength: def.Str, Dexterity: def.Dex, Constitution: def.Con,
			Intelligence: def.Int, Wisdom: def.Wis, Charisma: def.Cha,
			Sprite: sprite, Name: def.Name,
		},
		MaxMovementPoints: def.Move,
		AttackType:        def.AttackType,
		MaxRange:          def.MaxRange,
	}
	g.Enemies = append(g.Enemies, enemy)
	g.addCombatLog(fmt.Sprintf("%s appears!", enemy.Name))
}

func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move, w, h int, sprite *ebiten.Image) {
	conMod := getModifier(con)
	maxHp := baseHp + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, Width: w, Height: h,
			HP: maxHp, MaxHP: maxHp, AC: ac,
			Strength: str, Dexterity: dex, Constitution: con,
			Intelligence: intel, Wisdom: wis, Charisma: cha,
			Sprite: sprite, Name: name,
		},
		MaxMovementPoints: move,
	}
	g.Enemies = append(g.Enemies, enemy)
}

func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
}

func (g *Game) isTileBlocked(checkX, checkY, movingEnemyIndex int) bool {
	if g.Player.HP > 0 && checkX >= g.Player.X && checkX < g.Player.X+g.Player.Width &&
		checkY >= g.Player.Y && checkY < g.Player.Y+g.Player.Height {
		return true
	}
	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if i == movingEnemyIndex {
			continue
		}
		if checkX >= enemy.X && checkX < enemy.X+enemy.Width &&
			checkY >= enemy.Y && checkY < enemy.Y+enemy.Height {
			return true
		}
	}
	return false
}

func (g *Game) isTileFullyBlocked(checkX, checkY, entityWidth, entityHeight, movingEnemyIndex int) bool {
	for w := 0; w < entityWidth; w++ {
		for h := 0; h < entityHeight; h++ {
			tileX, tileY := checkX+w, checkY+h
			if tileX < 0 || tileX >= mapWidth || tileY < 0 || tileY >= mapHeight {
				return true
			}
			if g.isTileBlocked(tileX, tileY, movingEnemyIndex) {
				return true
			}
		}
	}
	return false
}

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

func isAdjacentSimple(x1, y1, x2, y2 int) bool {
	return distance(x1, y1, x2, y2) == 1
}

func isAdjacentToEntity(px, py int, entity *Entity) bool {
	if entity == nil || entity.HP <= 0 {
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
	if attacker == nil || defender == nil || attacker.HP <= 0 || defender.HP <= 0 {
		return false
	}

	var attackAbilityMod int
	var abilityName string
	switch attackType {
	case "ranged":
		attackAbilityMod = getModifier(attacker.Dexterity)
		abilityName = "DEX"
	default:
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
	rollString := fmt.Sprintf("%d%s%s(%s)=%d", roll, profString, modString, abilityName, attackRoll)

	attackVerb := "attacks"
	if attackType == "ranged" {
		attackVerb = "shoots"
	}
	logMsg := fmt.Sprintf("%s %s %s(AC%d). Roll: %s.", attacker.Name, attackVerb, defender.Name, defender.AC, rollString)

	if hit {
		var damage int
		damageRoll := 0
		damageLog := ""

		isPlayer := attacker == &g.Player.Entity

		if isPlayer {
			switch attackType {
			case "ranged":
				damageRoll = rand.Intn(6) + 1
				damage = damageRoll + attackAbilityMod
				damageLog = fmt.Sprintf(" (1d6[%d]%+d)", damageRoll, attackAbilityMod)
			default:
				damageRoll = rand.Intn(8) + 1
				damage = damageRoll + attackAbilityMod
				damageLog = fmt.Sprintf(" (1d8[%d]%+d)", damageRoll, attackAbilityMod)
			}
		} else {
			if attacker.Name == "Melee Skeleton" {
				damageRoll = rand.Intn(6) + 1
				damage = damageRoll + getModifier(attacker.Dexterity)
				damageLog = fmt.Sprintf(" (1d6[%d]%+d)", damageRoll, getModifier(attacker.Dexterity))
			} else if attacker.Name == "Ranged Skeleton" {
				damageRoll = rand.Intn(6) + 1
				damage = damageRoll + getModifier(attacker.Dexterity)
				damageLog = fmt.Sprintf(" (1d6[%d]%+d)", damageRoll, getModifier(attacker.Dexterity))
			} else {
				damageRoll = 0
				damage = attackAbilityMod
				damageLog = fmt.Sprintf(" (%+d)", attackAbilityMod)
			}
		}

		damage = max(1, damage)

		defender.HP -= damage
		logMsg += fmt.Sprintf(" Hit! Deals %d%s dmg.", damage, damageLog)
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

func (g *Game) executeAction(actionDef *ActionDefinition, targetX, targetY int) bool {
	if actionDef == nil {
		g.addCombatLog("Error: Tried to execute nil action.")
		return false
	}

	classDef := ClassDefinitions[g.Player.Class]
	var classAction *ClassAction
	if classDef != nil {
		classAction = classDef.ClassActions[actionDef.ID]
	}

	if classAction != nil && classAction.ResourceType == ResourceClassFeature {
		if classAction.UsesPerRest > 0 && g.Player.ClassResources[actionDef.ID] < classAction.ResourceCost {
			g.addCombatLog(fmt.Sprintf("Not enough uses left for %s.", actionDef.Name))
			return false
		}
	}

	success := actionDef.Execute(g, targetX, targetY)

	if success {
		g.lastExecutedActionID = actionDef.ID

		if classAction != nil && classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
			g.Player.ClassResources[actionDef.ID] -= classAction.ResourceCost
		}

		switch actionDef.ActionType {
		case ActionTypeStandard:
			g.Player.ActionTaken = true
		case ActionTypeBonus:
			g.Player.BonusActionTaken = true
		case ActionTypeFree:
		case ActionTypeReaction:
		}

		if actionDef.ID == "action_surge" {
			g.Player.ActionTaken = false
		}

		if actionDef.ID == "wait" {
			g.endPlayerTurn()
		}
	}

	return success
}

func executeMeleeAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil && isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		killed := g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "melee")
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
		dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
		if dist <= playerRangedRange {
			killed := g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "ranged")
			if killed {
				g.cleanupDeadEnemies()
			}
			return true
		}
		g.addCombatLog(fmt.Sprintf("Target %s out of range (%d > %d).", targetEnemy.Name, dist, playerRangedRange))
		return false
	}
	g.addCombatLog("No valid target selected at cursor for ranged attack.")
	return false
}

func executeDash(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Dash!")
	g.Player.MovementPoints += g.Player.MaxMovementPoints
	return true
}

func executeDisengage(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Disengage!")
	g.Player.IsDisengaging = true
	return true
}

func executeWait(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player waits, ending turn.")
	return true
}

func executeSecondWind(g *Game, targetX, targetY int) bool {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return false
	}
	classAction := classDef.ClassActions["second_wind"]
	if classAction == nil {
		return false
	}

	conMod := getModifier(g.Player.Constitution)
	healRoll := rand.Intn(classDef.HitDieSize) + 1
	healAmount := max(1, healRoll+conMod)
	actualHeal := min(healAmount, g.Player.MaxHP-g.Player.HP)
	actualHeal = max(0, actualHeal)

	if actualHeal > 0 {
		g.Player.HP += actualHeal
		g.addCombatLog(fmt.Sprintf("Used Second Wind! Healed %d HP (Rolled %d%+d).", actualHeal, healRoll, conMod))
	} else {
		g.addCombatLog("Used Second Wind, but already at full HP.")
	}

	return true
}

func executeActionSurge(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Action Surge! Gains an extra action.")
	return true
}

func (g *Game) handlePlayerInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if g.InputMode == InputModeCharacterSheet {
			g.InputMode = InputModeMap
			g.primedActionID = ""
		} else if g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
			g.InputMode = InputModeCharacterSheet
			g.primedActionID = ""
		}
		return
	}

	switch g.InputMode {
	case InputModeLevelUp:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.addCombatLog("Level up acknowledged. Preparing for next challenge...")
			g.longRest()
			if g.CurrentWaveIndex+1 < len(g.WaveDefinitions) {
				g.startNextWave()
			} else {
				g.addCombatLog("All challenges overcome! VICTORY!")
				g.CurrentTurn = GameOver
			}
		}
		return

	case InputModeRestPrompt:
		if inpututil.IsKeyJustPressed(ebiten.KeyY) {
			g.shortRest()
			g.startNextWave()
		} else if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.addCombatLog("Skipped short rest.")
			g.startNextWave()
		}
		return

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
					g.executeAction(selectedActionDef, -1, -1)
					if g.CurrentTurn != GameOver && selectedActionDef.ID != "wait" && g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
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
			g.buildAvailableActions()
			if len(g.availableActions) > 0 {
				foundLast := false
				if g.lastExecutedActionID != "" {
					for i, actionDef := range g.availableActions {
						if actionDef.ID == g.lastExecutedActionID {
							g.selectedActionIndex = i
							foundLast = true
							break
						}
					}
				}
				if !foundLast {
					g.selectedActionIndex = 0
				}
				g.InputMode = InputModeActionSelect
				g.primedActionID = ""
			} else {
				g.addCombatLog("No actions available!")
			}
			return
		}

		actionExecutedByClick := false
		if g.primedActionID != "" {
			actionDef, exists := ActionTable[g.primedActionID]
			if !exists {
				g.addCombatLog(fmt.Sprintf("Error: Unknown primed action ID '%s'. Cancelling.", g.primedActionID))
				g.primedActionID = ""
			} else {
				canUseAction := false
				switch actionDef.ActionType {
				case ActionTypeStandard:
					canUseAction = !g.Player.ActionTaken
				case ActionTypeBonus:
					canUseAction = !g.Player.BonusActionTaken
				case ActionTypeFree, ActionTypeReaction:
					canUseAction = true
				}

				if !canUseAction {
					g.addCombatLog(fmt.Sprintf("Cannot target for %s: %s already used.", actionDef.Name, actionDef.ActionType.String()))
					g.primedActionID = ""
				} else {
					if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
						cursorX, cursorY := ebiten.CursorPosition()
						gridX := (cursorX - g.MapOffsetX) / tileSize
						gridY := (cursorY - g.MapOffsetY) / tileSize

						if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight {
							actionExecutedByClick = g.executeAction(actionDef, gridX, gridY)
							g.primedActionID = ""
						} else {
							g.addCombatLog("Clicked outside map. Targeting cancelled.")
							g.primedActionID = ""
						}
					} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
						g.addCombatLog(fmt.Sprintf("Targeting for %s cancelled.", actionDef.Name))
						g.primedActionID = ""
					}
				}
			}
		}

		if actionExecutedByClick {
			if g.CurrentTurn == GameOver || g.InputMode == InputModeRestPrompt || g.InputMode == InputModeLevelUp {
				return
			}
		}

		if g.primedActionID == "" && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			waitAction, exists := ActionTable["wait"]
			if exists {
				g.executeAction(waitAction, -1, -1)
			} else {
				g.addCombatLog("Error: Wait action not found!")
				g.endPlayerTurn()
			}
			return
		}

		if g.primedActionID == "" && g.Player.MovementPoints > 0 {
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
				if targetX >= 0 && targetX+g.Player.Width <= mapWidth && targetY >= 0 && targetY+g.Player.Height <= mapHeight {
					if !g.isTileFullyBlocked(targetX, targetY, g.Player.Width, g.Player.Height, -1) {
						performAoOCheck := !g.Player.IsDisengaging
						if performAoOCheck {
							for _, enemy := range g.Enemies {
								if enemy.HP <= 0 {
									continue
								}
								wasAdj := isAdjacentToEntity(startX, startY, &enemy.Entity)
								isStillAdj := isAdjacentToEntity(targetX, targetY, &enemy.Entity)

								if wasAdj && !isStillAdj {
									if g.Player.HP > 0 {
										g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
										killedByAoO := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
										if killedByAoO {
											g.Player.MovementPoints--
											return
										}
									}
								}
							}
						}

						if g.Player.HP > 0 {
							g.Player.X = targetX
							g.Player.Y = targetY
							g.Player.MovementPoints--
						}
					} else {
						g.addCombatLog("Movement blocked.")
					}
				} else {
					g.addCombatLog("Cannot move outside map boundaries.")
				}
			}
		}
	}
}

func (g *Game) buildAvailableActions() {
	g.availableActions = []*ActionDefinition{}
	possibleActions := make(map[string]bool)

	for id, actionDef := range ActionTable {
		isBaseAction := true
		if classDef := ClassDefinitions[g.Player.Class]; classDef != nil {
			if _, isClassAction := classDef.ClassActions[id]; isClassAction {
				isBaseAction = false
			}
		}
		if actionDef.Execute == nil {
			isBaseAction = false
		}

		if isBaseAction {
			possibleActions[id] = true
		}
	}

	classDef, exists := ClassDefinitions[g.Player.Class]
	if exists {
		for actionID, classAction := range classDef.ClassActions {
			if g.Player.Level >= classAction.RequiredLevel {
				if _, actionDefExists := ActionTable[actionID]; actionDefExists {
					possibleActions[actionID] = true
				} else {
					log.Printf("Warning: ClassAction '%s' defined for class '%s' but not found in main ActionTable.", actionID, g.Player.Class)
				}
			}
		}
	}

	tempAvailableActions := []*ActionDefinition{}
	for id := range possibleActions {
		actionDef := ActionTable[id]
		isAvailable := true

		switch actionDef.ActionType {
		case ActionTypeStandard:
			if g.Player.ActionTaken {
				isAvailable = false
			}
		case ActionTypeBonus:
			if g.Player.BonusActionTaken {
				isAvailable = false
			}
		case ActionTypeFree, ActionTypeReaction:
		}
		if !isAvailable {
			continue
		}

		if classDef != nil {
			if classAction, ok := classDef.ClassActions[id]; ok {
				if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
					if g.Player.ClassResources[id] < classAction.ResourceCost {
						isAvailable = false
					}
				}
			}
		}
		if !isAvailable {
			continue
		}

		if actionDef.ID == "disengage" && !isPlayerAdjacentToEnemy(g) {
			isAvailable = false
		}

		if isAvailable {
			tempAvailableActions = append(tempAvailableActions, actionDef)
		}
	}

	sort.Slice(tempAvailableActions, func(i, j int) bool {
		if tempAvailableActions[i].ActionType != tempAvailableActions[j].ActionType {
			return tempAvailableActions[i].ActionType < tempAvailableActions[j].ActionType
		}
		return tempAvailableActions[i].Name < tempAvailableActions[j].Name
	})

	g.availableActions = tempAvailableActions
}

func (g *Game) findPathStep(startX, startY, targetX, targetY, entityWidth, entityHeight, movingEnemyIndex int) (int, int, bool) {
	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	currentDist := distance(startX, startY, targetX, targetY)
	bestMoveX, bestMoveY := startX, startY
	minDist := currentDist

	possibleMoves := []image.Point{}
	for _, offset := range moveOffsets {
		nextX, nextY := startX+offset.X, startY+offset.Y
		if !g.isTileFullyBlocked(nextX, nextY, entityWidth, entityHeight, movingEnemyIndex) {
			possibleMoves = append(possibleMoves, image.Point{X: nextX, Y: nextY})
		}
	}

	if len(possibleMoves) == 0 {
		return startX, startY, false
	}

	bestMoves := []image.Point{}
	for _, move := range possibleMoves {
		dist := distance(move.X, move.Y, targetX, targetY)
		if dist < minDist {
			minDist = dist
		}
	}
	for _, move := range possibleMoves {
		if distance(move.X, move.Y, targetX, targetY) == minDist {
			bestMoves = append(bestMoves, move)
		}
	}

	if len(bestMoves) > 0 {
		dx := targetX - startX
		dy := targetY - startY
		preferredMoveFound := false
		if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
			for _, move := range bestMoves {
				if move.X != startX {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}
		if !preferredMoveFound && math.Abs(float64(dy)) >= math.Abs(float64(dx)) {
			for _, move := range bestMoves {
				if move.Y != startY {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}
		if !preferredMoveFound {
			bestMoveX, bestMoveY = bestMoves[0].X, bestMoves[0].Y
		}
		if bestMoveX != startX || bestMoveY != startY {
			return bestMoveX, bestMoveY, true
		}
	}

	return startX, startY, false
}

func (g *Game) findRetreatStep(startX, startY, targetX, targetY, entityWidth, entityHeight, movingEnemyIndex int) (int, int, bool) {
	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	currentDist := distance(startX, startY, targetX, targetY)
	bestMoveX, bestMoveY := startX, startY
	maxDist := currentDist

	possibleMoves := []image.Point{}
	for _, offset := range moveOffsets {
		nextX, nextY := startX+offset.X, startY+offset.Y
		if !g.isTileFullyBlocked(nextX, nextY, entityWidth, entityHeight, movingEnemyIndex) {
			possibleMoves = append(possibleMoves, image.Point{X: nextX, Y: nextY})
		}
	}

	if len(possibleMoves) == 0 {
		return startX, startY, false
	}

	bestMoves := []image.Point{}
	for _, move := range possibleMoves {
		dist := distance(move.X, move.Y, targetX, targetY)
		if dist > maxDist {
			maxDist = dist
		}
	}

	if maxDist == currentDist {
		maxDist = -1
		for _, move := range possibleMoves {
			dist := distance(move.X, move.Y, targetX, targetY)
			if dist > maxDist {
				maxDist = dist
			}
		}
	}

	for _, move := range possibleMoves {
		if distance(move.X, move.Y, targetX, targetY) == maxDist {
			bestMoves = append(bestMoves, move)
		}
	}

	if len(bestMoves) > 0 {

		bestMoveX, bestMoveY = bestMoves[0].X, bestMoves[0].Y
		if bestMoveX != startX || bestMoveY != startY {
			return bestMoveX, bestMoveY, true
		}
	}

	return startX, startY, false
}

func (g *Game) handleEnemyTurns() {
	if g.Player.HP <= 0 {
		return
	}

	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}

		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		actedThisTurn := false

		for turnPhase := 0; turnPhase < 2; turnPhase++ {
			if actedThisTurn && enemy.MovementPoints <= 0 {
				break
			}

			distToPlayer := distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
			isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

			if enemy.AttackType == "ranged" {
				moved := false
				if isAdj && enemy.MovementPoints > 0 {
					nextX, nextY, foundMove := g.findRetreatStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						moved = true
						isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
						distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
					}
				}

				if !isAdj && distToPlayer <= enemy.MaxRange && enemy.ActionAvailable {
					killedPlayer := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
					enemy.ActionAvailable = false
					actedThisTurn = true
					if killedPlayer {
						return
					}
				}

				if !moved && enemy.ActionAvailable && distToPlayer > enemy.MaxRange && enemy.MovementPoints > 0 {
					nextX, nextY, foundMove := g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						moved = true
						distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
						if !isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity) && distToPlayer <= enemy.MaxRange && enemy.ActionAvailable {
							killedPlayer := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
							enemy.ActionAvailable = false
							actedThisTurn = true
							if killedPlayer {
								return
							}
						}
					}
				}
			} else { // Melee AI

				if isAdj && enemy.ActionAvailable {
					killedPlayer := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
					enemy.ActionAvailable = false
					actedThisTurn = true
					if killedPlayer {
						return
					}
				}

				if !isAdj && enemy.MovementPoints > 0 {
					nextX, nextY, foundMove := g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--

						isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
						if isAdj && enemy.ActionAvailable {
							killedPlayer := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
							enemy.ActionAvailable = false
							actedThisTurn = true
							if killedPlayer {
								return
							}
						}
					}
				}
			}
			if enemy.ActionAvailable {
				enemy.ActionAvailable = false
			}
		}
	}
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
	// Handle Game Over state first
	if g.CurrentTurn == GameOver {
		// Potentially handle restart input here later
		return nil
	}

	// Handle Level Up state separately - only process Enter key
	if g.InputMode == InputModeLevelUp {
		g.handlePlayerInput() // This now only checks for Enter in this mode
		return nil            // Halt further game logic until Enter is pressed
	}

	// Handle other input modes or player turn actions
	if g.CurrentTurn == PlayerTurn || g.InputMode == InputModeActionSelect || g.InputMode == InputModeCharacterSheet || g.InputMode == InputModeRestPrompt {
		g.handlePlayerInput()
	}

	// If input handling resulted in a state change that should halt further processing
	if g.CurrentTurn == GameOver || g.InputMode == InputModeLevelUp || g.InputMode == InputModeRestPrompt {
		return nil
	}

	// Process enemy turn if it's their turn
	if g.CurrentTurn == EnemyTurn {
		g.handleEnemyTurns()
		// Check player death immediately after enemy actions
		if g.Player.HP <= 0 && g.CurrentTurn != GameOver {
			g.addCombatLog("Player has died! Game Over.")
			g.CurrentTurn = GameOver
			return nil
		}
		// End enemy turn only if not waiting for rest/level up
		if g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
			g.endEnemyTurn()
		}
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
			originX, originY := g.Player.X, g.Player.Y

			for x := 0; x < mapWidth; x++ {
				for y := 0; y < mapHeight; y++ {
					dist := distance(originX, originY, x, y)
					isInRange := dist > 0 && dist <= actionDef.Range

					if isInRange {
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

	sort.Slice(entitiesToDraw, func(i, j int) bool {
		return entitiesToDraw[i].Y < entitiesToDraw[j].Y
	})

	entityOpts := &ebiten.DrawImageOptions{}
	for _, entity := range entitiesToDraw {
		if entity.Sprite == nil {
			continue
		}

		entityScreenX := float64(mapOffsetX + entity.X*tileSize)
		entityScreenY := float64(mapOffsetY + entity.Y*tileSize)

		entityOpts.GeoM.Reset()
		if entity.Width > 1 || entity.Height > 1 {
			entityOpts.GeoM.Scale(float64(entity.Width), float64(entity.Height))
		}
		entityOpts.GeoM.Translate(entityScreenX, entityScreenY)
		screen.DrawImage(entity.Sprite, entityOpts)

		hpBarBaseX := float64(mapOffsetX + entity.X*tileSize)
		hpBarBaseY := float64(mapOffsetY + (entity.Y+entity.Height)*tileSize)
		hpBarX := float32(hpBarBaseX)
		hpBarY := float32(hpBarBaseY + hpBarOffsetY)
		hpBarWidth := float32(tileSize * entity.Width)
		hpRatio := float32(entity.HP) / float32(entity.MaxHP)
		hpRatio = maxF(0.0, minF(1.0, hpRatio))

		vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, color.RGBA{R: 80, G: 0, B: 0, A: 255}, false)
		vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth*hpRatio, hpBarHeight, color.RGBA{R: 0, G: 200, B: 0, A: 255}, false)
		vector.StrokeRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, 1, color.Black, false)
	}

	uiStartY := 10
	uiLineHeight := 15
	statusStartY := 10

	waveText := fmt.Sprintf("Wave: %d / %d", g.CurrentWaveIndex+1, len(g.WaveDefinitions))
	waveTextWidth := text.BoundString(basicfont.Face7x13, waveText).Dx()
	ebitenutil.DebugPrintAt(screen, waveText, screenWidth-waveTextWidth-10, statusStartY)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Turn: %s", g.CurrentTurn.String()), 10, uiStartY)
	playerHpVal := max(0, g.Player.HP)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP: %d/%d AC: %d", playerHpVal, g.Player.MaxHP, g.Player.AC), 10, uiStartY+uiLineHeight*1)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Move: %d/%d", g.Player.MovementPoints, g.Player.MaxMovementPoints), 10, uiStartY+uiLineHeight*2)

	actionStatusText := "Action: Available"
	if g.Player.ActionTaken {
		actionStatusText = "Action: Used"
	}
	ebitenutil.DebugPrintAt(screen, actionStatusText, 10, uiStartY+uiLineHeight*3)

	bonusActionStatusText := "Bonus Action: Available"
	if g.Player.BonusActionTaken {
		bonusActionStatusText = "Bonus Action: Used"
	}
	ebitenutil.DebugPrintAt(screen, bonusActionStatusText, 10, uiStartY+uiLineHeight*4)

	primedActionText := "Primed: None"
	if g.primedActionID != "" {
		if actionDef, exists := ActionTable[g.primedActionID]; exists {
			primedActionText = fmt.Sprintf("Primed: %s", actionDef.Name)
		} else {
			primedActionText = fmt.Sprintf("Primed: ??? (%s)", g.primedActionID)
		}
	}
	ebitenutil.DebugPrintAt(screen, primedActionText, 10, uiStartY+uiLineHeight*5)

	statusText := ""
	if g.Player.IsDisengaging {
		statusText = "Status: Disengaging"
	}
	if statusText != "" {
		ebitenutil.DebugPrintAt(screen, statusText, 10, uiStartY+uiLineHeight*6)
	}

	if g.InputMode == InputModeActionSelect {
		menuW, menuH := screenWidth/2, screenHeight/2+20
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 20, B: 30, A: 220}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, color.White, false)

		title := "Select Action ([Tab] / [Esc] to Cancel)"
		titleX := menuX + 10
		titleY := menuY + 15
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, color.White)

		itemStartY := titleY + 25
		itemLineHeight := 18
		for i, actionDef := range g.availableActions {
			actionText := actionDef.Name

			resourceText := ""
			classDef := ClassDefinitions[g.Player.Class]
			if classDef != nil {
				if classAction, ok := classDef.ClassActions[actionDef.ID]; ok {
					if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
						resourceText = fmt.Sprintf(" (%d/%d)", g.Player.ClassResources[actionDef.ID], classAction.UsesPerRest)
					}
				}
			}
			actionText += resourceText

			var itemColor color.Color = color.Gray{Y: 180}
			if i == g.selectedActionIndex {
				actionText = "> " + actionText
				itemColor = color.White
			}

			itemX := menuX + 15
			itemY := itemStartY + (i * itemLineHeight)
			text.Draw(screen, actionText, basicfont.Face7x13, itemX, itemY, itemColor)
		}
	}

	if g.InputMode == InputModeCharacterSheet {
		menuW, menuH := screenWidth/2+40, screenHeight/2+60
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 30, G: 20, B: 20, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, color.White, false)

		title := fmt.Sprintf("%s - Level %d %s ([C] / [Esc] to Close)", g.Player.Name, g.Player.Level, g.Player.Class)
		titleX := menuX + 10
		titleY := menuY + 15
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, color.White)

		infoStartY := titleY + 25
		infoLineHeight := 14
		col1X := menuX + 15
		col2X := menuX + menuW/2

		lineNum := 0
		text.Draw(screen, fmt.Sprintf("HP: %d / %d", max(0, g.Player.HP), g.Player.MaxHP), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("AC: %d", g.Player.AC), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("Movement: %d", g.Player.MaxMovementPoints), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("Prof Bonus: +%d", g.Player.ProficiencyBonus), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("Hit Dice: %d / %d (d%d)", g.Player.HitDice, g.Player.MaxHitDice, ClassDefinitions[g.Player.Class].HitDieSize), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		lineNum++

		meleeMod := getModifier(g.Player.Strength)
		rangedMod := getModifier(g.Player.Dexterity)
		text.Draw(screen, fmt.Sprintf("Melee Atk: +%d (1d8%+d)", g.Player.ProficiencyBonus+meleeMod, meleeMod), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("Ranged Atk: +%d (1d6%+d)", g.Player.ProficiencyBonus+rangedMod, rangedMod), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		lineNum++

		text.Draw(screen, "Class Features:", basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), color.Gray{Y: 200})
		lineNum++
		classDef := ClassDefinitions[g.Player.Class]
		if classDef != nil {
			featureIDs := make([]string, 0, len(classDef.ClassActions))
			for id := range classDef.ClassActions {
				featureIDs = append(featureIDs, id)
			}
			sort.Slice(featureIDs, func(i, j int) bool {
				ca1 := classDef.ClassActions[featureIDs[i]]
				ca2 := classDef.ClassActions[featureIDs[j]]
				if ca1.RequiredLevel != ca2.RequiredLevel {
					return ca1.RequiredLevel < ca2.RequiredLevel
				}
				return ca1.Name < ca2.Name
			})

			for _, id := range featureIDs {
				classAction := classDef.ClassActions[id]
				if g.Player.Level >= classAction.RequiredLevel {
					featureText := fmt.Sprintf(" L%d: %s", classAction.RequiredLevel, classAction.Name)
					if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
						featureText += fmt.Sprintf(" (%d/%d)", g.Player.ClassResources[id], classAction.UsesPerRest)
						if classAction.RefreshesOn == RestTypeNever {
							featureText += " (Per Lvl)"
						} else {
							// Indicate short/long rest if needed later
						}
					}
					text.Draw(screen, featureText, basicfont.Face7x13, col1X+5, infoStartY+(lineNum*infoLineHeight), color.White)
					lineNum++
				}
			}
		}

		lineNum = 0
		text.Draw(screen, "Attributes:", basicfont.Face7x13, col2X, infoStartY+(lineNum*infoLineHeight), color.Gray{Y: 200})
		lineNum++
		text.Draw(screen, fmt.Sprintf("STR: %d (%+d)", g.Player.Strength, getModifier(g.Player.Strength)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("DEX: %d (%+d)", g.Player.Dexterity, getModifier(g.Player.Dexterity)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("CON: %d (%+d)", g.Player.Constitution, getModifier(g.Player.Constitution)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("INT: %d (%+d)", g.Player.Intelligence, getModifier(g.Player.Intelligence)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("WIS: %d (%+d)", g.Player.Wisdom, getModifier(g.Player.Wisdom)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, fmt.Sprintf("CHA: %d (%+d)", g.Player.Charisma, getModifier(g.Player.Charisma)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++

	}

	logLineHeight := 13
	logStartY := screenHeight - (combatLogLength * logLineHeight) - 10
	logX := 10
	for i, msg := range g.CombatLog {
		isRestPrompt := g.InputMode == InputModeRestPrompt && i == len(g.CombatLog)-1 && msg == fmt.Sprintf("Wave Cleared! Spend 1 Hit Die (of %d) to heal? [Y/N]", g.Player.HitDice)
		isLevelUpMsg := g.InputMode == InputModeLevelUp && i == len(g.CombatLog)-1 && msg == fmt.Sprintf("LEVEL UP! Reached Level %d!", g.Player.Level)

		var msgColor color.Color = color.White
		if isRestPrompt || isLevelUpMsg {
			msgColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
		}
		text.Draw(screen, msg, basicfont.Face7x13, logX, logStartY+(i*logLineHeight), msgColor)
	}

	if g.InputMode == InputModeLevelUp {
		menuW, menuH := screenWidth/2, screenHeight/4
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 30, B: 20, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, color.White, false)

		levelUpTitle := fmt.Sprintf("Level %d Reached!", g.Player.Level)
		titleBounds := text.BoundString(basicfont.Face7x13, levelUpTitle)
		titleX := menuX + (menuW-titleBounds.Dx())/2
		titleY := menuY + 20
		text.Draw(screen, levelUpTitle, basicfont.Face7x13, titleX, titleY, color.White)

		summaryText := "Check Character Sheet [C] for details."
		summaryBounds := text.BoundString(basicfont.Face7x13, summaryText)
		summaryX := menuX + (menuW-summaryBounds.Dx())/2
		summaryY := titleY + 25
		text.Draw(screen, summaryText, basicfont.Face7x13, summaryX, summaryY, color.Gray{Y: 200})

		continueMsg := "Press [Enter] to Continue"
		continueBounds := text.BoundString(basicfont.Face7x13, continueMsg)
		continueX := menuX + (menuW-continueBounds.Dx())/2
		continueY := menuY + menuH - 30
		text.Draw(screen, continueMsg, basicfont.Face7x13, continueX, continueY, color.White)
	}

	if g.CurrentTurn == GameOver {
		gameOverMsg := "GAME OVER"
		isVictory := false
		if g.Player.HP > 0 {
			if g.CurrentWaveIndex >= len(g.WaveDefinitions)-1 {
				isVictory = true
			}
		}

		if isVictory {
			gameOverMsg = "VICTORY!"
		}

		if g.InputMode != InputModeLevelUp {
			msgFont := basicfont.Face7x13
			bounds := text.BoundString(msgFont, gameOverMsg)
			msgX := (screenWidth - bounds.Dx()) / 2
			msgY := (screenHeight - bounds.Dy()) / 2
			text.Draw(screen, gameOverMsg, msgFont, msgX+1, msgY+1, color.Black)
			text.Draw(screen, gameOverMsg, msgFont, msgX, msgY, color.White)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minF(a, b float32) float32 {
	if a < b {
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
	ebiten.SetWindowTitle("Slumb Gate - Class System PoC")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
