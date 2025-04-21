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
	// Could add Class string, Level int etc. here later
}

type Enemy struct {
	Entity
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
	g.CombatLog = make([]string, 0, 5)

	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)

	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)
	g.Player = &Player{
		Entity: Entity{
			X:     mapWidth / 2,
			Y:     mapHeight / 2,
			HP:    20,
			MaxHP: 20,
			// Base AC 12 + 1 from hardcoded "Defense" Fighting Style = 13
			AC:          13,
			AttackPower: 4,
			Sprite:      playerSprite,
			Name:        "Player",
		},
		// Hardcoded Level 1 Proficiency Bonus
		ProficiencyBonus: 2,
	}

	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)

	g.spawnEnemy(2, 2, "Enemy 0", 8, 10, 3)
	g.spawnEnemy(mapWidth-3, mapHeight-3, "Enemy 1", 8, 10, 3)

	g.CurrentTurn = PlayerTurn
	return g
}

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

func (g *Game) addCombatLog(msg string) {
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > 5 {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-5:]
	}
	fmt.Println("CombatLog:", msg)
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

func (g *Game) findAdjacentEnemy() *Enemy {
	for _, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		dx := enemy.X - g.Player.X
		dy := enemy.Y - g.Player.Y
		if math.Abs(float64(dx))+math.Abs(float64(dy)) == 1 {
			return enemy
		}
	}
	return nil
}

// resolveAttack handles the logic for an attack attempt.
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

// handlePlayerInput processes player actions. Returns true if an action was taken.
func (g *Game) handlePlayerInput() bool {
	actionTaken := false

	// Wait Action (W Key)
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.addCombatLog("Player waits.")
		actionTaken = true
	}

	// Attack Input (Spacebar)
	if !actionTaken && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		targetEnemy := g.findAdjacentEnemy()
		if targetEnemy != nil {
			g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus)
			actionTaken = true
		} else {
			g.addCombatLog("Player attacks... nothing in range!")
		}
	}

	// Movement Input (Arrow Keys)
	if !actionTaken {
		moved := false
		targetX, targetY := g.Player.X, g.Player.Y
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
					actionTaken = true
				}
			}
		}
	}
	return actionTaken
}

// handleEnemyTurns processes actions for all enemies.
func (g *Game) handleEnemyTurns() {
	for i, enemy := range g.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		if g.Player.HP <= 0 {
			continue
		}

		dx := g.Player.X - enemy.X
		dy := g.Player.Y - enemy.Y

		if math.Abs(float64(dx))+math.Abs(float64(dy)) <= 1 {
			g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0) // Enemy attacks with 0 prof bonus
		} else {
			// Move towards player
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
			if targetX >= 0 && targetX < mapWidth && targetY >= 0 && targetY < mapHeight {
				if !g.isTileBlocked(targetX, targetY, i) {
					enemy.X = targetX
					enemy.Y = targetY
				}
			}
		}
	}
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
		if g.handlePlayerInput() {
			g.CurrentTurn = EnemyTurn
		}
	case EnemyTurn:
		g.handleEnemyTurns()
		g.cleanupDeadEnemies()
		if g.Player.HP <= 0 {
			if g.CurrentTurn != GameOver {
				g.addCombatLog("Player has died! Game Over.")
				g.CurrentTurn = GameOver
			}
		} else {
			g.CurrentTurn = PlayerTurn
		}
	}

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

	// --- Draw Enemies ---
	for _, enemy := range g.Enemies {
		enemyScreenX := float64(mapOffsetX + enemy.X*tileSize)
		enemyScreenY := float64(mapOffsetY + enemy.Y*tileSize)
		enemy.DrawOpts.GeoM.Reset()
		enemy.DrawOpts.GeoM.Translate(enemyScreenX, enemyScreenY)
		screen.DrawImage(enemy.Sprite, &enemy.DrawOpts)
		hpText := fmt.Sprintf("%d/%d", enemy.HP, enemy.MaxHP)
		ebitenutil.DebugPrintAt(screen, hpText, int(enemyScreenX), int(enemyScreenY)-12)
	}

	// --- Draw Player ---
	playerScreenX := float64(mapOffsetX + g.Player.X*tileSize)
	playerScreenY := float64(mapOffsetY + g.Player.Y*tileSize)
	g.Player.DrawOpts.GeoM.Reset()
	g.Player.DrawOpts.GeoM.Translate(playerScreenX, playerScreenY)
	screen.DrawImage(g.Player.Sprite, &g.Player.DrawOpts)

	// --- Draw UI ---
	// Define starting Y position for UI text block
	uiStartY := 10
	uiLineHeight := 15 // Approx height of DebugPrint line

	// Turn state
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrintAt(screen, turnText, 10, uiStartY)

	// Player Info Block
	// Hardcoded values based on current implementation
	playerInfoText := "Lvl: 1 Fighter (Defense Style)"
	ebitenutil.DebugPrintAt(screen, playerInfoText, 10, uiStartY+uiLineHeight)

	playerStatsText := fmt.Sprintf("HP: %d/%d AC: %d Prof: +%d",
		max(0, g.Player.HP), // Clamp HP display at 0
		g.Player.MaxHP,
		g.Player.AC,
		g.Player.ProficiencyBonus)
	ebitenutil.DebugPrintAt(screen, playerStatsText, 10, uiStartY+uiLineHeight*2)

	// Position (Optional, maybe less important now)
	// posText := fmt.Sprintf("Pos: (%d, %d)", g.Player.X, g.Player.Y)
	// ebitenutil.DebugPrintAt(screen, posText, 10, uiStartY + uiLineHeight*3)

	// Combat Log (at bottom)
	logStartY := screenHeight - (len(g.CombatLog) * 15) - 10
	for i, msg := range g.CombatLog {
		ebitenutil.DebugPrintAt(screen, msg, 10, logStartY+(i*15))
	}

	// --- Draw Game Over ---
	if g.CurrentTurn == GameOver {
		gameOverMsg := "GAME OVER"
		bounds := text.BoundString(basicfont.Face7x13, gameOverMsg)
		msgX := (screenWidth - bounds.Dx()) / 2
		msgY := screenHeight / 2
		text.Draw(screen, gameOverMsg, basicfont.Face7x13, msgX, msgY, color.White)
	}
}

// Helper function (alternative to math.Max if needed, or just inline)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Layout specifies the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Main Function ---

func main() {
	rand.Seed(time.Now().UnixNano())
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate MVP - L1 Fighter Features UI") // Updated title
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
