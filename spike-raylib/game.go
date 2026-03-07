package main

import "container/heap"

type CameraMode int

const (
	CameraLocal    CameraMode = iota // 3D tactical view
	CameraStrategic                  // 2D top-down map
)

type AutoTier int

const (
	TierRecruit    AutoTier = iota // Full manual — can't auto-resolve anything
	TierSoldier                    // Supervised — handles weak threats
	TierVeteran                    // Autonomous — handles most threats
	TierLieutenant                 // Command node — detects before arrival
)

type TaskType int

const (
	TaskIdle   TaskType = iota
	TaskMoveTo
)

type Task struct {
	Type    TaskType
	TargetX int
	TargetZ int
	Path    [][2]int
	PathIdx int
}

type Entity struct {
	Name         string
	X, Z         int
	PrevX, PrevZ int
	StepProgress float32
	Moving       bool
	FacingAngle  float32
	Tier         AutoTier
	Task         *Task
}

type Alert struct {
	EntityIdx int
	ThreatX   int
	ThreatZ   int
	Message   string
}

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

	Entities    []*Entity
	SelectedEnt int // -1 = none
	TickAccum   float32
	Alert       *Alert
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

const tickRate = 0.25 // seconds per simulation tick in Live mode

// TickEntities advances all entities one step. Returns an alert if triggered.
func (g *GameState) TickEntities(w *World) *Alert {
	for i, ent := range g.Entities {
		if ent.Task == nil || ent.Task.Type != TaskMoveTo {
			continue
		}
		if ent.Task.PathIdx >= len(ent.Task.Path) {
			ent.Task = nil
			continue
		}

		next := ent.Task.Path[ent.Task.PathIdx]
		nx, nz := next[0], next[1]

		// Check if path is still walkable
		if !w.IsWalkable(nx, nz) {
			ent.Task = nil
			continue
		}

		dx, dz := nx-ent.X, nz-ent.Z
		ent.FacingAngle = FacingAngleFromDir(dx, dz)
		ent.PrevX, ent.PrevZ = ent.X, ent.Z
		ent.X, ent.Z = nx, nz
		ent.StepProgress = 0
		ent.Moving = true
		ent.Task.PathIdx++

		w.RevealAround(ent.X, ent.Z)

		// Check for threat at new position
		if threat, ok := w.GetThreat(ent.X, ent.Z); ok {
			if int(ent.Tier) >= threat.Difficulty {
				// Auto-clear: tier is high enough
				w.RemoveThreat(ent.X, ent.Z)
			} else {
				// Can't handle it — ALERT
				ent.Task = nil
				return &Alert{
					EntityIdx: i,
					ThreatX:   ent.X,
					ThreatZ:   ent.Z,
					Message:   ent.Name + " encountered a threat they can't handle!",
				}
			}
		}

		if ent.Task != nil && ent.Task.PathIdx >= len(ent.Task.Path) {
			ent.Task = nil
		}
	}
	return nil
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
