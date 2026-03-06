package main

const (
	ChunkSize    = 16 // tiles per chunk side
	LoadRadius   = 3  // chunks around player to keep loaded
	UnloadRadius = 5  // chunks beyond this get unloaded
)

type ChunkCoord struct {
	X, Z int
}

type Chunk struct {
	CX, CZ   int
	Explored bool // player has been here
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

// TileToChunk converts a tile coordinate to its chunk coordinate
func TileToChunk(tx, tz int) (int, int) {
	cx := tx >> 4 // divide by 16, works for negatives too
	cz := tz >> 4
	if tx < 0 && tx&0xF != 0 {
		cx--
	}
	if tz < 0 && tz&0xF != 0 {
		cz--
	}
	return cx, cz
}

// ChunkOrigin returns the tile coordinate of a chunk's (0,0) corner
func ChunkOrigin(cx, cz int) (int, int) {
	return cx * ChunkSize, cz * ChunkSize
}

// EnsureChunksAround loads chunks in radius around the given chunk coord
func (w *World) EnsureChunksAround(cx, cz int) {
	for dz := -LoadRadius; dz <= LoadRadius; dz++ {
		for dx := -LoadRadius; dx <= LoadRadius; dx++ {
			coord := ChunkCoord{cx + dx, cz + dz}
			if _, ok := w.Chunks[coord]; !ok {
				w.Chunks[coord] = &Chunk{CX: coord.X, CZ: coord.Z}
			}
		}
	}
}

// UnloadFarChunks removes chunks too far from the player
func (w *World) UnloadFarChunks(cx, cz int) {
	for coord := range w.Chunks {
		dx := coord.X - cx
		dz := coord.Z - cz
		if dx < -UnloadRadius || dx > UnloadRadius || dz < -UnloadRadius || dz > UnloadRadius {
			delete(w.Chunks, coord)
		}
	}
}

// MarkExplored marks the chunk containing the tile as explored
func (w *World) MarkExplored(tx, tz int) {
	cx, cz := TileToChunk(tx, tz)
	coord := ChunkCoord{cx, cz}
	if c, ok := w.Chunks[coord]; ok {
		c.Explored = true
	}
}

// IsWalkable returns true if the tile exists in a loaded chunk
func (w *World) IsWalkable(tx, tz int) bool {
	cx, cz := TileToChunk(tx, tz)
	_, ok := w.Chunks[ChunkCoord{cx, cz}]
	return ok
}
