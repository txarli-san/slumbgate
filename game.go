package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
		return "Unknown"
	}
}

func (at ActionType) String() string {
	switch at {
	case ActionTypeStandard:
		return "Action"
	case ActionTypeBonus:
		return "Bonus Action"
	case ActionTypeFree:
		return "Free Action"
	case ActionTypeReaction:
		return "Reaction"
	default:
		return "Unknown Action Type"
	}
}

func (g *Game) InitGame(playerClassName string) {
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)
	g.MapOffsetX = (screenWidth - (mapWidth * tileSize)) / 2
	g.MapOffsetY = (screenHeight - (mapHeight * tileSize)) / 2
	g.InputMode = InputModeMap
	g.availableActions = make([]*ActionDefinition, 0)
	g.lastExecutedActionID = ""
	g.CurrentWaveIndex = -1
	g.FloatingTexts = make([]*FloatingText, 0)
	g.isVictory = false

	g.WaveDefinitions = []WaveDefinition{
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Scout", SpawnPointIdx: 0}, {TypeName: "Goblin Scout", SpawnPointIdx: 1}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Scout", SpawnPointIdx: 2}, {TypeName: "Goblin Scout", SpawnPointIdx: 3}, {TypeName: "Goblin Archer", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Archer", SpawnPointIdx: 2}, {TypeName: "Goblin Archer", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Chieftain", SpawnPointIdx: 4}}, IsBossWave: true},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Scout", SpawnPointIdx: 0}, {TypeName: "Goblin Scout", SpawnPointIdx: 1}, {TypeName: "Goblin Archer", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Archer", SpawnPointIdx: 4}, {TypeName: "Goblin Archer", SpawnPointIdx: 2}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Warlock", SpawnPointIdx: 4}, {TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Scout", SpawnPointIdx: 2}, {TypeName: "Goblin Scout", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Chieftain", SpawnPointIdx: 4}, {TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Brute", SpawnPointIdx: 1}, {TypeName: "Goblin Warlock", SpawnPointIdx: 3}}, IsBossWave: true},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Small Slime", SpawnPointIdx: 0}, {TypeName: "Small Slime", SpawnPointIdx: 1}, {TypeName: "Spore Mushroom", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Tough Slime", SpawnPointIdx: 0}, {TypeName: "Tough Slime", SpawnPointIdx: 1}, {TypeName: "Spore Mushroom", SpawnPointIdx: 2}, {TypeName: "Spore Mushroom", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goo Spitter", SpawnPointIdx: 4}, {TypeName: "Tough Slime", SpawnPointIdx: 0}, {TypeName: "Tough Slime", SpawnPointIdx: 1}, {TypeName: "Small Slime", SpawnPointIdx: 2}, {TypeName: "Small Slime", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Elder Spore Mushroom", SpawnPointIdx: 4}, {TypeName: "Goo Spitter", SpawnPointIdx: 2}, {TypeName: "Goo Spitter", SpawnPointIdx: 3}, {TypeName: "Tough Slime", SpawnPointIdx: 0}, {TypeName: "Tough Slime", SpawnPointIdx: 1}}, IsBossWave: true},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Centaur", SpawnPointIdx: 2}, {TypeName: "Centaur", SpawnPointIdx: 3}, {TypeName: "Druid", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Dire Wolf", SpawnPointIdx: 0}, {TypeName: "Faun", SpawnPointIdx: 2}, {TypeName: "Faun", SpawnPointIdx: 3}, {TypeName: "Druid", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Centaur", SpawnPointIdx: 2}, {TypeName: "Centaur", SpawnPointIdx: 3}, {TypeName: "Naga", SpawnPointIdx: 1}, {TypeName: "Druid", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Tauren", SpawnPointIdx: 4}, {TypeName: "Druid", SpawnPointIdx: 2}, {TypeName: "Druid", SpawnPointIdx: 3}, {TypeName: "Dire Wolf", SpawnPointIdx: 0}, {TypeName: "Dire Wolf", SpawnPointIdx: 1}}, IsBossWave: true},
	}

	var err error
	rogueSheetPath := "assets/rogues.png"
	monsterSheetPath := "assets/monsters.png"
	tilePath := "assets/tiles.png"

	g.rogueSheet, err = loadImage(rogueSheetPath)
	if err != nil {
		log.Printf("Error loading rogue sheet: %v. Using fallback.", err)
		g.rogueSheet = ebiten.NewImage(spriteSize*sheetWidthInSprites, spriteSize*10)
		g.rogueSheet.Fill(color.NRGBA{R: 50, G: 50, B: 50, A: 255})
	}
	g.monsterSheet, err = loadImage(monsterSheetPath)
	if err != nil {
		log.Printf("Error loading monster sheet: %v. Using fallback.", err)
		g.monsterSheet = ebiten.NewImage(spriteSize*sheetWidthInSprites, spriteSize*15)
		g.monsterSheet.Fill(color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	}
	tileSheet, err := loadImage(tilePath)
	if err != nil {
		log.Printf("Error loading tile sheet: %v. Creating fallback tile.", err)
		g.TileImage = ebiten.NewImage(tileSize, tileSize)
		vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.NRGBA{R: 50, G: 50, B: 50, A: 255}, false)
		vector.StrokeRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), 1, color.NRGBA{R: 80, G: 80, B: 80, A: 255}, false)
	} else {
		floorSprite := getSpriteFromSheet(tileSheet, 0, 0)
		g.TileImage = floorSprite
	}

	g.RangeOverlayTile = ebiten.NewImage(tileSize, tileSize)
	overlayColor := color.NRGBA{R: 0, G: 100, B: 200, A: 80}
	vector.DrawFilledRect(g.RangeOverlayTile, 0, 0, float32(tileSize), float32(tileSize), overlayColor, false)

	classDef, classExists := ClassDefinitions[playerClassName]
	if !classExists {
		log.Fatalf("FATAL: Player class '%s' not found in ClassDefinitions!", playerClassName)
	}

	playerSprite := getSpriteFromSheet(g.rogueSheet, classDef.SpriteSheetX, classDef.SpriteSheetY)
	playerStr, playerDex, playerCon := 10, 10, 10
	playerInt, playerWis, playerCha := 10, 10, 10
	startLevel := 1
	knownSpells := make([]string, 0)
	knownCantrips := make([]string, 0)
	maxSlotsL1 := 0

	if playerClassName == "Fighter" {
		playerStr, playerDex, playerCon = 16, 14, 14
		playerInt, playerWis, playerCha = 8, 8, 10
	} else if playerClassName == "Mage" {
		playerStr, playerDex, playerCon = 8, 14, 14
		playerInt, playerWis, playerCha = 16, 10, 8
		knownCantrips = append(knownCantrips, "firebolt")
		knownSpells = append(knownSpells, "magic_missile", "shield")
		maxSlotsL1 = 2
	}

	playerConMod := getModifier(playerCon)
	playerDexMod := getModifier(playerDex)
	playerMaxHP := classDef.HitDieSize + playerConMod
	playerAC := 10 + playerDexMod

	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2, Width: 1, Height: 1,
			HP: playerMaxHP, MaxHP: playerMaxHP, AC: playerAC,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Sprite: playerSprite, Name: "Player", CurrentAlpha: 1.0,
		},
		Level:                startLevel,
		Class:                playerClassName,
		ProficiencyBonus:     calculateProficiencyBonus(startLevel),
		MaxMovementPoints:    playerBaseMovement,
		ClassResources:       make(map[string]int),
		MaxHitDice:           startLevel,
		HitDice:              startLevel,
		MaxSpellSlotsL1:      maxSlotsL1,
		SpellSlotsL1:         maxSlotsL1,
		KnownSpells:          knownSpells,
		KnownCantrips:        knownCantrips,
		UsedReaction:         false,
		UsedArcaneRecovery:   false,
		ACBonusUntilNextTurn: 0,
		LastSpellCastID:      "",
		CombatStyle:          "",
		CombatTechnique:      "",
		IsSlowed:             false,
		SlowDuration:         0,
		NextElementType:      "Fire",
	}

	g.initializePlayerResources()

	g.CurrentTurn = PlayerTurn
	g.SpawnNextWave()
	g.startPlayerTurn()
	g.CurrentGameState = StatePlaying
}

func (g *Game) initializePlayerResources() {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}
	for actionID, classAction := range classDef.ClassActions {
		if classAction.ResourceType == ResourceClassFeature {
			if classAction.RequiredLevel <= g.Player.Level {
				if classAction.UsesPerRest > 0 {
					g.Player.ClassResources[actionID] = classAction.UsesPerRest
				} else {
					g.Player.ClassResources[actionID] = 0
				}
			} else {
				g.Player.ClassResources[actionID] = 0
			}
		}
	}
	if g.Player.Class == "Mage" {
		g.Player.MaxSpellSlotsL1 = 0
		if g.Player.Level >= 1 {
			g.Player.MaxSpellSlotsL1 = 2
		}
		if g.Player.Level >= 2 {
			g.Player.MaxSpellSlotsL1 = 3
		}

	}
}

func (g *Game) refreshLevelUpResources() {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}

	refreshedSomething := false
	logMsg := "Level Up! Resources refreshed: "

	for actionID, classAction := range classDef.ClassActions {
		if classAction.RefreshesOn == RestTypeNever {
			if g.Player.Level >= classAction.RequiredLevel {
				currentUses := 0
				if val, ok := g.Player.ClassResources[actionID]; ok {
					currentUses = val
				}
				maxUses := classAction.UsesPerRest
				if currentUses < maxUses || g.Player.Level == classAction.RequiredLevel {
					g.Player.ClassResources[actionID] = maxUses
					logMsg += fmt.Sprintf("%s, ", classAction.Name)
					refreshedSomething = true
				}
			}
		}
	}

	if g.Player.Class == "Mage" {
		oldMaxSlots := g.Player.MaxSpellSlotsL1
		newMaxSlots := 0
		if g.Player.Level >= 1 {
			newMaxSlots = 2
		}
		if g.Player.Level >= 3 {
			newMaxSlots = 3
		}

		if newMaxSlots > oldMaxSlots {
			g.Player.MaxSpellSlotsL1 = newMaxSlots
			g.Player.SpellSlotsL1 = newMaxSlots
			logMsg += "Spell Slots, "
			refreshedSomething = true
		} else {
			g.Player.SpellSlotsL1 = g.Player.MaxSpellSlotsL1
		}
	}

	if refreshedSomething {
		if len(logMsg) > 2 {
			logMsg = logMsg[:len(logMsg)-2] + "."
		}
		g.addCombatLog(logMsg)
	}
}

func (g *Game) resetPlayerResources(restType ResourceRestType) {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}
	logMsg := ""
	if restType == RestTypeShort {
		logMsg = "Short Rest complete. Resources refreshed: "
	} else if restType == RestTypeLong {
		logMsg = "Long Rest complete. Resources refreshed: "
	}

	refreshedSomething := false
	for actionID, classAction := range classDef.ClassActions {
		if classAction.RefreshesOn == RestTypeNever {
			continue
		}

		shouldReset := false
		if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
			if classAction.RefreshesOn == restType {
				shouldReset = true
			}
			if restType == RestTypeLong && classAction.RefreshesOn == RestTypeShort {
				shouldReset = true
			}
		}

		if shouldReset && g.Player.Level >= classAction.RequiredLevel {
			currentUses := g.Player.ClassResources[actionID]
			maxUses := classAction.UsesPerRest

			if currentUses < maxUses {
				g.Player.ClassResources[actionID] = maxUses
				logMsg += fmt.Sprintf("%s, ", classAction.Name)
				refreshedSomething = true
			}
		}
	}

	if g.Player.Class == "Mage" && restType == RestTypeLong {
		if g.Player.SpellSlotsL1 < g.Player.MaxSpellSlotsL1 {
			g.Player.SpellSlotsL1 = g.Player.MaxSpellSlotsL1
			logMsg += "Spell Slots, "
			refreshedSomething = true
		}
		if g.Player.UsedArcaneRecovery {
			g.Player.UsedArcaneRecovery = false
			logMsg += "Arcane Recovery Use, "
			refreshedSomething = true
		}
	}

	if refreshedSomething {
		if len(logMsg) > 2 {
			logMsg = logMsg[:len(logMsg)-2] + "."
		}
		g.addCombatLog(logMsg)
	}

	if restType == RestTypeLong {
		diceToRecover := max(1, g.Player.MaxHitDice/2)
		initialDice := g.Player.HitDice
		g.Player.HitDice = min(g.Player.MaxHitDice, g.Player.HitDice+diceToRecover)
		recoveredCount := g.Player.HitDice - initialDice
		if recoveredCount > 0 {
			g.addCombatLog(fmt.Sprintf("Recovered %d Hit Dice.", recoveredCount))
		}
	}
}

func (g *Game) shortRest() {
	g.addCombatLog("Player takes a Short Rest...")
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		g.addCombatLog("Cannot determine class for Hit Dice.")
		return
	}

	if g.Player.HitDice > 0 {
		conMod := getModifier(g.Player.Constitution)
		healRoll := rand.Intn(classDef.HitDieSize) + 1
		healAmount := max(1, healRoll+conMod)
		actualHeal := min(healAmount, g.Player.MaxHP-g.Player.HP)
		actualHeal = max(0, actualHeal)

		if actualHeal > 0 {
			g.Player.HP += actualHeal
			g.addCombatLog(fmt.Sprintf("Spent 1 Hit Die (d%d), recovered %d HP (Rolled %d%+d).", classDef.HitDieSize, actualHeal, healRoll, conMod))
		} else {
			g.addCombatLog("Spent 1 Hit Die, but already at full HP.")
		}
		g.Player.HitDice--
	} else {
		g.addCombatLog("No Hit Dice left to spend.")
	}

	if g.Player.Class == "Mage" && g.Player.Level >= 2 && !g.Player.UsedArcaneRecovery {
		slotsToRecoverFloat := math.Ceil(float64(g.Player.Level) / 2.0)
		slotsToRecover := int(slotsToRecoverFloat)

		if slotsToRecover > 0 && g.Player.SpellSlotsL1 < g.Player.MaxSpellSlotsL1 {
			recovered := min(slotsToRecover, g.Player.MaxSpellSlotsL1-g.Player.SpellSlotsL1)
			if recovered > 0 {
				g.Player.SpellSlotsL1 += recovered
				g.Player.UsedArcaneRecovery = true
				g.addCombatLog(fmt.Sprintf("Used Arcane Recovery! Recovered %d L1 spell slot(s).", recovered))
			} else {
				g.addCombatLog("Arcane Recovery: Already at max spell slots.")
			}
		} else if slotsToRecover > 0 {
			g.addCombatLog("Arcane Recovery: Already at max spell slots.")
		}
	}
}

func (g *Game) longRest() {
	g.addCombatLog("Player takes a Long Rest...")
	g.Player.HP = g.Player.MaxHP
	g.resetPlayerResources(RestTypeLong)
	g.addCombatLog("HP fully restored.")
}

func (g *Game) levelUpPlayer() {
	if g.Player.Level >= maxLevel {
		g.addCombatLog("Already at max level!")
		return
	}

	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		log.Printf("Error: Cannot find class definition %s for level up.", g.Player.Class)
		return
	}

	g.Player.Level++
	g.addCombatLog(fmt.Sprintf("LEVEL UP! Reached Level %d!", g.Player.Level))

	conMod := getModifier(g.Player.Constitution)
	hpRoll := rand.Intn(classDef.HitDieSize) + 1
	hpIncrease := max(1, hpRoll+conMod)
	g.Player.MaxHP += hpIncrease
	g.Player.HP += hpIncrease
	g.addCombatLog(fmt.Sprintf("Max HP increased by %d (Rolled %d%+d).", hpIncrease, hpRoll, conMod))

	g.Player.MaxHitDice++
	g.Player.HitDice++
	g.addCombatLog("Gained 1 Hit Die.")

	newProfBonus := calculateProficiencyBonus(g.Player.Level)
	if newProfBonus > g.Player.ProficiencyBonus {
		g.Player.ProficiencyBonus = newProfBonus
		g.addCombatLog(fmt.Sprintf("Proficiency Bonus increased to +%d.", newProfBonus))
	}

	g.refreshLevelUpResources()
}

func (g *Game) SpawnNextWave() {
	g.CurrentWaveIndex++

	if g.CurrentWaveIndex >= len(g.WaveDefinitions) {
		log.Printf("Attempted to spawn wave index %d, but only %d waves are defined.", g.CurrentWaveIndex, len(g.WaveDefinitions))
		g.addCombatLog("No more waves defined.")
		if g.CurrentGameState != StateGameOverScreen {
			g.setGameOver(true)
		}
		return
	}

	waveDef := g.WaveDefinitions[g.CurrentWaveIndex]
	g.addCombatLog(fmt.Sprintf("--- Starting Wave %d ---", g.CurrentWaveIndex+1))
	g.Enemies = make([]*Enemy, 0)

	spawnPoints := []image.Point{
		{X: 2, Y: 2},
		{X: mapWidth - 3, Y: mapHeight - 3},
		{X: 2, Y: mapHeight - 3},
		{X: mapWidth - 3, Y: 2},
		{X: mapWidth / 2, Y: 1},
	}

	for _, spawnInfo := range waveDef.EnemiesToSpawn {
		enemyDef, exists := EnemyDefinitions[spawnInfo.TypeName]
		if !exists {
			log.Printf("Warning: Enemy type '%s' not found in definitions.", spawnInfo.TypeName)
			continue
		}

		spawnX, spawnY := -1, -1
		if spawnInfo.SpawnPointIdx >= 0 && spawnInfo.SpawnPointIdx < len(spawnPoints) {
			spawnPoint := spawnPoints[spawnInfo.SpawnPointIdx]
			spawnX = spawnPoint.X
			spawnY = spawnPoint.Y
			if enemyDef.Width > 1 {
				spawnX -= (enemyDef.Width - 1) / 2
			}
			if enemyDef.Height > 1 {
				spawnY -= (enemyDef.Height - 1) / 2
			}
			if spawnX < 0 {
				spawnX = 0
			}
			if spawnY < 0 {
				spawnY = 0
			}
			if spawnX+enemyDef.Width > mapWidth {
				spawnX = mapWidth - enemyDef.Width
			}
			if spawnY+enemyDef.Height > mapHeight {
				spawnY = mapHeight - enemyDef.Height
			}

		} else {
			log.Printf("Warning: Invalid spawn point index %d for enemy '%s'. Skipping.", spawnInfo.SpawnPointIdx, enemyDef.Name)
			continue
		}

		if g.isTileFullyBlocked(spawnX, spawnY, enemyDef.Width, enemyDef.Height, -1) {
			log.Printf("Warning: Spawn point %d (%d,%d) for %s is blocked. Trying fallback.", spawnInfo.SpawnPointIdx, spawnX, spawnY, enemyDef.Name)
			foundAlt := false
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					altX, altY := spawnX+dx, spawnY+dy
					if altX >= 0 && altX+enemyDef.Width <= mapWidth && altY >= 0 && altY+enemyDef.Height <= mapHeight {
						if !g.isTileFullyBlocked(altX, altY, enemyDef.Width, enemyDef.Height, -1) {
							spawnX, spawnY = altX, altY
							foundAlt = true
							break
						}
					}
				}
				if foundAlt {
					break
				}
			}
			if !foundAlt {
				log.Printf("Error: Could not find alternative spawn location for %s near (%d,%d). Skipping.", enemyDef.Name, spawnX, spawnY)
				continue
			}
		}

		sprite := getSpriteFromSheet(g.monsterSheet, enemyDef.SpriteSheetX, enemyDef.SpriteSheetY)
		g.spawnEnemyFromDef(spawnX, spawnY, enemyDef, sprite)
	}
}

func (g *Game) spawnEnemyFromDef(x, y int, def EnemyDefinition, sprite *ebiten.Image) {
	conMod := getModifier(def.Con)
	maxHp := def.BaseHP + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, Width: def.Width, Height: def.Height,
			HP: maxHp, MaxHP: maxHp, AC: def.AC,
			Strength: def.Str, Dexterity: def.Dex, Constitution: def.Con,
			Intelligence: def.Int, Wisdom: def.Wis, Charisma: def.Cha,
			Sprite: sprite, Name: def.Name,
			CurrentAlpha: 1.0,
		},
		MaxMovementPoints: def.Move,
		AttackType:        def.AttackType,
		MaxRange:          def.MaxRange,
	}
	g.Enemies = append(g.Enemies, enemy)
	g.addCombatLog(fmt.Sprintf("%s appears!", enemy.Name))
}

func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move, w, h int, sprite *ebiten.Image) {
	conMod := getModifier(con)
	maxHp := baseHp + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, Width: w, Height: h,
			HP: maxHp, MaxHP: maxHp, AC: ac,
			Strength: str, Dexterity: dex, Constitution: con,
			Intelligence: intel, Wisdom: wis, Charisma: cha,
			Sprite: sprite, Name: name,
			CurrentAlpha: 1.0,
		},
		MaxMovementPoints: move,
	}
	g.Enemies = append(g.Enemies, enemy)
}

func (g *Game) addCombatLog(msg string) {
	if g.CombatLog == nil {
		g.CombatLog = make([]string, 0, combatLogLength)
	}
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
}

func (g *Game) isTileBlocked(checkX, checkY, movingEnemyIndex int) bool {
	if g.Player != nil && !g.Player.IsDying && g.Player.HP > 0 && checkX >= g.Player.X && checkX < g.Player.X+g.Player.Width &&
		checkY >= g.Player.Y && checkY < g.Player.Y+g.Player.Height {
		return true
	}
	for i, enemy := range g.Enemies {
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}
		if i == movingEnemyIndex {
			continue
		}
		if checkX >= enemy.X && checkX < enemy.X+enemy.Width &&
			checkY >= enemy.Y && checkY < enemy.Y+enemy.Height {
			return true
		}
	}
	return false
}

func (g *Game) isTileFullyBlocked(checkX, checkY, entityWidth, entityHeight, movingEnemyIndex int) bool {
	for w := 0; w < entityWidth; w++ {
		for h := 0; h < entityHeight; h++ {
			tileX, tileY := checkX+w, checkY+h
			if tileX < 0 || tileX >= mapWidth || tileY < 0 || tileY >= mapHeight {
				return true
			}
			if g.isTileBlocked(tileX, tileY, movingEnemyIndex) {
				return true
			}
		}
	}
	return false
}

func (g *Game) getEnemyAt(x, y int) *Enemy {
	for i, enemy := range g.Enemies {
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}
		if x >= enemy.X && x < enemy.X+enemy.Width &&
			y >= enemy.Y && y < enemy.Y+enemy.Height {
			return g.Enemies[i]
		}
	}
	return nil
}

func (g *Game) buildAvailableActions() {
	g.availableActions = []*ActionDefinition{}
	possibleActions := make(map[string]bool)

	alwaysAvailable := map[string]bool{"wait": true}
	if g.Player.Class == "Mage" {
	} else {
		alwaysAvailable["melee_attack"] = true
		alwaysAvailable["ranged_attack"] = true
		alwaysAvailable["dash"] = true
		alwaysAvailable["disengage"] = true
	}

	for id := range alwaysAvailable {
		if _, exists := ActionTable[id]; exists {
			possibleActions[id] = true
		}
	}
	for _, cantripID := range g.Player.KnownCantrips {
		if _, exists := ActionTable[cantripID]; exists {
			possibleActions[cantripID] = true
		}
	}
	for _, spellID := range g.Player.KnownSpells {
		if _, exists := ActionTable[spellID]; exists {
			possibleActions[spellID] = true
		}
	}

	classDef, exists := ClassDefinitions[g.Player.Class]
	if exists {
		for actionID, classAction := range classDef.ClassActions {
			if g.Player.Level >= classAction.RequiredLevel {
				if _, actionDefExists := ActionTable[actionID]; actionDefExists {
					possibleActions[actionID] = true
				} else {
					log.Printf("Warning: ClassAction '%s' defined for class '%s' but not found in main ActionTable.", actionID, g.Player.Class)
				}
			}
		}
	}

	if g.Player.CombatTechnique == "Quick Strike" {
		if _, exists := ActionTable["quick_strike"]; exists {
			possibleActions["quick_strike"] = true
		}
	}

	tempAvailableActions := []*ActionDefinition{}
	for id := range possibleActions {
		actionDef := ActionTable[id]
		isAvailable := true

		switch actionDef.ActionType {
		case ActionTypeStandard:
			if g.Player.ActionTaken {
				isAvailable = false
			}
		case ActionTypeBonus:
			if g.Player.BonusActionTaken {
				isAvailable = false
			}
		case ActionTypeReaction:
			isAvailable = false
		case ActionTypeFree:
		}
		if !isAvailable {
			continue
		}

		if actionDef.ResourceType == ResourceSpellSlotL1 {
			if g.Player.SpellSlotsL1 < actionDef.ResourceCost {
				isAvailable = false
			}
		} else if actionDef.ResourceType == ResourceClassFeature {
			if classDef != nil {
				if classAction, ok := classDef.ClassActions[id]; ok {
					if classAction.UsesPerRest > 0 {
						if _, resOk := g.Player.ClassResources[id]; !resOk {
							isAvailable = false
						} else if g.Player.ClassResources[id] < classAction.ResourceCost {
							isAvailable = false
						}
					}
				}
			}
		}

		if !isAvailable {
			continue
		}
		if actionDef.ID == "disengage" && !isPlayerAdjacentToEnemy(g) {
			isAvailable = false
		}
		if isAvailable {
			tempAvailableActions = append(tempAvailableActions, actionDef)
		}
	}

	sort.Slice(tempAvailableActions, func(i, j int) bool {
		if tempAvailableActions[i].ActionType != tempAvailableActions[j].ActionType {
			return tempAvailableActions[i].ActionType < tempAvailableActions[j].ActionType
		}
		return tempAvailableActions[i].Name < tempAvailableActions[j].Name
	})

	g.availableActions = tempAvailableActions
}

func (g *Game) cleanupDeadEnemies() {
	initialCount := len(g.Enemies)
	aliveEnemies := make([]*Enemy, 0, len(g.Enemies))
	for _, enemy := range g.Enemies {
		if !enemy.IsDying || enemy.CurrentAlpha > 0 {
			aliveEnemies = append(aliveEnemies, enemy)
		}
	}
	if len(aliveEnemies) != initialCount {
		g.Enemies = aliveEnemies
	}
}

func (g *Game) Update() error {
	activeTexts := make([]*FloatingText, 0, len(g.FloatingTexts))
	for _, ft := range g.FloatingTexts {
		ft.Life--
		if ft.Life > 0 {
			ft.Y += ft.VelocityY
			activeTexts = append(activeTexts, ft)
		}
	}
	g.FloatingTexts = activeTexts

	if g.Player != nil {
		if g.Player.AttackBumpTimer > 0 {
			g.Player.AttackBumpTimer--
		}
		if g.Player.IsDying && g.Player.CurrentAlpha > 0 {
			g.Player.CurrentAlpha -= 1.0 / float64(deathFadeDuration)
			if g.Player.CurrentAlpha < 0 {
				g.Player.CurrentAlpha = 0
			}
		}
	}

	for _, enemy := range g.Enemies {
		if enemy.AttackBumpTimer > 0 {
			enemy.AttackBumpTimer--
		}
		if enemy.IsDying && enemy.CurrentAlpha > 0 {
			enemy.CurrentAlpha -= 1.0 / float64(deathFadeDuration)
			if enemy.CurrentAlpha < 0 {
				enemy.CurrentAlpha = 0
			}
		}
	}

	switch g.CurrentGameState {
	case StateClassSelection:
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.classSelectionIndex--
			if g.classSelectionIndex < 0 {
				g.classSelectionIndex = len(g.selectableClasses) - 1
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.classSelectionIndex++
			if g.classSelectionIndex >= len(g.selectableClasses) {
				g.classSelectionIndex = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			selected := g.selectableClasses[g.classSelectionIndex]
			if selected.IsAvailable {
				// Start game!
				g.InitGame(selected.Name)
			}
		}
	case StatePlaying:
		g.UpdatePlaying()
	case StateGameOverScreen:
		break
	}

	return nil
}

func (g *Game) UpdatePlaying() {
	if g.reactionPending {
		g.handlePlayerInput()
		return
	}

	if g.CurrentTurn == GameOver || g.CurrentGameState == StateGameOverScreen {
		return
	}

	if g.InputMode == InputModeLevelUp {
		g.handlePlayerInput()
		return
	}
	if g.InputMode == InputModeRestPrompt {
		g.handlePlayerInput()
		return
	}

	if g.CurrentTurn == PlayerTurn || g.InputMode == InputModeActionSelect || g.InputMode == InputModeCharacterSheet {
		g.handlePlayerInput()
	}

	if g.CurrentGameState != StatePlaying || g.InputMode == InputModeLevelUp || g.InputMode == InputModeRestPrompt {
		return
	}

	if g.CurrentTurn == EnemyTurn {
		g.handleEnemyTurns()
		if g.reactionPending {
			return
		}
		if g.Player.IsDying && g.Player.CurrentAlpha <= 0 && g.CurrentGameState != StateGameOverScreen {
			g.setGameOver(false)
			return
		}
		if g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
			g.endEnemyTurn()
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}
