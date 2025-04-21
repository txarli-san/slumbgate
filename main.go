package main

import (
	"fmt"
	"image/color"
	"log"

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

// TurnState represents the current turn in the game
type TurnState int

const (
	PlayerTurn TurnState = iota // Player's turn to act
	EnemyTurn                   // Enemy's turn to act
)

// String method for TurnState for easier debugging display
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

// Player represents the player character
type Player struct {
	X        int // Grid X coordinate
	Y        int // Grid Y coordinate
	Sprite   *ebiten.Image
	DrawOpts ebiten.DrawImageOptions
	// Add HP, AC etc. later
}

// Enemy represents a single enemy unit
type Enemy struct {
	X        int // Grid X coordinate
	Y        int // Grid Y coordinate
	Sprite   *ebiten.Image
	DrawOpts ebiten.DrawImageOptions
	// Add HP, AC, AI state etc. later
}

// Game holds all game state
type Game struct {
	Player      *Player
	Enemies     []*Enemy                 // Slice to hold multiple enemies
	GameMap     [mapWidth][mapHeight]int // 0: Floor
	TileImage   *ebiten.Image
	EnemySprite *ebiten.Image // Shared sprite for all enemies (for now)
	CurrentTurn TurnState
}

// --- Game Logic ---

// NewGame initializes the game state
func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0) // Initialize the enemy slice

	// Initialize the map (all floors)
	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false) // Dark grey floor

	// Create player sprite
	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false) // Green player

	// Initialize the player
	g.Player = &Player{
		X:      mapWidth / 2,
		Y:      mapHeight / 2,
		Sprite: playerSprite,
	}

	// Create a shared enemy sprite
	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false) // Red enemy

	// Spawn some enemies
	// Enemy positions should ideally not overlap with player start or each other
	g.spawnEnemy(2, 2)
	g.spawnEnemy(mapWidth-3, mapHeight-3)

	// Set the initial turn state
	g.CurrentTurn = PlayerTurn

	return g
}

// spawnEnemy creates a new enemy and adds it to the game's enemy list
func (g *Game) spawnEnemy(x, y int) {
	enemy := &Enemy{
		X:      x,
		Y:      y,
		Sprite: g.EnemySprite, // Use the shared sprite
	}
	g.Enemies = append(g.Enemies, enemy)
}

// isTileOccupied checks if a given tile coordinate is occupied by an enemy
func (g *Game) isTileOccupied(x, y int) bool {
	for _, enemy := range g.Enemies {
		if enemy.X == x && enemy.Y == y {
			return true
		}
	}
	// Could also check for player position if needed, or other obstacles
	return false
}

// handleInput checks for player input and updates player position.
// Returns true if the player performed an action (like moving), false otherwise.
func (g *Game) handleInput() bool {
	targetX, targetY := g.Player.X, g.Player.Y
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
		// Check map boundaries first
		if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
			// Check if the target tile is occupied by an enemy
			if !g.isTileOccupied(targetX, targetY) {
				// Update player position
				g.Player.X = targetX
				g.Player.Y = targetY
				return true // Player successfully moved
			} else {
				fmt.Println("Blocked by enemy!") // Feedback that move failed
				return false                     // Move failed (blocked) - doesn't consume turn
			}
		}
		// Move failed (out of bounds) - doesn't consume turn
		return false
	}

	// No movement key was pressed
	return false
}

// Update proceeds the game state.
func (g *Game) Update() error {
	switch g.CurrentTurn {
	case PlayerTurn:
		playerActed := g.handleInput()
		if playerActed {
			g.CurrentTurn = EnemyTurn
		}
	case EnemyTurn:
		// --- Enemy AI Logic (Phase 3 - Step 7) ---
		// TODO: Implement enemy actions here (move, attack)
		fmt.Println("Enemy Turn (doing nothing yet)") // Debug print

		// Simulate enemy turn finishing immediately
		g.CurrentTurn = PlayerTurn
	}

	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	mapOffsetX := (screenWidth - (mapWidth * tileSize)) / 2
	mapOffsetY := (screenHeight - (mapHeight * tileSize)) / 2

	// 1. Draw the map tiles
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

	// 2. Draw the enemies
	for _, enemy := range g.Enemies {
		enemyScreenX := float64(mapOffsetX + enemy.X*tileSize)
		enemyScreenY := float64(mapOffsetY + enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
	}

	// 3. Draw the player (drawn after enemies so player appears on top if on same tile - shouldn't happen with collision)
	playerScreenX := float64(mapOffsetX + g.Player.X*tileSize)
	playerScreenY := float64(mapOffsetY + g.Player.Y*tileSize)
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)

	// --- UI Drawing (Phase 4) ---
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrint(screen, turnText)
	posText := fmt.Sprintf("Player Pos: (%d, %d)", g.Player.X, g.Player.Y)
	ebitenutil.DebugPrint(screen, "\n"+posText)
}

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Main Function ---

func main() {
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP - Step 6: Basic Enemy") // Updated title
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
