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
	// InputModeLevelUpChoice // Future mode
)

// --- Entity & Character Structs ---
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
	ProficiencyBonus  int
	MovementPoints    int
	MaxMovementPoints int
	ActionTaken       bool // TODO: Replace with ActionsAvailable for Action Surge
	IsDisengaging     bool
	// TODO: Add Equipment later
	// TODO: Add HitDiceAvailable, HitDiceMax, ShortRestsAvailable later
}

type Enemy struct {
	Entity
	MovementPoints    int
	MaxMovementPoints int
	ActionAvailable   bool
	AttackType        string
	MaxRange          int
}

// --- Game State & Actions ---
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
	Targeting      TargetType
	Range          int
	RequiresTarget bool
	Execute        ActionExecuteFunc
}

// --- Wave & Enemy Definitions ---
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
	// TODO: Add Short Rest action later
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
	g.CurrentWaveIndex = -1

	g.WaveDefinitions = []WaveDefinition{
		{
			EnemiesToSpawn: []EnemySpawnInfo{
				{TypeName: "Small Slime", SpawnPointIdx: 0},
				{TypeName: "Small Slime", SpawnPointIdx: 1},
			},
			IsBossWave: false,
		},
		{
			EnemiesToSpawn: []EnemySpawnInfo{
				{TypeName: "Tough Slime", SpawnPointIdx: 2},
				{TypeName: "Tough Slime", SpawnPointIdx: 3},
			},
			IsBossWave: false,
		},
		{
			EnemiesToSpawn: []EnemySpawnInfo{
				{TypeName: "Tough Slime", SpawnPointIdx: 0},
				{TypeName: "Goo Spitter", SpawnPointIdx: 4},
				{TypeName: "Tough Slime", SpawnPointIdx: 1},
			},
			IsBossWave: false,
		},
		{
			EnemiesToSpawn: []EnemySpawnInfo{
				{TypeName: "Big Slime Boss", SpawnPointIdx: 4},
			},
			IsBossWave: true,
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
	playerMaxHP := 10 + playerConMod
	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2, Width: 1, Height: 1,
			HP: playerMaxHP, MaxHP: playerMaxHP, AC: 13,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Sprite: playerSprite, Name: "Player",
		},
		Level:             1,
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
	// TODO: Reset Action Surge availability based on rest mechanic later
}

func (g *Game) endPlayerTurn() {
	g.Player.ActionTaken = true
	g.cleanupDeadEnemies()

	if len(g.Enemies) == 0 {
		currentWaveDef := g.WaveDefinitions[g.CurrentWaveIndex]
		isFinalWave := g.CurrentWaveIndex+1 >= len(g.WaveDefinitions)

		if currentWaveDef.IsBossWave && isFinalWave {
			// TODO: Initiate Long Rest / Level Up sequence here
			g.addCombatLog("Final Boss Defeated! Long Rest...")
			g.addCombatLog("VICTORY!")
			g.CurrentTurn = GameOver
		} else if !isFinalWave {
			g.SpawnNextWave()
			if len(g.Enemies) > 0 {
				g.startEnemyTurn()
			} else {
				g.addCombatLog("Error spawning next wave or wave empty.")
				g.CurrentTurn = GameOver
			}
		} else {
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
		currentWaveDef := g.WaveDefinitions[g.CurrentWaveIndex]
		isFinalWave := g.CurrentWaveIndex+1 >= len(g.WaveDefinitions)

		if currentWaveDef.IsBossWave && isFinalWave {
			// TODO: Initiate Long Rest / Level Up sequence here
			g.addCombatLog("Final Boss Defeated! Long Rest...")
			g.addCombatLog("VICTORY!")
			g.CurrentTurn = GameOver
		} else if !isFinalWave {
			g.SpawnNextWave()
			if len(g.Enemies) > 0 {
				g.startPlayerTurn()
			} else {
				g.addCombatLog("Error spawning next wave or wave empty.")
				g.CurrentTurn = GameOver
			}
		} else {
			g.addCombatLog("All defined waves cleared! VICTORY!")
			g.CurrentTurn = GameOver
		}
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
	if g.Player.HP > 0 && g.Player.X == checkX && g.Player.Y == checkY {
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
	rollString := fmt.Sprintf("%d%s%s(%s)=%d", roll, profString, modString, abilityName, attackRoll)
	attackVerb := "attacks"
	if attackType == "ranged" {
		attackVerb = "shoots"
	}
	logMsg := fmt.Sprintf("%s %s %s(AC%d). %s.", attacker.Name, attackVerb, defender.Name, defender.AC, rollString)

	if hit {
		var damage int
		var damageRoll int = 0
		damageLog := ""

		isPlayer := attacker == &g.Player.Entity

		if isPlayer {
			switch attackType {
			case "ranged":
				damageRoll = rand.Intn(6) + 1
				damage = damageRoll + attackAbilityMod
				damageLog = fmt.Sprintf(" (%d%+d)", damageRoll, attackAbilityMod)
			default: // Melee
				damageRoll = rand.Intn(8) + 1
				damage = damageRoll + attackAbilityMod
				damageLog = fmt.Sprintf(" (%d%+d)", damageRoll, attackAbilityMod)
			}
		} else {
			// TODO: Implement enemy damage dice later
			damage = attackAbilityMod
			damageLog = fmt.Sprintf(" (%+d)", attackAbilityMod)
		}

		damage = max(1, damage)

		defender.HP -= damage
		logMsg += fmt.Sprintf(" Hit! %d%s dmg.", damage, damageLog)
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
					if !g.isTileBlocked(targetX, targetY, -1) {
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
					g.addCombatLog("Cannot move outside map.")
				}
			}
		}
	}
}

func (g *Game) handleEnemyTurns() {
	if g.Player.HP <= 0 {
		return
	}

	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}

	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}

		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		distToPlayer := distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
		isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

		// --- Action Phase ---
		if enemy.ActionAvailable {
			if enemy.AttackType == "ranged" {
				// TODO: Add Line of Sight check
				if distToPlayer <= enemy.MaxRange && distToPlayer > 1 {
					g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
					enemy.ActionAvailable = false
				} else if isAdj {
				}
			} else {
				if isAdj {
					g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
					enemy.ActionAvailable = false
				}
			}
		}

		// --- Movement Phase ---
		needsToMove := enemy.ActionAvailable || (enemy.AttackType == "ranged" && (distToPlayer < 2 || distToPlayer > enemy.MaxRange))

		if needsToMove {
			for enemy.MovementPoints > 0 {
				currentX, currentY := enemy.X, enemy.Y
				currentDist := distance(currentX, currentY, g.Player.X, g.Player.Y)
				bestMoveX, bestMoveY := currentX, currentY

				potentialMoves := []image.Point{}

				for _, offset := range moveOffsets {
					nextX, nextY := currentX+offset.X, currentY+offset.Y
					if !g.isTileFullyBlocked(nextX, nextY, enemy.Width, enemy.Height, i) {
						potentialMoves = append(potentialMoves, image.Point{X: nextX, Y: nextY})
					}
				}

				if len(potentialMoves) == 0 {
					break
				}

				chosenMove := false
				if enemy.AttackType == "ranged" {
					targetDist := enemy.MaxRange
					bestScore := -1.0

					for _, move := range potentialMoves {
						dist := distance(move.X, move.Y, g.Player.X, g.Player.Y)
						score := math.Abs(float64(dist - targetDist))
						if dist <= 1 {
							score += 1000
						}

						if bestScore < 0 || score < bestScore {
							bestScore = score
							bestMoveX = move.X
							bestMoveY = move.Y
							chosenMove = true
						}
					}
					if bestMoveX == currentX && bestMoveY == currentY {
						chosenMove = false
					}

				} else {
					minDist := currentDist
					bestMoves := []image.Point{}

					for _, move := range potentialMoves {
						dist := distance(move.X, move.Y, g.Player.X, g.Player.Y)
						if dist < minDist {
							minDist = dist
						}
					}
					for _, move := range potentialMoves {
						if distance(move.X, move.Y, g.Player.X, g.Player.Y) == minDist {
							bestMoves = append(bestMoves, move)
						}
					}
					if len(bestMoves) > 0 {
						if len(bestMoves) == 1 {
							bestMoveX = bestMoves[0].X
							bestMoveY = bestMoves[0].Y
							chosenMove = true
						} else {
							dx := g.Player.X - currentX
							dy := g.Player.Y - currentY
							preferredMoveFound := false
							if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
								for _, move := range bestMoves {
									if move.X != currentX {
										bestMoveX, bestMoveY, chosenMove, preferredMoveFound = move.X, move.Y, true, true
										break
									}
								}
							} else {
								for _, move := range bestMoves {
									if move.Y != currentY {
										bestMoveX, bestMoveY, chosenMove, preferredMoveFound = move.X, move.Y, true, true
										break
									}
								}
							}
							if !preferredMoveFound {
								bestMoveX, bestMoveY, chosenMove = bestMoves[0].X, bestMoves[0].Y, true
							}
						}
					}
					if bestMoveX == currentX && bestMoveY == currentY {
						chosenMove = false
					}
				}

				if chosenMove {
					enemy.X = bestMoveX
					enemy.Y = bestMoveY
					enemy.MovementPoints--
					isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
					distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)

					if enemy.AttackType == "melee" && isAdj {
						break
					}
					if enemy.AttackType == "ranged" && distToPlayer == enemy.MaxRange {
					}
				} else {
					break
				}
			}
		}

		if enemy.ActionAvailable {
			if enemy.AttackType == "ranged" {
				distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
				if distToPlayer <= enemy.MaxRange && distToPlayer > 1 {
					// TODO: Add Line of Sight check
					g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
					enemy.ActionAvailable = false
				}
			}
			if enemy.ActionAvailable {
				enemy.ActionAvailable = false
			}
		}

		if g.Player.HP <= 0 {
			return
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
	if g.CurrentTurn == GameOver {
		// TODO: Handle restart input?
		return nil
	}

	turnBeforeInput := g.CurrentTurn

	if turnBeforeInput == PlayerTurn || g.InputMode == InputModeActionSelect || g.InputMode == InputModeCharacterSheet {
		g.handlePlayerInput()
	}

	if g.CurrentTurn == EnemyTurn {
		g.handleEnemyTurns()
		g.endEnemyTurn()
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
			entityOpts.GeoM.Scale(float64(entity.Width), float64(entity.Height))
		}
		entityOpts.GeoM.Translate(entityScreenX, entityScreenY)
		screen.DrawImage(entity.Sprite, entityOpts)

		hpBarBaseX := float64(mapOffsetX + entity.X*tileSize)
		hpBarBaseY := float64(mapOffsetY + entity.Y*tileSize)
		hpBarX := float32(hpBarBaseX)
		hpBarY := float32(hpBarBaseY + float64(entity.Height*tileSize) + hpBarOffsetY)
		hpBarWidth := float32(tileSize * entity.Width)
		hpRatio := float32(entity.HP) / float32(entity.MaxHP)
		if hpRatio < 0 {
			hpRatio = 0
		}
		if hpRatio > 1 {
			hpRatio = 1
		}
		vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, color.RGBA{R: 80, G: 0, B: 0, A: 255}, false)
		vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth*hpRatio, hpBarHeight, color.RGBA{R: 0, G: 200, B: 0, A: 255}, false)
		vector.StrokeRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, 1, color.Black, false)
	}

	// --- Draw UI ---
	uiStartY := 10
	uiLineHeight := 15
	statusStartY := 10

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Wave: %d", g.CurrentWaveIndex+1), screenWidth-100, statusStartY)

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
	if statusText != "" {
		ebitenutil.DebugPrintAt(screen, statusText, 10, uiStartY+uiLineHeight*5)
	}

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
			var itemColor color.Color = color.Gray{Y: 180}

			if i == g.selectedActionIndex {
				actionText = "> " + actionText
				itemColor = color.White
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

	if g.InputMode == InputModeCharacterSheet {
		menuW, menuH := screenWidth/2, screenHeight/2
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 30, G: 20, B: 20, A: 220}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, color.White, false)

		title := fmt.Sprintf("%s - Character Sheet", g.Player.Name)
		titleX := menuX + 10
		titleY := menuY + 20
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, color.White)

		infoStartY := titleY + 25
		infoLineHeight := 14
		infoX := menuX + 15
		lineNum := 0

		healthStr := fmt.Sprintf("HP: %d / %d", max(0, g.Player.HP), g.Player.MaxHP)
		acStr := fmt.Sprintf("AC: %d", g.Player.AC)
		moveStr := fmt.Sprintf("Movement: %d", g.Player.MaxMovementPoints)
		profStr := fmt.Sprintf("Proficiency Bonus: +%d", g.Player.ProficiencyBonus)
		levelStr := fmt.Sprintf("Level: %d", g.Player.Level)
		text.Draw(screen, levelStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, healthStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, acStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, moveStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, profStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++

		meleeDmgStr := fmt.Sprintf("Melee Damage: 1d8 %+d", getModifier(g.Player.Strength))
		rangedDmgStr := fmt.Sprintf("Ranged Damage: 1d6 %+d", getModifier(g.Player.Dexterity))
		text.Draw(screen, meleeDmgStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		text.Draw(screen, rangedDmgStr, basicfont.Face7x13, infoX, infoStartY+(lineNum*infoLineHeight), color.White)
		lineNum++
		lineNum++

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
		lineNum++

		closeMsg := "Press [C] or [Esc] to close"
		closeY := float32(menuY + menuH - 20)
		closeX := float32(menuX + (menuW-text.BoundString(basicfont.Face7x13, closeMsg).Dx())/2)
		text.Draw(screen, closeMsg, basicfont.Face7x13, int(closeX), int(closeY), color.Gray{Y: 150})
	}

	logLineHeight := 13
	logStartY := screenHeight - (combatLogLength * logLineHeight) - 10
	logX := 10
	for i, msg := range g.CombatLog {
		text.Draw(screen, msg, basicfont.Face7x13, logX, logStartY+(i*logLineHeight), color.White)
	}

	if g.CurrentTurn == GameOver {
		gameOverMsg := "GAME OVER"
		if g.Player.HP > 0 && len(g.Enemies) == 0 {
			isVictory := false
			if g.CurrentWaveIndex >= 0 && g.CurrentWaveIndex < len(g.WaveDefinitions) {
				if g.WaveDefinitions[g.CurrentWaveIndex].IsBossWave && g.CurrentWaveIndex+1 >= len(g.WaveDefinitions) {
					isVictory = true
				} else if g.CurrentWaveIndex+1 >= len(g.WaveDefinitions) {
					isVictory = true
				}
			} else if g.CurrentWaveIndex == -1 && len(g.Enemies) == 0 {
				isVictory = true
			}

			if isVictory {
				gameOverMsg = "VICTORY!"
			} else {
				gameOverMsg = "GAME OVER?"
			}
		}

		msgFont := basicfont.Face7x13
		bounds := text.BoundString(msgFont, gameOverMsg)
		msgX := (screenWidth - bounds.Dx()) / 2
		msgY := (screenHeight - bounds.Dy()) / 2

		text.Draw(screen, gameOverMsg, msgFont, msgX+1, msgY+1, color.Black)
		text.Draw(screen, gameOverMsg, msgFont, msgX, msgY, color.White)

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
	ebiten.SetWindowTitle("Slumb Gate - Improved IA ranged mobs")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
