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
	enemyBaseMovement  = 4
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

// --- Structs ---

type Entity struct {
	X     int
	Y     int
	HP    int
	MaxHP int
	AC    int
	// AttackPower removed, damage derived from ability scores
	Strength     int // Added Ability Scores
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
	Player      *Player
	Enemies     []*Enemy
	GameMap     [mapWidth][mapHeight]int
	TileImage   *ebiten.Image
	EnemySprite *ebiten.Image
	CurrentTurn TurnState
	CombatLog   []string
}

// --- Game Logic ---

// getModifier calculates the D&D ability modifier for a given score.
func getModifier(score int) int {
	return (score - 10) / 2 // Integer division handles floor
}

func NewGame() *Game {
	g := &Game{}
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)

	g.TileImage = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)

	playerSprite := ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(playerSprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)

	// Assign Player Stats (Example Fighter Array)
	playerStr := 15
	playerDex := 14
	playerCon := 13
	playerInt := 8
	playerWis := 12
	playerCha := 10

	// Calculate initial HP based on Con
	playerConMod := getModifier(playerCon)
	playerMaxHP := 10 + playerConMod // Fighter L1 HP

	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2,
			HP:           playerMaxHP,
			MaxHP:        playerMaxHP,
			AC:           13, // Base AC + Defense Style (hardcoded)
			Strength:     playerStr,
			Dexterity:    playerDex,
			Constitution: playerCon,
			Intelligence: playerInt,
			Wisdom:       playerWis,
			Charisma:     playerCha,
			Sprite:       playerSprite,
			Name:         "Player",
		},
		ProficiencyBonus:  2,
		MaxMovementPoints: playerBaseMovement,
	}

	g.EnemySprite = ebiten.NewImage(tileSize, tileSize)
	vector.DrawFilledRect(g.EnemySprite, 0, 0, float32(tileSize), float32(tileSize), color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)

	// Spawn enemies with basic stats
	// Args: x, y, name, base_hp, ac, str, dex, con, int, wis, cha, move
	g.spawnEnemy(2, 2, "Enemy 0", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement) // Basic melee enemy stats
	g.spawnEnemy(mapWidth-3, mapHeight-3, "Enemy 1", 6, 10, 12, 10, 11, 8, 8, 8, enemyBaseMovement)

	g.CurrentTurn = PlayerTurn
	g.resetPlayerTurnState()
	return g
}

// resetPlayerTurnState resets state for the start of the player's turn.
func (g *Game) resetPlayerTurnState() {
	g.Player.MovementPoints = g.Player.MaxMovementPoints
	g.Player.ActionTaken = false
}

// spawnEnemy creates a new enemy with specified ability scores.
func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move int) {
	conMod := getModifier(con)
	maxHp := baseHp + conMod // Adjust base HP by Con modifier
	if maxHp < 1 {
		maxHp = 1
	} // Ensure minimum 1 HP

	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, HP: maxHp, MaxHP: maxHp, AC: ac,
			Strength: str, Dexterity: dex, Constitution: con,
			Intelligence: intel, Wisdom: wis, Charisma: cha,
			Sprite: g.EnemySprite, Name: name,
		},
		MaxMovementPoints: move,
	}
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

// resolveAttack handles attack logic using Ability Modifiers + Proficiency Bonus.
// Returns true if defender died.
func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int) bool {
	// Determine attack ability modifier (Using Strength for melee for now)
	// TODO: Implement logic for choosing Str/Dex based on weapon (Finesse)
	attackAbilityMod := getModifier(attacker.Strength)

	// Calculate attack roll: d20 + Proficiency + Ability Mod
	roll := rand.Intn(20) + 1
	attackRoll := roll + attackerProfBonus + attackAbilityMod
	hit := attackRoll >= defender.AC

	// Log message showing calculation
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

	rollString := fmt.Sprintf("Roll: %d%s%s = %d", roll, profString, modString, attackRoll)
	logMsg := fmt.Sprintf("%s attacks %s (AC %d). %s.", attacker.Name, defender.Name, defender.AC, rollString)

	if hit {
		// Calculate damage: Ability Mod (min 1)
		// TODO: Add weapon damage dice later
		damage := max(1, attackAbilityMod) // Ensure at least 1 damage
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

// handlePlayerInput processes player actions (Wait, Attack, Move).
func (g *Game) handlePlayerInput() {
	// Action Input
	if !g.Player.ActionTaken {
		if inpututil.IsKeyJustPressed(ebiten.KeyW) {
			g.addCombatLog("Player waits.")
			g.Player.ActionTaken = true
		}
		if !g.Player.ActionTaken && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			targetEnemy := g.findAdjacentEnemy()
			if targetEnemy != nil {
				// Player uses Str mod (assumed melee) + Prof Bonus
				g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus)
				g.Player.ActionTaken = true
			} else {
				g.addCombatLog("Player attacks... nothing in range!")
			}
		}
	}
	// Movement Input
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
								g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0)
							}
						} // Enemy uses 0 prof bonus for AoO for now
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
	}
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

		// Action Phase
		if isAdjacent(g.Player.X, g.Player.Y, enemy.X, enemy.Y) && enemy.ActionAvailable {
			fmt.Printf("  %s is adjacent, attempting attack.\n", enemy.Name)
			// Enemy uses Str mod (assumed melee) + 0 Prof Bonus for now
			g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0)
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
	if g.CurrentTurn == GameOver {
		return nil
	}
	previousTurn := g.CurrentTurn
	switch g.CurrentTurn {
	case PlayerTurn:
		g.handlePlayerInput()
		if g.Player.ActionTaken {
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
			g.resetPlayerTurnState()
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

	// Turn state
	turnText := fmt.Sprintf("Turn: %s", g.CurrentTurn.String())
	ebitenutil.DebugPrintAt(screen, turnText, 10, uiStartY)

	// Player Info Block
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

	// Player Ability Scores Display
	p := g.Player.Entity // Shortcut for player entity stats
	scoresText1 := fmt.Sprintf("STR:%d(%+d) DEX:%d(%+d) CON:%d(%+d)", p.Strength, getModifier(p.Strength), p.Dexterity, getModifier(p.Dexterity), p.Constitution, getModifier(p.Constitution))
	scoresText2 := fmt.Sprintf("INT:%d(%+d) WIS:%d(%+d) CHA:%d(%+d)", p.Intelligence, getModifier(p.Intelligence), p.Wisdom, getModifier(p.Wisdom), p.Charisma, getModifier(p.Charisma))
	ebitenutil.DebugPrintAt(screen, scoresText1, 10, uiStartY+uiLineHeight*5)
	ebitenutil.DebugPrintAt(screen, scoresText2, 10, uiStartY+uiLineHeight*6)

	// Combat Log (at bottom)
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
	ebiten.SetWindowTitle("Slumb Gate MVP - Ability Scores") // Updated title
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
