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
		return "Unknown Turn State"
	}
}

type InputMode int

const (
	InputModeMap InputMode = iota
	InputModeActionSelect
)

type Entity struct {
	X            int
	Y            int
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
}

type Game struct {
	Player               *Player
	Enemies              []*Enemy
	GameMap              [mapWidth][mapHeight]int
	TileImage            *ebiten.Image
	CurrentTurn          TurnState
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
		// basic floor tile is at (0, 0) on the tile sheet
		floorSprite := getSpriteFromSheet(tileSheet, 0, 0)
		g.TileImage = ebiten.NewImage(tileSize, tileSize)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(spriteScale, spriteScale)
		g.TileImage.DrawImage(floorSprite, opts)
	}

	g.RangeOverlayTile = ebiten.NewImage(tileSize, tileSize)
	overlayColor := color.NRGBA{R: 0, G: 100, B: 200, A: 80}
	vector.DrawFilledRect(g.RangeOverlayTile, 0, 0, float32(tileSize), float32(tileSize), overlayColor, false)

	// Player: "2.b. male fighter" -> on coords (1, 1)
	playerSprite := getSpriteFromSheet(g.rogueSheet, 1, 1)
	playerStr, playerDex, playerCon := 15, 14, 13
	playerInt, playerWis, playerCha := 8, 12, 10
	playerConMod := getModifier(playerCon)
	playerMaxHP := 10 + playerConMod
	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2, HP: playerMaxHP, MaxHP: playerMaxHP, AC: 13,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Sprite: playerSprite, Name: "Player",
		},
		ProficiencyBonus:  2,
		MaxMovementPoints: playerBaseMovement,
	}

	// Enemies: "3.a. small slime" -> on coords (0, 2)
	slimeSprite := getSpriteFromSheet(g.monsterSheet, 0, 2)
	g.spawnEnemy(2, 2, "Slime 1", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement, slimeSprite)
	g.spawnEnemy(mapWidth-3, mapHeight-3, "Slime 2", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement, slimeSprite)

	g.CurrentTurn = PlayerTurn
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
		if g.CurrentTurn != GameOver {
			g.addCombatLog("All enemies defeated! VICTORY!")
			g.CurrentTurn = GameOver
		}
	} else {
		g.startEnemyTurn()
	}
	g.primedActionID = ""
	if g.CurrentTurn != GameOver { // Avoid resetting mode if game just ended
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
	} else {
		g.startPlayerTurn()
	}
}

func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move int, sprite *ebiten.Image) {
	conMod := getModifier(con)
	maxHp := baseHp + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, HP: maxHp, MaxHP: maxHp, AC: ac,
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

func (g *Game) isTileBlocked(x, y, movingEnemyIndex int) bool {
	if g.Player.HP > 0 && g.Player.X == x && g.Player.Y == y {
		return true
	}
	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if i == movingEnemyIndex {
			continue
		}
		if enemy.X == x && enemy.Y == y {
			return true
		}
	}
	return false
}

func (g *Game) getEnemyAt(x, y int) *Enemy {
	for _, enemy := range g.Enemies {
		if enemy.HP > 0 && enemy.X == x && enemy.Y == y {
			return enemy
		}
	}
	return nil
}

func distance(x1, y1, x2, y2 int) int {
	return int(math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2)))
}

func isAdjacent(x1, y1, x2, y2 int) bool {
	return distance(x1, y1, x2, y2) == 1
}

func isPlayerAdjacentToEnemy(g *Game) bool {
	for _, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
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
	rollString := fmt.Sprintf("Roll: %d%s%s(%s) = %d", roll, profString, modString, abilityName, attackRoll)

	attackVerb := "attacks"
	if attackType == "ranged" {
		attackVerb = "shoots"
	}
	logMsg := fmt.Sprintf("%s %s %s (AC %d). %s.", attacker.Name, attackVerb, defender.Name, defender.AC, rollString)

	if hit {
		damage := max(1, attackAbilityMod) // Basic damage
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
	if targetEnemy != nil && isAdjacent(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y) {
		killed := g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "melee")
		g.lastExecutedActionID = "melee_attack"
		if killed {
			g.cleanupDeadEnemies()
			if len(g.Enemies) == 0 {
				g.addCombatLog("All enemies defeated! VICTORY!")
				g.CurrentTurn = GameOver
				return true
			}
		}
		return true
	}
	g.addCombatLog("Invalid target for melee attack.")
	return false
}

func executeRangedAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil {
		dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
		if dist <= playerRangedRange {
			killed := g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "ranged")
			g.lastExecutedActionID = "ranged_attack"
			if killed {
				g.cleanupDeadEnemies()
				if len(g.Enemies) == 0 {
					g.addCombatLog("All enemies defeated! VICTORY!")
					g.CurrentTurn = GameOver
					return true
				}
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

func (g *Game) handlePlayerInput() {
	switch g.InputMode {
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
					// Wait action calls endPlayerTurn which sets ActionTaken
					if success && selectedActionDef.ID != "wait" {
						// TODO: Differentiate action types (Main, Bonus, Reaction) - currently all consume the single 'ActionTaken' flag
						g.Player.ActionTaken = true
					}
					// Mode is handled by Execute for Wait/endPlayerTurn, otherwise reset here
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
					if actionDef.ID == "disengage" && !isPlayerAdjacentToEnemy(g) {
						isAvailable = false
					}
					// TODO: Add more complex availability checks (cost, etc.)

					if isAvailable {
						g.availableActions = append(g.availableActions, actionDef)
					}
				}

				if len(g.availableActions) > 0 {
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
				} else {
					g.addCombatLog("No actions available!")
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
							// TODO: Differentiate action types (Main, Bonus, Reaction) - currently all consume the single 'ActionTaken' flag
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
				g.addCombatLog("Player ends turn (Manual - Wait action missing!).")
				g.endPlayerTurn()
			}
			return
		}

		if g.Player.MovementPoints > 0 {
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
								wasAdj := isAdjacent(startX, startY, enemy.X, enemy.Y)
								if wasAdj && !isAdjacent(targetX, targetY, enemy.X, enemy.Y) {
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

	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}

		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true

		for enemy.MovementPoints > 0 && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			targetX, targetY := enemy.X, enemy.Y
			dx := g.Player.X - enemy.X
			dy := g.Player.Y - enemy.Y
			movedStep := false
			tempTargetX, tempTargetY := targetX, targetY

			if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
				if dx > 0 {
					tempTargetX++
				} else {
					tempTargetX--
				}
				if tempTargetX >= 0 && tempTargetX < mapWidth && !g.isTileBlocked(tempTargetX, tempTargetY, i) {
					targetX = tempTargetX
					movedStep = true
				} else {
					tempTargetX = targetX
				}
			}
			if !movedStep || math.Abs(float64(dy)) >= math.Abs(float64(dx)) {
				tempTargetY = targetY
				if dy > 0 {
					tempTargetY++
				} else {
					tempTargetY--
				}
				if tempTargetY >= 0 && tempTargetY < mapHeight && !g.isTileBlocked(targetX, tempTargetY, i) {
					targetY = tempTargetY
					movedStep = true
				}
			}

			if movedStep && (targetX != enemy.X || targetY != enemy.Y) {
				enemy.X = targetX
				enemy.Y = targetY
				enemy.MovementPoints--
			} else {
				break
			}
		}

		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) && enemy.ActionAvailable {
			if g.Player.HP > 0 {
				g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
				enemy.ActionAvailable = false
			} else {
				enemy.ActionAvailable = false
			}
		} else if enemy.ActionAvailable {
			enemy.ActionAvailable = false
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
		return nil
	}

	turnBeforeInput := g.CurrentTurn

	if turnBeforeInput == PlayerTurn || g.InputMode == InputModeActionSelect {
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

	entityOpts := &ebiten.DrawImageOptions{}

	for _, entity := range entitiesToDraw {
		if entity.Sprite == nil {
			continue
		}

		entityScreenX := float64(mapOffsetX + entity.X*tileSize)
		entityScreenY := float64(mapOffsetY + entity.Y*tileSize)

		entityOpts.GeoM.Reset()
		entityOpts.GeoM.Scale(spriteScale, spriteScale)
		entityOpts.GeoM.Translate(entityScreenX, entityScreenY)
		screen.DrawImage(entity.Sprite, entityOpts)

		hpBarX := float32(entityScreenX)
		hpBarY := float32(entityScreenY + tileSize + hpBarOffsetY)
		hpBarWidth := float32(tileSize)

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

	uiStartY := 10
	uiLineHeight := 15
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Turn: %s", g.CurrentTurn.String()), 10, uiStartY)
	ebitenutil.DebugPrintAt(screen, "Player (Warrior)", 10, uiStartY+uiLineHeight)
	playerHpVal := max(0, g.Player.HP)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP: %d/%d AC: %d", playerHpVal, g.Player.MaxHP, g.Player.AC), 10, uiStartY+uiLineHeight*2)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Move: %d/%d", g.Player.MovementPoints, g.Player.MaxMovementPoints), 10, uiStartY+uiLineHeight*3)
	actionStatusText := "Action: Available"
	if g.Player.ActionTaken {
		actionStatusText = "Action: Used"
	}
	ebitenutil.DebugPrintAt(screen, actionStatusText, 10, uiStartY+uiLineHeight*4)
	primedActionText := "Primed: None"
	if g.primedActionID != "" {
		actionDef, exists := ActionTable[g.primedActionID]
		if exists {
			primedActionText = fmt.Sprintf("Primed: %s", actionDef.Name)
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
			itemColor := color.Color(color.Gray{Y: 180})
			if i == g.selectedActionIndex {
				actionText = "> " + actionText
				itemColor = color.White
			}
			itemX := menuX + 15
			itemY := itemStartY + (i * itemLineHeight)
			text.Draw(screen, actionText, basicfont.Face7x13, itemX, itemY, itemColor)
		}
	}

	logStartY := screenHeight - (combatLogLength * 15) - 10
	logX := 10
	for i, msg := range g.CombatLog {
		text.Draw(screen, msg, basicfont.Face7x13, logX, logStartY+(i*15), color.White)
	}

	if g.CurrentTurn == GameOver {
		gameOverMsg := "GAME OVER"
		if g.Player.HP > 0 && len(g.Enemies) == 0 {
			gameOverMsg = "VICTORY!"
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
	ebiten.SetWindowTitle("Slumb Gate - Visual Polish")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
