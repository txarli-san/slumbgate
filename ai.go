package main

import (
	"fmt"
	"image"
	"math"
)

func (g *Game) stepEnemyTurn() {
	if g.currentEnemyTurn.Index < 0 || g.currentEnemyTurn.Index >= len(g.Enemies) {
		g.currentEnemyTurn.Index = -1
		return
	}

	enemy := g.Enemies[g.currentEnemyTurn.Index]
	if enemy == nil || enemy.IsDying || enemy.HP <= 0 {
		if g.currentEnemyTurn.Index >= 0 && g.currentEnemyTurn.Index < len(g.enemiesActedThisTurn) {
			g.enemiesActedThisTurn[g.currentEnemyTurn.Index] = true
		}
		g.currentEnemyTurn.Phase = PhaseEnemyDone
		g.currentEnemyTurn.Index = -1
		return
	}

	if g.currentEnemyTurn.Phase == PhaseEnemyPausedForReaction {
		if !g.reactionPending {
			g.addCombatLog(enemy.Name + " turn continues after reaction.")

			g.currentEnemyTurn.Phase = PhaseEnemyDone
		} else {
			return
		}
	}

	if g.Player == nil || g.Player.IsDying || g.Player.HP <= 0 {
		if g.currentEnemyTurn.Index >= 0 && g.currentEnemyTurn.Index < len(g.enemiesActedThisTurn) {
			g.enemiesActedThisTurn[g.currentEnemyTurn.Index] = true
		}
		g.currentEnemyTurn.Phase = PhaseEnemyDone
		g.currentEnemyTurn.Index = -1
		return
	}

	switch g.currentEnemyTurn.Phase {
	case PhaseEnemyStartTurn:
		TickConditions(g, &enemy.Entity)
		if HasCondition(&enemy.Entity, ConditionStunned) {
			g.addCombatLog(fmt.Sprintf("%s is Stunned!", enemy.Name))
			if g.currentEnemyTurn.Index >= 0 && g.currentEnemyTurn.Index < len(g.enemiesActedThisTurn) {
				g.enemiesActedThisTurn[g.currentEnemyTurn.Index] = true
			}
			g.currentEnemyTurn.Phase = PhaseEnemyDone
			g.currentEnemyTurn.Index = -1
			return
		}
		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		g.currentEnemyTurn.Phase = PhaseEnemyDecideAction

	case PhaseEnemyDecideAction:

		distToPlayer := distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
		isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

		canAttackNow := false
		if enemy.AttackType == "ranged" {
			canAttackNow = !isAdj && distToPlayer <= enemy.MaxRange && enemy.ActionAvailable
		} else {
			canAttackNow = isAdj && enemy.ActionAvailable
		}

		if canAttackNow {
			g.currentEnemyTurn.Phase = PhaseEnemyAction
		} else {
			g.currentEnemyTurn.Phase = PhaseEnemyMove
		}

	case PhaseEnemyMove:
		movedThisStep := false
		targetReached := false

		distToPlayer := distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
		isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

		if enemy.MovementPoints > 0 {
			var nextX, nextY int
			var foundMove bool

			if enemy.AttackType == "ranged" {
				if isAdj {
					nextX, nextY, foundMove = g.findRetreatStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, g.currentEnemyTurn.Index)
				} else if distToPlayer > enemy.MaxRange {
					nextX, nextY, foundMove = g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, g.currentEnemyTurn.Index)
				} else {

					foundMove = false
					targetReached = true
				}
			} else {
				if !isAdj {
					nextX, nextY, foundMove = g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, g.currentEnemyTurn.Index)
				} else {

					foundMove = false
					targetReached = true
				}
			}

			if foundMove {
				enemy.X, enemy.Y = nextX, nextY
				enemy.MovementPoints--
				movedThisStep = true
			}
		}

		if !movedThisStep || enemy.MovementPoints <= 0 || targetReached {

			distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
			isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
			canAttackNow := false
			if enemy.AttackType == "ranged" {
				canAttackNow = !isAdj && distToPlayer <= enemy.MaxRange
			} else {
				canAttackNow = isAdj
			}

			if enemy.ActionAvailable && canAttackNow {
				g.currentEnemyTurn.Phase = PhaseEnemyAction
			} else {

				if g.currentEnemyTurn.Index >= 0 && g.currentEnemyTurn.Index < len(g.enemiesActedThisTurn) {
					g.enemiesActedThisTurn[g.currentEnemyTurn.Index] = true
				}
				g.currentEnemyTurn.Phase = PhaseEnemyDone
				g.currentEnemyTurn.Index = -1
			}
		} else {
		}

	case PhaseEnemyAction:
		if !enemy.ActionAvailable {
			g.currentEnemyTurn.Phase = PhaseEnemyDone
			return
		}

		distToPlayer := distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
		isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
		killedPlayer := false
		attackType := "melee"
		canAttack := false

		if enemy.AttackType == "ranged" {
			if !isAdj && distToPlayer <= enemy.MaxRange {
				canAttack = true
				attackType = "ranged"
			}
		} else {
			if isAdj {
				canAttack = true
				attackType = "melee"
			}
		}

		if canAttack {
			killedPlayer, _ = g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, attackType)
			enemy.ActionAvailable = false

			if g.reactionPending {
				g.currentEnemyTurn.Phase = PhaseEnemyPausedForReaction

				return
			}
			if killedPlayer {

				if g.currentEnemyTurn.Index >= 0 && g.currentEnemyTurn.Index < len(g.enemiesActedThisTurn) {
					g.enemiesActedThisTurn[g.currentEnemyTurn.Index] = true
				}
				g.currentEnemyTurn.Phase = PhaseEnemyDone
				g.currentEnemyTurn.Index = -1
				return
			}
		}

		if g.currentEnemyTurn.Index >= 0 && g.currentEnemyTurn.Index < len(g.enemiesActedThisTurn) {
			g.enemiesActedThisTurn[g.currentEnemyTurn.Index] = true
		}
		g.currentEnemyTurn.Phase = PhaseEnemyDone
		g.currentEnemyTurn.Index = -1

	case PhaseEnemyDone:

		if g.currentEnemyTurn.Index >= 0 && g.currentEnemyTurn.Index < len(g.enemiesActedThisTurn) {
			g.enemiesActedThisTurn[g.currentEnemyTurn.Index] = true
		}
		g.currentEnemyTurn.Index = -1

	}
}

func (g *Game) findPathStep(startX, startY, targetX, targetY, entityWidth, entityHeight, movingEnemyIndex int) (int, int, bool) {
	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	bestMoveX, bestMoveY := startX, startY

	possibleMoves := []image.Point{}
	for _, offset := range moveOffsets {
		nextX, nextY := startX+offset.X, startY+offset.Y
		if !g.isTileFullyBlocked(nextX, nextY, entityWidth, entityHeight, movingEnemyIndex) {
			possibleMoves = append(possibleMoves, image.Point{X: nextX, Y: nextY})
		}
	}

	if len(possibleMoves) == 0 {
		return startX, startY, false
	}

	minDist := math.MaxInt32
	for _, move := range possibleMoves {
		dist := distance(move.X, move.Y, targetX, targetY)
		if dist < minDist {
			minDist = dist
		}
	}

	currentDist := distance(startX, startY, targetX, targetY)
	if currentDist <= 1 && minDist >= currentDist {
		return startX, startY, false
	}

	bestMoves := []image.Point{}
	for _, move := range possibleMoves {
		if distance(move.X, move.Y, targetX, targetY) == minDist {
			bestMoves = append(bestMoves, move)
		}
	}

	if len(bestMoves) > 0 {
		dx := targetX - startX
		dy := targetY - startY
		preferredMoveFound := false
		if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
			for _, move := range bestMoves {
				if move.X != startX {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}
		if !preferredMoveFound && math.Abs(float64(dy)) > math.Abs(float64(dx)) {
			for _, move := range bestMoves {
				if move.Y != startY {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}

		if !preferredMoveFound {
			canMoveY := false
			for _, move := range bestMoves {
				if move.Y != startY {
					canMoveY = true
					break
				}
			}
			if canMoveY && math.Abs(float64(dy)) >= math.Abs(float64(dx)) {
				for _, move := range bestMoves {
					if move.Y != startY {
						bestMoveX, bestMoveY = move.X, move.Y
						preferredMoveFound = true
						break
					}
				}
			}
		}

		if !preferredMoveFound && len(bestMoves) > 0 {
			bestMoveX, bestMoveY = bestMoves[0].X, bestMoves[0].Y
		}

		if bestMoveX != startX || bestMoveY != startY {
			return bestMoveX, bestMoveY, true
		}
	}

	return startX, startY, false
}

func (g *Game) findRetreatStep(startX, startY, targetX, targetY, entityWidth, entityHeight, movingEnemyIndex int) (int, int, bool) {
	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	bestMoveX, bestMoveY := startX, startY

	possibleMoves := []image.Point{}
	for _, offset := range moveOffsets {
		nextX, nextY := startX+offset.X, startY+offset.Y
		if !g.isTileFullyBlocked(nextX, nextY, entityWidth, entityHeight, movingEnemyIndex) {
			possibleMoves = append(possibleMoves, image.Point{X: nextX, Y: nextY})
		}
	}

	if len(possibleMoves) == 0 {
		return startX, startY, false
	}

	maxDist := -1
	for _, move := range possibleMoves {
		dist := distance(move.X, move.Y, targetX, targetY)
		if dist > maxDist {
			maxDist = dist
		}
	}

	if maxDist == 0 {
		return startX, startY, false
	}

	bestMoves := []image.Point{}
	for _, move := range possibleMoves {
		if distance(move.X, move.Y, targetX, targetY) == maxDist {
			bestMoves = append(bestMoves, move)
		}
	}

	if len(bestMoves) > 0 {
		dx := targetX - startX
		dy := targetY - startY
		preferredMoveFound := false

		if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
			for _, move := range bestMoves {
				if (dx > 0 && move.X < startX) || (dx < 0 && move.X > startX) || (dx == 0 && move.X != startX) {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}
		if !preferredMoveFound && math.Abs(float64(dy)) > math.Abs(float64(dx)) {
			for _, move := range bestMoves {
				if (dy > 0 && move.Y < startY) || (dy < 0 && move.Y > startY) || (dy == 0 && move.Y != startY) {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}

		if !preferredMoveFound {
			if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
				for _, move := range bestMoves {
					if (dy > 0 && move.Y < startY) || (dy < 0 && move.Y > startY) || (dy == 0 && move.Y != startY) {
						bestMoveX, bestMoveY = move.X, move.Y
						preferredMoveFound = true
						break
					}
				}
			} else {
				for _, move := range bestMoves {
					if (dx > 0 && move.X < startX) || (dx < 0 && move.X > startX) || (dx == 0 && move.X != startX) {
						bestMoveX, bestMoveY = move.X, move.Y
						preferredMoveFound = true
						break
					}
				}
			}
		}

		if !preferredMoveFound {
			bestMoveX, bestMoveY = bestMoves[0].X, bestMoves[0].Y
		}

		if bestMoveX != startX || bestMoveY != startY {
			return bestMoveX, bestMoveY, true
		}
	}

	return startX, startY, false
}
