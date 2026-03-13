package main

import (
	"fmt"
	"testing"
)

// Wall breaking and pathing tests: return paths, path preference, breach cost.

// After breaching into a room, can Brynn path back to spawn?
// This validates return path viability — the path back should use
// the doorways she already created.
func TestWall_SupplyLineBackToSpawn(t *testing.T) {
	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]
			spawnX, spawnZ := brynn.X, brynn.Z

			// Get pickaxe
			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				t.Logf("couldn't reach pickaxe — skipping")
				return
			}

			// Find a room and breach to it
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
				t.Logf("no suitable room — skipping")
				return
			}

			tx, tz := targetRoom.X+targetRoom.W/2, targetRoom.Z+targetRoom.H/2
			walkToBreakable(g, w, 0, tx, tz, 500)

			if g.Combat != nil {
				runCombatToEnd(g, w, 200)
			}
			if g.GameOver {
				t.Logf("died during breach")
				return
			}

			// Now try to path back to spawn using normal (non-breakable) pathfinding
			// This should work through the doorways created during breach
			brynn = g.Entities[0]
			pathBack := FindPath(w, brynn.X, brynn.Z, spawnX, spawnZ)

			if pathBack == nil {
				// Try breakable as fallback to see if there's any path at all
				pathBreakable := FindPathBreakable(w, brynn.X, brynn.Z, spawnX, spawnZ)
				if pathBreakable == nil {
					t.Errorf("no path back to spawn at all — completely stranded at (%d,%d)", brynn.X, brynn.Z)
				} else {
					t.Errorf("no walkable path back to spawn (need to break %d more walls)",
						countWallsInPath(w, pathBreakable))
				}
			} else {
				t.Logf("return path OK — %d tiles back to spawn from (%d,%d)",
					len(pathBack), brynn.X, brynn.Z)
			}
		})
	}
}

// Does breakable pathfinding prefer existing openings over breaking new walls?
// Compare: path from A to B with an existing doorway nearby vs path through solid.
func TestWall_PrefersExistingOpenings(t *testing.T) {
	g, w := newSeededGame(testSeed)
	brynn := g.Entities[0]

	// Get pickaxe first
	walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
	if !g.HasPickaxe {
		t.Skip("couldn't reach pickaxe")
	}

	// Breach to first room — creates doorways
	var firstRoom Room
	for _, room := range w.Rooms {
		if room.W > 2 && room.H > 2 {
			firstRoom = room
			break
		}
	}
	tx, tz := firstRoom.X+firstRoom.W/2, firstRoom.Z+firstRoom.H/2
	walkToBreakable(g, w, 0, tx, tz, 500)

	if g.Combat != nil {
		runCombatToEnd(g, w, 200)
	}
	if g.GameOver {
		t.Skip("died during first breach")
	}

	brynn = g.Entities[0]

	// Now find a second room nearby and path to it
	var secondRoom Room
	secondFound := false
	for _, room := range w.Rooms {
		if room.W > 2 && room.H > 2 && room.X != firstRoom.X {
			// Check it's somewhat close
			rx, rz := room.X+room.W/2, room.Z+room.H/2
			dist := abs(brynn.X-rx) + abs(brynn.Z-rz)
			if dist < 40 {
				secondRoom = room
				secondFound = true
				break
			}
		}
	}
	if !secondFound {
		t.Skip("no second room nearby")
	}

	sx, sz := secondRoom.X+secondRoom.W/2, secondRoom.Z+secondRoom.H/2
	path := FindPathBreakable(w, brynn.X, brynn.Z, sx, sz)
	if path == nil {
		t.Skip("no breakable path to second room")
	}

	wallsInPath := countWallsInPath(w, path)
	totalLen := len(path)

	t.Logf("path to second room: %d tiles, %d walls to break (%.0f%% reuse)",
		totalLen, wallsInPath, float64(totalLen-wallsInPath)*100/float64(totalLen))

	// The path should mostly reuse existing openings rather than breaking fresh
	// With wall cost +5, the A* should strongly prefer walkable tiles
	if totalLen > 5 && wallsInPath > totalLen/2 {
		t.Errorf("path breaks too many walls: %d/%d — not reusing existing openings",
			wallsInPath, totalLen)
	}
}

// How many walls broken per room reached across seeds?
func TestWall_BreachCostPerRoom(t *testing.T) {
	totalWalls := 0
	totalRooms := 0

	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)

			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				t.Logf("couldn't reach pickaxe — skipping")
				return
			}

			brynn := g.Entities[0]

			// Count doorway tiles before breaching
			doorwaysBefore := countDoorways(w)

			// Breach to first 3 rooms
			roomsReached := 0
			for _, room := range w.Rooms {
				if room.W <= 2 || room.H <= 2 {
					continue
				}
				if roomsReached >= 3 {
					break
				}

				tx, tz := room.X+room.W/2, room.Z+room.H/2
				walkToBreakable(g, w, 0, tx, tz, 300)

				if g.Combat != nil {
					runCombatToEnd(g, w, 200)
				}
				if g.GameOver {
					break
				}

				// Check if Brynn actually reached the room
				brynn = g.Entities[0]
				if abs(brynn.X-tx) <= room.W/2 && abs(brynn.Z-tz) <= room.H/2 {
					roomsReached++
				}
			}

			doorwaysAfter := countDoorways(w)
			wallsBroken := doorwaysAfter - doorwaysBefore

			if roomsReached > 0 {
				totalWalls += wallsBroken
				totalRooms += roomsReached
				t.Logf("rooms=%d walls=%d avg=%.1f",
					roomsReached, wallsBroken, float64(wallsBroken)/float64(roomsReached))
			} else {
				t.Logf("no rooms reached — walls=%d", wallsBroken)
			}
		})
	}

	if totalRooms > 0 {
		t.Logf("TOTAL: %d rooms, %d walls broken, avg %.1f walls/room",
			totalRooms, totalWalls, float64(totalWalls)/float64(totalRooms))
	}
}

// --- Helpers ---

func countWallsInPath(w *World, path [][2]int) int {
	walls := 0
	for _, p := range path {
		if t, ok := w.GetTile(p[0], p[1]); ok && t == TileSolid {
			walls++
		}
	}
	return walls
}

func countDoorways(w *World) int {
	count := 0
	for _, chunk := range w.Chunks {
		ox, oz := ChunkOrigin(chunk.CX, chunk.CZ)
		for lz := 0; lz < ChunkSize; lz++ {
			for lx := 0; lx < ChunkSize; lx++ {
				if chunk.Tiles[lz][lx] == TileDoorway {
					_ = ox + lx
					_ = oz + lz
					count++
				}
			}
		}
	}
	return count
}
