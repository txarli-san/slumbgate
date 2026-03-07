package main

import (
	"container/heap"
	"math"
)

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
	TaskIdle    TaskType = iota
	TaskMoveTo
	TaskExplore
)

type Task struct {
	Type         TaskType
	TargetX      int
	TargetZ      int
	Path         [][2]int
	PathIdx      int
	ExploreAngle float64 // current angle for perimeter patrol
}

type Entity struct {
	Name         string
	X, Z         int
	PrevX, PrevZ int
	StepProgress float32
	Moving       bool
	FacingAngle  float32
	Tier         AutoTier
	RevealDist   int
	Task         *Task
	Scouted map[[2]int]bool // coarse 8x8 cells this entity has covered
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

// AnyEntityBusy returns true if any entity has an active task
func (g *GameState) AnyEntityBusy() bool {
	for _, ent := range g.Entities {
		if ent.Task != nil {
			return true
		}
	}
	return false
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

// WorldStep advances the global clock by one tick and lets all entities act.
// Called once per player action (move, break wall, wait). Time only moves when someone acts.
func (g *GameState) WorldStep(w *World) *Alert {
	g.TimeTicks++
	return g.TickEntities(w)
}

// TickEntities advances all entities one step. Returns an alert if triggered.
func (g *GameState) TickEntities(w *World) *Alert {
	for i, ent := range g.Entities {
		if ent.Task == nil {
			continue
		}

		// Explore: patrol dungeon perimeter, advance angle each waypoint
		if ent.Task.Type == TaskExplore && (ent.Task.Path == nil || ent.Task.PathIdx >= len(ent.Task.Path)) {
			// Init angle from entity position relative to origin
			if ent.Task.ExploreAngle == 0 {
				ent.Task.ExploreAngle = math.Atan2(float64(ent.Z), float64(ent.X))
			}
			// Advance ~20 degrees along the perimeter
			ent.Task.ExploreAngle += 0.35
			outerR := w.OuterEdge(ent.Task.ExploreAngle)
			// Stay a few tiles outside the wall — perception does the rest
			patrolR := outerR + float64(ent.RevealDist)/2
			tx := int(math.Cos(ent.Task.ExploreAngle) * patrolR)
			tz := int(math.Sin(ent.Task.ExploreAngle) * patrolR)
			// Find nearest walkable tile to the target
			fx, fz, found := nearestWalkable(w, tx, tz)
			if found {
				path := FindPath(w, ent.X, ent.Z, fx, fz)
				if path != nil {
					ent.Task.Path = path
					ent.Task.PathIdx = 0
				}
			}
		}

		if ent.Task.Type != TaskMoveTo && ent.Task.Type != TaskExplore {
			continue
		}
		if ent.Task.Path == nil || ent.Task.PathIdx >= len(ent.Task.Path) {
			continue
		}

		next := ent.Task.Path[ent.Task.PathIdx]
		nx, nz := next[0], next[1]

		if !w.IsWalkable(nx, nz) {
			ent.Task.Path = nil // force re-plan on next tick
			continue
		}

		dx, dz := nx-ent.X, nz-ent.Z
		ent.FacingAngle = FacingAngleFromDir(dx, dz)
		ent.PrevX, ent.PrevZ = ent.X, ent.Z
		ent.X, ent.Z = nx, nz
		ent.StepProgress = 0
		ent.Moving = true
		ent.Task.PathIdx++

		cx, cz := TileToChunk(ent.X, ent.Z)
		w.EnsureChunksAround(cx, cz)
		w.RevealAroundDist(ent.X, ent.Z, ent.RevealDist)

		// Track scouted area
		if ent.Scouted != nil {
			ent.Scouted[[2]int{ent.X >> 3, ent.Z >> 3}] = true
		}

		// Pickaxe pickup
		if !g.HasPickaxe && ent.X == w.PickaxeX && ent.Z == w.PickaxeZ {
			g.HasPickaxe = true
			g.SetMessage(ent.Name + " picked up the pickaxe!")
		}

		// Exploring: check for pickaxe within perception range every step — divert if spotted
		if ent.Task != nil && ent.Task.Type == TaskExplore && !g.HasPickaxe &&
			withinRange(ent.X, ent.Z, w.PickaxeX, w.PickaxeZ, ent.RevealDist) {
			path := FindPath(w, ent.X, ent.Z, w.PickaxeX, w.PickaxeZ)
			if path != nil {
				ent.Task.Path = path
				ent.Task.PathIdx = 0
				g.SetMessage(ent.Name + " spotted something interesting!")
			}
		}

		// Check for threat at new position
		if threat, ok := w.GetThreat(ent.X, ent.Z); ok {
			if int(ent.Tier) >= threat.Difficulty {
				w.RemoveThreat(ent.X, ent.Z)
			} else {
				ent.Task = nil
				return &Alert{
					EntityIdx: i,
					ThreatX:   ent.X,
					ThreatZ:   ent.Z,
					Message:   ent.Name + " encountered a threat they can't handle!",
				}
			}
		}

		// MoveTo: clear task when path is done. Explore: will re-plan next tick.
		if ent.Task != nil && ent.Task.Type == TaskMoveTo && ent.Task.PathIdx >= len(ent.Task.Path) {
			ent.Task = nil
		}
	}
	return nil
}

func withinRange(ax, az, bx, bz, r int) bool {
	dx, dz := ax-bx, az-bz
	return dx*dx+dz*dz <= r*r
}

// nearestWalkable spirals out from (tx,tz) to find the closest walkable tile.
func nearestWalkable(w *World, tx, tz int) (int, int, bool) {
	if w.IsWalkable(tx, tz) {
		return tx, tz, true
	}
	for r := 1; r <= 8; r++ {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if abs(dx) != r && abs(dz) != r {
					continue // only check the ring edge
				}
				if w.IsWalkable(tx+dx, tz+dz) {
					return tx + dx, tz + dz, true
				}
			}
		}
	}
	return 0, 0, false
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
