package main

import (
	"testing"
)

// Bug: rooms at the dungeon ring boundary can have floor tiles directly
// adjacent to ground tiles with no solid wall between them. This lets
// entities on the outside see and fight threats inside without a pickaxe.
//
// Every dungeon floor tile must have at least one solid tile between it
// and any ground tile. No floor should be directly adjacent to ground.

func TestBug_NoFloorAdjacentToGround(t *testing.T) {
	seeds := []int64{42, 123, 999, 7777, 54321}
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for _, seed := range seeds {
		w := NewWorld(seed)

		// Load all chunks that contain rooms
		for _, room := range w.Rooms {
			cx, cz := TileToChunk(room.X, room.Z)
			w.EnsureChunksAround(cx, cz)
		}

		violations := 0
		for _, room := range w.Rooms {
			for z := room.Z; z < room.Z+room.H; z++ {
				for x := room.X; x < room.X+room.W; x++ {
					tile := w.TileTypeAt(x, z)
					if tile != TileFloor {
						continue
					}
					for _, d := range dirs {
						nx, nz := x+d[0], z+d[1]
						neighbor := w.TileTypeAt(nx, nz)
						if neighbor == TileGround {
							violations++
							if violations <= 5 {
								t.Errorf("seed %d: floor at (%d,%d) adjacent to ground at (%d,%d)",
									seed, x, z, nx, nz)
							}
						}
					}
				}
			}
		}

		if violations > 5 {
			t.Errorf("seed %d: %d total floor/ground adjacency violations (showing first 5)", seed, violations)
		}
	}
}

// The scout should never trigger combat without a pickaxe. If she's
// patrolling outside the dungeon walls, no threat should be visible.

func TestBug_NoLOSToThreatsFromOutside(t *testing.T) {
	seeds := []int64{42, 123, 999, 7777, 54321}

	for _, seed := range seeds {
		w := NewWorld(seed)
		spawnX := OuterRadius + DungeonWarpAmp + 5

		// Load chunks around the dungeon perimeter
		for angle := 0.0; angle < 6.3; angle += 0.2 {
			outerR := w.OuterEdge(angle)
			px := int(float64(spawnX) * outerR / float64(spawnX))
			pz := int(float64(0))
			cx, cz := TileToChunk(px, pz)
			w.EnsureChunksAround(cx, cz)
		}
		// Also load around all threats
		for _, threat := range w.Threats {
			cx, cz := TileToChunk(threat.X, threat.Z)
			w.EnsureChunksAround(cx, cz)
		}

		// Walk every ground tile and check if any threat is visible within aggro range
		violations := 0
		for _, chunk := range w.Chunks {
			ox, oz := ChunkOrigin(chunk.CX, chunk.CZ)
			for lz := 0; lz < ChunkSize; lz++ {
				for lx := 0; lx < ChunkSize; lx++ {
					if chunk.Tiles[lz][lx] != TileGround {
						continue
					}
					gx, gz := ox+lx, oz+lz
					for _, threat := range w.Threats {
						if withinRange(gx, gz, threat.X, threat.Z, 4) &&
							w.HasLineOfSight(gx, gz, threat.X, threat.Z) {
							violations++
							if violations <= 3 {
								t.Errorf("seed %d: ground (%d,%d) has LOS to threat at (%d,%d) room %d",
									seed, gx, gz, threat.X, threat.Z, threat.RoomIdx)
							}
						}
					}
				}
			}
		}

		if violations > 3 {
			t.Errorf("seed %d: %d total ground-to-threat LOS violations", seed, violations)
		}
	}
}
