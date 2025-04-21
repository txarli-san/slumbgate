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

type TurnState int

const (
	PlayerTurn TurnState = iota // Player's turn to act
	EnemyTurn                   // Enemy's turn to act
	// Add other states like GameOver, Victory etc. later
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
}

// Game holds all game state
type Game struct {
	Player      *Player
	GameMap     [mapWidth][mapHeight]int // 0: Floor
	TileImage   *ebiten.Image
	CurrentTurn TurnState // Added to manage turns
}

// --- Game Logic ---

// NewGame initializes the game state
func NewGame() *Game {
	g := &Game{}

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

	// Set the initial turn state
	g.CurrentTurn = PlayerTurn

	return g
}

// handleInput checks for player input and updates player position.
// Returns true if the player performed an action (like moving), false otherwise.
func (g *Game) handleInput() bool {
	// Target coordinates after potential move
	targetX, targetY := g.Player.X, g.Player.Y
	moved := false // Flag to track if player attempted to move

	// Check for arrow key presses using IsKeyJustPressed for single action per press
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
	// Add other actions later (e.g., Attack, Wait)

	// If a move key was pressed, check boundaries and update player position
	if moved {
		// Check map boundaries
		if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
			// TODO: Later, check if the target tile is walkable (e.g., not a wall or occupied by enemy)

			// Update player position
			g.Player.X = targetX
			g.Player.Y = targetY
			return true // Player successfully performed an action
		}
		// If move was attempted but invalid (e.g., out of bounds), action was still attempted but failed.
		// Depending on game rules, this might still consume the turn or not.
		// For simplicity now, only a *successful* move consumes the turn.
		return false // Move failed (e.g., out of bounds)
	}

	// No movement key was pressed
	return false // Player did not perform an action
}

// Update proceeds the game state.
func (g *Game) Update() error {
	switch g.CurrentTurn {
	case PlayerTurn:
		// --- Input Handling ---
		playerActed := g.handleInput() // Process player movement input
		if playerActed {
			// If the player performed an action, switch to Enemy turn
			g.CurrentTurn = EnemyTurn
		}
	case EnemyTurn:
		// --- Enemy AI Logic ---
		// TODO: Implement enemy actions here (move, attack)

		// For now, enemies do nothing and the turn immediately switches back.
		fmt.Println("Enemy Turn (doing nothing yet)") // Debug print
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

	// 2. Draw the player
	playerScreenX := float64(mapOffsetX + g.Player.X*tileSize)
	playerScreenY := float64(mapOffsetY + g.Player.Y*tileSize)
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)

	// --- UI Drawing ---
	// Display current turn state
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrint(screen, turnText)

	// Display player coordinates for debugging
	posText := fmt.Sprintf("Player Pos: (%d, %d)", g.Player.X, g.Player.Y)
	ebitenutil.DebugPrint(screen, "\n"+posText) // Display on the next line
}

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Main Function ---

func main() {
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP - Turn Management")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
