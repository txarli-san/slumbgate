package main

import "container/heap"

// Game state for the exploration PoC
type GameState struct {
	PlayerX, PlayerZ int
	Path             [][2]int // queued path to walk
	StepTimer        float32  // time until next step
	HasPickaxe       bool
	TimeTicks        int // each step = 1 tick
	PickaxeX         int // ground item position
	PickaxeZ         int
	BreakableWallX   int // which cell has the breakable wall
	BreakableWallZ   int
	BreakableWallDir int // 0=N, 1=E, 2=S, 3=W
	WallBroken       bool
	Message          string
	MessageTimer     float32
}

const stepInterval = 0.15 // seconds between steps

func (g *GameState) SetMessage(msg string) {
	g.Message = msg
	g.MessageTimer = 3.0
}

// CanWalk checks if moving from (fx,fz) to (tx,tz) is allowed (no wall crossing)
func CanWalk(d *Dungeon, fx, fz, tx, tz int) bool {
	if tx < 0 || tx >= d.Width || tz < 0 || tz >= d.Height {
		return false
	}
	// Can't walk into unrevealed floor (unless adjacent to current pos and ground)
	target := d.Cells[tz][tx]
	if target.Type != CellGround && target.Type != CellFloor {
		return false
	}

	// Check wall between from and to
	from := d.Cells[fz][fx]
	dx, dz := tx-fx, tz-fz

	// Moving north (dz=-1): check from's north wall
	if dz == -1 && dx == 0 && from.WallN {
		return false
	}
	// Moving south
	if dz == 1 && dx == 0 && from.WallS {
		return false
	}
	// Moving west
	if dx == -1 && dz == 0 && from.WallW {
		return false
	}
	// Moving east
	if dx == 1 && dz == 0 && from.WallE {
		return false
	}

	// Also check target cell's wall from the opposite side
	if dz == -1 && dx == 0 && target.WallS {
		return false
	}
	if dz == 1 && dx == 0 && target.WallN {
		return false
	}
	if dx == -1 && dz == 0 && target.WallE {
		return false
	}
	if dx == 1 && dz == 0 && target.WallW {
		return false
	}

	return true
}

// A* pathfinding
type astarNode struct {
	x, z, g, f int
	parent     *astarNode
}

type astarHeap []*astarNode

func (h astarHeap) Len() int            { return len(h) }
func (h astarHeap) Less(i, j int) bool   { return h[i].f < h[j].f }
func (h astarHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i] }
func (h *astarHeap) Push(x interface{})  { *h = append(*h, x.(*astarNode)) }
func (h *astarHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func FindPath(d *Dungeon, sx, sz, gx, gz int) [][2]int {
	type key struct{ x, z int }
	closed := map[key]bool{}
	open := &astarHeap{}
	heap.Init(open)

	start := &astarNode{x: sx, z: sz, g: 0, f: abs(gx-sx) + abs(gz-sz)}
	heap.Push(open, start)

	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for open.Len() > 0 {
		cur := heap.Pop(open).(*astarNode)
		if cur.x == gx && cur.z == gz {
			// Reconstruct path (skip start)
			var path [][2]int
			for n := cur; n != nil && (n.x != sx || n.z != sz); n = n.parent {
				path = append([][2]int{{n.x, n.z}}, path...)
			}
			return path
		}

		k := key{cur.x, cur.z}
		if closed[k] {
			continue
		}
		closed[k] = true

		for _, dir := range dirs {
			nx, nz := cur.x+dir[0], cur.z+dir[1]
			if closed[(key{nx, nz})] {
				continue
			}
			if !CanWalk(d, cur.x, cur.z, nx, nz) {
				continue
			}
			ng := cur.g + 1
			nf := ng + abs(gx-nx) + abs(gz-nz)
			heap.Push(open, &astarNode{x: nx, z: nz, g: ng, f: nf, parent: cur})
		}
	}
	return nil // no path
}
