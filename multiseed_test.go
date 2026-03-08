package main

import (
	"fmt"
	"testing"
)

// Multi-seed tests run core game flows across many worlds.
// Same game logic, different terrain — catches seed-dependent bugs.

var testSeeds = []int64{
	42, 123, 456, 789, 999,
	1337, 2024, 3141, 4269, 5555,
	7777, 9001, 10101, 12345, 13013,
	20000, 31415, 54321, 65536, 99999,
}

func newSeededGame(seed int64) (*GameState, *World) {
	w := NewWorld(seed)
	spawnX := OuterRadius + DungeonWarpAmp + 5
	w.PlacePickaxe(spawnX, 0)

	g := &GameState{SelectedEnt: 0}
	g.Entities = []*Entity{
		{Name: "Brynn", X: spawnX, Z: 0, Tier: TierVeteran, RevealDist: 8,
			Scouted: map[[2]int]bool{},
			Stats: &CombatStats{
				HP: 28, MaxHP: 28, AC: 16,
				STR: 16, DEX: 12, CON: 14, INT: 10, WIS: 12, CHA: 10,
				Level: 3, ProfBonus: 2, MoveSpeed: 5,
				Class: "Fighter", ClassCharges: 2, MaxClassCharges: 2,
				HitDice: 3, MaxHitDice: 3, HitDieSize: 10,
			}},
	}
	for _, ent := range g.Entities {
		cx, cz := TileToChunk(ent.X, ent.Z)
		w.EnsureChunksAround(cx, cz)
		w.RevealAround(ent.X, ent.Z)
	}
	return g, w
}

// --- World generation integrity across seeds ---

func TestMultiSeed_WorldGeneration(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			w := NewWorld(seed)

			// Must have rooms
			if len(w.Rooms) < 10 {
				t.Errorf("only %d rooms", len(w.Rooms))
			}

			// Must have threats
			if len(w.Threats) < 3 {
				t.Errorf("only %d threats", len(w.Threats))
			}

			// All rooms must fit in ring (with buffer)
			for i, r := range w.Rooms {
				if r.W <= 1 && r.H <= 1 {
					continue // corridor
				}
				for z := r.Z; z < r.Z+r.H; z++ {
					for x := r.X; x < r.X+r.W; x++ {
						base := w.BaseTileType(x, z)
						if base != TileSolid {
							t.Errorf("room %d tile (%d,%d) base=%d, want TileSolid", i, x, z, base)
						}
					}
				}
			}

			// All threats must be on floor tiles
			spawnX := OuterRadius + DungeonWarpAmp + 5
			w.PlacePickaxe(spawnX, 0)
			for _, room := range w.Rooms {
				cx, cz := TileToChunk(room.X, room.Z)
				w.EnsureChunksAround(cx, cz)
			}
			for key, threat := range w.Threats {
				tile := w.TileTypeAt(threat.X, threat.Z)
				if tile != TileFloor {
					t.Errorf("threat at (%d,%d) on tile %d, want TileFloor", key[0], key[1], tile)
				}
			}
		})
	}
}

// --- Pickaxe reachability ---

func TestMultiSeed_PickaxeReachable(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]

			path := FindPath(w, brynn.X, brynn.Z, w.PickaxeX, w.PickaxeZ)
			if path == nil {
				t.Errorf("no path from Brynn (%d,%d) to pickaxe (%d,%d)",
					brynn.X, brynn.Z, w.PickaxeX, w.PickaxeZ)
			}
		})
	}
}

// --- Dungeon breachable ---

func TestMultiSeed_DungeonBreachable(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]

			// Find a real room
			found := false
			for _, room := range w.Rooms {
				if room.W <= 1 || room.H <= 1 {
					continue
				}
				tx, tz := room.X+room.W/2, room.Z+room.H/2
				path := FindPathBreakable(w, brynn.X, brynn.Z, tx, tz)
				if path != nil {
					found = true
					break
				}
			}
			if !found {
				t.Error("no room reachable via breakable pathfinding")
			}
		})
	}
}

// --- No combat without pickaxe ---

func TestMultiSeed_NoCombatWithoutPickaxe(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]

			// Scout for 500 ticks without pickaxe
			brynn.Task = &Task{Type: TaskExplore}
			for i := 0; i < 500; i++ {
				if g.Combat != nil {
					t.Errorf("tick %d: combat triggered without pickaxe at (%d,%d)",
						i, brynn.X, brynn.Z)
					return
				}
				if g.HasPickaxe {
					return // picked up pickaxe, test is no longer valid
				}
				if g.GameOver {
					t.Error("game over without combat — shouldn't happen")
					return
				}
				g.WorldStep(w)
			}
		})
	}
}

// --- Full session: pickaxe → breach → combat → survive ---

func TestMultiSeed_FullSession(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]

			// Phase 1: Walk to pickaxe
			reached := walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !reached || !g.HasPickaxe {
				t.Errorf("couldn't reach pickaxe")
				return
			}

			// Phase 2: Breach into first room
			var targetRoom Room
			roomFound := false
			for _, room := range w.Rooms {
				if room.W > 2 && room.H > 2 {
					targetRoom = room
					roomFound = true
					break
				}
			}
			if !roomFound {
				t.Error("no suitable room")
				return
			}

			tx, tz := targetRoom.X+targetRoom.W/2, targetRoom.Z+targetRoom.H/2
			walkToBreakable(g, w, 0, tx, tz, 500)

			// Phase 3: Handle combat if triggered
			if g.Combat != nil {
				runCombatToEnd(g, w, 200)
			}
			if g.GameOver {
				t.Logf("Brynn fell — valid outcome")
				return
			}

			// Phase 4: Rest and verify recovery
			beforeHP := brynn.Stats.HP
			brynn.Stats.ClassCharges = 0
			g.TryRest(w, 0)
			ambushed := g.Combat != nil
			if ambushed {
				runCombatToEnd(g, w, 100)
			}
			if g.GameOver {
				t.Logf("fell during rest ambush")
				return
			}

			if !ambushed && brynn.Stats.ClassCharges != brynn.Stats.MaxClassCharges {
				t.Errorf("charges not restored: got %d, want %d",
					brynn.Stats.ClassCharges, brynn.Stats.MaxClassCharges)
			}

			// Invariants
			if v := checkInvariants(g, w); len(v) > 0 {
				for _, msg := range v {
					t.Error(msg)
				}
			}

			t.Logf("tick=%d pos=(%d,%d) HP=%d/%d",
				g.TimeTicks, brynn.X, brynn.Z, brynn.Stats.HP, brynn.Stats.MaxHP)
			_ = beforeHP
		})
	}
}

// --- Extended scout: 2000 steps with invariant checks ---

func TestMultiSeed_ExtendedScout(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]
			brynn.Task = &Task{Type: TaskExplore}

			combatCount := 0
			for i := 0; i < 2000; i++ {
				if g.GameOver {
					break
				}
				if g.Combat != nil {
					combatCount++
					runCombatToEnd(g, w, 200)
					if g.GameOver {
						break
					}
					// Resume scouting
					brynn.Task = &Task{Type: TaskExplore}
					continue
				}
				g.WorldStep(w)
			}

			if !g.GameOver {
				if v := checkInvariants(g, w); len(v) > 0 {
					for _, msg := range v {
						t.Error(msg)
					}
				}
			}

			t.Logf("tick=%d combats=%d pos=(%d,%d) HP=%d/%d pickaxe=%v alive=%v",
				g.TimeTicks, combatCount, brynn.X, brynn.Z,
				brynn.Stats.HP, brynn.Stats.MaxHP, g.HasPickaxe, !g.GameOver)
		})
	}
}

// --- Mage rescue across seeds ---

func TestMultiSeed_MageRescue(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)

			mageRescueRoom := -1
			g.Events = []*Event{
				{
					ID: "mage_rescue_enter", Trigger: TriggerRoomEntered, OneShot: true,
					Check: func(g *GameState, w *World, ctx EventContext) bool {
						return mageRescueRoom < 0
					},
					Fire: func(g *GameState, w *World, ctx EventContext) {
						mageRescueRoom = ctx.RoomIdx
						for key, thr := range w.Threats {
							if thr.RoomIdx == ctx.RoomIdx {
								delete(w.Threats, key)
							}
						}
						ent := g.Entities[ctx.EntityIdx]
						for _, off := range [2][2]int{{1, 0}, {0, 1}} {
							fx, fz, ok := nearestWalkable(w, ent.X+off[0], ent.Z+off[1])
							if !ok {
								continue
							}
							w.Threats[[2]int{fx, fz}] = Threat{
								X: fx, Z: fz, Type: SkeletonMinion, RoomIdx: ctx.RoomIdx,
								HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
								MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
							}
						}
					},
				},
				{
					ID: "mage_rescue_clear", Trigger: TriggerRoomCleared, OneShot: true,
					Check: func(g *GameState, w *World, ctx EventContext) bool {
						return mageRescueRoom >= 0 && ctx.RoomIdx == mageRescueRoom
					},
					Fire: func(g *GameState, w *World, ctx EventContext) {
						room := w.Rooms[ctx.RoomIdx]
						mx, mz := room.X+room.W/2, room.Z+room.H/2
						mx, mz, _ = nearestClearTile(g, w, mx, mz)
						g.Entities = append(g.Entities, &Entity{
							Name: "Elara", X: mx, Z: mz, Tier: TierSoldier, RevealDist: 5,
							Scouted: map[[2]int]bool{},
							Stats: &CombatStats{
								HP: 18, MaxHP: 18, AC: 12,
								STR: 8, DEX: 14, CON: 12, INT: 16, WIS: 14, CHA: 10,
								Level: 3, ProfBonus: 2, MoveSpeed: 5, Class: "Mage",
								ClassCharges: 3, MaxClassCharges: 3,
								HitDice: 3, MaxHitDice: 3, HitDieSize: 6,
							},
						})
					},
				},
			}

			// Get pickaxe
			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				t.Logf("couldn't reach pickaxe — skipping")
				return
			}

			// Find first threat room and breach toward it
			_, tx, tz, found := findFirstThreatRoom(w)
			if !found {
				t.Error("no threats")
				return
			}

			brynn := g.Entities[0]
			walkToBreakable(g, w, 0, tx, tz, 500)

			// If no combat yet, force it
			if g.Combat == nil && mageRescueRoom < 0 {
				for _, threat := range w.Threats {
					fx, fz := threat.X-1, threat.Z
					if w.IsWalkable(fx, fz) && !w.IsThreatAt(fx, fz) {
						brynn.X, brynn.Z = fx, fz
						cx, cz := TileToChunk(brynn.X, brynn.Z)
						w.EnsureChunksAround(cx, cz)
						g.StartCombat(w, 0, threat.RoomIdx)
						break
					}
				}
			}

			if mageRescueRoom < 0 {
				t.Error("rescue event never fired")
				return
			}

			// Fight
			if g.Combat != nil {
				runCombatToEnd(g, w, 200)
			}
			if g.GameOver {
				t.Logf("Brynn fell during rescue")
				return
			}

			// Elara should exist (unless enemies leashed and room wasn't cleared)
			if len(g.Entities) < 2 {
				t.Logf("Elara not in roster — room may not have cleared (enemies leashed)")
				return
			}
			if g.Entities[1].Name != "Elara" {
				t.Errorf("expected Elara, got %s", g.Entities[1].Name)
			}

			// Invariants
			if v := checkInvariants(g, w); len(v) > 0 {
				for _, msg := range v {
					t.Error(msg)
				}
			}
		})
	}
}
