package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand" // For dice rolls
	"time"      // For seeding random

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// --- Constants ---

const (
	screenWidth  = 640
	screenHeight = 480
	tileSize     = 32
	mapWidth     = 10
	mapHeight    = 10
)

// --- Types ---

type TurnState int

const (
	PlayerTurn TurnState = iota
	EnemyTurn
	// TODO: Add GameOver state later
)

func (ts TurnState) String() string {
	switch ts {
	case PlayerTurn:
		return "Player Turn"
	case EnemyTurn:
		return "Enemy Turn"
	default:
		return "Unknown Turn State"
	}
}

// --- Structs ---

// Entity represents common properties for player and enemies
type Entity struct {
	X           int
	Y           int
	HP          int // Hit Points
	MaxHP       int // Max Hit Points
	AC          int // Armor Class
	AttackPower int // Damage dealt on hit
	Sprite      *ebiten.Image
	DrawOpts    ebiten.DrawImageOptions
}

// Player specific data (if any needed beyond Entity)
type Player struct {
	Entity
}

// Enemy specific data (if any needed beyond Entity)
type Enemy struct {
	Entity
	// TODO: Add AI state etc.
}

type Game struct {
	Player      *Player
	Enemies     []*Enemy
	GameMap     [mapWidth][mapHeight]int
	TileImage   *ebiten.Image
	EnemySprite *ebiten.Image
	CurrentTurn TurnState
	CombatLog   []string // Simple log for combat messages
}

// --- Game Logic ---

func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, 5) // Pre-allocate capacity for a few messages

	// Create floor tile sprite
	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)

	// Create player sprite and initialize stats
	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)
	g.Player = &Player{
		Entity: Entity{
			X:           mapWidth / 2,
			Y:           mapHeight / 2,
			HP:          20, // Player stats
			MaxHP:       20,
			AC:          12,
			AttackPower: 4,
			Sprite:      playerSprite,
		},
	}

	// Create enemy sprite
	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)

	// Spawn initial enemies with stats
	g.spawnEnemy(2, 2)
	g.spawnEnemy(mapWidth-3, mapHeight-3)

	g.CurrentTurn = PlayerTurn
	return g
}

func (g *Game) spawnEnemy(x, y int) {
	enemy := &Enemy{
		Entity: Entity{
			X:           x,
			Y:           y,
			HP:          8, // Enemy stats
			MaxHP:       8,
			AC:          10,
			AttackPower: 3,
			Sprite:      g.EnemySprite,
		},
	}
	g.Enemies = append(g.Enemies, enemy)
}

// addCombatLog adds a message to the log, keeping it short
func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > 5 { // Keep only the last 5 messages
		g.CombatLog = g.CombatLog[len(g.CombatLog)-5:]
	}
	fmt.Println("CombatLog:", msg) // Also print to console for debugging
}

// isTileBlocked checks for player or other enemies. Pass movingEnemyIndex = -1 for player checks.
func (g *Game) isTileBlocked(x, y, movingEnemyIndex int) bool {
	if g.Player.X == x && g.Player.Y == y {
		return true
	}
	for i, enemy := range g.Enemies {
		if i == movingEnemyIndex {
			continue
		}
		if enemy.X == x && enemy.Y == y {
			return true
		}
	}
	return false
}

// findAdjacentEnemy checks if any enemy is adjacent to the player. Returns the first one found or nil.
func (g *Game) findAdjacentEnemy() *Enemy {
	for _, enemy := range g.Enemies {
		dx := enemy.X - g.Player.X
		dy := enemy.Y - g.Player.Y
		// Check adjacency (Manhattan distance == 1)
		if math.Abs(float64(dx))+math.Abs(float64(dy)) == 1 {
			return enemy
		}
	}
	return nil
}

// resolveAttack handles the logic for an attack attempt.
func (g *Game) resolveAttack(attackerName string, attackerPower, defenderAC int, defender *Entity) {
	roll := rand.Intn(20) + 1 // d20 roll
	hit := roll >= defenderAC

	logMsg := fmt.Sprintf("%s attacks %s (AC %d). Roll: %d.", attackerName, "Defender", defenderAC, roll) // Generic defender name for now

	if hit {
		damage := attackerPower // Simple fixed damage for now
		defender.HP -= damage
		logMsg += fmt.Sprintf(" Hit! Deals %d damage.", damage)
		// TODO: Add death check here (Step 9)
	} else {
		logMsg += " Miss!"
	}
	g.addCombatLog(logMsg)
}

// handlePlayerInput processes player actions. Returns true if an action was taken.
func (g *Game) handlePlayerInput() bool {
	targetX, targetY := g.Player.X, g.Player.Y
	actionTaken := false

	// --- Attack Input (Spacebar) ---
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		targetEnemy := g.findAdjacentEnemy()
		if targetEnemy != nil {
			g.resolveAttack("Player", g.Player.AttackPower, targetEnemy.AC, &targetEnemy.Entity)
			actionTaken = true
		} else {
			g.addCombatLog("Player attacks... nothing in range!")
			// Attacking empty space might still consume the turn depending on rules,
			// for now, only a successful attack attempt consumes the turn.
		}
	}

	// --- Movement Input (Only if no action already taken) ---
	if !actionTaken {
		moved := false
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
					g.Player.X = targetX
					g.Player.Y = targetY
					actionTaken = true // Successful move is an action
				} else {
					// fmt.Println("Blocked!") // Reduce console spam
				}
			}
		}
	}

	// --- Other Actions (Wait etc.) ---
	// TODO: Add Wait action (e.g., pressing 'W') that sets actionTaken = true

	return actionTaken
}

// handleEnemyTurns processes actions for all enemies.
func (g *Game) handleEnemyTurns() {
	// fmt.Println("Enemy Turn Start") // Reduce console spam
	for i, enemy := range g.Enemies {
		// Check if enemy is alive first (relevant after Step 9)
		// if enemy.HP <= 0 { continue }

		dx := g.Player.X - enemy.X
		dy := g.Player.Y - enemy.Y

		// Check if adjacent
		if math.Abs(float64(dx))+math.Abs(float64(dy)) <= 1 {
			// Attack player
			enemyName := fmt.Sprintf("Enemy %d", i) // Simple name
			g.resolveAttack(enemyName, enemy.AttackPower, g.Player.AC, &g.Player.Entity)
		} else {
			// Move towards player (if not adjacent)
			targetX, targetY := enemy.X, enemy.Y
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

			// Check validity and move
			if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
				if !g.isTileBlocked(targetX, targetY, i) {
					enemy.X = targetX
					enemy.Y = targetY
				} else {
					// Wait if move blocked
				}
			} else {
				// Wait if move out of bounds
			}
		}
	}
	// fmt.Println("Enemy Turn End") // Reduce console spam
}

// Update proceeds the game state by one tick.
func (g *Game) Update() error {
	switch g.CurrentTurn {
	case PlayerTurn:
		if g.handlePlayerInput() {
			g.CurrentTurn = EnemyTurn
		}
	case EnemyTurn:
		g.handleEnemyTurns()
		g.CurrentTurn = PlayerTurn
		// TODO: Check for player death after enemy turn (Step 9)
	}
	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	mapOffsetX := (screenWidth - (mapWidth * tileSize)) / 2
	mapOffsetY := (screenHeight - (mapHeight * tileSize)) / 2

	// Draw map tiles
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

	// Draw enemies & their HP
	for _, enemy := range g.Enemies {
		// Draw sprite
		enemyScreenX := float64(mapOffsetX + enemy.X*tileSize)
		enemyScreenY := float64(mapOffsetY + enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
		// Draw HP above enemy
		hpText := fmt.Sprintf("%d/%d", enemy.HP, enemy.MaxHP)
		ebitenutil.DebugPrintAt(screen, hpText, int(enemyScreenX), int(enemyScreenY)-12) // Position text above sprite
	}

	// Draw player & HP
	playerScreenX := float64(mapOffsetX + g.Player.X*tileSize)
	playerScreenY := float64(mapOffsetY + g.Player.Y*tileSize)
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)
	playerHpText := fmt.Sprintf("HP: %d/%d", g.Player.HP, g.Player.MaxHP)
	// Draw player HP below other debug text
	ebitenutil.DebugPrint(screen, "\n\n"+playerHpText) // Extra newlines for spacing

	// Draw UI text (Turn, Position)
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrint(screen, turnText)
	posText := fmt.Sprintf("Player Pos: (%d, %d)", g.Player.X, g.Player.Y)
	ebitenutil.DebugPrint(screen, "\n"+posText)

	// Draw Combat Log
	logStartY := screenHeight - (len(g.CombatLog) * 15) - 10 // Position log at bottom
	for i, msg := range g.CombatLog {
		ebitenutil.DebugPrintAt(screen, msg, 10, logStartY+(i*15))
	}
}

// Layout specifies the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Main Function ---

func main() {
	rand.Seed(time.Now().UnixNano()) // Seed random number generator
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP - Step 8: Combat") // Updated title
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
