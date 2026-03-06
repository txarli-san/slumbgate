package main

import "math"

const (
	ChunkSize    = 16
	LoadRadius   = 3
	UnloadRadius = 5

	// Dungeon shape — center at world origin (0,0)
	DungeonRadius    = 80  // base radius in tiles
	DungeonWarpScale = 8.0 // noise frequency (lower = smoother)
	DungeonWarpAmp   = 15  // max deviation from base radius
)

type TileType int

const (
	TileGround TileType = iota // outside — walkable open terrain
	TileFloor                  // dungeon interior
)

type ChunkCoord struct {
	X, Z int
}

type Chunk struct {
	CX, CZ   int
	Tiles     [ChunkSize][ChunkSize]TileType
	Explored  bool
}

type World struct {
	Seed   int64
	Chunks map[ChunkCoord]*Chunk
}

func NewWorld(seed int64) *World {
	return &World{
		Seed:   seed,
		Chunks: make(map[ChunkCoord]*Chunk),
	}
}

// DungeonEdge returns the wall distance from center at a given angle.
// Uses seeded noise to create an irregular boundary.
func (w *World) DungeonEdge(angle float64) float64 {
	// Simple layered sine noise seeded by world seed
	s := float64(w.Seed % 10000)
	n := math.Sin(angle*DungeonWarpScale+s)*0.5 +
		math.Sin(angle*DungeonWarpScale*2.3+s*1.7)*0.25 +
		math.Sin(angle*DungeonWarpScale*4.1+s*0.3)*0.125
	// Normalize to -1..1 range (max amplitude ~0.875)
	n /= 0.875
	return float64(DungeonRadius) + n*float64(DungeonWarpAmp)
}

// TileTypeAt determines what a tile is based on its distance from dungeon center
func (w *World) TileTypeAt(tx, tz int) TileType {
	dx := float64(tx) + 0.5
	dz := float64(tz) + 0.5
	dist := math.Sqrt(dx*dx + dz*dz)
	angle := math.Atan2(dz, dx)

	edge := w.DungeonEdge(angle)

	if dist > edge {
		return TileGround
	}
	return TileFloor
}

func TileToChunk(tx, tz int) (int, int) {
	cx := tx >> 4
	cz := tz >> 4
	if tx < 0 && tx&0xF != 0 {
		cx--
	}
	if tz < 0 && tz&0xF != 0 {
		cz--
	}
	return cx, cz
}

func ChunkOrigin(cx, cz int) (int, int) {
	return cx * ChunkSize, cz * ChunkSize
}

func (w *World) generateChunk(cx, cz int) *Chunk {
	c := &Chunk{CX: cx, CZ: cz}
	ox, oz := ChunkOrigin(cx, cz)
	for lz := range ChunkSize {
		for lx := range ChunkSize {
			c.Tiles[lz][lx] = w.TileTypeAt(ox+lx, oz+lz)
		}
	}
	return c
}

func (w *World) EnsureChunksAround(cx, cz int) {
	for dz := -LoadRadius; dz <= LoadRadius; dz++ {
		for dx := -LoadRadius; dx <= LoadRadius; dx++ {
			coord := ChunkCoord{cx + dx, cz + dz}
			if _, ok := w.Chunks[coord]; !ok {
				w.Chunks[coord] = w.generateChunk(coord.X, coord.Z)
			}
		}
	}
}

func (w *World) UnloadFarChunks(cx, cz int) {
	for coord := range w.Chunks {
		dx := coord.X - cx
		dz := coord.Z - cz
		if dx < -UnloadRadius || dx > UnloadRadius || dz < -UnloadRadius || dz > UnloadRadius {
			delete(w.Chunks, coord)
		}
	}
}

func (w *World) MarkExplored(tx, tz int) {
	cx, cz := TileToChunk(tx, tz)
	if c, ok := w.Chunks[ChunkCoord{cx, cz}]; ok {
		c.Explored = true
	}
}

// IsWalkable — ground is walkable, walls and floor are not (floor will be walkable once inside)
func (w *World) IsWalkable(tx, tz int) bool {
	cx, cz := TileToChunk(tx, tz)
	c, ok := w.Chunks[ChunkCoord{cx, cz}]
	if !ok {
		return false
	}
	ox, oz := ChunkOrigin(cx, cz)
	lx, lz := tx-ox, tz-oz
	if lx < 0 || lx >= ChunkSize || lz < 0 || lz >= ChunkSize {
		return false
	}
	return c.Tiles[lz][lx] == TileGround
}
