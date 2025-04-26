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

type pathResult struct {
	finalX, finalY, finalDist int
	steps                     int
}

func simulateMultiStepPath(g *Game, startX, startY, targetX, targetY, entityW, entityH, movingIdx, maxSteps int) pathResult {
	currX, currY := startX, startY
	stepsUsed := 0

	for stepsUsed < maxSteps {
		nextX, nextY, found := g.findPathStep(currX, currY, targetX, targetY, entityW, entityH, movingIdx)
		if !found || (nextX == currX && nextY == currY) {
			break
		}

		currX, currY = nextX, nextY
		stepsUsed++

		if distance(currX, currY, targetX, targetY) <= 1 {
			break
		}
	}

	return pathResult{
		finalX:    currX,
		finalY:    currY,
		finalDist: distance(currX, currY, targetX, targetY),
		steps:     stepsUsed,
	}
}

func TestIsTileFullyBlocked(t *testing.T) {
	blockers := []image.Point{{X: 3, Y: 3}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)

	cases := []struct {
		x, y, w, h int
		expected   bool
		desc       string
	}{
		{2, 2, 1, 1, false, "Empty space"},
		{3, 3, 1, 1, true, "Directly on blocker"},
		{2, 2, 2, 2, true, "Overlaps blocker"},
		{-1, 3, 1, 1, true, "Off map left"},
		{mapWidth, 3, 1, 1, true, "Off map right"},
		{3, -1, 1, 1, true, "Off map top"},
		{3, mapHeight, 1, 1, true, "Off map bottom"},
		{mapWidth - 1, mapHeight - 1, 2, 2, true, "Partially off map bottom-right"},
		{5, 5, 1, 1, true, "On player position"},
	}

	for _, tc := range cases {
		result := g.isTileFullyBlocked(tc.x, tc.y, tc.w, tc.h, -1)
		if result != tc.expected {
			t.Errorf("%s: expected=%v, got=%v for pos=(%d,%d) size=%dx%d",
				tc.desc, tc.expected, result, tc.x, tc.y, tc.w, tc.h)
		}
	}
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

func TestFindPathStep_ComplexObstacleConfiguration(t *testing.T) {
	blockers := []image.Point{{X: 3, Y: 2}, {X: 3, Y: 3}, {X: 3, Y: 4}, {X: 4, Y: 4}, {X: 5, Y: 4}}
	g := setupTestGamePathfinding(image.Point{X: 7, Y: 7}, blockers)
	startX, startY := 1, 3
	targetX, targetY := 6, 3
	initialDist := distance(startX, startY, targetX, targetY)

	result := simulateMultiStepPath(g, startX, startY, targetX, targetY, 1, 1, -1, 15)

	if result.finalDist >= initialDist && result.finalDist > 1 {
		t.Errorf("Failed to navigate around obstacle wall: start dist=%d, final dist=%d after %d steps, ended at (%d,%d)",
			initialDist, result.finalDist, result.steps, result.finalX, result.finalY)
	}
	if result.finalDist > 1 && result.steps == 0 {
		t.Errorf("Failed to make any progress around obstacle wall: start dist=%d, final dist=%d", initialDist, result.finalDist)
	}
}

func TestFindPathStep_DiagonalCorner(t *testing.T) {
	blockers := []image.Point{{X: 2, Y: 1}, {X: 1, Y: 2}}
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, blockers)
	startX, startY := 1, 1
	targetX, targetY := 3, 3
	entityW, entityH := 1, 1
	movingIdx := -1

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if !found {
		t.Errorf("Entity should find a way out of diagonal corner, but got stuck")
	}
	if found && (nextX == startX && nextY == startY) {
		t.Errorf("Entity found a path but didn't move out of diagonal corner, stayed at (%d,%d)", nextX, nextY)
	}
}

func TestFindPathStep_LargeEntityMovement(t *testing.T) {
	blockers := []image.Point{{X: 3, Y: 3}}
	g := setupTestGamePathfinding(image.Point{X: 7, Y: 7}, blockers)
	startX, startY := 1, 1
	targetX, targetY := 4, 4
	entityW, entityH := 2, 2
	movingIdx := -1

	nextX, nextY, found := g.findPathStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if found {
		if nextX < 0 || nextY < 0 || nextX+entityW > mapWidth || nextY+entityH > mapHeight {
			t.Errorf("Large entity moved partially out of bounds: (%d,%d)", nextX, nextY)
		}

		for w := 0; w < entityW; w++ {
			for h := 0; h < entityH; h++ {
				if g.isTileBlocked(nextX+w, nextY+h, movingIdx) {
					t.Errorf("Large entity moved into a blocked tile at (%d,%d)", nextX+w, nextY+h)
				}
			}
		}
	} else {
		t.Logf("Large entity movement test: No path found from (%d,%d) towards (%d,%d), which might be expected depending on blocking.", startX, startY, targetX, targetY)
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

func TestFindRetreatStep_NotAdjacent(t *testing.T) {
	g := setupTestGamePathfinding(image.Point{X: 5, Y: 5}, nil)
	startX, startY := 3, 3
	targetX, targetY := 5, 5
	entityW, entityH := 1, 1
	movingIdx := -1

	nextX, nextY, found := g.findRetreatStep(startX, startY, targetX, targetY, entityW, entityH, movingIdx)

	if !found {
		t.Errorf("findRetreatStep NotAdjacent failed: Expected a move to be found (to maximize distance)")
	}

	initialDist := distance(startX, startY, targetX, targetY)
	newDist := distance(nextX, nextY, targetX, targetY)

	if found && newDist < initialDist {
		t.Errorf("findRetreatStep NotAdjacent failed: Moved closer (%d,%d -> dist %d) instead of further/same (start dist %d)",
			nextX, nextY, newDist, initialDist)
	}
}
