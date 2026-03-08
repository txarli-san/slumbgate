package main

import (
	"fmt"
	"testing"
)

// Combat balance tests: measure win rates, HP remaining, and fight stats
// across seeds to understand if combat is fair and balanced.

func TestBalance_BrynnSoloFirstRoom(t *testing.T) {
	wins, losses, leashes := 0, 0, 0
	totalHPRemaining := 0
	totalEnemies := 0
	fought := 0

	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)

			// Get pickaxe
			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				t.Logf("couldn't reach pickaxe — skipping")
				return
			}

			// Breach to first threat room
			roomIdx, tx, tz, found := findFirstThreatRoom(w)
			if !found {
				t.Logf("no threat room — skipping")
				return
			}

			// Count enemies in this room
			enemies := 0
			for _, thr := range w.Threats {
				if thr.RoomIdx == roomIdx {
					enemies++
				}
			}
			totalEnemies += enemies

			walkToBreakable(g, w, 0, tx, tz, 500)

			// Force combat if not triggered naturally
			if g.Combat == nil && !g.GameOver {
				brynn := g.Entities[0]
				for _, threat := range w.Threats {
					if threat.RoomIdx == roomIdx {
						fx, fz := threat.X-1, threat.Z
						if w.IsWalkable(fx, fz) && !w.IsThreatAt(fx, fz) {
							brynn.X, brynn.Z = fx, fz
							cx, cz := TileToChunk(fx, fz)
							w.EnsureChunksAround(cx, cz)
							g.StartCombat(w, 0, roomIdx)
							break
						}
					}
				}
			}

			if g.Combat == nil {
				t.Logf("no combat triggered — skipping")
				return
			}

			fought++
			runCombatToEnd(g, w, 200)

			if g.GameOver {
				losses++
				t.Logf("LOSS — enemies=%d", enemies)
				return
			}

			brynn := g.Entities[0]
			if g.Combat != nil {
				// Combat still going = timeout
				t.Logf("TIMEOUT — enemies=%d", enemies)
			} else {
				// Check if enemies leashed or were killed
				roomThreatsLeft := 0
				for _, thr := range w.Threats {
					if thr.RoomIdx == roomIdx {
						roomThreatsLeft++
					}
				}
				if roomThreatsLeft > 0 {
					leashes++
					t.Logf("LEASH — enemies=%d left=%d HP=%d/%d",
						enemies, roomThreatsLeft, brynn.Stats.HP, brynn.Stats.MaxHP)
				} else {
					wins++
					totalHPRemaining += brynn.Stats.HP
					t.Logf("WIN — enemies=%d HP=%d/%d",
						enemies, brynn.Stats.HP, brynn.Stats.MaxHP)
				}
			}
		})
	}

	avgHP := 0
	if wins > 0 {
		avgHP = totalHPRemaining / wins
	}
	t.Logf("SOLO BRYNN: fought=%d wins=%d losses=%d leashes=%d avgHPrem=%d avgEnemies=%.1f",
		fought, wins, losses, leashes, avgHP, float64(totalEnemies)/float64(max(fought, 1)))
}

func TestBalance_BrynnElaraFirstRoom(t *testing.T) {
	wins, losses, leashes := 0, 0, 0
	totalHPBrynn, totalHPElara := 0, 0
	totalEnemies := 0
	fought := 0

	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			addElara(g, w)

			// Get pickaxe
			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				t.Logf("couldn't reach pickaxe — skipping")
				return
			}

			// Breach to first threat room
			roomIdx, tx, tz, found := findFirstThreatRoom(w)
			if !found {
				t.Logf("no threat room — skipping")
				return
			}

			enemies := 0
			for _, thr := range w.Threats {
				if thr.RoomIdx == roomIdx {
					enemies++
				}
			}
			totalEnemies += enemies

			walkToBreakable(g, w, 0, tx, tz, 500)

			// Move Elara near Brynn so she joins combat
			brynn := g.Entities[0]
			elara := g.Entities[1]
			ex, ez, _ := nearestClearTile(g, w, brynn.X, brynn.Z)
			elara.X, elara.Z = ex, ez

			if g.Combat == nil && !g.GameOver {
				for _, threat := range w.Threats {
					if threat.RoomIdx == roomIdx {
						fx, fz := threat.X-1, threat.Z
						if w.IsWalkable(fx, fz) && !w.IsThreatAt(fx, fz) {
							brynn.X, brynn.Z = fx, fz
							ex, ez, _ = nearestClearTile(g, w, brynn.X, brynn.Z)
							elara.X, elara.Z = ex, ez
							cx, cz := TileToChunk(fx, fz)
							w.EnsureChunksAround(cx, cz)
							g.StartCombat(w, 0, roomIdx)
							break
						}
					}
				}
			}

			if g.Combat == nil {
				t.Logf("no combat triggered — skipping")
				return
			}

			fought++
			runCombatToEnd(g, w, 200)

			if g.GameOver {
				losses++
				t.Logf("LOSS — enemies=%d", enemies)
			} else if g.Combat != nil {
				t.Logf("TIMEOUT — enemies=%d", enemies)
			} else {
				roomThreatsLeft := 0
				for _, thr := range w.Threats {
					if thr.RoomIdx == roomIdx {
						roomThreatsLeft++
					}
				}
				if roomThreatsLeft > 0 {
					leashes++
					t.Logf("LEASH — enemies=%d left=%d", enemies, roomThreatsLeft)
				} else {
					wins++
					// Check who's alive
					brynnHP, elaraHP := 0, 0
					elaraAlive := false
					for _, ent := range g.Entities {
						if ent.Name == "Brynn" {
							brynnHP = ent.Stats.HP
						}
						if ent.Name == "Elara" {
							elaraHP = ent.Stats.HP
							elaraAlive = true
						}
					}
					totalHPBrynn += brynnHP
					if elaraAlive {
						totalHPElara += elaraHP
					}
					t.Logf("WIN — enemies=%d Brynn=%d/%d Elara=%d/%d alive=%v",
						enemies, brynnHP, 28, elaraHP, 18, elaraAlive)
				}
			}
		})
	}

	avgBrynn, avgElara := 0, 0
	if wins > 0 {
		avgBrynn = totalHPBrynn / wins
		avgElara = totalHPElara / wins
	}
	t.Logf("BRYNN+ELARA: fought=%d wins=%d losses=%d leashes=%d avgBrynnHP=%d avgElaraHP=%d avgEnemies=%.1f",
		fought, wins, losses, leashes, avgBrynn, avgElara, float64(totalEnemies)/float64(max(fought, 1)))
}

func TestBalance_EnemyLeashBehavior(t *testing.T) {
	leashed, killed, stuck := 0, 0, 0

	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]

			// Place one enemy 3 tiles away (not adjacent — should pursue then leash)
			ex, ez := brynn.X+3, brynn.Z
			w.SetTile(ex, ez, TileGround)
			cx, cz := TileToChunk(ex, ez)
			w.EnsureChunksAround(cx, cz)

			w.Threats[[2]int{ex, ez}] = Threat{
				X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
				HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
				MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
			}

			g.StartCombat(w, 0, 0)
			if g.Combat == nil {
				t.Logf("combat didn't start — skipping")
				return
			}

			// Run combat but DON'T move the player — just end turns
			for turn := 0; turn < 50; turn++ {
				if g.Combat == nil || g.GameOver {
					break
				}
				c := g.Combat
				cur := c.Current()
				if cur.IsEnemy {
					g.RunEnemyTurn(w)
				} else {
					// Player does nothing — just end turn
					c.NextTurn(g, w)
				}
			}

			if g.GameOver {
				killed++
				t.Logf("Brynn died")
			} else if g.Combat == nil {
				// Combat ended — check if enemy is still alive (leashed) or dead
				if w.IsThreatAt(ex, ez) || len(w.LeashingThreats) > 0 {
					leashed++
					t.Logf("LEASH — enemy gave up")
				} else {
					killed++
					t.Logf("KILLED — enemy died somehow")
				}
			} else {
				stuck++
				t.Logf("STUCK — combat still going after 50 turns")
			}
		})
	}

	t.Logf("LEASH TEST: leashed=%d killed=%d stuck=%d (of %d seeds)",
		leashed, killed, stuck, len(testSeeds))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
