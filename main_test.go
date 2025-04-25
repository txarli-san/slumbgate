package main

import (
	"fmt"
	"image"
	"testing"
)

func setupTestGamePathfinding(playerPos image.Point, blockers []image.Point) *Game {
	g := &Game{
		Player: &Player{
			Entity: Entity{X: playerPos.X, Y: playerPos.Y, Width: 1, Height: 1, HP: 10},
		},
		Enemies: make([]*Enemy, 0),
	}
	for i, pos := range blockers {
		g.Enemies = append(g.Enemies, &Enemy{
			Entity: Entity{Name: fmt.Sprintf("Blocker%d", i), X: pos.X, Y: pos.Y, Width: 1, Height: 1, HP: 1},
		})
	}
	return g
}

func TestFindPathStep_DirectPath(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 1, 1
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 2, 1
	expectedFound := true

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findPathStep DirectPath failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindPathStep_BlockedDirect_Sideways(t *testing.T) {
	blockers := []image.Point{{X: 4, Y: 5}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 3, 5
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 2, 5
	expectedFound := true

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findPathStep BlockedDirect failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindPathStep_TargetAdjacent(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 4, 5
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 4, 5
	expectedFound := false

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound {
		t.Errorf("findPathStep TargetAdjacent failed: expected found %t, got %t. Position: (%d,%d)",
			expectedFound, found, nextX, nextY)
	}
	if !found && (nextX != expectedX || nextY != expectedY) {
		t.Errorf("findPathStep TargetAdjacent failed: expected position (%d,%d) when found=false, got (%d,%d)",
			expectedX, expectedY, nextX, nextY)
	}
}

func TestFindPathStep_CompletelyBlocked(t *testing.T) {
	blockers := []image.Point{
		{X: 0, Y: 1}, {X: 1, Y: 0}, {X: 2, Y: 1}, {X: 1, Y: 2},
	}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 1, 1
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 1, 1
	expectedFound := false

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findPathStep CompletelyBlocked failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindRetreatStep_DirectRetreat(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 5, 6
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 5, 7
	expectedFound := true

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findRetreatStep DirectRetreat failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}

func TestFindRetreatStep_BlockedDirect_Sideways(t *testing.T) {
	blockers := []image.Point{{X: 5, Y: 7}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 5, 6
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX1, expectedY1 := 4, 6
	expectedX2, expectedY2 := 6, 6
	expectedFound := true

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound {
		t.Errorf("findRetreatStep BlockedDirect failed: expected found %t, got %t", expectedFound, found)
	}
	if !((nextX == expectedX1 && nextY == expectedY1) || (nextX == expectedX2 && nextY == expectedY2)) {
		t.Errorf("findRetreatStep BlockedDirect failed: expected (%d,%d) or (%d,%d), got (%d,%d)",
			expectedX1, expectedY1, expectedX2, expectedY2, nextX, nextY)
	}
}

func TestFindRetreatStep_Cornered(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 1, Y: 1}, nil)
	startX, startY := 0, 0
	targetX, targetY := 1, 1
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX1, expectedY1 := 1, 0
	expectedX2, expectedY2 := 0, 1
	expectedFound := true

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound {
		t.Errorf("findRetreatStep Cornered failed: expected found %t, got %t", expectedFound, found)
	}
	if !((nextX == expectedX1 && nextY == expectedY1) || (nextX == expectedX2 && nextY == expectedY2)) {
		t.Errorf("findRetreatStep Cornered failed: expected (%d,%d) or (%d,%d), got (%d,%d)",
			expectedX1, expectedY1, expectedX2, expectedY2, nextX, nextY)
	}
}

func TestFindRetreatStep_NoValidMoves(t *testing.T) {
	blockers := []image.Point{{X: 1, Y: 0}, {X: 0, Y: 1}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 0, 0
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	expectedX, expectedY := 0, 0
	expectedFound := false

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found != expectedFound || nextX != expectedX || nextY != expectedY {
		t.Errorf("findRetreatStep NoValidMoves failed: expected (%d,%d, %t), got (%d,%d, %t)",
			expectedX, expectedY, expectedFound, nextX, nextY, found)
	}
}
