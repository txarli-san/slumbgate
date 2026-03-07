package main

import (
	"container/heap"
	"math"
	"math/rand"
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

type CombatStats struct {
	HP, MaxHP    int
	AC           int
	STR, DEX, CON, INT, WIS, CHA int
	Level        int
	ProfBonus    int
	MoveSpeed    int // tiles per combat turn
	Class        string // "Fighter", "Mage", etc.
}

func (s CombatStats) Mod(stat int) int { return (stat - 10) / 2 }

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
	Scouted      map[[2]int]bool
	Stats        *CombatStats // nil = non-combatant
}

type Alert struct {
	EntityIdx int
	ThreatX   int
	ThreatZ   int
	Message   string
}

// Combat mode types

type CombatTurnPhase int

const (
	PhaseMovement CombatTurnPhase = iota
	PhaseAction
)

type Combatant struct {
	IsEnemy    bool
	EntityIdx  int      // index into GameState.Entities (allies)
	ThreatKey  [2]int   // key into World.Threats (enemies)
	Initiative int
}

type Combat struct {
	Combatants  []Combatant
	TurnIndex   int // whose turn in initiative order
	Phase       CombatTurnPhase
	MoveLeft    int // remaining movement for current combatant
	ActionTaken bool
	RoomIdx     int // which room this fight is in
}

func (c *Combat) Current() *Combatant {
	return &c.Combatants[c.TurnIndex]
}

func (c *Combat) NextTurn() {
	c.TurnIndex = (c.TurnIndex + 1) % len(c.Combatants)
	c.Phase = PhaseMovement
	c.ActionTaken = false
}

// StartCombat initiates combat with an entity entering a room with threats.
// Rolls initiative (d20 + DEX mod) for all participants.
func (g *GameState) StartCombat(w *World, entityIdx int, roomIdx int) {
	ent := g.Entities[entityIdx]
	ent.Task = nil // stop whatever they were doing

	var combatants []Combatant

	// Add the entity as ally
	dexMod := 0
	if ent.Stats != nil {
		dexMod = ent.Stats.Mod(ent.Stats.DEX)
	}
	combatants = append(combatants, Combatant{
		IsEnemy:    false,
		EntityIdx:  entityIdx,
		Initiative: rollD20() + dexMod,
	})

	// Add all threats in the same room
	for key, threat := range w.Threats {
		if threat.RoomIdx == roomIdx {
			combatants = append(combatants, Combatant{
				IsEnemy:    true,
				ThreatKey:  key,
				Initiative: rollD20() + (threat.DEX-10)/2,
			})
		}
	}

	// Sort by initiative descending
	for i := 0; i < len(combatants); i++ {
		for j := i + 1; j < len(combatants); j++ {
			if combatants[j].Initiative > combatants[i].Initiative {
				combatants[i], combatants[j] = combatants[j], combatants[i]
			}
		}
	}

	// Set move points for first combatant
	moveLeft := 0
	first := combatants[0]
	if !first.IsEnemy {
		if s := g.Entities[first.EntityIdx].Stats; s != nil {
			moveLeft = s.MoveSpeed
		}
	} else {
		if t, ok := w.Threats[first.ThreatKey]; ok {
			moveLeft = t.MoveSpeed
		}
	}

	g.Combat = &Combat{
		Combatants: combatants,
		TurnIndex:  0,
		Phase:      PhaseMovement,
		MoveLeft:   moveLeft,
		RoomIdx:    roomIdx,
	}
	g.SelectedEnt = entityIdx
	g.SetMessage("Combat started! Roll initiative!")
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
	Combat      *Combat // nil = continuous mode
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

func rollD20() int  { return rand.Intn(20) + 1 }
func rollDice(sides int) int {
	if sides <= 0 {
		return 0
	}
	return rand.Intn(sides) + 1
}

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

		// Explore: patrol dungeon perimeter (or push inward with pickaxe)
		if ent.Task.Type == TaskExplore && (ent.Task.Path == nil || ent.Task.PathIdx >= len(ent.Task.Path)) {
			if ent.Task.ExploreAngle == 0 {
				ent.Task.ExploreAngle = math.Atan2(float64(ent.Z), float64(ent.X))
			}

			if g.HasPickaxe {
				// With pickaxe: push inward at current angle, then advance
				midR := (w.OuterEdge(ent.Task.ExploreAngle) + w.InnerEdge(ent.Task.ExploreAngle)) / 2
				tx := int(math.Cos(ent.Task.ExploreAngle) * midR)
				tz := int(math.Sin(ent.Task.ExploreAngle) * midR)
				path := FindPathBreakable(w, ent.X, ent.Z, tx, tz)
				if path != nil {
					ent.Task.Path = path
					ent.Task.PathIdx = 0
				}
				// Only advance angle after committing to a breach
				ent.Task.ExploreAngle += 0.35
			} else {
				// No pickaxe: patrol outside the wall
				ent.Task.ExploreAngle += 0.35
				outerR := w.OuterEdge(ent.Task.ExploreAngle)
				patrolR := outerR + float64(ent.RevealDist)/2
				tx := int(math.Cos(ent.Task.ExploreAngle) * patrolR)
				tz := int(math.Sin(ent.Task.ExploreAngle) * patrolR)
				fx, fz, found := nearestWalkable(w, tx, tz)
				if found {
					path := FindPath(w, ent.X, ent.Z, fx, fz)
					if path != nil {
						ent.Task.Path = path
						ent.Task.PathIdx = 0
					}
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
			// Break wall if we have the pickaxe
			if g.HasPickaxe {
				if t, ok := w.GetTile(nx, nz); ok && t == TileSolid {
					w.SetTile(nx, nz, TileDoorway)
					w.RevealAround(nx, nz)
					// Wall broken — check if we just revealed threats
					if ent.Stats != nil {
						for _, threat := range w.Threats {
							if w.IsRevealed(threat.X, threat.Z) &&
								withinRange(ent.X, ent.Z, threat.X, threat.Z, ent.RevealDist+3) {
								g.StartCombat(w, i, threat.RoomIdx)
								return nil
							}
						}
					}
					continue // spend this tick breaking, move next tick
				}
			}
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
		w.UnloadFarChunks(cx, cz)
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

		// Check if entity can see any revealed threats
		if ent.Stats != nil {
			for _, threat := range w.Threats {
				if w.IsRevealed(threat.X, threat.Z) &&
					withinRange(ent.X, ent.Z, threat.X, threat.Z, ent.RevealDist) {
					g.StartCombat(w, i, threat.RoomIdx)
					return nil
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

func findPath(w *World, sx, sz, gx, gz int, breakWalls bool) [][2]int {
	canPass := func(tx, tz int) bool {
		if w.IsWalkable(tx, tz) {
			return true
		}
		if breakWalls {
			t, ok := w.GetTile(tx, tz)
			return ok && t == TileSolid
		}
		return false
	}

	if !canPass(gx, gz) {
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
			if closed[key{nx, nz}] || !canPass(nx, nz) {
				continue
			}
			ng := cur.g + 1
			// Wall tiles cost more so entities prefer existing paths
			if breakWalls {
				if t, ok := w.GetTile(nx, nz); ok && t == TileSolid {
					ng += 5
				}
			}
			nf := ng + abs(gx-nx) + abs(gz-nz)
			heap.Push(open, &astarNode{x: nx, z: nz, g: ng, f: nf, parent: cur})
		}
	}
	return nil
}

func FindPath(w *World, sx, sz, gx, gz int) [][2]int {
	return findPath(w, sx, sz, gx, gz, false)
}

func FindPathBreakable(w *World, sx, sz, gx, gz int) [][2]int {
	return findPath(w, sx, sz, gx, gz, true)
}
