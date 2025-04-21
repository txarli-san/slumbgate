package main

import (
	"fmt"
	"image/color"
	"log"
	"math" // Needed for Abs

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

type Player struct {
	X        int
	Y        int
	Sprite   *ebiten.Image
	DrawOpts ebiten.DrawImageOptions
	// TODO: Add HP, AC etc.
}

type Enemy struct {
	X        int
	Y        int
	Sprite   *ebiten.Image
	DrawOpts ebiten.DrawImageOptions
	// TODO: Add HP, AC, AI state etc.
}

type Game struct {
	Player      *Player
	Enemies     []*Enemy
	GameMap     [mapWidth][mapHeight]int // Using 0 for Floor
	TileImage   *ebiten.Image
	EnemySprite *ebiten.Image
	CurrentTurn TurnState
}

// --- Game Logic ---

func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)

	// Create floor tile sprite
	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)

	// Create player sprite
	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)

	g.Player = &Player{
		X:      mapWidth / 2,
		Y:      mapHeight / 2,
		Sprite: playerSprite,
	}

	// Create enemy sprite
	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)

	// Spawn initial enemies
	g.spawnEnemy(2, 2)
	g.spawnEnemy(mapWidth-3, mapHeight-3)

	g.CurrentTurn = PlayerTurn
	return g
}

func (g *Game) spawnEnemy(x, y int) {
	enemy := &Enemy{
		X:      x,
		Y:      y,
		Sprite: g.EnemySprite,
	}
	g.Enemies = append(g.Enemies, enemy)
}

// isTileBlocked checks if a given tile coordinate is occupied by the player or another enemy.
// Pass the index of the currently moving enemy to avoid self-collision check.
func (g *Game) isTileBlocked(x, y, movingEnemyIndex int) bool {
	// Check against player
	if g.Player.X == x && g.Player.Y == y {
		return true
	}
	// Check against other enemies
	for i, enemy := range g.Enemies {
		if i == movingEnemyIndex { // Don't check collision with self
			continue
		}
		if enemy.X == x && enemy.Y == y {
			return true
		}
	}
	return false
}

// handlePlayerInput processes player actions. Returns true if an action was taken.
func (g *Game) handlePlayerInput() bool {
	targetX, targetY := g.Player.X, g.Player.Y
	actionTaken := false

	// --- Movement Input ---
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
		// Check map boundaries
		if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
			// Check collision with enemies
			if !g.isTileBlocked(targetX, targetY, -1) { // -1 indicates check against all enemies (player move)
				g.Player.X = targetX
				g.Player.Y = targetY
				actionTaken = true // Successful move is an action
			} else {
				fmt.Println("Blocked!") // Move failed (blocked)
				// Failed move does not consume the turn in this design
			}
		}
		// Invalid move (out of bounds) also doesn't consume turn
	}

	// --- Other Actions (Attack, Wait etc.) ---
	// TODO: Add other player actions here

	return actionTaken
}

// handleEnemyTurns processes actions for all enemies.
func (g *Game) handleEnemyTurns() {
	fmt.Println("Enemy Turn Start")
	for i, enemy := range g.Enemies {
		// --- Simple AI: Move towards player if not adjacent ---
		dx := g.Player.X - enemy.X
		dy := g.Player.Y - enemy.Y

		// Check if already adjacent (Manhattan distance <= 1)
		if math.Abs(float64(dx))+math.Abs(float64(dy)) <= 1 {
			// TODO: Implement attack logic here (Step 8)
			fmt.Printf("Enemy %d at (%d, %d) is adjacent. Skipping move.\n", i, enemy.X, enemy.Y)
			continue // Skip movement if adjacent
		}

		// Determine target move coordinate (move one step orthogonally)
		targetX, targetY := enemy.X, enemy.Y
		if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
			// Move horizontally
			if dx > 0 {
				targetX++
			} else {
				targetX--
			}
		} else {
			// Move vertically
			if dy > 0 {
				targetY++
			} else {
				targetY--
			}
		}

		// Check if target tile is valid (within bounds and not blocked)
		if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
			if !g.isTileBlocked(targetX, targetY, i) { // Pass enemy index 'i'
				// Move enemy
				fmt.Printf("Enemy %d moving from (%d, %d) to (%d, %d)\n", i, enemy.X, enemy.Y, targetX, targetY)
				enemy.X = targetX
				enemy.Y = targetY
			} else {
				fmt.Printf("Enemy %d at (%d, %d) move blocked to (%d, %d).\n", i, enemy.X, enemy.Y, targetX, targetY)
				// Enemy waits if move is blocked
			}
		} else {
			fmt.Printf("Enemy %d at (%d, %d) attempted out-of-bounds move to (%d, %d).\n", i, enemy.X, enemy.Y, targetX, targetY)
			// Enemy waits if move is out of bounds
		}
	}
	fmt.Println("Enemy Turn End")
}

// Update proceeds the game state by one tick.
func (g *Game) Update() error {
	switch g.CurrentTurn {
	case PlayerTurn:
		if g.handlePlayerInput() { // If player acted
			g.CurrentTurn = EnemyTurn // Switch to enemy turn
		}
	case EnemyTurn:
		g.handleEnemyTurns()       // Process all enemy actions
		g.CurrentTurn = PlayerTurn // Switch back to player turn
	}
	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	// Calculate offsets to center the map
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

	// Draw enemies
	for _, enemy := range g.Enemies {
		enemyScreenX := float64(mapOffsetX + enemy.X*tileSize)
		enemyScreenY := float64(mapOffsetY + enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
	}

	// Draw player
	playerScreenX := float64(mapOffsetX + g.Player.X*tileSize)
	playerScreenY := float64(mapOffsetY + g.Player.Y*tileSize)
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)

	// Draw UI text
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrint(screen, turnText)
	posText := fmt.Sprintf("Player Pos: (%d, %d)", g.Player.X, g.Player.Y)
	ebitenutil.DebugPrint(screen, "\n"+posText)
}

// Layout specifies the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Main Function ---

func main() {
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP - Step 7: Simple Enemy AI") // Updated title
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
