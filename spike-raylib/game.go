package main

import "container/heap"

type CameraMode int

const (
	CameraLocal    CameraMode = iota // 3D tactical view
	CameraStrategic                  // 2D top-down map
)

type GameState struct {
	PlayerX, PlayerZ int
	PrevX, PrevZ     int
	StepProgress     float32
	Moving           bool
	FacingAngle      float32
	FacingDX         int // grid direction player faces
	FacingDZ         int
	TimeTicks        int
	Path             [][2]int
	Camera           CameraMode
	HasPickaxe       bool
	Message          string
	MessageTimer     float32
}

func (g *GameState) SetMessage(msg string) {
	g.Message = msg
	g.MessageTimer = 3.0
}

const stepInterval = 0.08

func FacingAngleFromDir(dx, dz int) float32 {
	switch {
	case dx == 0 && dz == -1:
		return 180
	case dx == 0 && dz == 1:
		return 0
	case dx == -1 && dz == 0:
		return 90
	case dx == 1 && dz == 0:
		return -90
	case dx == -1 && dz == -1:
		return 135
	case dx == 1 && dz == -1:
		return -135
	case dx == -1 && dz == 1:
		return 45
	case dx == 1 && dz == 1:
		return -45
	}
	return 0
}

// A* pathfinding with cardinal movement
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

func FindPath(w *World, sx, sz, gx, gz int) [][2]int {
	if !w.IsWalkable(gx, gz) {
		return nil
	}

	type key struct{ x, z int }
	closed := map[key]bool{}
	open := &astarHeap{}
	heap.Init(open)

	start := &astarNode{x: sx, z: sz, g: 0, f: abs(gx-sx) + abs(gz-sz)}
	heap.Push(open, start)

	dirs := [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for open.Len() > 0 {
		cur := heap.Pop(open).(*astarNode)
		if cur.x == gx && cur.z == gz {
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
			if closed[key{nx, nz}] || !w.IsWalkable(nx, nz) {
				continue
			}
			ng := cur.g + 1
			nf := ng + abs(gx-nx) + abs(gz-nz)
			heap.Push(open, &astarNode{x: nx, z: nz, g: ng, f: nf, parent: cur})
		}
	}
	return nil
}
