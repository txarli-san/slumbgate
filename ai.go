package main

import (
	"image"
	"math"
)

func (g *Game) handleEnemyTurns() {
	if g.Player == nil || g.Player.IsDying || g.Player.HP <= 0 {
		return
	}

	for i, enemy := range g.Enemies {
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}

		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		actedThisTurn := false

		distToPlayer := distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
		isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

		if enemy.AttackType == "ranged" {
			if !isAdj && distToPlayer <= enemy.MaxRange && enemy.ActionAvailable {
				killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
				if g.reactionPending {
					return
				}
				enemy.ActionAvailable = false
				actedThisTurn = true
				if killedPlayer {
					return
				}
			}
		} else {
			if isAdj && enemy.ActionAvailable {
				killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
				if g.reactionPending {
					return
				}
				enemy.ActionAvailable = false
				actedThisTurn = true
				if killedPlayer {
					return
				}
			}
		}

		for enemy.MovementPoints > 0 {
			if g.Player.IsDying || g.Player.HP <= 0 {
				break
			}

			distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
			isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
			movedThisStep := false

			if enemy.AttackType == "ranged" {
				if isAdj {
					nextX, nextY, foundMove := g.findRetreatStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						movedThisStep = true
					} else {
						break
					}
				} else if distToPlayer > enemy.MaxRange {
					nextX, nextY, foundMove := g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						movedThisStep = true
					} else {
						break
					}
				} else {
					break
				}
			} else {
				if !isAdj {
					nextX, nextY, foundMove := g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						movedThisStep = true
					} else {
						break
					}
				} else {
					break
				}
			}
			if !movedThisStep {
				break
			}
		}

		if !actedThisTurn && enemy.ActionAvailable {
			if g.Player.IsDying || g.Player.HP <= 0 {
				continue
			}

			distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
			isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

			if enemy.AttackType == "ranged" {
				if !isAdj && distToPlayer <= enemy.MaxRange {
					killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
					if g.reactionPending {
						return
					}
					enemy.ActionAvailable = false
					if killedPlayer {
						return
					}
				}
			} else {
				if isAdj {
					killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
					if g.reactionPending {
						return
					}
					enemy.ActionAvailable = false
					if killedPlayer {
						return
					}
				}
			}
		}
		enemy.ActionAvailable = false

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
	if currentDist <= 1 && minDist > currentDist {
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
			if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
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
	currentDist := distance(startX, startY, targetX, targetY)
	bestMoveX, bestMoveY := startX, startY
	maxDist := currentDist

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

	bestMoves := []image.Point{}
	for _, move := range possibleMoves {
		dist := distance(move.X, move.Y, targetX, targetY)
		if dist > maxDist {
			maxDist = dist
		}
	}

	if maxDist == currentDist {

		maxDist = -1
		for _, move := range possibleMoves {
			dist := distance(move.X, move.Y, targetX, targetY)
			if dist > maxDist {
				maxDist = dist
			}
		}
	}

	for _, move := range possibleMoves {
		if distance(move.X, move.Y, targetX, targetY) == maxDist {
			bestMoves = append(bestMoves, move)
		}
	}

	if len(bestMoves) > 0 {

		bestMoveX, bestMoveY = bestMoves[0].X, bestMoves[0].Y
		if bestMoveX != startX || bestMoveY != startY {
			return bestMoveX, bestMoveY, true
		}
	}

	return startX, startY, false
}
