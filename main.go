package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640 // Window width
	screenHeight = 480 // Window height
	tileSize     = 32  // Pixel size of each grid tile
	mapWidth     = 10
	mapHeight    = 10
)

type Player struct {
	X        int
	Y        int
	Sprite   *ebiten.Image
	DrawOpts ebiten.DrawImageOptions
}

type Game struct {
	Player    *Player
	GameMap   [mapWidth][mapHeight]int // 0: Floor (for now)
	TileImage *ebiten.Image            // Image for a floor tile
}

// --- Game Logic ---

func NewGame() *Game {
	g := &Game{}

	// Initialize the map (all floors for now)
	// for x := 0; x < mapWidth; x++ {
	// 	for y := 0; y < mapHeight; y++ {
	// 		g.GameMap[x][y] = 0 // Explicitly setting to 0 (Floor)
	// 	}
	// }

	// Create a simple placeholder image for the floor tile
	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	// Use vector package for simple rectangle fill
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false) // Dark grey floor

	// Create a simple placeholder sprite for the player
	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false) // Green player

	// Initialize the player
	g.Player = &Player{
		X:      mapWidth / 2,  // Start in the middle-ish
		Y:      mapHeight / 2, // Start in the middle-ish
		Sprite: playerSprite,
	}

	return g
}

func (g *Game) Update() error {
	// --- Input Handling ---
	// Will be added later

	// --- Game Logic ---
	// Turn management, AI, Combat etc. will go here

	return nil // Return error to stop the game
}

func (g *Game) Draw(screen *ebiten.Image) {
	mapOffsetX := (screenWidth - (mapWidth * tileSize)) / 2
	mapOffsetY := (screenHeight - (mapHeight * tileSize)) / 2

	// 1. Draw the map tiles
	tileOpts := &ebiten.DrawImageOptions{}
	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			// Calculate screen coordinates for the tile
			screenX := float64(mapOffsetX + x*tileSize)
			screenY := float64(mapOffsetY + y*tileSize)

			// Set position for drawing the tile
			tileOpts.GeoM.Reset()
			tileOpts.GeoM.Translate(screenX, screenY)

			// Get tile type (currently always 0)
			// tileType := g.GameMap[x][y]
			// TODO: Later, select different tile images based on tileType

			// Draw the floor tile
			screen.DrawImage(g.TileImage, tileOpts)
		}
	}

	// 2. Draw the player
	// Calculate screen coordinates for the player based on their grid position
	playerScreenX := float64(mapOffsetX + g.Player.X*tileSize)
	playerScreenY := float64(mapOffsetY + g.Player.Y*tileSize)

	// Reset and set position for drawing the player
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)

	// --- UI Drawing ---
	ebitenutil.DebugPrint(screen, "Slumb Gate MVP - Step 1") // Basic title/debug text
	// Player HP etc. will be drawn here later
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	game := NewGame()

	// Set window properties
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
