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
	screenWidth     = 640
	screenHeight    = 480
	tileSize        = 32
	mapWidth        = 10
	mapHeight       = 10
	combatLogLength = 7 // Increased log length slightly
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
	AttackPower int // Base damage value
	Sprite      *ebiten.Image
	DrawOpts    ebiten.DrawImageOptions
	Name        string
}

type Player struct {
	Entity
	ProficiencyBonus int // Added for SRD alignment
}

type Enemy struct {
	Entity
	// Enemies currently don't use proficiency bonus
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
	// Use constant for combat log size
	g.CombatLog = make([]string, 0, combatLogLength)

	// Initialize the map (all floors)
	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false) // Dark grey floor

	// Create player sprite and initialize stats
	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false) // Green player
	g.Player = &Player{
		Entity: Entity{
			X:     mapWidth / 2,
			Y:     mapHeight / 2,
			HP:    20, // Base HP (Not SRD calculation yet)
			MaxHP: 20,
			// Base AC 12 + 1 from hardcoded "Defense" Fighting Style = 13
			AC:          13,
			AttackPower: 4, // Base damage (Not Str/Dex based yet)
			Sprite:      playerSprite,
			Name:        "Player",
		},
		// Hardcoded Level 1 Proficiency Bonus
		ProficiencyBonus: 2,
	}

	// Create enemy sprite
	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false) // Red enemy

	// Spawn initial enemies with stats
	g.spawnEnemy(2, 2, "Enemy 0", 8, 10, 3)
	g.spawnEnemy(mapWidth-3, mapHeight-3, "Enemy 1", 8, 10, 3)

	g.CurrentTurn = PlayerTurn
	return g
}

// spawnEnemy creates a new enemy and adds it to the game's enemy list
func (g *Game) spawnEnemy(x, y int, name string, hp, ac, attackPower int) {
	enemy := &Enemy{
		Entity: Entity{
			X:           x,
			Y:           y,
			HP:          hp,
			MaxHP:       hp,
			AC:          ac,
			AttackPower: attackPower,
			Sprite:      g.EnemySprite,
			Name:        name,
		},
	}
	g.Enemies = append(g.Enemies, enemy)
}

// addCombatLog adds a message to the log, keeping it short
func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	// Use constant for combat log size
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
	fmt.Println("CombatLog:", msg) // Keep console log for debugging
}

// isTileBlocked checks if a given tile coordinate is occupied by the player or another *living* enemy.
// Pass movingEnemyIndex = -1 for player checks, or the index of the moving enemy.
func (g *Game) isTileBlocked(x, y, movingEnemyIndex int) bool {
	// Check against living player
	if g.Player.HP > 0 && g.Player.X == x && g.Player.Y == y {
		return true
	}
	// Check against other living enemies
	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 { // Skip dead enemies
			continue
		}
		if i == movingEnemyIndex { // Skip self
			continue
		}
		if enemy.X == x && enemy.Y == y { // Check collision
			return true
		}
	}
	return false
}

// findAdjacentEnemy checks if any *living* enemy is adjacent to the player. Returns the first one found or nil.
func (g *Game) findAdjacentEnemy() *Enemy {
	for _, enemy := range g.Enemies {
		if enemy.HP <= 0 { // Skip dead enemies
			continue
		}
		// Check adjacency using helper function
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			return enemy
		}
	}
	return nil
}

// resolveAttack handles the logic for an attack attempt.
// Takes attacker/defender entities and attacker's proficiency bonus. Returns true if defender died.
func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int) bool {
	// Note: We assume checks for attacker/defender being alive happen *before* calling this

	roll := rand.Intn(20) + 1
	attackRoll := roll + attackerProfBonus // Apply proficiency bonus to the roll
	hit := attackRoll >= defender.AC

	// Log message includes proficiency bonus if non-zero
	var rollString string
	if attackerProfBonus != 0 {
		rollString = fmt.Sprintf("Roll: %d + %d = %d", roll, attackerProfBonus, attackRoll)
	} else {
		rollString = fmt.Sprintf("Roll: %d", roll)
	}
	logMsg := fmt.Sprintf("%s attacks %s (AC %d). %s.", attacker.Name, defender.Name, defender.AC, rollString)

	if hit {
		damage := attacker.AttackPower // Using base damage for now
		defender.HP -= damage
		logMsg += fmt.Sprintf(" Hit! Deals %d damage.", damage)
		if defender.HP <= 0 {
			logMsg += fmt.Sprintf(" %s dies!", defender.Name)
			g.addCombatLog(logMsg)
			return true // Defender died this turn
		}
	} else {
		logMsg += " Miss!"
	}
	g.addCombatLog(logMsg)
	return false // Defender survived this attack
}

// isAdjacent checks Manhattan distance between two points
func isAdjacent(x1, y1, x2, y2 int) bool {
	dx := x1 - x2
	dy := y1 - y2
	// Adjacent if distance is exactly 1 horizontally or vertically
	return math.Abs(float64(dx))+math.Abs(float64(dy)) == 1
}

// handlePlayerInput processes player actions (Wait, Attack, Move).
// Handles Attacks of Opportunity triggered by movement.
// Returns true if an action was taken that should end the turn.
func (g *Game) handlePlayerInput() bool {
	actionTaken := false
	startX, startY := g.Player.X, g.Player.Y // Store starting position for AoO checks

	// --- Wait Action (W Key) ---
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.addCombatLog("Player waits.")
		actionTaken = true
	}

	// --- Attack Input (Spacebar) ---
	// Check only if no action already taken this turn
	if !actionTaken && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		targetEnemy := g.findAdjacentEnemy()
		if targetEnemy != nil {
			// Player attacks using their proficiency bonus
			g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus)
			actionTaken = true // Attack is an action
		} else {
			g.addCombatLog("Player attacks... nothing in range!")
			// Attacking empty space doesn't consume turn in this design
		}
	}

	// --- Movement Input (Arrow Keys) ---
	// Check only if no action already taken this turn
	if !actionTaken {
		moved := false
		targetX, targetY := startX, startY // Potential destination

		// Determine target square based on input
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

		// If a move was attempted:
		if moved {
			// 1. Check map bounds and if the destination tile is blocked
			if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
				if !g.isTileBlocked(targetX, targetY, -1) { // Check if destination is valid (not blocked)

					// 2. --- Attack of Opportunity Check ---
					// Check if moving away from adjacent enemies BEFORE updating player position
					for _, enemy := range g.Enemies {
						if enemy.HP <= 0 {
							continue
						} // Skip dead enemies

						// Was enemy adjacent to starting position?
						wasAdj := isAdjacent(startX, startY, enemy.X, enemy.Y)
						// Will enemy be adjacent to ending position?
						willBeAdj := isAdjacent(targetX, targetY, enemy.X, enemy.Y)

						// If was adjacent AND will NOT be adjacent, trigger AoO
						if wasAdj && !willBeAdj {
							// Check if player is still alive before AoO resolves
							if g.Player.HP > 0 {
								g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
								// Resolve AoO immediately (enemy uses 0 prof bonus)
								g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0)
								// Player death state checked later in Update()
							}
						}
					} // End AoO check loop

					// 3. --- Update Player Position ---
					// Only complete the move if the player is still alive after potential AoOs
					if g.Player.HP > 0 {
						g.Player.X = targetX
						g.Player.Y = targetY
						actionTaken = true // Successful move is an action
					} else {
						// Player died from AoO, action still counts as taken (turn ends), but position doesn't update.
						actionTaken = true
					}
				} // End if !isTileBlocked
			} // End if bounds check
		} // End if moved
	} // End if !actionTaken (movement)

	return actionTaken // Return true if Wait, Attack, or Move was successful
}

// handleEnemyTurns processes actions for all *living* enemies.
func (g *Game) handleEnemyTurns() {
	for i, enemy := range g.Enemies {
		// Skip dead enemies
		if enemy.HP <= 0 {
			continue
		}
		// Skip turn if player is already dead (e.g. from AoO)
		if g.Player.HP <= 0 {
			continue
		}

		// Decide action: Attack if adjacent, otherwise move
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			// Attack player (enemy uses 0 prof bonus)
			g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0)
		} else {
			// Move towards player: Calculate target
			targetX, targetY := enemy.X, enemy.Y
			dx := g.Player.X - enemy.X
			dy := g.Player.Y - enemy.Y
			if math.Abs(float64(dx)) > math.Abs(float64(dy)) { // Move horizontally closer
				if dx > 0 {
					targetX++
				} else {
					targetX--
				}
			} else { // Move vertically closer
				if dy > 0 {
					targetY++
				} else {
					targetY--
				}
			}

			// Check validity (bounds, blocked) and move if possible
			if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
				if !g.isTileBlocked(targetX, targetY, i) { // Pass self index 'i'
					enemy.X = targetX
					enemy.Y = targetY
				}
			}
		}
	} // End loop through enemies
}

// cleanupDeadEnemies removes enemies with HP <= 0 from the game state.
func (g *Game) cleanupDeadEnemies() {
	initialCount := len(g.Enemies)
	aliveEnemies := make([]*Enemy, 0, len(g.Enemies)) // Create new slice
	for _, enemy := range g.Enemies {
		if enemy.HP > 0 { // Keep only living enemies
			aliveEnemies = append(aliveEnemies, enemy)
		}
	}
	// Only report if changes were made
	if len(aliveEnemies) != initialCount {
		fmt.Printf("[INFO] Cleanup removed %d dead enemies.\n", initialCount-len(aliveEnemies))
	}
	g.Enemies = aliveEnemies // Replace old slice with the filtered one
}

// Update proceeds the game state by one tick.
func (g *Game) Update() error {
	// If game is over, freeze updates (except maybe restart input)
	if g.CurrentTurn == GameOver {
		// Check for restart key? e.g. if inpututil.IsKeyJustPressed(ebiten.KeyR) { g = NewGame() }
		return nil
	}

	previousTurn := g.CurrentTurn // Store state before actions

	// Process turn based on current state
	switch g.CurrentTurn {
	case PlayerTurn:
		if g.handlePlayerInput() { // Process player action (Move, Attack, Wait), includes AoO checks
			g.CurrentTurn = EnemyTurn // End player turn if action was taken
		}
		// If no action taken, turn remains PlayerTurn
	case EnemyTurn:
		g.handleEnemyTurns()   // Process all enemy actions
		g.cleanupDeadEnemies() // Remove enemies killed by other enemies during their turn (rare case now)

		// Check player death AFTER enemy actions and cleanup
		if g.Player.HP <= 0 {
			if g.CurrentTurn != GameOver { // Prevent multiple logs/state changes
				g.addCombatLog("Player has died! Game Over.")
				g.CurrentTurn = GameOver
			}
		} else {
			// If player survived, switch back to player turn
			g.CurrentTurn = PlayerTurn
		}
	}

	// Cleanup enemies killed by the PLAYER *after* the player turn ends
	// (including those killed by AoOs during player move)
	// This ensures they are removed before the enemy turn starts.
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
			screenX := float64(mapOffsetX + x*tileSize)
			screenY := float64(mapOffsetY + y*tileSize)
			tileOpts.GeoM.Reset()
			tileOpts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(g.TileImage, tileOpts)
		}
	}

	// --- Draw Enemies --- (g.Enemies slice should only contain living ones)
	for _, enemy := range g.Enemies {
		enemyScreenX := float64(mapOffsetX + enemy.X*tileSize)
		enemyScreenY := float64(mapOffsetY + enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
		// Display HP above enemy
		hpText := fmt.Sprintf("%d/%d", enemy.HP, enemy.MaxHP)
		ebitenutil.DebugPrintAt(screen, hpText, int(enemyScreenX), int(enemyScreenY)-12)
	}

	// --- Draw Player ---
	playerScreenX := float64(mapOffsetX + g.Player.X*tileSize)
	playerScreenY := float64(mapOffsetY + g.Player.Y*tileSize)
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	// Optionally change sprite or add effect if HP <= 0
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)

	// --- Draw UI ---
	uiStartY := 10
	uiLineHeight := 15

	// Turn state
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrintAt(screen, turnText, 10, uiStartY)

	// Player Info Block
	playerInfoText := "Lvl: 1 Fighter (Defense Style)"
	ebitenutil.DebugPrintAt(screen, playerInfoText, 10, uiStartY+uiLineHeight)

	playerHpVal := g.Player.HP
	if playerHpVal < 0 {
		playerHpVal = 0
	} // Clamp display value
	playerStatsText := fmt.Sprintf("HP: %d/%d AC: %d Prof: +%d",
		playerHpVal,
		g.Player.MaxHP,
		g.Player.AC,
		g.Player.ProficiencyBonus)
	ebitenutil.DebugPrintAt(screen, playerStatsText, 10, uiStartY+uiLineHeight*2)

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
		// Draw slightly offset for basic shadow effect
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
	ebiten.SetWindowTitle("Slumb Gate MVP - AoO Implemented") // Updated title
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
