package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

//go:embed assets/*
var assetsFS embed.FS

func loadImage(path string) (*ebiten.Image, error) {
	file, err := assetsFS.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded image %s: %w", path, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode embedded image %s: %w", path, err)
	}
	return ebiten.NewImageFromImage(img), nil
}

func getSpriteFromSheet(sheet *ebiten.Image, sx, sy int) *ebiten.Image {
	if sheet == nil {
		log.Println("Warning: Sprite sheet not loaded.")
		img := ebiten.NewImage(spriteSize, spriteSize)
		img.Fill(colorWhite)
		return img
	}
	x := sx * spriteSize
	y := sy * spriteSize
	rect := image.Rect(x, y, x+spriteSize, y+spriteSize)
	bounds := sheet.Bounds()
	if !rect.In(bounds) {
		log.Printf("Warning: Sprite index (%d, %d) out of sheet bounds (%v).", sx, sy, bounds)
		img := ebiten.NewImage(spriteSize, spriteSize)
		img.Fill(colorWhite)
		return img
	}
	return sheet.SubImage(rect).(*ebiten.Image)
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.CurrentGameState {
	case StateClassSelection:
		g.DrawClassSelection(screen)
	case StatePlaying:
		g.DrawPlaying(screen)
	case StateGameOverScreen:
		g.DrawPlaying(screen)
		g.DrawGameOver(screen)
	}
}

func (g *Game) DrawClassSelection(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 10, G: 10, B: 20, A: 255})
	title := "Select Your Class"
	titleFont := basicfont.Face7x13
	titleBounds := text.BoundString(titleFont, title)
	titleX := (screenWidth - titleBounds.Dx()) / 2
	titleY := screenHeight / 4
	text.Draw(screen, title, titleFont, titleX, titleY, colorWhite)

	itemStartY := titleY + 40
	itemLineHeight := 20
	itemX := screenWidth / 3

	for i, class := range g.selectableClasses {
		lineText := class.Name
		lineColor := colorGray

		if !class.IsAvailable {
			lineText += " (Locked)"
			lineColor = colorLocked
		}

		if i == g.classSelectionIndex {
			lineText = "> " + lineText
			if class.IsAvailable {
				lineColor = colorWhite
			} else {
				lineColor = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
			}
		}

		text.Draw(screen, lineText, titleFont, itemX, itemStartY+(i*itemLineHeight), lineColor)
	}
	helpText := "Up/Down to navigate, Enter to select"
	helpBounds := text.BoundString(titleFont, helpText)
	helpX := (screenWidth - helpBounds.Dx()) / 2
	helpY := screenHeight - 40
	text.Draw(screen, helpText, titleFont, helpX, helpY, colorGray)
}

func (g *Game) DrawPlaying(screen *ebiten.Image) {
	mapOffsetX, mapOffsetY := g.MapOffsetX, g.MapOffsetY
	tileOpts := &ebiten.DrawImageOptions{}
	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			screenX := float64(mapOffsetX + x*tileSize)
			screenY := float64(mapOffsetY + y*tileSize)
			tileOpts.GeoM.Reset()
			tileOpts.GeoM.Translate(screenX, screenY)
			if g.TileImage != nil {
				screen.DrawImage(g.TileImage, tileOpts)
			}
		}
	}

	if len(g.Player.Path) > 0 {
		pathColor := color.NRGBA{R: 100, G: 100, B: 255, A: 100}
		for i, pt := range g.Player.Path {
			screenX := float32(mapOffsetX + pt.X*tileSize)
			screenY := float32(mapOffsetY + pt.Y*tileSize)
			if i == len(g.Player.Path)-1 {
				vector.StrokeRect(screen, screenX, screenY, float32(tileSize), float32(tileSize), 2, pathColor, false)
			} else {
				vector.DrawFilledRect(screen, screenX+float32(tileSize/4), screenY+float32(tileSize/4), float32(tileSize/2), float32(tileSize/2), pathColor, false)
			}
		}
	}

	if g.CurrentTurn == PlayerTurn && g.InputMode == InputModeMap {
		cursorX, cursorY := ebiten.CursorPosition()
		gridX := (cursorX - g.MapOffsetX) / tileSize
		gridY := (cursorY - g.MapOffsetY) / tileSize
		if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight {
			hoverScreenX := float32(mapOffsetX + gridX*tileSize)
			hoverScreenY := float32(mapOffsetY + gridY*tileSize)
			hoverColor := color.NRGBA{R: 255, G: 255, B: 255, A: 100}

			if g.primedActionID != "" {
				hoverColor = color.NRGBA{R: 255, G: 100, B: 100, A: 100}
			} else if len(g.Player.Path) == 0 {
				if !g.isTileFullyBlocked(gridX, gridY, 1, 1, -1) {
					hoverColor = color.NRGBA{R: 100, G: 255, B: 100, A: 100}
				} else {
					hoverColor = color.NRGBA{R: 200, G: 50, B: 50, A: 100}
				}
			} else {
				hoverColor = color.NRGBA{A: 0}
			}

			vector.StrokeRect(screen, hoverScreenX, hoverScreenY, float32(tileSize), float32(tileSize), 1, hoverColor, false)
		}
	}
	if g.CurrentTurn == PlayerTurn && g.primedActionID != "" && g.RangeOverlayTile != nil {
		actionDef, exists := ActionTable[g.primedActionID]
		if exists && actionDef.RequiresTarget && actionDef.Range >= 0 && g.Player != nil {
			overlayOpts := &ebiten.DrawImageOptions{}
			originX, originY := g.Player.X, g.Player.Y
			for x := 0; x < mapWidth; x++ {
				for y := 0; y < mapHeight; y++ {
					dist := distance(originX, originY, x, y)
					isInRange := false
					if actionDef.Range == 0 && actionDef.Targeting == TargetSelf {
						continue
					} else if actionDef.Targeting == TargetEmptyTile {
						isInRange = dist > 0 && dist <= actionDef.Range
					} else {
						isInRange = dist <= actionDef.Range
					}

					if isInRange {
						isTargetValid := false
						switch actionDef.Targeting {
						case TargetSelf:
							isTargetValid = (x == originX && y == originY)
						case TargetEnemyAdjacent:
							enemy := g.getEnemyAt(x, y)
							if enemy != nil && isAdjacentToEntity(originX, originY, &enemy.Entity) {
								isTargetValid = true
							}
						case TargetEnemyRange:
							enemy := g.getEnemyAt(x, y)
							if enemy != nil {
								isTargetValid = true
							}
						case TargetEmptyTile:
							if !g.isTileFullyBlocked(x, y, 1, 1, -1) {
								isTargetValid = true
							}
						default:

							isTargetValid = true
						}

						if isTargetValid {
							screenX := float64(mapOffsetX + x*tileSize)
							screenY := float64(mapOffsetY + y*tileSize)
							overlayOpts.GeoM.Reset()
							overlayOpts.GeoM.Translate(screenX, screenY)
							screen.DrawImage(g.RangeOverlayTile, overlayOpts)
						}
					}
				}
			}
		}
	}

	entitiesToDraw := make([]*Entity, 0, len(g.Enemies)+1)
	for _, enemy := range g.Enemies {
		if !enemy.IsDying || enemy.CurrentAlpha > 0 {
			entitiesToDraw = append(entitiesToDraw, &enemy.Entity)
		}
	}
	if g.Player != nil && (!g.Player.IsDying || g.Player.CurrentAlpha > 0) {
		entitiesToDraw = append(entitiesToDraw, &g.Player.Entity)
	}
	sort.Slice(entitiesToDraw, func(i, j int) bool {
		if entitiesToDraw[i].Y != entitiesToDraw[j].Y {
			return entitiesToDraw[i].Y < entitiesToDraw[j].Y
		}
		return entitiesToDraw[i].X < entitiesToDraw[j].X
	})
	entityOpts := &ebiten.DrawImageOptions{}
	for _, entity := range entitiesToDraw {
		if entity.Sprite == nil {

			tempSprite := ebiten.NewImage(spriteSize*entity.Width, spriteSize*entity.Height)
			if entity == &g.Player.Entity {
				tempSprite.Fill(color.NRGBA{B: 200, A: 255})
			} else {
				tempSprite.Fill(color.NRGBA{R: 200, A: 255})
			}
			entity.Sprite = tempSprite
		}
		entityScreenX := float64(mapOffsetX + entity.X*tileSize)
		entityScreenY := float64(mapOffsetY + entity.Y*tileSize)
		bumpOffsetX := 0.0
		if entity.AttackBumpTimer > 0 {
			progress := float64(attackBumpDuration-entity.AttackBumpTimer) / float64(attackBumpDuration)
			bumpOffsetX = math.Sin(progress*math.Pi) * 4.0
			entityScreenX += bumpOffsetX
		}
		entityOpts.GeoM.Reset()

		scaleX := float64(entity.Width)
		scaleY := float64(entity.Height)
		entityOpts.GeoM.Scale(scaleX, scaleY)

		entityOpts.GeoM.Translate(entityScreenX, entityScreenY)
		entityOpts.ColorM.Reset()
		if entity.CurrentAlpha < 1.0 {
			entityOpts.ColorM.Scale(1, 1, 1, entity.CurrentAlpha)
		}
		screen.DrawImage(entity.Sprite, entityOpts)
		if entity.CurrentAlpha > 0 && entity.MaxHP > 0 {
			hpBarBaseX := float64(mapOffsetX+entity.X*tileSize) + bumpOffsetX
			hpBarBaseY := float64(mapOffsetY + (entity.Y+entity.Height)*tileSize)
			hpBarX, hpBarY := float32(hpBarBaseX), float32(hpBarBaseY+hpBarOffsetY)
			hpBarWidth := float32(tileSize * entity.Width)
			hpRatio := float32(max(0, entity.HP)) / float32(entity.MaxHP)
			hpRatio = maxF(0.0, minF(1.0, hpRatio))
			var hpColor color.NRGBA
			if hpRatio > 0.6 {
				hpColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
			} else if hpRatio > 0.3 {
				hpColor = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
			} else {
				hpColor = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
			}
			hpBgColor := color.NRGBA{R: 50, G: 50, B: 50, A: 255}
			hpBarAlpha := uint8(entity.CurrentAlpha * 255)
			hpColor.A, hpBgColor.A = hpBarAlpha, hpBarAlpha
			outlineColor := color.NRGBA{R: 0, G: 0, B: 0, A: hpBarAlpha}
			vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, hpBgColor, false)
			vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth*hpRatio, hpBarHeight, hpColor, false)
			vector.StrokeRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, 1, outlineColor, false)
		}
	}

	uiTextFont := basicfont.Face7x13
	uiLineHeight := 15

	statusStartY := 10
	if g.Player != nil {
		waveText := fmt.Sprintf("Wave: %d / %d", g.CurrentWaveIndex+1, len(g.WaveDefinitions))
		waveTextWidth := text.BoundString(uiTextFont, waveText).Dx()
		text.Draw(screen, waveText, uiTextFont, screenWidth-waveTextWidth-10, statusStartY, colorWhite)

		text.Draw(screen, fmt.Sprintf("Turn: %s", g.CurrentTurn.String()), uiTextFont, 10, statusStartY, colorWhite)

		currentY := statusStartY + uiLineHeight
		playerHpVal := max(0, g.Player.HP)

		effectiveAC := GetEffectiveAC(&g.Player.Entity)
		acString := fmt.Sprintf("%d", g.Player.AC)
		acBonusFromConditions := effectiveAC - g.Player.AC
		if acBonusFromConditions > 0 {
			acString = fmt.Sprintf("%d (+%d Buff)", effectiveAC, acBonusFromConditions)
		}

		text.Draw(screen, fmt.Sprintf("HP: %d/%d AC: %s", playerHpVal, g.Player.MaxHP, acString), uiTextFont, 10, currentY, colorWhite)
		currentY += uiLineHeight
		text.Draw(screen, fmt.Sprintf("Move: %d/%d", g.Player.MovementPoints, g.Player.MaxMovementPoints), uiTextFont, 10, currentY, colorWhite)
		currentY += uiLineHeight
		if g.Player.Class == "Mage" {
			text.Draw(screen, fmt.Sprintf("L1 Slots: %d/%d", g.Player.SpellSlotsL1, g.Player.MaxSpellSlotsL1), uiTextFont, 10, currentY, colorWhite)
		} else {
			text.Draw(screen, fmt.Sprintf("Hit Dice: %d/%d", g.Player.HitDice, g.Player.MaxHitDice), uiTextFont, 10, currentY, colorWhite)
		}
		currentY += uiLineHeight
		actionStatusText := "Action: Available"
		if g.Player.ActionTaken {
			actionStatusText = "Action: Used"
		}
		text.Draw(screen, actionStatusText, uiTextFont, 10, currentY, colorWhite)
		currentY += uiLineHeight
		bonusActionStatusText := "Bonus Action: Available"
		if g.Player.BonusActionTaken {
			bonusActionStatusText = "Bonus Action: Used"
		}
		text.Draw(screen, bonusActionStatusText, uiTextFont, 10, currentY, colorWhite)
		currentY += uiLineHeight
		reactionStatusText := "Reaction: Available"
		if g.Player.UsedReaction {
			reactionStatusText = "Reaction: Used"
		}
		text.Draw(screen, reactionStatusText, uiTextFont, 10, currentY, colorWhite)
		currentY += uiLineHeight
		primedActionText := "Primed: None"
		if g.primedActionID != "" {
			if actionDef, exists := ActionTable[g.primedActionID]; exists {
				primedActionText = fmt.Sprintf("Primed: %s", actionDef.Name)
			} else {
				primedActionText = fmt.Sprintf("Primed: ??? (%s)", g.primedActionID)
			}
		}
		text.Draw(screen, primedActionText, uiTextFont, 10, currentY, colorWhite)
		currentY += uiLineHeight

		statusText := ""
		for _, cond := range g.Player.Conditions {
			switch cond.Name {
			case ConditionSlowed:
				statusText += "Slowed "
			case ConditionMagicArmor:
				statusText += "MagicArmor "
			case ConditionExpeditiousRetreat:
				statusText += "Speedy "
			case ConditionFeatherFall:
				statusText += "Evasive "
			case ConditionShielded:
				statusText += "Shielded "
			}
		}
		if g.Player.IsDisengaging {
			statusText += "Disengaging "
		}

		if statusText != "" {
			text.Draw(screen, "Status: "+statusText, uiTextFont, 10, currentY, colorWhite)
		}
	}

	g.DrawUIBar(screen)

	if g.InputMode == InputModeCharacterSheet && g.Player != nil {
		charSheetFont := basicfont.Face7x13
		charSheetLineHeight := 14

		menuW, menuH := screenWidth/2+40, screenHeight/2+60
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 30, G: 20, B: 20, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, colorWhite, false)
		title := fmt.Sprintf("%s - Level %d %s ([C] / [Esc] to Close)", g.Player.Name, g.Player.Level, g.Player.Class)
		titleX, titleYCharSheet := menuX+10, menuY+15
		text.Draw(screen, title, charSheetFont, titleX, titleYCharSheet, colorWhite)

		infoStartY, col1X, col2X := titleYCharSheet+25, menuX+15, menuX+menuW/2
		currentLineNum := 0
		drawCharSheetLine := func(txt string, colX int, lineNum int) int {
			text.Draw(screen, txt, charSheetFont, colX, infoStartY+(lineNum*charSheetLineHeight), colorWhite)
			return lineNum + 1
		}

		currentLineNum = drawCharSheetLine(fmt.Sprintf("HP: %d / %d", max(0, g.Player.HP), g.Player.MaxHP), col1X, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("AC: %d", GetEffectiveAC(&g.Player.Entity)), col1X, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("Movement: %d", g.Player.MaxMovementPoints), col1X, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("Prof Bonus: +%d", g.Player.ProficiencyBonus), col1X, currentLineNum)
		classHitDieSize := 0
		if cd, ok := ClassDefinitions[g.Player.Class]; ok {
			classHitDieSize = cd.HitDieSize
		}
		currentLineNum = drawCharSheetLine(fmt.Sprintf("Hit Dice: %d / %d (d%d)", g.Player.HitDice, g.Player.MaxHitDice, classHitDieSize), col1X, currentLineNum)
		if g.Player.Class == "Mage" {
			currentLineNum = drawCharSheetLine(fmt.Sprintf("L1 Slots: %d / %d", g.Player.SpellSlotsL1, g.Player.MaxSpellSlotsL1), col1X, currentLineNum)
		}
		if g.Player.CombatStyle != "" {
			currentLineNum = drawCharSheetLine(fmt.Sprintf("Style: %s", g.Player.CombatStyle), col1X, currentLineNum)
		}
		if g.Player.CombatTechnique != "" {
			currentLineNum = drawCharSheetLine(fmt.Sprintf("Technique: %s", g.Player.CombatTechnique), col1X, currentLineNum)
		}
		currentLineNum = max(currentLineNum, 8)
		if g.Player.Class == "Mage" {
			intMod := getModifier(g.Player.Intelligence)
			dc := spellSaveDC(g)
			currentLineNum = drawCharSheetLine(fmt.Sprintf("Spell Atk: +%d", g.Player.ProficiencyBonus+intMod), col1X, currentLineNum)
			currentLineNum = drawCharSheetLine(fmt.Sprintf("Spell Save DC: %d", dc), col1X, currentLineNum)
		} else {
			meleeMod := getModifier(g.Player.Strength)
			rangedMod := getModifier(g.Player.Dexterity)
			currentLineNum = drawCharSheetLine(fmt.Sprintf("Melee Atk: +%d (1d8%+d)", g.Player.ProficiencyBonus+meleeMod, meleeMod), col1X, currentLineNum)
			currentLineNum = drawCharSheetLine(fmt.Sprintf("Ranged Atk: +%d (1d6%+d)", g.Player.ProficiencyBonus+rangedMod, rangedMod), col1X, currentLineNum)
		}
		currentLineNum = max(currentLineNum, 11)
		text.Draw(screen, "Class Features:", charSheetFont, col1X, infoStartY+(currentLineNum*charSheetLineHeight), colorGray)
		currentLineNum++
		classDef := ClassDefinitions[g.Player.Class]
		if classDef != nil {
			featureIDs := make([]string, 0, len(classDef.ClassActions))
			for id := range classDef.ClassActions {
				featureIDs = append(featureIDs, id)
			}
			sort.Slice(featureIDs, func(i, j int) bool {
				ca1, ca2 := classDef.ClassActions[featureIDs[i]], classDef.ClassActions[featureIDs[j]]
				if ca1.RequiredLevel != ca2.RequiredLevel {
					return ca1.RequiredLevel < ca2.RequiredLevel
				}
				return ca1.Name < ca2.Name
			})
			for _, id := range featureIDs {
				classAction := classDef.ClassActions[id]
				if g.Player.Level >= classAction.RequiredLevel {
					featureText := fmt.Sprintf(" L%d: %s", classAction.RequiredLevel, classAction.Name)
					if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
						uses := 0
						if val, ok := g.Player.ClassResources[id]; ok {
							uses = val
						}
						featureText += fmt.Sprintf(" (%d/%d)", uses, classAction.UsesPerRest)
						if classAction.RefreshesOn == RestTypeLong {
							featureText += " (LR)"
						} else if classAction.RefreshesOn == RestTypeShort {
							featureText += " (SR)"
						} else if classAction.RefreshesOn == RestTypeNever {
							featureText += " (Per Lvl)"
						}
					}
					text.Draw(screen, featureText, charSheetFont, col1X+5, infoStartY+(currentLineNum*charSheetLineHeight), colorWhite)
					currentLineNum++
				}
			}
		}
		if len(g.Player.KnownCantrips) > 0 {
			text.Draw(screen, "Known Cantrips:", charSheetFont, col1X, infoStartY+(currentLineNum*charSheetLineHeight), colorGray)
			currentLineNum++
			for _, cantripID := range g.Player.KnownCantrips {
				spellName := "??"
				if spellDef, exists := ActionTable[cantripID]; exists {
					spellName = spellDef.Name
				}
				text.Draw(screen, fmt.Sprintf(" - %s", spellName), charSheetFont, col1X+5, infoStartY+(currentLineNum*charSheetLineHeight), colorWhite)
				currentLineNum++
			}
		}
		if len(g.Player.KnownSpells) > 0 {
			text.Draw(screen, "Known Spells:", charSheetFont, col1X, infoStartY+(currentLineNum*charSheetLineHeight), colorGray)
			currentLineNum++
			for _, spellID := range g.Player.KnownSpells {
				spellName := "??"
				if spellDef, exists := ActionTable[spellID]; exists {
					spellName = spellDef.Name
				}
				text.Draw(screen, fmt.Sprintf(" - %s", spellName), charSheetFont, col1X+5, infoStartY+(currentLineNum*charSheetLineHeight), colorWhite)
				currentLineNum++
			}
		}

		currentLineNum = 0
		text.Draw(screen, "Attributes:", charSheetFont, col2X, infoStartY+(currentLineNum*charSheetLineHeight), colorGray)
		currentLineNum++
		currentLineNum = drawCharSheetLine(fmt.Sprintf("STR: %d (%+d)", g.Player.Strength, getModifier(g.Player.Strength)), col2X+5, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("DEX: %d (%+d)", g.Player.Dexterity, getModifier(g.Player.Dexterity)), col2X+5, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("CON: %d (%+d)", g.Player.Constitution, getModifier(g.Player.Constitution)), col2X+5, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("INT: %d (%+d)", g.Player.Intelligence, getModifier(g.Player.Intelligence)), col2X+5, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("WIS: %d (%+d)", g.Player.Wisdom, getModifier(g.Player.Wisdom)), col2X+5, currentLineNum)
		currentLineNum = drawCharSheetLine(fmt.Sprintf("CHA: %d (%+d)", g.Player.Charisma, getModifier(g.Player.Charisma)), col2X+5, currentLineNum)
	}

	logActiveFont := g.CombatLogFont
	if logActiveFont == nil {
		logActiveFont = basicfont.Face7x13
	}
	logLineHeight := 10
	if metrics := logActiveFont.Metrics(); metrics.Height != 0 {
		logLineHeight = metrics.Height.Ceil()
		if logLineHeight < 8 {
			logLineHeight = 8
		}
		logLineHeight += 1
	}

	logScreenHeight := combatLogLength * logLineHeight
	logPanelPadding := 5
	logPanelY := screenHeight - uiPanelHeight - logScreenHeight - logPanelPadding
	logPanelX := logPanelPadding

	if g.CombatLog != nil {
		for i, msg := range g.CombatLog {
			isRestPrompt, isLevelUpMsg, isReactionPrompt := false, false, false
			if g.Player != nil {
				isRestPrompt = g.InputMode == InputModeRestPrompt && i == len(g.CombatLog)-1 && msg == fmt.Sprintf("Wave Cleared! Spend 1 Hit Die (of %d) to heal? [Y/N]", g.Player.HitDice)
				levelUpMsgPattern := "LEVEL UP! Reached Level"
				isLevelUpMsg = g.InputMode == InputModeLevelUp && i == len(g.CombatLog)-1 && (strings.HasPrefix(msg, levelUpMsgPattern) || strings.HasPrefix(msg, "Learned") || strings.HasPrefix(msg, "Chosen"))
			}
			isReactionPrompt = g.InputMode == InputModeReactionPrompt && i == len(g.CombatLog)-1
			var msgColor color.Color = colorWhite
			if isRestPrompt || isLevelUpMsg || isReactionPrompt {
				msgColor = colorYellow
			}
			text.Draw(screen, msg, logActiveFont, logPanelX, logPanelY+(i*logLineHeight), msgColor)
		}
	}

	ftFont := basicfont.Face7x13
	for _, ft := range g.FloatingTexts {
		alpha := uint8(255 * (float64(ft.Life) / float64(ft.MaxLife)))
		nrgbaColor := color.NRGBAModel.Convert(ft.Color).(color.NRGBA)
		finalColor := nrgbaColor
		finalColor.A = alpha
		shadowColor := color.NRGBA{R: 0, G: 0, B: 0, A: alpha / 2}
		bounds := text.BoundString(ftFont, ft.Text)
		textX, textY := int(ft.X)-bounds.Dx()/2, int(ft.Y)
		text.Draw(screen, ft.Text, ftFont, textX+1, textY+1, shadowColor)
		text.Draw(screen, ft.Text, ftFont, textX, textY, finalColor)
	}

	levelUpFont := basicfont.Face7x13
	if g.InputMode == InputModeLevelUp && g.Player != nil {
		menuW, menuH := screenWidth/2, screenHeight/3+20
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 30, B: 20, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, colorWhite, false)

		levelUpTitle := fmt.Sprintf("Level %d Reached!", g.Player.Level)
		titleBounds := text.BoundString(levelUpFont, levelUpTitle)
		titleX := menuX + (menuW-titleBounds.Dx())/2
		titleY := menuY + 15
		text.Draw(screen, levelUpTitle, levelUpFont, titleX, titleY, colorWhite)
		lineStartY := titleY + 25
		lineNum := 0
		lineSpacing := 15
		choiceX := titleX

		choicePending := false
		if g.Player.Class == "Mage" {
			if g.Player.Level == 2 && len(g.Player.KnownSpells) < 3 {
				choicePending = true
				spellChoiceText := "Choose L1 Spell:"
				text.Draw(screen, spellChoiceText, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				level2Spells := []string{"arcane_blink", "burning_hands", "frost_nova", "mind_spike"}
				for i, spellID := range level2Spells {
					spellName := ActionTable[spellID].Name
					choiceLine := fmt.Sprintf("[%d] %s", i+1, spellName)
					text.Draw(screen, choiceLine, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			} else if g.Player.Level == 3 && len(g.Player.KnownCantrips) < 2 {
				choicePending = true
				cantripChoiceText := "Choose Cantrip:"
				text.Draw(screen, cantripChoiceText, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				level3Cantrips := []string{"ray_of_frost", "shocking_grasp"}
				for i, cantripID := range level3Cantrips {
					cantripName := ActionTable[cantripID].Name
					choiceLine := fmt.Sprintf("[%d] %s", i+1, cantripName)
					text.Draw(screen, choiceLine, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			} else if g.Player.Level == 4 && len(g.Player.KnownSpells) < 4 {
				choicePending = true
				spellChoiceText := "Choose L1 Spell:"
				text.Draw(screen, spellChoiceText, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				level4Spells := []string{"magic_armor", "elemental_strike", "feather_fall", "expeditious_retreat"}
				for i, spellID := range level4Spells {
					spellName := ActionTable[spellID].Name
					choiceLine := fmt.Sprintf("[%d] %s", i+1, spellName)
					text.Draw(screen, choiceLine, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			}
		} else if g.Player.Class == "Fighter" {
			if g.Player.Level == 3 && g.Player.CombatStyle == "" {
				choicePending = true
				styleChoiceText := "Choose Combat Style:"
				text.Draw(screen, styleChoiceText, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				styles := []string{"Gladiator (+2 Melee Dmg)", "Ranger (+2 Ranged Hit)", "Juggernaut (+2 AC)"}
				for i, styleDesc := range styles {
					choiceLine := fmt.Sprintf("[%d] %s", i+1, styleDesc)
					text.Draw(screen, choiceLine, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			} else if g.Player.Level == 4 && g.Player.CombatTechnique == "" {
				choicePending = true
				techChoiceText := "Choose Combat Technique:"
				text.Draw(screen, techChoiceText, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				techniques := []string{"Power Attack (-2 Hit/+50% Dmg)", "Defensive Stance (+2 AC/-1 Move)", "Quick Strike (Bonus Atk/Half Dmg)"}
				for i, techDesc := range techniques {
					choiceLine := fmt.Sprintf("[%d] %s", i+1, techDesc)
					text.Draw(screen, choiceLine, levelUpFont, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			}
		}

		if !choicePending {
			summaryText := "Check Character Sheet [C] for details."
			summaryBounds := text.BoundString(levelUpFont, summaryText)
			summaryX := menuX + (menuW-summaryBounds.Dx())/2
			summaryY := titleY + 25
			text.Draw(screen, summaryText, levelUpFont, summaryX, summaryY, colorGray)
			continueMsg := "Press [Enter] to Continue"
			continueBounds := text.BoundString(levelUpFont, continueMsg)
			continueX := menuX + (menuW-continueBounds.Dx())/2
			continueY := menuY + menuH - 30
			text.Draw(screen, continueMsg, levelUpFont, continueX, continueY, colorWhite)
		}
	}

	reactionFont := basicfont.Face7x13
	if g.InputMode == InputModeReactionPrompt {
		menuW, menuH := screenWidth/3, screenHeight/6
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 30, G: 30, B: 50, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, colorWhite, false)
		promptText := "Use Reaction? [Y/N]"
		if g.Player.Class == "Fighter" {
			promptText = "Use Riposte Reaction? [Y/N]"
		} else {
			promptText = "Use Shield Reaction? [Y/N]"
		}
		promptBounds := text.BoundString(reactionFont, promptText)
		promptX := menuX + (menuW-promptBounds.Dx())/2
		promptY := menuY + (menuH-promptBounds.Dy())/2
		text.Draw(screen, promptText, reactionFont, promptX, promptY, colorYellow)
	}
}

func (g *Game) DrawUIBar(screen *ebiten.Image) {
	panelY := float32(screenHeight - uiPanelHeight)
	vector.DrawFilledRect(screen, 0, panelY, float32(screenWidth), float32(uiPanelHeight), colorUIPanel, false)

	cursorX, cursorY := ebiten.CursorPosition()
	cursorPoint := image.Point{X: cursorX, Y: cursorY}
	tooltipText := ""
	var hoveredButton *UIButton = nil

	for i := range g.ActionButtons {
		btn := &g.ActionButtons[i]
		if cursorPoint.In(btn.Rect) {
			hoveredButton = btn
			tooltipText = btn.Tooltip
		}
	}

	for i := range g.ActionButtons {
		btn := &g.ActionButtons[i]
		btnColor := colorGray
		if hoveredButton == btn {
			btnColor = colorButtonHover
		}

		actionDef, primedExists := ActionTable[g.primedActionID]
		if g.primedActionID != "" && btn.ID == g.primedActionID {
			btnColor = colorYellow
		} else if g.primedActionID != "" && primedExists && !actionDef.RequiresTarget && btn.ID == g.primedActionID {
			btnColor = colorYellow
		}

		vector.DrawFilledRect(screen, float32(btn.Rect.Min.X), float32(btn.Rect.Min.Y), float32(uiButtonSize), float32(uiButtonSize), btnColor, false)
		vector.StrokeRect(screen, float32(btn.Rect.Min.X), float32(btn.Rect.Min.Y), float32(uiButtonSize), float32(uiButtonSize), 1, colorWhite, false)

		iconRune, iconExists := actionIconMap[btn.ID]
		if iconExists && g.IconFont.Face != nil {
			iconStr := string(iconRune)
			bounds, _ := font.BoundString(g.IconFont.Face, iconStr)

			boundWidth := bounds.Max.X.Ceil() - bounds.Min.X.Ceil()
			boundHeight := bounds.Max.Y.Ceil() - bounds.Min.Y.Ceil()

			manualOffsetX := -1
			manualOffsetY := -4

			baseX := btn.Rect.Min.X + (uiButtonSize-boundWidth)/2
			metrics := g.IconFont.Face.Metrics()
			baseY := btn.Rect.Min.Y + (uiButtonSize-boundHeight)/2 + metrics.Ascent.Ceil()

			iconX := baseX + manualOffsetX
			iconY := baseY + manualOffsetY

			text.Draw(screen, iconStr, g.IconFont.Face, iconX, iconY, colorIconDefault)
		} else {
			label := btn.ID
			if len(label) > 6 {
				label = label[:6]
			}
			tooltipFont := basicfont.Face7x13
			labelBounds := text.BoundString(tooltipFont, label)
			labelX := btn.Rect.Min.X + (uiButtonSize-labelBounds.Dx())/2
			labelY := btn.Rect.Min.Y + (uiButtonSize-labelBounds.Dy())/2 + labelBounds.Dy()
			text.Draw(screen, label, tooltipFont, labelX, labelY, colorBlack)
		}
	}

	if tooltipText != "" {
		tooltipFont := basicfont.Face7x13 // Or g.CombatLogFont if you decide to change tooltip font too
		// If you were to use g.CombatLogFont here and it might be nil:
		// if g.CombatLogFont != nil {
		//    tooltipFont = g.CombatLogFont
		// }

		tooltipBounds := text.BoundString(tooltipFont, tooltipText)
		tooltipHeight := tooltipBounds.Dy() // This is the height of the text block itself

		tooltipX := cursorX + 10
		tooltipY := cursorY - 5 // Position tooltip slightly above cursor

		// Adjust if tooltip goes off screen (simplified logic)
		if tooltipY-tooltipHeight < 0 { // If the top of the tooltip goes above screen
			tooltipY = cursorY + 15 + tooltipHeight // Move below cursor
		}
		if tooltipX+tooltipBounds.Dx()+4 > screenWidth {
			tooltipX = cursorX - tooltipBounds.Dx() - 10
		}
		if tooltipX < 0 {
			tooltipX = 0
		}

		padding := 3
		bgX := float32(tooltipX - padding)
		// The Y for text.Draw is the baseline, so bgY needs to account for that.
		// Ascent is distance from baseline to top.
		ascent := tooltipFont.Metrics().Ascent.Ceil()
		bgY := float32(tooltipY - ascent - padding) // Background starts above the baseline
		bgW := float32(tooltipBounds.Dx() + padding*2)
		bgH := float32(tooltipHeight + padding*2) // tooltipHeight is the full block height

		vector.DrawFilledRect(screen, bgX, bgY, bgW, bgH, colorBlack, false)
		// For text.Draw, Y is the baseline.
		text.Draw(screen, tooltipText, tooltipFont, tooltipX, tooltipY, colorWhite)
	}
}

func (g *Game) DrawGameOver(screen *ebiten.Image) {
	gameOverMsg := "GAME OVER"
	if g.isVictory {
		gameOverMsg = "VICTORY!"
	}
	quitMsg := "Press [Q] to Quit"

	msgFont := basicfont.Face7x13
	msgBounds := text.BoundString(msgFont, gameOverMsg)
	quitBounds := text.BoundString(msgFont, quitMsg)

	msgX := (screenWidth - msgBounds.Dx()) / 2
	msgY := (screenHeight / 2) - msgBounds.Dy()

	quitX := (screenWidth - quitBounds.Dx()) / 2
	quitY := (screenHeight / 2) + 5

	overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	vector.DrawFilledRect(screen, 0, 0, float32(screenWidth), float32(screenHeight), overlayColor, false)

	text.Draw(screen, gameOverMsg, msgFont, msgX+1, msgY+1, colorBlack)
	text.Draw(screen, gameOverMsg, msgFont, msgX, msgY, colorWhite)

	text.Draw(screen, quitMsg, msgFont, quitX+1, quitY+1, colorBlack)
	text.Draw(screen, quitMsg, msgFont, quitX, quitY, colorGray)
}

func StartScreen(startInGodMode bool) *Game {
	g := &Game{}
	g.CurrentGameState = StateClassSelection
	g.godModeEnabled = startInGodMode
	if g.godModeEnabled {
		log.Println("Game starting with GOD MODE ENABLED via command-line flag.")
	}
	g.selectableClasses = []SelectableClass{
		{Name: "Fighter", IsAvailable: true},
		{Name: "Mage", IsAvailable: true},
		{Name: "Druid", IsAvailable: false},
		{Name: "Cleric", IsAvailable: false},
		{Name: "Rogue", IsAvailable: false},
	}
	g.classSelectionIndex = 0

	for i, class := range g.selectableClasses {
		if class.IsAvailable {
			g.classSelectionIndex = i
			break
		}
	}
	return g
}
