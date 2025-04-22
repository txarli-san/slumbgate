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
	playerBaseMovement = 5 // Represents 25ft if 1 tile = 5ft
	enemyBaseMovement  = 4 // Represents 20ft
	playerRangedRange  = 5 // In tiles
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
	ActionTaken       bool // Tracks if the main Action slot has been used
	IsDisengaging     bool // Tracks if Disengage action was used this turn
}

type Enemy struct {
	Entity
	MovementPoints    int
	MaxMovementPoints int
	ActionAvailable   bool // Tracks if the enemy can still take an action this turn
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
	availableActions    []*Action // Dynamically populated list of actions the player can take NOW
	selectedActionIndex int
	primedActionID      string // ID of the action selected (e.g., Melee), ready for targeting
}

// --- Game Data ---

// Define available player actions (static list, availability checked dynamically)
// Defined once globally
var playerActionList = []*Action{
	{ID: "melee_attack", Name: "Melee Attack"},
	{ID: "ranged_attack", Name: "Ranged Attack"},
	{ID: "dash", Name: "Dash"},
	{ID: "disengage", Name: "Disengage"},
	{ID: "wait", Name: "Wait (End Turn)"}, // Clarified name
	// {ID: "dodge", Name: "Dodge (N/A)"}, // Example for future
	// {ID: "help", Name: "Help (N/A)"}, // Example for future
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
	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2, HP: playerMaxHP, MaxHP: playerMaxHP, AC: 13,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Sprite: playerSprite, Name: "Player",
		},
		ProficiencyBonus:  2,
		MaxMovementPoints: playerBaseMovement,
		// MovementPoints, ActionTaken, IsDisengaging are set in resetPlayerTurnState
	}

	// Create Enemy Sprite & Spawn Enemies
	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)
	g.spawnEnemy(2, 2, "Slime 1", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement)
	g.spawnEnemy(mapWidth-3, mapHeight-3, "Slime 2", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement)

	g.CurrentTurn = PlayerTurn
	g.resetPlayerTurnState() // Initialize player state for the first turn
	return g
}

// resetPlayerTurnState resets state for the start of the player's turn.
// Called when transitioning from EnemyTurn to PlayerTurn.
func (g *Game) resetPlayerTurnState() {
	g.Player.MovementPoints = g.Player.MaxMovementPoints
	g.Player.ActionTaken = false
	g.Player.IsDisengaging = false
	g.InputMode = InputModeMap
	g.primedActionID = ""
	fmt.Println("Player turn state reset.")
}

// spawnEnemy creates a new enemy.
func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move int) {
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
			Sprite: g.EnemySprite, Name: name,
		},
		MaxMovementPoints: move,
		// MovementPoints and ActionAvailable are reset at the start of enemy turn/phase
	}
	g.Enemies = append(g.Enemies, enemy)
}

// addCombatLog adds a message to the log.
func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
	fmt.Println("CombatLog:", msg) // Also print to console for debugging
}

// isTileBlocked checks if a tile is occupied by a living unit.
// movingEnemyIndex is used to allow an enemy to move *from* its own square. Pass -1 if checking for player.
func (g *Game) isTileBlocked(x, y, movingEnemyIndex int) bool {
	// Check player position
	if g.Player.HP > 0 && g.Player.X == x && g.Player.Y == y {
		return true
	}
	// Check enemy positions
	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 { // Ignore dead enemies
			continue
		}
		if i == movingEnemyIndex { // Ignore the enemy doing the moving
			continue
		}
		if enemy.X == x && enemy.Y == y {
			return true
		}
	}
	return false
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

// isAdjacent checks Manhattan distance == 1.
func isAdjacent(x1, y1, x2, y2 int) bool {
	return distance(x1, y1, x2, y2) == 1
}

// isPlayerAdjacentToEnemy checks if the player is adjacent to any living enemy.
func isPlayerAdjacentToEnemy(g *Game) bool {
	for _, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			return true // Found at least one adjacent enemy
		}
	}
	return false // No adjacent enemies found
}

// resolveAttack handles attack logic for different types (melee/ranged).
// Returns true if the target was killed by the attack.
func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int, attackType string) bool {
	if attacker.HP <= 0 || defender.HP <= 0 {
		return false // Can't attack or be attacked if dead
	}

	var attackAbilityMod int
	var abilityName string
	switch attackType {
	case "ranged":
		attackAbilityMod = getModifier(attacker.Dexterity)
		abilityName = "DEX"
	case "melee":
		fallthrough // Default to Strength for melee
	default:
		attackAbilityMod = getModifier(attacker.Strength)
		abilityName = "STR"
	}

	roll := rand.Intn(20) + 1
	attackRoll := roll + attackerProfBonus + attackAbilityMod
	hit := attackRoll >= defender.AC

	// Build log message components
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
		// Calculate damage (using the same ability mod as the attack for simplicity, min 1)
		damage := max(1, attackAbilityMod) // Basic damage = ability mod, min 1
		// TODO: Implement weapon damage dice later (e.g., 1d8 + STR mod)
		defender.HP -= damage
		logMsg += fmt.Sprintf(" Hit! Deals %d damage.", damage)
		if defender.HP <= 0 {
			logMsg += fmt.Sprintf(" %s dies!", defender.Name)
			g.addCombatLog(logMsg)
			return true // Target died
		}
	} else {
		logMsg += " Miss!"
	}
	g.addCombatLog(logMsg)
	return false // Target survived
}

// --- Player Turn Ending Logic ---
// Centralized function to handle the transition from Player to Enemy turn or Game Over
func (g *Game) endPlayerTurn() {
	fmt.Println("Ending player turn...")
	g.Player.ActionTaken = true // Ensure action is marked used if ending turn manually
	g.cleanupDeadEnemies()      // Cleanup any enemies killed during the turn
	if len(g.Enemies) == 0 {    // Check win condition
		g.addCombatLog("All enemies defeated! VICTORY!")
		g.CurrentTurn = GameOver
	} else {
		g.CurrentTurn = EnemyTurn // Transition to enemy turn
	}
	g.primedActionID = ""      // Clear any primed action when turn ends
	g.InputMode = InputModeMap // Ensure back to map mode
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
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.selectedActionIndex++
			if g.selectedActionIndex >= len(g.availableActions) {
				g.selectedActionIndex = 0
			}
		}

		// Select Action (Enter Key)
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if g.selectedActionIndex >= 0 && g.selectedActionIndex < len(g.availableActions) {
				selectedAction := g.availableActions[g.selectedActionIndex]
				closeMenu := true // Assume menu closes unless action needs targeting

				switch selectedAction.ID {
				case "wait":
					g.addCombatLog("Player ends turn (Wait).")
					g.endPlayerTurn() // Use the centralized end turn function
					fmt.Println("[Input] Wait Action selected and executed. Turn ended.")

				case "dash":
					g.addCombatLog("Player uses Dash!")
					g.Player.ActionTaken = true // Mark action as used
					g.Player.MovementPoints += g.Player.MaxMovementPoints
					fmt.Println("[Input] Dash Action selected and executed.")

				case "disengage":
					g.addCombatLog("Player uses Disengage!")
					g.Player.ActionTaken = true   // Mark action as used
					g.Player.IsDisengaging = true // Set the status for this turn
					fmt.Println("[Input] Disengage Action selected and executed.")

				// Actions that require targeting are primed:
				case "melee_attack", "ranged_attack":
					g.primedActionID = selectedAction.ID
					fmt.Printf("[Input] Action Primed: %s\n", g.primedActionID)
					closeMenu = true // Close menu, wait for click target

				default:
					g.addCombatLog(fmt.Sprintf("Action '%s' not fully implemented.", selectedAction.Name))
					closeMenu = true // Close menu even for unimplemented actions
				}

				// Close menu if needed and clear priming if action wasn't one that primes
				if closeMenu {
					g.InputMode = InputModeMap
					// Clear priming ONLY if the action taken wasn't melee/ranged
					if selectedAction.ID != "melee_attack" && selectedAction.ID != "ranged_attack" {
						g.primedActionID = ""
					}
				}
			}
		} // End Enter Key Check

		// Close menu without selection (Tab or Escape)
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fmt.Println("[Input] Closing Action Select Menu (No Selection).")
			g.InputMode = InputModeMap
			g.primedActionID = "" // Ensure no action is primed if menu is closed manually
		}
		return // Prevent other inputs while menu is open

	case InputModeMap:
		// --- Input Handling when in Normal Map Mode ---

		// 1. Open Action Select Menu with Tab (if action available)
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			if !g.Player.ActionTaken {
				fmt.Println("[Input] Opening Action Select Menu.")
				g.availableActions = []*Action{} // Start fresh each time menu opens
				for _, action := range playerActionList {
					isAvailable := true
					if action.ID == "disengage" && !isPlayerAdjacentToEnemy(g) {
						isAvailable = false
					}
					// Add more checks here...
					if isAvailable {
						g.availableActions = append(g.availableActions, action)
					}
				}
				if len(g.availableActions) > 0 {
					g.selectedActionIndex = 0
					g.InputMode = InputModeActionSelect
					g.primedActionID = ""
					return // Stop processing for this frame
				} else {
					g.addCombatLog("No actions available!")
				}
			} else {
				g.addCombatLog("Action already taken this turn.")
			}
		}

		// 2. Action Execution (via Left Mouse Click if action is primed)
		actionExecutedByClick := false
		if g.primedActionID != "" && !g.Player.ActionTaken {
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				cursorX, cursorY := ebiten.CursorPosition()
				gridX := (cursorX - g.MapOffsetX) / tileSize
				gridY := (cursorY - g.MapOffsetY) / tileSize

				if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight {
					fmt.Printf("[Exec] Left Click at (%d, %d) while action '%s' is primed.\n", gridX, gridY, g.primedActionID)
					switch g.primedActionID {
					case "melee_attack":
						targetEnemy := g.getEnemyAt(gridX, gridY)
						if targetEnemy != nil && isAdjacent(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y) {
							fmt.Printf("[Exec] Executing Primed Melee Attack on %s\n", targetEnemy.Name)
							g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "melee")
							g.Player.ActionTaken = true
							actionExecutedByClick = true
						} else {
							g.addCombatLog("Invalid target for melee attack.")
						}
					case "ranged_attack":
						targetEnemy := g.getEnemyAt(gridX, gridY)
						if targetEnemy != nil {
							dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
							if dist <= playerRangedRange {
								fmt.Printf("[Exec] Firing at %s\n", targetEnemy.Name)
								g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "ranged")
								g.Player.ActionTaken = true
								actionExecutedByClick = true
							} else {
								g.addCombatLog(fmt.Sprintf("Target %s out of range.", targetEnemy.Name))
							}
						} else {
							g.addCombatLog("No target selected at cursor.")
						}
					default:
						g.addCombatLog(fmt.Sprintf("Unknown primed action '%s'.", g.primedActionID))
					}
					g.primedActionID = "" // Clear priming after attempt
				} else {
					g.addCombatLog("Clicked outside map.")
					g.primedActionID = ""
				}
			} // End if Left Mouse Click
		} // End if Primed Action

		// If an action was just executed via click, stop processing further input for this frame
		if actionExecutedByClick {
			return
		}

		// 3. End Turn Manually with Enter (if nothing is primed) - NEW
		if g.primedActionID == "" && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			fmt.Println("[Input] Enter pressed to end turn.")
			g.addCombatLog("Player ends turn.")
			g.endPlayerTurn() // Use the centralized end turn function
			return            // Stop processing for this frame as turn has ended
		}

		// 4. Movement Input (Arrow Keys)
		if g.Player.MovementPoints > 0 {
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
						// --- Opportunity Attack (AoO) Check ---
						performAoOCheck := !g.Player.IsDisengaging
						if performAoOCheck {
							for _, enemy := range g.Enemies {
								if enemy.HP <= 0 {
									continue
								}
								wasAdj := isAdjacent(startX, startY, enemy.X, enemy.Y)
								willBeAdj := isAdjacent(targetX, targetY, enemy.X, enemy.Y)
								if wasAdj && !willBeAdj {
									if g.Player.HP > 0 {
										g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
										killedByAoO := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
										if killedByAoO {
											g.Player.MovementPoints-- // Consume point for attempt
											return                    // Exit input handling
										}
									}
								}
							} // End AoO loop
						} // --- End AoO Check Block ---

						// Complete the move if alive
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
			} // End if moved
		} // End Movement check
	} // End InputModeMap case
}

// handleEnemyTurns processes enemy actions using simple Move then Action logic.
func (g *Game) handleEnemyTurns() {
	fmt.Println("--- Start Enemy Turn Processing ---")
	if g.Player.HP <= 0 {
		fmt.Println("Player is dead, skipping enemy turns.")
		return // Skip enemy turns if player is already dead
	}

	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}

		// Reset enemy state for their turn/action phase
		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		// fmt.Printf("Processing %s (HP: %d, Pos: %d,%d, Move: %d, Action: %t)\n", enemy.Name, enemy.HP, enemy.X, enemy.Y, enemy.MovementPoints, enemy.ActionAvailable) // Verbose

		// --- Enemy AI Logic ---
		// 1. Movement Phase
		movedThisTurn := false // FIX: Declare movedThisTurn here
		for enemy.MovementPoints > 0 && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			targetX, targetY := enemy.X, enemy.Y
			dx := g.Player.X - enemy.X
			dy := g.Player.Y - enemy.Y
			movedStep := false
			tempTargetX := targetX
			tempTargetY := targetY

			if math.Abs(float64(dx)) > math.Abs(float64(dy)) { // Try X first
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
				} // Revert if blocked
			}
			if !movedStep || math.Abs(float64(dy)) >= math.Abs(float64(dx)) { // Try Y if X didn't move or Y is >= X dist
				tempTargetY = targetY // Reset Y for check
				if dy > 0 {
					tempTargetY++
				} else {
					tempTargetY--
				}
				if tempTargetY >= 0 && tempTargetY < mapHeight && !g.isTileBlocked(targetX, tempTargetY, i) { // Use potentially updated targetX
					targetY = tempTargetY
					movedStep = true
				}
			}

			if movedStep && (targetX != enemy.X || targetY != enemy.Y) {
				enemy.X = targetX
				enemy.Y = targetY
				enemy.MovementPoints--
				movedThisTurn = true // FIX: Assign value here
			} else {
				break
			} // No valid step
		}
		// Log movement outcome (less verbose) - FIX: Uncommented lines using movedThisTurn
		if !movedThisTurn && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			fmt.Printf("  %s could not move closer.\n", enemy.Name)
		}
		// else if enemy.MovementPoints == 0 && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) { fmt.Printf("  %s ran out of movement.\n", enemy.Name) }

		// 2. Action Phase
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) && enemy.ActionAvailable {
			if g.Player.HP > 0 {
				fmt.Printf("  %s attacks Player.\n", enemy.Name)
				killedPlayer := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
				enemy.ActionAvailable = false
				if killedPlayer {
					fmt.Println("  Player killed by enemy attack!")
					// Game over state checked in Update loop
				}
			} else {
				enemy.ActionAvailable = false
			} // Player dead, consume action
		} else if enemy.ActionAvailable {
			enemy.ActionAvailable = false
		} // Consume action if unused

	} // End loop through enemies
	fmt.Println("--- End Enemy Turn Processing ---")
}

// cleanupDeadEnemies removes enemies with HP <= 0 from the game slice.
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
		g.Enemies = aliveEnemies
	}
}

// Update proceeds the game state by one tick.
func (g *Game) Update() error {
	// Skip all updates if game is over
	if g.CurrentTurn == GameOver {
		// TODO: Add logic for restarting the game? (e.g., press R)
		return nil
	}

	// Store turn state before input handling
	turnBeforeInput := g.CurrentTurn

	// Handle input processing (actions, movement)
	// This might change the CurrentTurn state if "Wait" or manual End Turn (Enter) is triggered.
	if g.CurrentTurn == PlayerTurn || g.InputMode == InputModeActionSelect {
		g.handlePlayerInput()
	}

	// --- Post-Input Turn Logic ---
	// Only run enemy turn logic if the state *was* EnemyTurn before input handling
	// OR if the state *became* EnemyTurn during input handling (via Wait/Enter)
	if turnBeforeInput == EnemyTurn || g.CurrentTurn == EnemyTurn {
		// Check if it's *still* EnemyTurn (might have become GameOver via Wait/Enter)
		if g.CurrentTurn == EnemyTurn {
			g.handleEnemyTurns()

			// Check player death *after* all enemies have acted
			if g.Player.HP <= 0 {
				if g.CurrentTurn != GameOver { // Avoid double logging
					g.addCombatLog("Player has died! Game Over.")
					g.CurrentTurn = GameOver
				}
			} else {
				// If player survived, switch back to Player turn and reset state
				fmt.Println("Enemy turn finished, starting player turn.")
				g.CurrentTurn = PlayerTurn
				g.resetPlayerTurnState() // Reset for the start of the player's turn
			}
		}
	}
	// Note: The transition PlayerTurn -> EnemyTurn is now handled within endPlayerTurn(),
	// which is called by the 'Wait' action or the manual Enter key press.

	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	mapOffsetX, mapOffsetY := g.MapOffsetX, g.MapOffsetY

	// --- Draw Map ---
	tileOpts := &ebiten.DrawImageOptions{}
	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			screenX, screenY := float64(mapOffsetX+x*tileSize), float64(mapOffsetY+y*tileSize)
			tileOpts.GeoM.Reset()
			tileOpts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(g.TileImage, tileOpts)
		}
	}

	// --- Draw Range Overlay (If Ranged Attack is Primed) ---
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

	// --- Draw Entities ---
	// Draw Enemies First
	for _, enemy := range g.Enemies {
		enemyScreenX, enemyScreenY := float64(mapOffsetX+enemy.X*tileSize), float64(mapOffsetY+enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
		hpText := fmt.Sprintf("%d/%d", enemy.HP, enemy.MaxHP)
		ebitenutil.DebugPrintAt(screen, hpText, int(enemyScreenX)+4, int(enemyScreenY)+tileSize+2)
	}

	// Draw Player
	if g.Player.HP > 0 {
		playerScreenX, playerScreenY := float64(mapOffsetX+g.Player.X*tileSize), float64(mapOffsetY+g.Player.Y*tileSize)
		g.Player.DrawOpts.GeoM.Reset()
		g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
		screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)
	}

	// --- Draw UI ---
	uiStartY := 10
	uiLineHeight := 15
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrintAt(screen, turnText, 10, uiStartY)
	playerInfoText := "Player (Warrior)"
	ebitenutil.DebugPrintAt(screen, playerInfoText, 10, uiStartY+uiLineHeight)
	playerHpVal := g.Player.HP
	if playerHpVal < 0 {
		playerHpVal = 0
	}
	playerStatsText := fmt.Sprintf("HP: %d/%d AC: %d", playerHpVal, g.Player.MaxHP, g.Player.AC)
	ebitenutil.DebugPrintAt(screen, playerStatsText, 10, uiStartY+uiLineHeight*2)
	playerMoveText := fmt.Sprintf("Move: %d/%d", g.Player.MovementPoints, g.Player.MaxMovementPoints)
	ebitenutil.DebugPrintAt(screen, playerMoveText, 10, uiStartY+uiLineHeight*3)
	actionStatusText := "Action: Available"
	if g.Player.ActionTaken {
		actionStatusText = "Action: Used"
	}
	ebitenutil.DebugPrintAt(screen, actionStatusText, 10, uiStartY+uiLineHeight*4)
	primedActionText := fmt.Sprintf("Primed: %s", g.primedActionID)
	if g.primedActionID == "" {
		primedActionText = "Primed: None"
	}
	ebitenutil.DebugPrintAt(screen, primedActionText, 10, uiStartY+uiLineHeight*5)
	statusText := ""
	if g.Player.IsDisengaging {
		statusText = "Status: Disengaging"
	}
	if statusText != "" {
		ebitenutil.DebugPrintAt(screen, statusText, 10, uiStartY+uiLineHeight*6)
	}

	// --- Draw Action Select Menu (If active) ---
	if g.InputMode == InputModeActionSelect {
		menuX, menuY := screenWidth/4, screenHeight/4
		menuW, menuH := screenWidth/2, screenHeight/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 20, B: 30, A: 220}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, color.White, false)
		title := "Select Action ([Up/Down], [Enter], [Esc]/[Tab])"
		titleX := menuX + 10
		titleY := menuY + 20
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, color.White)
		itemStartY := titleY + 25
		itemLineHeight := 18
		for i, action := range g.availableActions {
			actionText := action.Name
			var itemColor color.Color = color.Gray{Y: 180}
			if i == g.selectedActionIndex {
				actionText = "> " + actionText
				itemColor = color.White
			}
			itemX := menuX + 15
			itemY := itemStartY + (i * itemLineHeight)
			text.Draw(screen, actionText, basicfont.Face7x13, itemX, itemY, itemColor)
		}
	} // End Action Select Menu Draw

	// --- Combat Log ---
	logStartY := screenHeight - (combatLogLength * 15) - 10
	logX := 10
	for i, msg := range g.CombatLog {
		text.Draw(screen, msg, basicfont.Face7x13, logX, logStartY+(i*15), color.White)
	}

	// --- Draw Game Over / Victory Message ---
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

// Helper function: max returns the greater of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Layout specifies the logical screen size, used by Ebitengine.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Main Function ---
func main() {
	rand.Seed(time.Now().UnixNano())
	game := NewGame()
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Slumb Gate MVP - Dash/Disengage Implemented")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
