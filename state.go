package main

import "fmt"

func (g *Game) startPlayerTurn() {
	currentACBonus := 0
	currentMoveBonus := 0

	if g.Player.IsSlowed {
		g.Player.SlowDuration--
		if g.Player.SlowDuration <= 0 {
			g.Player.IsSlowed = false
			g.addCombatLog("Slow effect wears off.")
		}
	}
	if g.Player.MagicArmorDuration > 0 {
		g.Player.MagicArmorDuration--
		if g.Player.MagicArmorDuration <= 0 {
			g.addCombatLog("Magic Armor fades.")
		} else {
			currentACBonus += 2
		}
	}
	if g.Player.ExpeditiousRetreatDuration > 0 {
		g.Player.ExpeditiousRetreatDuration--
		if g.Player.ExpeditiousRetreatDuration <= 0 {
			g.addCombatLog("Expeditious Retreat ends.")
		} else {
			currentMoveBonus += 3
		}
	}
	if g.Player.FeatherFallDuration > 0 {
		g.Player.FeatherFallDuration--
		if g.Player.FeatherFallDuration <= 0 {
			g.addCombatLog("Feather Fall ends.")
		}
	}

	g.CurrentTurn = PlayerTurn
	g.Player.MovementPoints = g.Player.MaxMovementPoints + currentMoveBonus
	if g.Player.IsSlowed {
		g.Player.MovementPoints = max(1, g.Player.MovementPoints/2)
	}

	g.Player.ActionTaken = false
	g.Player.BonusActionTaken = false
	g.Player.IsDisengaging = false
	g.Player.UsedReaction = false
	g.Player.ACBonusUntilNextTurn = currentACBonus
	g.InputMode = InputModeMap
	g.primedActionID = ""
}

func (g *Game) endPlayerTurn() {
	g.cleanupDeadEnemies()

	if len(g.Enemies) == 0 {
		g.handleWaveCompletion()
	} else {
		g.startEnemyTurn()
	}

	g.primedActionID = ""
	if g.CurrentGameState == StatePlaying && g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
		g.InputMode = InputModeMap
	}
}

func (g *Game) startEnemyTurn() {
	g.CurrentTurn = EnemyTurn
	g.currentEnemyTurn = EnemyTurnContext{Index: -1, Phase: PhaseEnemyDone}
	g.enemiesActedThisTurn = make([]bool, len(g.Enemies))
}

func (g *Game) endEnemyTurn() {
	g.cleanupDeadEnemies()

	if g.Player.IsDying && g.Player.CurrentAlpha <= 0 {
		if g.CurrentGameState != StateGameOverScreen {
			g.addCombatLog("Player has faded away! Game Over.")
		}
		return
	}

	if g.CurrentTurn == GameOver || g.CurrentGameState == StateGameOverScreen {
		return
	}
	if len(g.Enemies) == 0 {
		if g.Player.IsDying {
			if g.CurrentGameState != StateGameOverScreen {
			}
			return
		}
		g.handleWaveCompletion()
	} else {
		if !g.Player.IsDying {
			g.startPlayerTurn()
		} else {
			if g.CurrentGameState != StateGameOverScreen {
			}
		}
	}
}

func (g *Game) setGameOver(victory bool) {
	g.CurrentTurn = GameOver
	g.CurrentGameState = StateGameOverScreen
	g.isVictory = victory
}

func (g *Game) handleWaveCompletion() {
	currentWaveDef := g.WaveDefinitions[g.CurrentWaveIndex]
	isFinalWave := g.CurrentWaveIndex+1 >= len(g.WaveDefinitions)

	if currentWaveDef.IsBossWave && g.Player.Level < maxLevel {
		g.addCombatLog("Final Boss Defeated!")
		g.levelUpPlayer()
		if g.Player.Class == "Mage" && g.Player.Level == 2 && len(g.Player.KnownSpells) == 0 {
			g.addCombatLog("Choose a new Level 1 spell!")
			g.pendingLevelUpSpellChoice = 0
		}
		g.InputMode = InputModeLevelUp
	} else if !isFinalWave {
		g.addCombatLog(fmt.Sprintf("Wave Cleared! Spend 1 Hit Die (of %d) to heal? [Y/N]", g.Player.HitDice))
		g.InputMode = InputModeRestPrompt
	} else {
		g.addCombatLog("All challenges overcome! VICTORY!")
		g.longRest()
		g.setGameOver(true)
	}
}

func (g *Game) startNextWave() {
	g.SpawnNextWave()
	if len(g.Enemies) > 0 {
		g.startPlayerTurn()
	} else {
		g.addCombatLog("Error spawning next wave or wave empty.")
		g.setGameOver(false)
	}
	g.InputMode = InputModeMap
}
