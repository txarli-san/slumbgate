package main

import (
	"container/heap"
	"fmt"
	"image"
	"math"
)

type aStarNode struct {
	pos    image.Point
	parent *aStarNode
	g, h   int
	f      int
	index  int
}

type priorityQueue []*aStarNode

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].f < pq[j].f
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	node := x.(*aStarNode)
	node.index = n
	*pq = append(*pq, node)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	node := old[n-1]
	old[n-1] = nil
	node.index = -1
	*pq = old[0 : n-1]
	return node
}

func (g *Game) FindPath(startX, startY, endX, endY int, entityWidth, entityHeight int) ([]image.Point, bool) {
	startPoint := image.Point{X: startX, Y: startY}
	endPoint := image.Point{X: endX, Y: endY}

	openSet := make(priorityQueue, 0)
	heap.Init(&openSet)
	closedSet := make(map[image.Point]bool)
	nodeMap := make(map[image.Point]*aStarNode)

	startNode := &aStarNode{pos: startPoint, g: 0, h: distance(startX, startY, endX, endY)}
	startNode.f = startNode.g + startNode.h
	nodeMap[startPoint] = startNode
	heap.Push(&openSet, startNode)

	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}

	for openSet.Len() > 0 {
		currentNode := heap.Pop(&openSet).(*aStarNode)

		if currentNode.pos == endPoint {
			path := []image.Point{}
			temp := currentNode
			for temp != nil {
				path = append(path, temp.pos)
				temp = temp.parent
			}

			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			if len(path) > 0 {
				return path[1:], true
			}
			return path, true
		}

		closedSet[currentNode.pos] = true

		for _, offset := range moveOffsets {
			neighborPos := image.Point{X: currentNode.pos.X + offset.X, Y: currentNode.pos.Y + offset.Y}

			if neighborPos.X < 0 || neighborPos.X+entityWidth > mapWidth || neighborPos.Y < 0 || neighborPos.Y+entityHeight > mapHeight {
				continue
			}

			if g.isTileFullyBlocked(neighborPos.X, neighborPos.Y, entityWidth, entityHeight, -1) {

				isTargetAdjacentBlocked := false
				for w := 0; w < entityWidth; w++ {
					for h := 0; h < entityHeight; h++ {
						tileX, tileY := neighborPos.X+w, neighborPos.Y+h
						if tileX == endX && tileY == endY {
							isTargetAdjacentBlocked = true
							break
						}
					}
					if isTargetAdjacentBlocked {
						break
					}
				}

				if !isTargetAdjacentBlocked {
					continue
				}

			}

			if closedSet[neighborPos] {
				continue
			}

			gCost := currentNode.g + 1
			hCost := distance(neighborPos.X, neighborPos.Y, endX, endY)
			fCost := gCost + hCost

			neighborNode, exists := nodeMap[neighborPos]
			if !exists {
				neighborNode = &aStarNode{pos: neighborPos, parent: currentNode, g: gCost, h: hCost, f: fCost}
				nodeMap[neighborPos] = neighborNode
				heap.Push(&openSet, neighborNode)
			} else if gCost < neighborNode.g {
				neighborNode.parent = currentNode
				neighborNode.g = gCost
				neighborNode.f = fCost
				heap.Fix(&openSet, neighborNode.index)
			}
		}
	}

	return nil, false
}

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
