package main

import (
	"fmt"
	"image/color"
	"math/rand"
)

func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int, attackType string) (bool, bool) {
	if attacker == nil || defender == nil || attacker.HP <= 0 || defender.HP <= 0 || defender.IsDying {
		return false, false
	}

	isPlayerAttacking := g.Player != nil && attacker == &g.Player.Entity
	isPlayerDefending := g.Player != nil && defender == &g.Player.Entity

	var attackAbilityMod int
	var abilityName string
	var enemyDef *EnemyDefinition

	attackPenalty := 0
	damageMultiplier := 1.0

	if isPlayerDefending && g.Player.FeatherFallDuration > 0 {
		if rand.Intn(2) == 0 {
			g.addCombatLog("Player nimbly dodges (Feather Fall)!")
			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
				Text: "Dodge!", X: float64(g.MapOffsetX + defender.X*tileSize + (defender.Width*tileSize)/2), Y: float64(g.MapOffsetY+defender.Y*tileSize) - 15, Life: 30, MaxLife: 30, Color: colorWhite, VelocityY: -0.5,
			})
			return false, false
		}
	}

	if isPlayerAttacking {
		if g.Player.CombatTechnique == "Power Attack" {
			attackPenalty = -2
			damageMultiplier = 1.5
		}
		switch attackType {
		case "ranged":
			attackAbilityMod = getModifier(attacker.Dexterity)
			abilityName = "DEX"
			if g.Player.CombatStyle == "Ranger" {
				attackerProfBonus += 2
			}
		default:
			attackAbilityMod = getModifier(attacker.Strength)
			abilityName = "STR"
		}
	} else {
		foundDef := false
		var tempDef EnemyDefinition
		for _, def := range EnemyDefinitions {
			if def.Name == attacker.Name {
				tempDef = def
				enemyDef = &tempDef
				foundDef = true
				break
			}
		}
		if foundDef {
			switch enemyDef.AttackAbilityMod {
			case "STR":
				attackAbilityMod = getModifier(attacker.Strength)
				abilityName = "STR"
			case "DEX":
				attackAbilityMod = getModifier(attacker.Dexterity)
				abilityName = "DEX"
			case "INT":
				attackAbilityMod = getModifier(attacker.Intelligence)
				abilityName = "INT"
			case "WIS":
				attackAbilityMod = getModifier(attacker.Wisdom)
				abilityName = "WIS"
			case "CHA":
				attackAbilityMod = getModifier(attacker.Charisma)
				abilityName = "CHA"
			default:
				if enemyDef.AttackType == "ranged" {
					attackAbilityMod = getModifier(attacker.Dexterity)
					abilityName = "DEX"
				} else {
					attackAbilityMod = getModifier(attacker.Strength)
					abilityName = "STR"
				}
			}
		} else {
			if attackType == "ranged" {
				attackAbilityMod = getModifier(attacker.Dexterity)
				abilityName = "DEX"
			} else {
				attackAbilityMod = getModifier(attacker.Strength)
				abilityName = "STR"
			}
		}
	}

	roll := rand.Intn(20) + 1
	effectiveAC := defender.AC
	if isPlayerDefending {
		effectiveAC += g.Player.ACBonusUntilNextTurn
	}

	attackRoll := roll + attackerProfBonus + attackAbilityMod + attackPenalty
	isCrit := roll == 20
	isFumble := roll == 1
	hit := !isFumble && (isCrit || attackRoll >= effectiveAC)

	modString := fmt.Sprintf("%+d", attackAbilityMod)
	profString := ""
	if attackerProfBonus != 0 {
		profString = fmt.Sprintf("+%d", attackerProfBonus)
	}
	penaltyString := ""
	if attackPenalty != 0 {
		penaltyString = fmt.Sprintf("%d", attackPenalty)
	}
	rollString := fmt.Sprintf("%d%s%s%s(%s)=%d", roll, profString, modString, penaltyString, abilityName, attackRoll)
	attackVerb := "attacks"
	if attackType == "ranged" {
		attackVerb = "shoots"
		if !isPlayerAttacking && attacker.Name == "Orc Shaman" {
			attackVerb = "casts Bone Chill at"
		}
	}
	logMsg := fmt.Sprintf("%s %s %s(AC%d). Roll: %s.", attacker.Name, attackVerb, defender.Name, effectiveAC, rollString)
	if isPlayerDefending && g.Player.ACBonusUntilNextTurn > 0 {
		logMsg = fmt.Sprintf("%s %s %s(AC%d+%d Buff=%d). Roll: %s.", attacker.Name, attackVerb, defender.Name, defender.AC, g.Player.ACBonusUntilNextTurn, effectiveAC, rollString)
	}

	textSpawnX := float64(g.MapOffsetX + defender.X*tileSize + (defender.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + defender.Y*tileSize)
	textLifetime := 60
	killed := false
	attackLanded := false

	if isPlayerDefending && hit && !g.Player.UsedReaction {
		playerKnowsShield := false
		for _, known := range g.Player.KnownSpells {
			if known == "shield" {
				playerKnowsShield = true
				break
			}
		}
		if !playerKnowsShield && g.Player.Class == "Mage" {
			playerKnowsShield = true
		}
		if playerKnowsShield && g.Player.SpellSlotsL1 > 0 {
			g.addCombatLog("Hit detected! Use Shield reaction? [Y/N]")
			g.reactionPending = true
			g.InputMode = InputModeReactionPrompt
			g.addCombatLog(logMsg + " (Pending Reaction...)")
			return false, false
		}
	}

	if isFumble {
		logMsg += " FUMBLE! Miss!"
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "FUMBLE!", X: textSpawnX, Y: textSpawnY - 15, Life: textLifetime / 2, MaxLife: textLifetime / 2, Color: color.NRGBA{R: 150, G: 0, B: 0, A: 255}, VelocityY: -0.5})
	} else if hit {
		attackLanded = true
		attacker.AttackBumpTimer = attackBumpDuration
		var damage int
		damageRoll1, damageRoll2 := 0, 0
		damageLog := ""
		baseDamage, styleDamageBonus := 0, 0
		if isCrit {
			logMsg += " CRITICAL!"
		}

		if isPlayerAttacking { // Player DMG Calculation
			switch attackType {
			case "ranged":
				damageRoll1 = rand.Intn(6) + 1
				if isCrit {
					damageRoll2 = rand.Intn(6) + 1
				}
				baseDamage = damageRoll1 + damageRoll2 + attackAbilityMod
				if isCrit {
					damageLog = fmt.Sprintf(" (2d6[%d,%d]%+d)", damageRoll1, damageRoll2, attackAbilityMod)
				} else {
					damageLog = fmt.Sprintf(" (1d6[%d]%+d)", damageRoll1, attackAbilityMod)
				}
			default:
				damageRoll1 = rand.Intn(8) + 1
				if isCrit {
					damageRoll2 = rand.Intn(8) + 1
				}
				if g.Player.CombatStyle == "Gladiator" {
					styleDamageBonus = 2
				}
				baseDamage = damageRoll1 + damageRoll2 + attackAbilityMod + styleDamageBonus
				if isCrit {
					damageLog = fmt.Sprintf(" (2d8[%d,%d]%+d", damageRoll1, damageRoll2, attackAbilityMod)
				} else {
					damageLog = fmt.Sprintf(" (1d8[%d]%+d", damageRoll1, attackAbilityMod)
				}
				if styleDamageBonus > 0 {
					damageLog += fmt.Sprintf("+%d Style", styleDamageBonus)
				}
				damageLog += ")"
			}
		} else { // Enemy DMG Calculation
			if enemyDef != nil && enemyDef.AttackDiceNum > 0 && enemyDef.AttackDiceSize > 0 {
				totalDiceRoll := 0
				diceRolls := []int{}
				numDiceToRoll := enemyDef.AttackDiceNum
				if isCrit {
					numDiceToRoll *= 2
				}
				for i := 0; i < numDiceToRoll; i++ {
					roll := rand.Intn(enemyDef.AttackDiceSize) + 1
					totalDiceRoll += roll
					diceRolls = append(diceRolls, roll)
				}
				baseDamage = totalDiceRoll + attackAbilityMod
				rollsStr := ""
				for i, r := range diceRolls {
					rollsStr += fmt.Sprintf("%d", r)
					if i < len(diceRolls)-1 {
						rollsStr += ","
					}
				}
				damageLog = fmt.Sprintf(" (%dd%d[%s]%+d)", numDiceToRoll, enemyDef.AttackDiceSize, rollsStr, attackAbilityMod)
			} else {
				baseDamage = max(1, attackAbilityMod)
				if isCrit {
					baseDamage = max(1, baseDamage*2)
				}
				damageLog = fmt.Sprintf(" (%+d)", baseDamage)
			}
		}

		damage = int(float64(max(1, baseDamage)) * damageMultiplier)
		actualDamage := min(damage, defender.HP)
		defender.HP -= actualDamage
		logMsg += fmt.Sprintf(" Hit! Deals %d%s dmg.", actualDamage, damageLog)
		if damageMultiplier != 1.0 {
			logMsg += fmt.Sprintf(" (%.1fx Power Attack)", damageMultiplier)
		}

		if isPlayerDefending && !isPlayerAttacking && attacker.Name == "Orc Shaman" {
			g.Player.IsSlowed = true
			g.Player.SlowDuration = 2
			logMsg += " Player is Slowed!"
			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Slowed!", X: textSpawnX, Y: textSpawnY + 15, Life: textLifetime, MaxLife: textLifetime, Color: visualEffectColors["Cold"], VelocityY: -0.5})
		}

		ftColor := visualEffectColors["Default"]
		if !isPlayerAttacking && attacker.Name == "Orc Shaman" {
			ftColor = visualEffectColors["Cold"]
		}
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: textLifetime, MaxLife: textLifetime, Color: ftColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: textLifetime / 2, MaxLife: textLifetime / 2, Color: hitTextColor, VelocityY: -0.5})

		if defender.HP <= 0 {
			logMsg += fmt.Sprintf(" %s dies!", defender.Name)
			defender.IsDying = true
			killed = true
			if isPlayerDefending {
				g.setGameOver(false)
			}
		}
	} else if !isFumble {
		logMsg += " Miss!"
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: textLifetime / 2, MaxLife: textLifetime / 2, Color: colorGray, VelocityY: -0.5})
	}
	g.addCombatLog(logMsg)
	return killed, attackLanded
}

func (g *Game) executeAction(actionDef *ActionDefinition, targetX, targetY int) bool {
	if actionDef == nil {
		g.addCombatLog("Error: Tried to execute nil action.")
		return false
	}

	classDef := ClassDefinitions[g.Player.Class]
	var classAction *ClassAction
	if classDef != nil {
		classAction = classDef.ClassActions[actionDef.ID]
	}

	if actionDef.ResourceType == ResourceSpellSlotL1 {
		if g.Player.SpellSlotsL1 < actionDef.ResourceCost {
			g.addCombatLog(fmt.Sprintf("Not enough L1 spell slots for %s (%d needed).", actionDef.Name, actionDef.ResourceCost))
			return false
		}
	} else if actionDef.ResourceType == ResourceClassFeature {
		if classAction != nil && classAction.UsesPerRest > 0 {
			if _, ok := g.Player.ClassResources[actionDef.ID]; !ok {
				g.addCombatLog(fmt.Sprintf("Resource %s not initialized for player.", actionDef.ID))
				return false
			}
			if g.Player.ClassResources[actionDef.ID] < classAction.ResourceCost {
				g.addCombatLog(fmt.Sprintf("Not enough uses left for %s.", actionDef.Name))
				return false
			}
		}
	}

	switch actionDef.ActionType {
	case ActionTypeStandard:
		if g.Player.ActionTaken {
			g.addCombatLog(fmt.Sprintf("Cannot use %s: Action already used.", actionDef.Name))
			return false
		}
	case ActionTypeBonus:
		if g.Player.BonusActionTaken {
			g.addCombatLog(fmt.Sprintf("Cannot use %s: Bonus Action already used.", actionDef.Name))
			return false
		}
	case ActionTypeReaction:
		if g.Player.UsedReaction {
			g.addCombatLog(fmt.Sprintf("Cannot use %s: Reaction already used.", actionDef.Name))
			return false
		}
	case ActionTypeFree:

	}

	success := actionDef.Execute(g, targetX, targetY)

	if success {
		g.lastExecutedActionID = actionDef.ID

		if actionDef.ResourceType == ResourceSpellSlotL1 {
			g.Player.SpellSlotsL1 -= actionDef.ResourceCost
		} else if actionDef.ResourceType == ResourceClassFeature {
			if classAction != nil && classAction.UsesPerRest > 0 {
				g.Player.ClassResources[actionDef.ID] -= classAction.ResourceCost
			}
		}

		switch actionDef.ActionType {
		case ActionTypeStandard:
			g.Player.ActionTaken = true
		case ActionTypeBonus:
			g.Player.BonusActionTaken = true
		case ActionTypeReaction:
			g.Player.UsedReaction = true
		case ActionTypeFree:

		}

		if actionDef.ID == "action_surge" {
			g.Player.ActionTaken = false
		}

		if actionDef.ID == "wait" {
			g.endPlayerTurn()
		}
	}

	return success
}

// --- Execute ---

func executeMeleeAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil && isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "melee")
		if g.reactionPending {
			return false
		}
		return true
	}
	g.addCombatLog("Invalid target for melee attack (or not adjacent).")
	return false
}

func executeRangedAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil {
		dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
		if dist <= playerRangedRange {
			g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "ranged")
			if g.reactionPending {
				return false
			}
			return true
		}
		g.addCombatLog(fmt.Sprintf("Target %s out of range (%d > %d).", targetEnemy.Name, dist, playerRangedRange))
		return false
	}
	g.addCombatLog("No valid target selected at cursor for ranged attack.")
	return false
}

func executeDash(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Dash!")
	g.Player.MovementPoints += g.Player.MaxMovementPoints
	return true
}

func executeDisengage(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Disengage!")
	g.Player.IsDisengaging = true
	return true
}

func executeWait(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player waits, ending turn.")
	return true
}

func executeSecondWind(g *Game, targetX, targetY int) bool {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return false
	}
	classAction := classDef.ClassActions["second_wind"]
	if classAction == nil {
		return false
	}

	conMod := getModifier(g.Player.Constitution)
	healRoll := rand.Intn(classDef.HitDieSize) + 1
	healAmount := max(1, healRoll+conMod)
	actualHeal := min(healAmount, g.Player.MaxHP-g.Player.HP)
	actualHeal = max(0, actualHeal)

	if actualHeal > 0 {
		g.Player.HP += actualHeal
		g.addCombatLog(fmt.Sprintf("Used Second Wind! Healed %d HP (Rolled %d%+d).", actualHeal, healRoll, conMod))
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("+%d", actualHeal),
			X:         float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2),
			Y:         float64(g.MapOffsetY + g.Player.Y*tileSize),
			Life:      60,
			MaxLife:   60,
			Color:     visualEffectColors["Heal"],
			VelocityY: -0.5,
		})
	} else {
		g.addCombatLog("Used Second Wind, but already at full HP.")
	}

	return true
}

func executeActionSurge(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Action Surge! Gains an extra action.")
	return true
}

func executeQuickStrike(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil || !isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		g.addCombatLog("Invalid target for Quick Strike (or not adjacent).")
		return false
	}

	attackAbilityMod := getModifier(g.Player.Strength)
	roll := rand.Intn(20) + 1
	isCrit := roll == 20
	isFumble := roll == 1
	attackRoll := roll + g.Player.ProficiencyBonus + attackAbilityMod
	effectiveAC := targetEnemy.AC

	rollString := fmt.Sprintf("%d+%d+%d(STR)=%d", roll, g.Player.ProficiencyBonus, attackAbilityMod, attackRoll)
	logMsg := fmt.Sprintf("Player Quick Strikes %s(AC%d). Roll: %s.", targetEnemy.Name, effectiveAC, rollString)
	hit := !isFumble && (isCrit || attackRoll >= effectiveAC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(8) + 1
		}
		baseDamage := damageRoll1 + damageRoll2 + attackAbilityMod
		finalDamage := max(1, baseDamage/2)
		actualDamage := min(finalDamage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (Half of 2d8[%d,%d]%+d) dmg.", actualDamage, damageRoll1, damageRoll2, attackAbilityMod)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (Half of 1d8[%d]%+d) dmg.", actualDamage, damageRoll1, attackAbilityMod)
		}
		logMsg += dmgLog

		ftColor := visualEffectColors["Default"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text: fmt.Sprintf("-%d", actualDamage),
			X:    textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: ftColor, VelocityY: -0.5,
		})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text: hitText,
			X:    textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5,
		})

		if targetEnemy.HP <= 0 {
			logMsg += fmt.Sprintf(" %s dies!", targetEnemy.Name)
			targetEnemy.IsDying = true
		}
	} else {
		logMsg += " Miss!"
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text: "Miss!",
			X:    textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5,
		})
	}

	g.addCombatLog(logMsg)
	return true
}

func executeFirebolt(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil {
		g.addCombatLog("Invalid target for Firebolt.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)

	actionRange := ActionTable["firebolt"].Range

	if dist > actionRange {
		g.addCombatLog("Target out of range for Firebolt.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Firebolt at %s!", targetEnemy.Name))
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, getModifier(g.Player.Intelligence), g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(10) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(10) + 1
		}
		damage := damageRoll1 + damageRoll2
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (2d10[%d,%d]) fire damage.", actualDamage, damageRoll1, damageRoll2)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (1d10[%d]) fire damage.", actualDamage, damageRoll1)
		}
		g.addCombatLog(attackLog + dmgLog)

		ftColor := visualEffectColors["Fire"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}

		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("-%d", actualDamage),
			X:         textSpawnX,
			Y:         textSpawnY,
			Life:      60,
			MaxLife:   60,
			Color:     ftColor,
			VelocityY: -0.5,
		})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      hitText,
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     hitTextColor,
			VelocityY: -0.5,
		})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      "Miss!",
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     colorGray,
			VelocityY: -0.5,
		})
	}

	return true
}

func executeMagicMissile(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil {
		g.addCombatLog("Invalid target for Magic Missile.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)

	actionRange := ActionTable["magic_missile"].Range

	if dist > actionRange {
		g.addCombatLog("Target out of range for Magic Missile.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Magic Missile at %s!", targetEnemy.Name))
	totalDamage := 0
	damageLog := " (["
	boltCount := 3

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	for i := 0; i < boltCount; i++ {
		boltDamage := rand.Intn(4) + 1 + 1
		totalDamage += boltDamage
		damageLog += fmt.Sprintf("%d", boltDamage)
		if i < boltCount-1 {
			damageLog += ","
		}

		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("-%d", boltDamage),
			X:         textSpawnX + float64(rand.Intn(10)-5),
			Y:         textSpawnY + float64(rand.Intn(10)-5),
			Life:      30 + i*5,
			MaxLife:   30 + i*5,
			Color:     visualEffectColors["Force"],
			VelocityY: -0.3 - float64(i)*0.05,
		})
	}
	damageLog += "])"

	actualDamage := min(totalDamage, targetEnemy.HP)
	targetEnemy.HP -= actualDamage
	g.addCombatLog(fmt.Sprintf(" Hits automatically! Deals %d total%s force damage.", actualDamage, damageLog))

	if targetEnemy.HP <= 0 {
		g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
		targetEnemy.IsDying = true
	}

	return true
}

func executeShield(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Used Reaction: Shield!")
	g.Player.ACBonusUntilNextTurn = 5
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text:      "+5 AC!",
		X:         float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2),
		Y:         float64(g.MapOffsetY+g.Player.Y*tileSize) - 15,
		Life:      60,
		MaxLife:   60,
		Color:     visualEffectColors["Arcane"],
		VelocityY: -0.5,
	})
	return true
}

func executeArcaneBlink(g *Game, targetX, targetY int) bool {
	if g.isTileFullyBlocked(targetX, targetY, g.Player.Width, g.Player.Height, -1) {
		g.addCombatLog("Cannot Arcane Blink into blocked tile.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetX, targetY)
	maxRange := playerBaseMovement
	if dist > maxRange {
		g.addCombatLog(fmt.Sprintf("Target tile too far for Arcane Blink (Max %d).", maxRange))
		return false
	}

	g.addCombatLog(fmt.Sprintf("Used Arcane Blink to (%d, %d)!", targetX, targetY))
	g.Player.X = targetX
	g.Player.Y = targetY
	g.Player.MovementPoints = 0
	return true
}

func executeBurningHands(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Burning Hands!")
	damageRoll1 := rand.Intn(6) + 1
	damageRoll2 := rand.Intn(6) + 1
	damageRoll3 := rand.Intn(6) + 1
	damage := damageRoll1 + damageRoll2 + damageRoll3
	damage = max(1, damage)
	g.addCombatLog(fmt.Sprintf("Deals %d (3d6[%d,%d,%d]) fire damage in a cone.", damage, damageRoll1, damageRoll2, damageRoll3))

	targetsHit := 0
	for i := len(g.Enemies) - 1; i >= 0; i-- {
		enemy := g.Enemies[i]
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}

		if isAdjacentSimple(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			actualDamage := min(damage, enemy.HP)
			enemy.HP -= actualDamage
			g.addCombatLog(fmt.Sprintf(" Hits %s for %d damage!", enemy.Name, actualDamage))
			targetsHit++

			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
				Text:      fmt.Sprintf("-%d", actualDamage),
				X:         float64(g.MapOffsetX + enemy.X*tileSize + (enemy.Width*tileSize)/2),
				Y:         float64(g.MapOffsetY + enemy.Y*tileSize),
				Life:      60,
				MaxLife:   60,
				Color:     visualEffectColors["Fire"],
				VelocityY: -0.5,
			})

			if enemy.HP <= 0 {
				g.addCombatLog(fmt.Sprintf(" %s burns away!", enemy.Name))
				enemy.IsDying = true
			}
		}
	}

	if targetsHit == 0 {
		g.addCombatLog(" No targets hit.")
	}

	return true
}

func executeFrostNova(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Frost Nova!")
	damageRoll1 := rand.Intn(6) + 1
	damageRoll2 := rand.Intn(6) + 1
	damage := damageRoll1 + damageRoll2
	damage = max(1, damage)
	g.addCombatLog(fmt.Sprintf("Deals %d (2d6[%d,%d]) cold damage to adjacent enemies.", damage, damageRoll1, damageRoll2))

	targetsHit := 0
	for i := len(g.Enemies) - 1; i >= 0; i-- {
		enemy := g.Enemies[i]
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}

		if isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity) {
			actualDamage := min(damage, enemy.HP)
			enemy.HP -= actualDamage
			g.addCombatLog(fmt.Sprintf(" Hits %s for %d damage!", enemy.Name, actualDamage))
			targetsHit++

			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
				Text:      fmt.Sprintf("-%d", actualDamage),
				X:         float64(g.MapOffsetX + enemy.X*tileSize + (enemy.Width*tileSize)/2),
				Y:         float64(g.MapOffsetY + enemy.Y*tileSize),
				Life:      60,
				MaxLife:   60,
				Color:     visualEffectColors["Cold"],
				VelocityY: -0.5,
			})

			if enemy.HP <= 0 {
				g.addCombatLog(fmt.Sprintf(" %s freezes and shatters!", enemy.Name))
				enemy.IsDying = true
			}
		}
	}

	if targetsHit == 0 {
		g.addCombatLog(" No targets hit.")
	}

	return true
}

func executeMindSpike(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil {
		g.addCombatLog("Invalid target for Mind Spike.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
	maxRange := playerRangedRange
	if dist > maxRange {
		g.addCombatLog("Target out of range for Mind Spike.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Mind Spike at %s!", targetEnemy.Name))
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, getModifier(g.Player.Intelligence), g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := rand.Intn(8) + 1
		damageRoll3 := 0
		damageRoll4 := 0
		if isCrit {
			damageRoll3 = rand.Intn(8) + 1
			damageRoll4 = rand.Intn(8) + 1
		}
		damage := damageRoll1 + damageRoll2 + damageRoll3 + damageRoll4
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (4d8[%d,%d,%d,%d]) psychic damage.", actualDamage, damageRoll1, damageRoll2, damageRoll3, damageRoll4)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (2d8[%d,%d]) psychic damage.", actualDamage, damageRoll1, damageRoll2)
		}
		g.addCombatLog(attackLog + dmgLog)

		ftColor := visualEffectColors["Psychic"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("-%d", actualDamage),
			X:         textSpawnX,
			Y:         textSpawnY,
			Life:      60,
			MaxLife:   60,
			Color:     ftColor,
			VelocityY: -0.5,
		})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      hitText,
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     hitTextColor,
			VelocityY: -0.5,
		})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s's mind is shattered!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      "Miss!",
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     colorGray,
			VelocityY: -0.5,
		})
	}
	return true
}

func executeRayOfFrost(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	actionDef := ActionTable["ray_of_frost"]
	if targetEnemy == nil {
		g.addCombatLog("Invalid target.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
	if dist > actionDef.Range {
		g.addCombatLog("Target out of range.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Ray of Frost at %s!", targetEnemy.Name))
	// casterAbilityMod := getModifier(g.Player.Wisdom)
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, getModifier(g.Player.Intelligence), g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(8) + 1
		}
		damage := damageRoll1 + damageRoll2
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (2d8[%d,%d]) cold dmg.", actualDamage, damageRoll1, damageRoll2)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (1d8[%d]) cold dmg.", actualDamage, damageRoll1)
		}
		logMsg := attackLog + dmgLog
		g.addCombatLog(logMsg)

		ftColor := visualEffectColors["Cold"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: ftColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5})
	}
	return true
}

func executeShockingGrasp(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil || !isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		g.addCombatLog("Invalid target (or not adjacent).")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Shocking Grasp on %s!", targetEnemy.Name))
	casterAbilityMod := getModifier(g.Player.Intelligence)
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, casterAbilityMod, g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(8) + 1
		}
		damage := damageRoll1 + damageRoll2
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (2d8[%d,%d]) lightning dmg.", actualDamage, damageRoll1, damageRoll2)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (1d8[%d]) lightning dmg.", actualDamage, damageRoll1)
		}
		logMsg := attackLog + dmgLog

		g.addCombatLog(logMsg)

		ftColor := colorYellow
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: ftColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5})
	}
	return true
}

func executeMagicArmor(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Magic Armor!")
	g.Player.MagicArmorDuration = 3
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text: "+2 AC!", X: float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2), Y: float64(g.MapOffsetY+g.Player.Y*tileSize) - 15, Life: 60, MaxLife: 60, Color: visualEffectColors["Arcane"], VelocityY: -0.5,
	})
	return true
}

func executeElementalStrike(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	actionDef := ActionTable["elemental_strike"]
	if targetEnemy == nil {
		g.addCombatLog("Invalid target.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
	if dist > actionDef.Range {
		g.addCombatLog("Target out of range.")
		return false
	}

	currentElement := g.Player.NextElementType
	g.addCombatLog(fmt.Sprintf("Casting Elemental Strike (%s) at %s!", currentElement, targetEnemy.Name))

	casterAbilityMod := getModifier(g.Player.Intelligence)
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, casterAbilityMod, g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)
	elementColor := visualEffectColors["Default"]
	if currentElement == "Fire" {
		elementColor = visualEffectColors["Fire"]
	}
	if currentElement == "Cold" {
		elementColor = visualEffectColors["Cold"]
	}
	if currentElement == "Lightning" {
		elementColor = colorYellow
	}
	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		d1, d2, d3 := rand.Intn(8)+1, rand.Intn(8)+1, rand.Intn(8)+1
		critD1, critD2, critD3 := 0, 0, 0
		if isCrit {
			critD1, critD2, critD3 = rand.Intn(8)+1, rand.Intn(8)+1, rand.Intn(8)+1
		}
		damage := d1 + d2 + d3 + critD1 + critD2 + critD3
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (6d8[%d,%d,%d,%d,%d,%d]) %s dmg.", actualDamage, d1, d2, d3, critD1, critD2, critD3, currentElement)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (3d8[%d,%d,%d]) %s dmg.", actualDamage, d1, d2, d3, currentElement)
		}
		g.addCombatLog(attackLog + dmgLog)

		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			elementColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: elementColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5})
	}

	if currentElement == "Fire" {
		g.Player.NextElementType = "Cold"
	}
	if currentElement == "Cold" {
		g.Player.NextElementType = "Lightning"
	}
	if currentElement == "Lightning" {
		g.Player.NextElementType = "Fire"
	}

	return true
}

func executeFeatherFall(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Feather Fall!")
	g.Player.FeatherFallDuration = 2
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text: "Evasive!", X: float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2), Y: float64(g.MapOffsetY+g.Player.Y*tileSize) - 15, Life: 60, MaxLife: 60, Color: colorWhite, VelocityY: -0.5,
	})
	return true
}

func executeExpeditiousRetreat(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Expeditious Retreat!")
	g.Player.ExpeditiousRetreatDuration = 2
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text: "Speedy!", X: float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2), Y: float64(g.MapOffsetY+g.Player.Y*tileSize) - 15, Life: 60, MaxLife: 60, Color: colorYellow, VelocityY: -0.5,
	})
	return true
}
