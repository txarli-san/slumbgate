package main

import (
	"math"
	"math/rand"
)

const (
	ChunkSize    = 16
	LoadRadius   = 3
	UnloadRadius = 5

	// Dungeon shape
	OuterRadius      = 80
	InnerRadius      = 58
	DungeonWarpScale = 8.0
	DungeonWarpAmp   = 12

	// Sectors
	NumSectors     = 16
	RoomsPerSector = 3
	MinRoomSize    = 4
	MaxRoomSize    = 8

	// Fog of war
	RevealRadius = 3
)

type TileType int

const (
	TileGround  TileType = iota // outside — walkable
	TileSolid                   // ring fill — breakable wall
	TileFloor                   // carved room/corridor — walkable
	TileDoorway                 // broken wall — walkable, no walls rendered
	TileCore                    // inner area — impenetrable
)

type ChunkCoord struct {
	X, Z int
}

type Chunk struct {
	CX, CZ   int
	Tiles     [ChunkSize][ChunkSize]TileType
	Revealed  [ChunkSize][ChunkSize]bool
	Explored  bool
}

type Room struct {
	X, Z, W, H int
}

type World struct {
	Seed             int64
	Chunks           map[ChunkCoord]*Chunk
	Rooms            []Room
	PickaxeX         int
	PickaxeZ         int
}

func NewWorld(seed int64) *World {
	w := &World{
		Seed:   seed,
		Chunks: make(map[ChunkCoord]*Chunk),
	}
	w.generateRooms()
	return w
}

// Edge returns the boundary distance at a given angle for a base radius
func (w *World) edge(baseRadius float64, angle float64) float64 {
	s := float64(w.Seed % 10000)
	n := math.Sin(angle*DungeonWarpScale+s)*0.5 +
		math.Sin(angle*DungeonWarpScale*2.3+s*1.7)*0.25 +
		math.Sin(angle*DungeonWarpScale*4.1+s*0.3)*0.125
	n /= 0.875
	return baseRadius + n*float64(DungeonWarpAmp)
}

func (w *World) OuterEdge(angle float64) float64 { return w.edge(OuterRadius, angle) }
func (w *World) InnerEdge(angle float64) float64  { return w.edge(InnerRadius, angle) }

// BaseTileType returns what a tile is before room carving
func (w *World) BaseTileType(tx, tz int) TileType {
	dx := float64(tx) + 0.5
	dz := float64(tz) + 0.5
	dist := math.Sqrt(dx*dx + dz*dz)
	angle := math.Atan2(dz, dx)

	outer := w.OuterEdge(angle)
	inner := w.InnerEdge(angle)

	if dist > outer {
		return TileGround
	}
	if dist > inner {
		return TileSolid
	}
	return TileCore
}

func (w *World) generateRooms() {
	rng := rand.New(rand.NewSource(w.Seed))
	sectorAngle := 2 * math.Pi / float64(NumSectors)

	for s := range NumSectors {
		angleMin := float64(s) * sectorAngle
		angleMid := angleMin + sectorAngle*0.5

		for range RoomsPerSector {
			// Random point within the ring band in this sector
			angle := angleMin + rng.Float64()*sectorAngle
			midRadius := (float64(InnerRadius) + float64(OuterRadius)) / 2
			radius := midRadius + (rng.Float64()-0.5)*float64(OuterRadius-InnerRadius)*0.6

			cx := int(math.Cos(angle)*radius)
			cz := int(math.Sin(angle)*radius)

			rw := MinRoomSize + rng.Intn(MaxRoomSize-MinRoomSize+1)
			rh := MinRoomSize + rng.Intn(MaxRoomSize-MinRoomSize+1)

			room := Room{X: cx - rw/2, Z: cz - rh/2, W: rw, H: rh}

			// Verify room fits within the ring
			if w.roomFitsInRing(room) {
				w.Rooms = append(w.Rooms, room)
			}
		}

		// Connect rooms in this sector with corridors
		sectorStart := s * RoomsPerSector
		// Find rooms that belong to this sector (might be fewer if some didn't fit)
		var sectorRooms []int
		for i := range w.Rooms {
			rx := float64(w.Rooms[i].X+w.Rooms[i].W/2) + 0.5
			rz := float64(w.Rooms[i].Z+w.Rooms[i].H/2) + 0.5
			ra := math.Atan2(rz, rx)
			if ra < 0 {
				ra += 2 * math.Pi
			}
			if ra >= angleMin && ra < angleMin+sectorAngle {
				sectorRooms = append(sectorRooms, i)
			}
		}
		_ = sectorStart
		_ = angleMid

		// Connect sequential rooms in this sector
		for j := 1; j < len(sectorRooms); j++ {
			a := w.Rooms[sectorRooms[j-1]]
			b := w.Rooms[sectorRooms[j]]
			w.carveCorridor(a.X+a.W/2, a.Z+a.H/2, b.X+b.W/2, b.Z+b.H/2)
		}
	}
}

func (w *World) roomFitsInRing(r Room) bool {
	for z := r.Z; z < r.Z+r.H; z++ {
		for x := r.X; x < r.X+r.W; x++ {
			t := w.BaseTileType(x, z)
			if t != TileSolid {
				return false
			}
		}
	}
	return true
}

// carveCorridor records corridor tiles as rooms of width 1
func (w *World) carveCorridor(x1, z1, x2, z2 int) {
	// L-shaped: horizontal then vertical
	if x1 > x2 {
		x1, x2 = x2, x1
		z1, z2 = z2, z1
	}
	for x := x1; x <= x2; x++ {
		w.Rooms = append(w.Rooms, Room{X: x, Z: z1, W: 1, H: 1})
	}
	if z1 > z2 {
		z1, z2 = z2, z1
	}
	for z := z1; z <= z2; z++ {
		w.Rooms = append(w.Rooms, Room{X: x2, Z: z, W: 1, H: 1})
	}
}

func (w *World) PlacePickaxe(spawnX, spawnZ int) {
	// Place near player spawn
	rng := rand.New(rand.NewSource(w.Seed + 999))
	for range 200 {
		tx := spawnX + rng.Intn(7) - 3
		tz := spawnZ + rng.Intn(7) - 3
		if w.BaseTileType(tx, tz) == TileGround && (tx != spawnX || tz != spawnZ) {
			w.PickaxeX = tx
			w.PickaxeZ = tz
			return
		}
	}
	// Fallback: right next to spawn
	w.PickaxeX = spawnX + 1
	w.PickaxeZ = spawnZ
}

func (w *World) isRoom(tx, tz int) bool {
	for _, r := range w.Rooms {
		if tx >= r.X && tx < r.X+r.W && tz >= r.Z && tz < r.Z+r.H {
			return true
		}
	}
	return false
}

func TileToChunk(tx, tz int) (int, int) {
	// Floor division by ChunkSize (16)
	cx := tx >> 4
	cz := tz >> 4
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
			tx, tz := ox+lx, oz+lz
			base := w.BaseTileType(tx, tz)
			if base == TileSolid && w.isRoom(tx, tz) {
				c.Tiles[lz][lx] = TileFloor
			} else {
				c.Tiles[lz][lx] = base
			}
			// Ground is always visible
			if base == TileGround {
				c.Revealed[lz][lx] = true
			}
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

// RevealAround reveals tiles near the player
func (w *World) RevealAround(tx, tz int) {
	for dz := -RevealRadius; dz <= RevealRadius; dz++ {
		for dx := -RevealRadius; dx <= RevealRadius; dx++ {
			nx, nz := tx+dx, tz+dz
			cx, cz := TileToChunk(nx, nz)
			c, ok := w.Chunks[ChunkCoord{cx, cz}]
			if !ok {
				continue
			}
			ox, oz := ChunkOrigin(cx, cz)
			lx, lz := nx-ox, nz-oz
			if lx >= 0 && lx < ChunkSize && lz >= 0 && lz < ChunkSize {
				c.Revealed[lz][lx] = true
			}
		}
	}
}

// GetTile returns tile type from loaded chunk
func (w *World) GetTile(tx, tz int) (TileType, bool) {
	cx, cz := TileToChunk(tx, tz)
	c, ok := w.Chunks[ChunkCoord{cx, cz}]
	if !ok {
		return TileGround, false
	}
	ox, oz := ChunkOrigin(cx, cz)
	lx, lz := tx-ox, tz-oz
	if lx < 0 || lx >= ChunkSize || lz < 0 || lz >= ChunkSize {
		return TileGround, false
	}
	return c.Tiles[lz][lx], true
}

// SetTile modifies a tile in a loaded chunk
func (w *World) SetTile(tx, tz int, t TileType) {
	cx, cz := TileToChunk(tx, tz)
	c, ok := w.Chunks[ChunkCoord{cx, cz}]
	if !ok {
		return
	}
	ox, oz := ChunkOrigin(cx, cz)
	lx, lz := tx-ox, tz-oz
	if lx >= 0 && lx < ChunkSize && lz >= 0 && lz < ChunkSize {
		c.Tiles[lz][lx] = t
	}
}

// IsRevealed checks if a tile has been revealed
func (w *World) IsRevealed(tx, tz int) bool {
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
	return c.Revealed[lz][lx]
}

// TileTypeAt returns tile type, checking loaded chunks first, falling back to base
func (w *World) TileTypeAt(tx, tz int) TileType {
	if t, ok := w.GetTile(tx, tz); ok {
		return t
	}
	return w.BaseTileType(tx, tz)
}

func (w *World) IsWalkable(tx, tz int) bool {
	t, ok := w.GetTile(tx, tz)
	if !ok {
		return false
	}
	return t == TileGround || t == TileFloor || t == TileDoorway
}
