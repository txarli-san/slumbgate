package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text" // Keep import for title/game over text
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont" // Keep import for title/game over text
)

// --- Constants ---

const (
	screenWidth        = 640
	screenHeight       = 480
	tileSize           = 32
	mapWidth           = 10
	mapHeight          = 10
	combatLogLength    = 7
	playerBaseMovement = 5
	enemyBaseMovement  = 4
	playerRangedRange  = 5
)

// --- Types ---

type TurnState int

const (
	PlayerTurn TurnState = iota
	EnemyTurn
	GameOver
)

// String method for TurnState
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

// InputMode manages UI states like showing menus
type InputMode int

const (
	InputModeMap InputMode = iota
	InputModeActionSelect
)

// Action struct definition
type Action struct {
	ID   string // Internal identifier
	Name string // Display name for UI
	// TODO: Add IsAvailable func, Execute func, TargetType, Range etc. later
}

// --- Structs ---

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
}

type Enemy struct {
	Entity
	MovementPoints    int
	MaxMovementPoints int
	ActionAvailable   bool
}

type Game struct {
	Player           *Player
	Enemies          []*Enemy
	GameMap          [mapWidth][mapHeight]int
	TileImage        *ebiten.Image
	EnemySprite      *ebiten.Image
	CurrentTurn      TurnState
	CombatLog        []string
	MapOffsetX       int
	MapOffsetY       int
	RangeOverlayTile *ebiten.Image
	InputMode        InputMode
	// Action Menu State
	availableActions    []*Action
	selectedActionIndex int
	primedActionID      string // ID of the action selected, ready for execution
}

// --- Game Data ---

// Define available player actions (static for now)
// Defined once globally
var playerActionList = []*Action{
	{ID: "melee_attack", Name: "1. Melee Attack"},
	{ID: "ranged_attack", Name: "2. Ranged Attack"},
	{ID: "wait", Name: "3. Wait"},
	{ID: "dash", Name: "4. Dash (N/A)"},           // Placeholder
	{ID: "disengage", Name: "5. Disengage (N/A)"}, // Placeholder
}

// --- Game Logic ---

// getModifier calculates the D&D ability modifier.
func getModifier(score int) int { return (score - 10) / 2 }

// NewGame initializes the game state.
func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)
	g.MapOffsetX = (screenWidth - (mapWidth * tileSize)) / 2
	g.MapOffsetY = (screenHeight - (mapHeight * tileSize)) / 2
	g.InputMode = InputModeMap              // Start in normal map mode
	g.availableActions = make([]*Action, 0) // Initialize empty

	// Create Tile Sprites
	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)
	g.RangeOverlayTile = ebiten.NewImage(tileSize, tileSize)
	overlayColor := color.NRGBA{R: 0, G: 100, B: 200, A: 80}
	vector.DrawFilledRect(g.RangeOverlayTile, 0, 0, float32(tileSize), float32(tileSize), overlayColor, false)

	// Create Player
	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)
	playerStr := 15
	playerDex := 14
	playerCon := 13
	playerInt := 8
	playerWis := 12
	playerCha := 10
	playerConMod := getModifier(playerCon)
	playerMaxHP := 10 + playerConMod
	g.Player = &Player{Entity: Entity{X: mapWidth / 2, Y: mapHeight / 2, HP: playerMaxHP, MaxHP: playerMaxHP, AC: 13, Strength: playerStr, Dexterity: playerDex, Constitution: playerCon, Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha, Sprite: playerSprite, Name: "Player"}, ProficiencyBonus: 2, MaxMovementPoints: playerBaseMovement}

	// Create Enemy Sprite & Spawn Enemies
	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)
	g.spawnEnemy(2, 2, "Enemy 0", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement)
	g.spawnEnemy(mapWidth-3, mapHeight-3, "Enemy 1", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement)

	g.CurrentTurn = PlayerTurn
	g.resetPlayerTurnState()
	return g
}

// resetPlayerTurnState resets state for the start of the player's turn.
func (g *Game) resetPlayerTurnState() {
	g.Player.MovementPoints = g.Player.MaxMovementPoints
	g.Player.ActionTaken = false
	g.InputMode = InputModeMap // Ensure player starts turn in map mode
	g.primedActionID = ""      // Clear any primed action
}

// spawnEnemy creates a new enemy.
func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move int) {
	conMod := getModifier(con)
	maxHp := baseHp + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{Entity: Entity{X: x, Y: y, HP: maxHp, MaxHP: maxHp, AC: ac, Strength: str, Dexterity: dex, Constitution: con, Intelligence: intel, Wisdom: wis, Charisma: cha, Sprite: g.EnemySprite, Name: name}, MaxMovementPoints: move}
	g.Enemies = append(g.Enemies, enemy)
}

// addCombatLog adds a message to the log.
func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
	fmt.Println("CombatLog:", msg)
}

// isTileBlocked checks if a tile is occupied by a living unit.
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

// findAdjacentEnemy finds the first living adjacent enemy.
func (g *Game) findAdjacentEnemy() *Enemy {
	for _, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			return enemy
		}
	}
	return nil
}

// getEnemyAt returns the living enemy at the specified grid coordinates, or nil.
func (g *Game) getEnemyAt(x, y int) *Enemy {
	for _, enemy := range g.Enemies {
		if enemy.HP > 0 && enemy.X == x && enemy.Y == y {
			return enemy
		}
	}
	return nil
}

// distance calculates Manhattan distance between two grid points.
func distance(x1, y1, x2, y2 int) int {
	return int(math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2)))
}

// resolveAttack handles attack logic for different types (melee/ranged).
func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int, attackType string) bool {
	var attackAbilityMod int
	var abilityName string
	switch attackType {
	case "ranged":
		attackAbilityMod = getModifier(attacker.Dexterity)
		abilityName = "DEX"
	case "melee":
		fallthrough
	default:
		attackAbilityMod = getModifier(attacker.Strength)
		abilityName = "STR"
	}
	roll := rand.Intn(20) + 1
	attackRoll := roll + attackerProfBonus + attackAbilityMod
	hit := attackRoll >= defender.AC
	var modString string
	if attackAbilityMod >= 0 {
		modString = fmt.Sprintf("+%d", attackAbilityMod)
	} else {
		modString = fmt.Sprintf("%d", attackAbilityMod)
	}
	var profString string
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
		damage := max(1, attackAbilityMod)
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

// isAdjacent checks Manhattan distance.
func isAdjacent(x1, y1, x2, y2 int) bool {
	dx := x1 - x2
	dy := y1 - y2
	return math.Abs(float64(dx))+math.Abs(float64(dy)) == 1
}

// handlePlayerInput processes player inputs based on the current InputMode.
func (g *Game) handlePlayerInput() {
	switch g.InputMode {
	case InputModeActionSelect:
		// --- Input Handling when Action Select Menu is OPEN ---
		// Navigate Menu (Up/Down Arrows)
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.selectedActionIndex--
			if g.selectedActionIndex < 0 {
				g.selectedActionIndex = len(g.availableActions) - 1
			}
			fmt.Printf("[Input] Menu Up. Selected Index: %d\n", g.selectedActionIndex)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.selectedActionIndex++
			if g.selectedActionIndex >= len(g.availableActions) {
				g.selectedActionIndex = 0
			}
			fmt.Printf("[Input] Menu Down. Selected Index: %d\n", g.selectedActionIndex)
		}

		// Select Action (Enter Key)
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if g.selectedActionIndex >= 0 && g.selectedActionIndex < len(g.availableActions) {
				selectedAction := g.availableActions[g.selectedActionIndex]
				// TODO: Add check here if action IsAvailable before priming

				// Handle "Wait" action immediately
				if selectedAction.ID == "wait" {
					g.addCombatLog("Player ends turn (Wait).")
					g.Player.ActionTaken = true // Wait consumes the action
					g.primedActionID = ""       // No action needs to be primed
					g.InputMode = InputModeMap  // Close menu
					fmt.Println("[Input] Wait Action selected and executed.")
				} else {
					// Prime other actions for execution later
					g.primedActionID = selectedAction.ID
					fmt.Printf("[Input] Action Primed: %s\n", g.primedActionID)
					g.InputMode = InputModeMap // Close menu after selection
				}
			}
		} // End Enter Key Check

		// Close menu without selection (Tab or Escape)
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fmt.Println("[Input] Closing Action Select Menu (No Selection).")
			g.InputMode = InputModeMap
			g.primedActionID = ""
		}
		return // Prevent other inputs while menu is open

	case InputModeMap:
		// --- Input Handling when in Normal Map Mode ---

		// Open Action Select Menu with Tab
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			if !g.Player.ActionTaken {
				fmt.Println("[Input] Opening Action Select Menu.")
				g.availableActions = playerActionList // Use static list for now
				g.selectedActionIndex = 0
				g.InputMode = InputModeActionSelect
				g.primedActionID = "" // Clear any previously primed action
				return                // Stop processing other inputs for this frame
			} else {
				g.addCombatLog("Action already taken this turn.")
			}
		}

		// --- Action Execution (via Left Mouse Click if action is primed) ---
		actionExecuted := false // Flag to prevent movement processing if action taken
		if g.primedActionID != "" && !g.Player.ActionTaken {
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				cursorX, cursorY := ebiten.CursorPosition()
				gridX := (cursorX - g.MapOffsetX) / tileSize
				gridY := (cursorY - g.MapOffsetY) / tileSize

				if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight { // Check if click is on the map
					fmt.Printf("[Exec] Left Click at (%d, %d) while action '%s' is primed.\n", gridX, gridY, g.primedActionID)

					switch g.primedActionID {
					case "melee_attack":
						targetEnemy := g.getEnemyAt(gridX, gridY)
						if targetEnemy != nil && isAdjacent(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y) {
							fmt.Printf("[Exec] Executing Primed Melee Attack on %s\n", targetEnemy.Name)
							g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "melee")
							g.Player.ActionTaken = true
							actionExecuted = true
						} else {
							g.addCombatLog("Invalid target for melee attack (must click adjacent enemy).")
						}
						g.primedActionID = "" // Clear after attempt

					case "ranged_attack":
						targetEnemy := g.getEnemyAt(gridX, gridY)
						if targetEnemy != nil {
							dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
							if dist <= playerRangedRange {
								fmt.Printf("[Exec] Firing at %s\n", targetEnemy.Name)
								g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "ranged")
								g.Player.ActionTaken = true
								actionExecuted = true
							} else {
								g.addCombatLog(fmt.Sprintf("Target %s out of range (%d > %d)", targetEnemy.Name, dist, playerRangedRange))
							}
						} else {
							g.addCombatLog("No target selected at cursor.")
						}
						g.primedActionID = "" // Clear after attempt

					case "dash", "disengage":
						g.addCombatLog(fmt.Sprintf("Action '%s' selected but not implemented yet.", g.primedActionID))
						g.primedActionID = "" // Clear unimplemented action

					default:
						g.primedActionID = "" // Clear unknown primed action
					} // End switch
				} else {
					g.addCombatLog("Clicked outside map.")
					g.primedActionID = ""
				} // Clear action if clicked outside map
			} // End if Left Mouse Click
		} // End if Primed Action

		// If an action was just executed via click, stop processing input for this frame
		if actionExecuted {
			return
		}

		// --- Movement Input (Arrow Keys) ---
		// Allow movement only if action hasn't been taken this turn
		if !g.Player.ActionTaken && g.Player.MovementPoints > 0 {
			moved := false
			startX, startY := g.Player.X, g.Player.Y
			targetX, targetY := startX, startY
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				targetY--
				moved = true
			} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				targetY++
				moved = true
			} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				targetX--
				moved = true
			} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				targetX++
				moved = true
			}
			if moved {
				if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
					if !g.isTileBlocked(targetX, targetY, -1) {
						// AoO Check
						for _, enemy := range g.Enemies {
							if enemy.HP <= 0 {
								continue
							}
							wasAdj := isAdjacent(startX, startY, enemy.X, enemy.Y)
							willBeAdj := isAdjacent(targetX, targetY, enemy.X, enemy.Y)
							if wasAdj && !willBeAdj {
								if g.Player.HP > 0 {
									g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
									g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
								}
							}
						}
						// Update Position & Movement if alive
						if g.Player.HP > 0 {
							g.Player.X = targetX
							g.Player.Y = targetY
							g.Player.MovementPoints--
						} else {
							g.Player.MovementPoints--
						}
					}
				}
			}
		} // End Movement check
	} // End InputModeMap case
}

// handleEnemyTurns processes enemy actions using Move then Action logic.
func (g *Game) handleEnemyTurns() {
	fmt.Println("--- Start Enemy Turn Processing ---")
	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if g.Player.HP <= 0 {
			continue
		}
		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		fmt.Printf("Processing %s (HP: %d, Move: %d, Action: %t)\n", enemy.Name, enemy.HP, enemy.MovementPoints, enemy.ActionAvailable)

		// Movement Phase
		movedThisTurn := false
		for enemy.MovementPoints > 0 && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			fmt.Printf("  %s has %d move points, attempting step.\n", enemy.Name, enemy.MovementPoints)
			targetX, targetY := enemy.X, enemy.Y
			dx := g.Player.X - enemy.X
			dy := g.Player.Y - enemy.Y
			if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
				if dx > 0 {
					targetX++
				} else {
					targetX--
				}
			} else {
				if dy > 0 {
					targetY++
				} else {
					targetY--
				}
			}
			if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
				if !g.isTileBlocked(targetX, targetY, i) {
					fmt.Printf("  %s moving to (%d, %d).\n", enemy.Name, targetX, targetY)
					enemy.X = targetX
					enemy.Y = targetY
					enemy.MovementPoints--
					movedThisTurn = true
				} else {
					fmt.Printf("  %s move to (%d, %d) blocked.\n", enemy.Name, targetX, targetY)
					break
				}
			} else {
				fmt.Printf("  %s move to (%d, %d) out of bounds.\n", enemy.Name, targetX, targetY)
				break
			}
		}
		if !movedThisTurn && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			fmt.Printf("  %s could not move (blocked or no path).\n", enemy.Name)
		} else if enemy.MovementPoints == 0 && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			fmt.Printf("  %s ran out of movement points.\n", enemy.Name)
		}

		// Action Phase - Enemy only does melee attack for now
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) && enemy.ActionAvailable {
			fmt.Printf("  %s is adjacent, attempting melee attack.\n", enemy.Name)
			g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee") // Enemy uses Str, 0 prof, melee type
			enemy.ActionAvailable = false
		} else if enemy.ActionAvailable {
			fmt.Printf("  %s is not adjacent or has no target, action available but unused.\n", enemy.Name)
		}
		fmt.Printf("Finished processing %s (Move left: %d, Action Available: %t)\n", enemy.Name, enemy.MovementPoints, enemy.ActionAvailable)
	}
	fmt.Println("--- End Enemy Turn Processing ---")
}

// cleanupDeadEnemies removes dead enemies.
func (g *Game) cleanupDeadEnemies() {
	initialCount := len(g.Enemies)
	aliveEnemies := make([]*Enemy, 0, len(g.Enemies))
	for _, enemy := range g.Enemies {
		if enemy.HP > 0 {
			aliveEnemies = append(aliveEnemies, enemy)
		}
	}
	if len(aliveEnemies) != initialCount {
		fmt.Printf("[INFO] Cleanup removed %d dead enemies.\n", initialCount-len(aliveEnemies))
	}
	g.Enemies = aliveEnemies
}

// Update proceeds the game state by one tick.
func (g *Game) Update() error {
	// Handle input only during Player's turn OR if menu is open (to allow closing it)
	// Also skip input processing if game is over
	if g.CurrentTurn == PlayerTurn || g.InputMode == InputModeActionSelect {
		if g.CurrentTurn != GameOver { // Don't process input if game is over
			g.handlePlayerInput()
		}
	}

	// --- Turn Logic ---
	// Only advance turns if not in a menu and not game over
	if g.InputMode == InputModeMap && g.CurrentTurn != GameOver {
		previousTurn := g.CurrentTurn
		switch g.CurrentTurn {
		case PlayerTurn:
			// Turn ends *only* if the player took their main Action
			if g.Player.ActionTaken {
				g.CurrentTurn = EnemyTurn
			}
		case EnemyTurn:
			g.handleEnemyTurns()
			g.cleanupDeadEnemies() // Cleanup after enemy actions
			if g.Player.HP <= 0 {  // Check player death
				if g.CurrentTurn != GameOver {
					g.addCombatLog("Player has died! Game Over.")
					g.CurrentTurn = GameOver
				}
			} else {
				// Switch back and reset player state for new turn
				g.CurrentTurn = PlayerTurn
				g.resetPlayerTurnState()
			}
		}
		// Cleanup enemies killed by Player *after* Player turn ends
		if previousTurn == PlayerTurn && g.CurrentTurn == EnemyTurn {
			g.cleanupDeadEnemies()
		}
	} // End turn logic block

	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	mapOffsetX, mapOffsetY := g.MapOffsetX, g.MapOffsetY

	// Draw Map
	tileOpts := &ebiten.DrawImageOptions{}
	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			screenX, screenY := float64(mapOffsetX+x*tileSize), float64(mapOffsetY+y*tileSize)
			tileOpts.GeoM.Reset()
			tileOpts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(g.TileImage, tileOpts)
		}
	}

	// Draw Range Overlay (If Ranged Attack is Primed)
	if g.CurrentTurn == PlayerTurn && g.primedActionID == "ranged_attack" {
		overlayOpts := &ebiten.DrawImageOptions{}
		for x := 0; x < mapWidth; x++ {
			for y := 0; y < mapHeight; y++ {
				dist := distance(g.Player.X, g.Player.Y, x, y)
				if dist > 0 && dist <= playerRangedRange {
					screenX := float64(mapOffsetX + x*tileSize)
					screenY := float64(mapOffsetY + y*tileSize)
					overlayOpts.GeoM.Reset()
					overlayOpts.GeoM.Translate(screenX, screenY)
					screen.DrawImage(g.RangeOverlayTile, overlayOpts)
				}
			}
		}
	}

	// Draw Enemies
	for _, enemy := range g.Enemies {
		enemyScreenX, enemyScreenY := float64(mapOffsetX+enemy.X*tileSize), float64(mapOffsetY+enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
		hpText := fmt.Sprintf("%d/%d", enemy.HP, enemy.MaxHP)
		ebitenutil.DebugPrintAt(screen, hpText, int(enemyScreenX), int(enemyScreenY)-12)
	}

	// Draw Player
	playerScreenX, playerScreenY := float64(mapOffsetX+g.Player.X*tileSize), float64(mapOffsetY+g.Player.Y*tileSize)
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)

	// --- Draw UI ---
	uiStartY := 10
	uiLineHeight := 15
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrintAt(screen, turnText, 10, uiStartY)
	playerInfoText := "Lvl: 1 Fighter (Defense Style)"
	ebitenutil.DebugPrintAt(screen, playerInfoText, 10, uiStartY+uiLineHeight)
	playerHpVal := g.Player.HP
	if playerHpVal < 0 {
		playerHpVal = 0
	}
	playerStatsText := fmt.Sprintf("HP: %d/%d AC: %d Prof: +%d", playerHpVal, g.Player.MaxHP, g.Player.AC, g.Player.ProficiencyBonus)
	ebitenutil.DebugPrintAt(screen, playerStatsText, 10, uiStartY+uiLineHeight*2)
	playerMoveText := fmt.Sprintf("Move: %d/%d", g.Player.MovementPoints, g.Player.MaxMovementPoints)
	ebitenutil.DebugPrintAt(screen, playerMoveText, 10, uiStartY+uiLineHeight*3)
	actionStatusText := "Action: Available"
	if g.Player.ActionTaken {
		actionStatusText = "Action: Used"
	}
	ebitenutil.DebugPrintAt(screen, actionStatusText, 10, uiStartY+uiLineHeight*4)
	p := g.Player.Entity
	scoresText1 := fmt.Sprintf("STR:%d(%+d) DEX:%d(%+d) CON:%d(%+d)", p.Strength, getModifier(p.Strength), p.Dexterity, getModifier(p.Dexterity), p.Constitution, getModifier(p.Constitution))
	scoresText2 := fmt.Sprintf("INT:%d(%+d) WIS:%d(%+d) CHA:%d(%+d)", p.Intelligence, getModifier(p.Intelligence), p.Wisdom, getModifier(p.Wisdom), p.Charisma, getModifier(p.Charisma))
	ebitenutil.DebugPrintAt(screen, scoresText1, 10, uiStartY+uiLineHeight*5)
	ebitenutil.DebugPrintAt(screen, scoresText2, 10, uiStartY+uiLineHeight*6)
	primedActionText := fmt.Sprintf("Action: %s", g.primedActionID)
	if g.primedActionID == "" {
		primedActionText = "Action: None"
	}
	ebitenutil.DebugPrintAt(screen, primedActionText, 10, uiStartY+uiLineHeight*7)

	// --- Draw Action Select Menu ---
	if g.InputMode == InputModeActionSelect {
		menuX, menuY := screenWidth/4, screenHeight/4
		menuW, menuH := screenWidth/2, screenHeight/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 20, B: 30, A: 200}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 1, color.White, false)

		// Draw Title using text.Draw (less likely to panic)
		title := "Select Action ([Tab]/[Esc] Close, [Enter] Select)"
		text.Draw(screen, title, basicfont.Face7x13, menuX+10, menuY+20, color.White)

		// Draw Action List using ebitenutil.DebugPrintAt
		itemY := menuY + 40
		itemLineHeight := 15 // Approximate line height for DebugPrint
		for i, action := range g.availableActions {
			actionText := action.Name
			if i == g.selectedActionIndex {
				actionText = "> " + actionText // Simple highlight prefix
			}
			// Use DebugPrintAt instead of text.Draw to avoid potential panic
			ebitenutil.DebugPrintAt(screen, actionText, menuX+10, itemY+(i*itemLineHeight))
		}
	} // End Action Select Menu Draw

	// Combat Log
	logStartY := screenHeight - (combatLogLength * 15) - 10
	for i, msg := range g.CombatLog {
		ebitenutil.DebugPrintAt(screen, msg, 10, logStartY+(i*15))
	}

	// Draw Game Over
	if g.CurrentTurn == GameOver {
		gameOverMsg := "GAME OVER"
		bounds := text.BoundString(basicfont.Face7x13, gameOverMsg)
		msgX := (screenWidth - bounds.Dx()) / 2
		msgY := screenHeight / 2
		text.Draw(screen, gameOverMsg, basicfont.Face7x13, msgX+1, msgY+1, color.Black)
		text.Draw(screen, gameOverMsg, basicfont.Face7x13, msgX, msgY, color.White)
	}
}

// Helper function: max returns the greater of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Layout specifies the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) { return screenWidth, screenHeight }

// --- Main Function ---
func main() {
	rand.Seed(time.Now().UnixNano())
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP - Debug Action Menu") // Title reflects debug state
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
