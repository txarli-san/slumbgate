package main

import "fmt"

func (g *Game) startPlayerTurn() {
	if g.Player == nil || g.Player.IsDying {
		return
	}

	g.CurrentTurn = PlayerTurn
	g.Player.MovementPoints = g.Player.MaxMovementPoints

	if cond := GetCondition(&g.Player.Entity, ConditionExpeditiousRetreat); cond != nil {
		if bonus, ok := cond.Data["MoveBonus"].(int); ok {
			g.Player.MovementPoints += bonus
		}
	}

	if HasCondition(&g.Player.Entity, ConditionSlowed) {
		g.Player.MovementPoints = max(1, g.Player.MovementPoints/2)
	}

	TickConditions(g, &g.Player.Entity)

	g.Player.ActionTaken = false
	g.Player.BonusActionTaken = false
	g.Player.IsDisengaging = false
	g.Player.UsedReaction = false
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
