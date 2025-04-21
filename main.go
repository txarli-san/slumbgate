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
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
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
	enemyBaseMovement  = 4 // Enemies might be slightly slower
)

// --- Types ---

type TurnState int

const (
	PlayerTurn TurnState = iota
	EnemyTurn
	GameOver
)

// String method for TurnState for easier debugging display
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

// --- Structs ---

type Entity struct {
	X           int
	Y           int
	HP          int
	MaxHP       int
	AC          int
	AttackPower int
	Sprite      *ebiten.Image
	DrawOpts    ebiten.DrawImageOptions
	Name        string
}

type Player struct {
	Entity
	ProficiencyBonus  int
	MovementPoints    int  // Current movement points for the turn
	MaxMovementPoints int  // Maximum movement points per turn
	ActionTaken       bool // Has the player taken their main Action this turn?
}

type Enemy struct {
	Entity
	MovementPoints    int  // Current movement points for the turn
	MaxMovementPoints int  // Maximum movement points per turn
	ActionAvailable   bool // Has the enemy taken its main Action this turn?
}

type Game struct {
	Player      *Player
	Enemies     []*Enemy
	GameMap     [mapWidth][mapHeight]int
	TileImage   *ebiten.Image
	EnemySprite *ebiten.Image
	CurrentTurn TurnState
	CombatLog   []string
}

// --- Game Logic ---

func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)

	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)

	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)
	g.Player = &Player{
		Entity: Entity{
			X:  mapWidth / 2,
			Y:  mapHeight / 2,
			HP: 20, MaxHP: 20, AC: 13, AttackPower: 4, Sprite: playerSprite, Name: "Player",
		},
		ProficiencyBonus:  2,
		MaxMovementPoints: playerBaseMovement,
	}

	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)

	// Spawn enemies with movement points
	g.spawnEnemy(2, 2, "Enemy 0", 8, 10, 3, enemyBaseMovement)
	g.spawnEnemy(mapWidth-3, mapHeight-3, "Enemy 1", 8, 10, 3, enemyBaseMovement)

	g.CurrentTurn = PlayerTurn
	g.resetPlayerTurnState()
	return g
}

// resetPlayerTurnState resets movement points and action status for the player's turn.
func (g *Game) resetPlayerTurnState() {
	g.Player.MovementPoints = g.Player.MaxMovementPoints
	g.Player.ActionTaken = false
}

// spawnEnemy creates a new enemy and adds it to the game's enemy list
func (g *Game) spawnEnemy(x, y int, name string, hp, ac, attackPower, move int) {
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, HP: hp, MaxHP: hp, AC: ac, AttackPower: attackPower, Sprite: g.EnemySprite, Name: name,
		},
		MaxMovementPoints: move,
		// Initial movement/action set at start of their turn processing
	}
	g.Enemies = append(g.Enemies, enemy)
}

// addCombatLog adds a message to the log, keeping it short
func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
	fmt.Println("CombatLog:", msg) // Keep console log for debugging
}

// isTileBlocked checks if a given tile coordinate is occupied by the player or another *living* enemy.
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

// findAdjacentEnemy checks if any *living* enemy is adjacent to the player. Returns the first one found or nil.
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

// resolveAttack handles the logic for an attack attempt. Returns true if defender died.
func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int) bool {
	roll := rand.Intn(20) + 1
	attackRoll := roll + attackerProfBonus
	hit := attackRoll >= defender.AC

	var rollString string
	if attackerProfBonus != 0 {
		rollString = fmt.Sprintf("Roll: %d + %d = %d", roll, attackerProfBonus, attackRoll)
	} else {
		rollString = fmt.Sprintf("Roll: %d", roll)
	}
	logMsg := fmt.Sprintf("%s attacks %s (AC %d). %s.", attacker.Name, defender.Name, defender.AC, rollString)

	if hit {
		damage := attacker.AttackPower
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

// isAdjacent checks Manhattan distance between two points
func isAdjacent(x1, y1, x2, y2 int) bool {
	dx := x1 - x2
	dy := y1 - y2
	return math.Abs(float64(dx))+math.Abs(float64(dy)) == 1
}

// handlePlayerInput processes player actions (Wait, Attack, Move).
func (g *Game) handlePlayerInput() {
	// --- Action Input (Attack/Wait) ---
	if !g.Player.ActionTaken {
		if inpututil.IsKeyJustPressed(ebiten.KeyW) {
			g.addCombatLog("Player waits.")
			g.Player.ActionTaken = true
		}
		if !g.Player.ActionTaken && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			targetEnemy := g.findAdjacentEnemy()
			if targetEnemy != nil {
				g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus)
				g.Player.ActionTaken = true
			} else {
				g.addCombatLog("Player attacks... nothing in range!")
			}
		}
	}

	// --- Movement Input (Arrow Keys) ---
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
					// AoO Check BEFORE moving
					for _, enemy := range g.Enemies {
						if enemy.HP <= 0 {
							continue
						}
						wasAdj := isAdjacent(startX, startY, enemy.X, enemy.Y)
						willBeAdj := isAdjacent(targetX, targetY, enemy.X, enemy.Y)
						if wasAdj && !willBeAdj {
							if g.Player.HP > 0 {
								g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
								g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0)
							}
						}
					}
					// Update Position & Movement Points if alive
					if g.Player.HP > 0 {
						g.Player.X = targetX
						g.Player.Y = targetY
						g.Player.MovementPoints--
					} else {
						// Died from AoO, consume point but don't move
						g.Player.MovementPoints--
					}
				}
			}
		}
	}
}

// handleEnemyTurns processes actions for all *living* enemies using Move + Action economy.
func (g *Game) handleEnemyTurns() {
	fmt.Println("--- Start Enemy Turn Processing ---") // Overall Start
	for i, enemy := range g.Enemies {
		// Skip dead enemies
		if enemy.HP <= 0 {
			continue
		}
		// Skip turn if player is already dead
		if g.Player.HP <= 0 {
			continue
		}

		// Reset enemy state for their turn
		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		fmt.Printf("Processing %s (HP: %d, Move: %d, Action: %t)\n", enemy.Name, enemy.HP, enemy.MovementPoints, enemy.ActionAvailable)

		// --- Movement Phase ---
		movedThisTurn := false // Track if any movement happened
		for enemy.MovementPoints > 0 && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			fmt.Printf("  %s has %d move points, attempting step.\n", enemy.Name, enemy.MovementPoints)
			targetX, targetY := enemy.X, enemy.Y
			dx := g.Player.X - enemy.X
			dy := g.Player.Y - enemy.Y

			// Calculate one step closer
			if math.Abs(float64(dx)) > math.Abs(float64(dy)) { // Move horizontally
				if dx > 0 {
					targetX++
				} else {
					targetX--
				}
			} else { // Move vertically
				if dy > 0 {
					targetY++
				} else {
					targetY--
				}
			}

			// Check validity (bounds, blocked)
			if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
				if !g.isTileBlocked(targetX, targetY, i) {
					fmt.Printf("  %s moving to (%d, %d).\n", enemy.Name, targetX, targetY)
					enemy.X = targetX
					enemy.Y = targetY
					enemy.MovementPoints-- // Consume point
					movedThisTurn = true
				} else {
					fmt.Printf("  %s move to (%d, %d) blocked.\n", enemy.Name, targetX, targetY)
					break // Stop moving if path is blocked
				}
			} else {
				fmt.Printf("  %s move to (%d, %d) out of bounds.\n", enemy.Name, targetX, targetY)
				break // Stop moving if trying to move out of bounds
			}
		} // End movement loop

		// Log outcome of movement phase
		if !movedThisTurn && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			fmt.Printf("  %s could not move (blocked or no path).\n", enemy.Name)
		} else if enemy.MovementPoints == 0 && !isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			fmt.Printf("  %s ran out of movement points.\n", enemy.Name)
		}

		// --- Action Phase ---
		// Check if adjacent (either started adjacent or moved adjacent) AND action is available
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) && enemy.ActionAvailable {
			fmt.Printf("  %s is adjacent, attempting attack.\n", enemy.Name)
			g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0) // Use Action
			enemy.ActionAvailable = false                       // Mark action as used
		} else if enemy.ActionAvailable {
			fmt.Printf("  %s is not adjacent or has no target, action available but unused.\n", enemy.Name)
		}

		fmt.Printf("Finished processing %s (Move left: %d, Action Available: %t)\n", enemy.Name, enemy.MovementPoints, enemy.ActionAvailable)

	} // End loop through enemies
	fmt.Println("--- End Enemy Turn Processing ---") // Overall End
}

// cleanupDeadEnemies removes enemies with HP <= 0 from the game state.
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
	if g.CurrentTurn == GameOver {
		return nil
	}

	previousTurn := g.CurrentTurn

	switch g.CurrentTurn {
	case PlayerTurn:
		g.handlePlayerInput() // Process inputs
		// Turn ends if player took their Action
		if g.Player.ActionTaken {
			g.CurrentTurn = EnemyTurn
		}
	case EnemyTurn:
		g.handleEnemyTurns()
		g.cleanupDeadEnemies() // Cleanup after enemy actions

		if g.Player.HP <= 0 { // Check player death
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

	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	mapOffsetX := (screenWidth - (mapWidth * tileSize)) / 2
	mapOffsetY := (screenHeight - (mapHeight * tileSize)) / 2

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

	// --- Draw Enemies ---
	for _, enemy := range g.Enemies {
		enemyScreenX, enemyScreenY := float64(mapOffsetX+enemy.X*tileSize), float64(mapOffsetY+enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
		hpText := fmt.Sprintf("%d/%d", enemy.HP, enemy.MaxHP)
		ebitenutil.DebugPrintAt(screen, hpText, int(enemyScreenX), int(enemyScreenY)-12)
	}

	// --- Draw Player ---
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

	// Combat Log (at bottom)
	logStartY := screenHeight - (combatLogLength * 15) - 10
	for i, msg := range g.CombatLog {
		ebitenutil.DebugPrintAt(screen, msg, 10, logStartY+(i*15))
	}

	// --- Draw Game Over ---
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

// Layout specifies the logical screen size. Ebitengine handles scaling.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Main Function ---

func main() {
	// Seed the random number generator once at startup
	rand.Seed(time.Now().UnixNano())

	// Initialize and run the game
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP - Enemy Action Economy") // Updated title
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
