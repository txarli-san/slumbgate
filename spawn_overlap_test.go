package main

import (
	"fmt"
	"testing"
)

// Bug: Mage rescue event spawns Elara at room center, which may
// already have a threat. Entities should never overlap threats.

func TestBug_ElaraSpawnsOnClearTile(t *testing.T) {
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
							},
						})
					},
				},
			}

			// Get pickaxe
			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				return
			}

			// Breach to first threat
			_, tx, tz, found := findFirstThreatRoom(w)
			if !found {
				return
			}
			walkToBreakable(g, w, 0, tx, tz, 500)

			// Force combat if not triggered
			if g.Combat == nil && mageRescueRoom < 0 {
				for _, threat := range w.Threats {
					brynn := g.Entities[0]
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

			// Fight
			if g.Combat != nil {
				runCombatToEnd(g, w, 200)
			}
			if g.GameOver {
				return
			}

			// Check: Elara doesn't overlap a threat (the specific spawn bug)
			for _, ent := range g.Entities {
				if ent.Name == "Elara" && w.IsThreatAt(ent.X, ent.Z) {
					t.Errorf("Elara at (%d,%d) overlaps a threat", ent.X, ent.Z)
				}
			}
		})
	}
}
