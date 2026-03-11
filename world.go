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
	RoomsPerSector = 6
	MinRoomSize    = 3
	MaxRoomSize    = 7

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

type SkeletonType int

const (
	SkeletonMinion  SkeletonType = iota // weak
	SkeletonWarrior                     // moderate
	SkeletonRogue                       // elite
	SkeletonMage                        // elite
)

type Threat struct {
	X, Z         int
	PrevX, PrevZ int
	StepProgress float32
	Moving       bool
	Type         SkeletonType
	RoomIdx      int
	HP, MaxHP    int
	AC           int
	STR, DEX     int
	MoveSpeed    int
	AttackDice   int // e.g. 6 = 1d6
	MaxRange     int // 1 = melee
	Anim         AnimState
}

func (t Threat) ThreatLevel() int {
	switch t.Type {
	case SkeletonMinion:
		return 0
	case SkeletonWarrior:
		return 1
	default:
		return 2
	}
}

func (t Threat) AttackMod() int { return (t.DEX - 10) / 2 }

type LeashingThreat struct {
	ThreatKey [2]int
	SpawnPos  [2]int
}

type World struct {
	Seed             int64
	Chunks           map[ChunkCoord]*Chunk
	Rooms            []Room
	Threats          map[[2]int]Threat
	LeashingThreats  []LeashingThreat
	PickaxeX         int
	PickaxeZ         int
}

func NewWorld(seed int64) *World {
	w := &World{
		Seed:    seed,
		Chunks:  make(map[ChunkCoord]*Chunk),
		Threats: make(map[[2]int]Threat),
	}
	w.generateRooms()
	w.placeThreats()
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

	// Phase 1: Place rooms in sectors
	for s := range NumSectors {
		angleMin := float64(s) * sectorAngle
		for range RoomsPerSector {
			for attempt := 0; attempt < 10; attempt++ {
				angle := angleMin + rng.Float64()*sectorAngle
				midRadius := (float64(InnerRadius) + float64(OuterRadius)) / 2
				radius := midRadius + (rng.Float64()-0.5)*float64(OuterRadius-InnerRadius)*0.5

				cx := int(math.Cos(angle) * radius)
				cz := int(math.Sin(angle) * radius)

				rw := MinRoomSize + rng.Intn(MaxRoomSize-MinRoomSize+1)
				rh := MinRoomSize + rng.Intn(MaxRoomSize-MinRoomSize+1)

				room := Room{X: cx - rw/2, Z: cz - rh/2, W: rw, H: rh}

				if w.roomFitsInRing(room) {
					w.Rooms = append(w.Rooms, room)
					break
				}
			}
		}
	}

	realRoomCount := len(w.Rooms)
	if realRoomCount < 2 {
		return
	}

	// Phase 2: MST — connect all rooms with shortest corridors
	roomCenter := func(i int) (int, int) {
		r := w.Rooms[i]
		return r.X + r.W/2, r.Z + r.H/2
	}
	roomDist2 := func(i, j int) int {
		ax, az := roomCenter(i)
		bx, bz := roomCenter(j)
		dx, dz := ax-bx, az-bz
		return dx*dx + dz*dz
	}

	connected := make([]bool, realRoomCount)
	connected[0] = true
	for numConnected := 1; numConnected < realRoomCount; numConnected++ {
		bestDist := math.MaxInt64
		bestFrom, bestTo := 0, 0
		for i := 0; i < realRoomCount; i++ {
			if !connected[i] {
				continue
			}
			for j := 0; j < realRoomCount; j++ {
				if connected[j] {
					continue
				}
				if d := roomDist2(i, j); d < bestDist {
					bestDist = d
					bestFrom, bestTo = i, j
				}
			}
		}
		connected[bestTo] = true
		ax, az := roomCenter(bestFrom)
		bx, bz := roomCenter(bestTo)
		w.carveCorridor(ax, az, bx, bz)
	}

	// Phase 3: Extra connections for loops (~30% of rooms)
	extras := realRoomCount * 3 / 10
	for range extras {
		i := rng.Intn(realRoomCount)
		j := rng.Intn(realRoomCount)
		if i == j {
			continue
		}
		// Only connect if reasonably close
		if roomDist2(i, j) < 25*25 {
			ax, az := roomCenter(i)
			bx, bz := roomCenter(j)
			w.carveCorridor(ax, az, bx, bz)
		}
	}

	// Phase 4: Fill empty solid areas with small rooms
	// Sample points around the ring and place tiny rooms where there's nothing nearby
	fillRoomCount := len(w.Rooms) // snapshot before adding fill rooms
	for range 200 {
		angle := rng.Float64() * 2 * math.Pi
		midRadius := (float64(InnerRadius) + float64(OuterRadius)) / 2
		radius := midRadius + (rng.Float64()-0.5)*float64(OuterRadius-InnerRadius)*0.4
		tx := int(math.Cos(angle) * radius)
		tz := int(math.Sin(angle) * radius)

		// Skip if near an existing room
		tooClose := false
		for i := 0; i < fillRoomCount; i++ {
			rx, rz := roomCenter(i)
			dx, dz := tx-rx, tz-rz
			if dx*dx+dz*dz < 8*8 {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}

		rw := 2 + rng.Intn(2) // 2-3
		rh := 2 + rng.Intn(2)
		room := Room{X: tx - rw/2, Z: tz - rh/2, W: rw, H: rh}
		if !w.roomFitsInRing(room) {
			continue
		}

		w.Rooms = append(w.Rooms, room)

		// Connect to nearest existing room
		bestDist := math.MaxInt64
		bestIdx := 0
		for i := 0; i < fillRoomCount; i++ {
			rx, rz := roomCenter(i)
			dx, dz := tx-rx, tz-rz
			d := dx*dx + dz*dz
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		bx, bz := roomCenter(bestIdx)
		w.carveCorridor(tx, tz, bx, bz)
	}
}

func (w *World) roomFitsInRing(r Room) bool {
	// Check room tiles + 1-tile border are all solid.
	// This guarantees a wall between carved rooms and the outside.
	for z := r.Z - 1; z < r.Z+r.H+1; z++ {
		for x := r.X - 1; x < r.X+r.W+1; x++ {
			t := w.BaseTileType(x, z)
			if t != TileSolid {
				return false
			}
		}
	}
	return true
}

// solidWithBuffer returns true if the tile and all 4 cardinal neighbors are TileSolid.
func (w *World) solidWithBuffer(x, z int) bool {
	for _, d := range [5][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if w.BaseTileType(x+d[0], z+d[1]) != TileSolid {
			return false
		}
	}
	return true
}

// carveCorridor records corridor tiles as rooms of width 1, only within solid ring.
// Each tile must have a solid buffer to prevent breaching the outer boundary.
func (w *World) carveCorridor(x1, z1, x2, z2 int) {
	// L-shaped: horizontal then vertical
	if x1 > x2 {
		x1, x2 = x2, x1
		z1, z2 = z2, z1
	}
	for x := x1; x <= x2; x++ {
		if w.solidWithBuffer(x, z1) {
			w.Rooms = append(w.Rooms, Room{X: x, Z: z1, W: 1, H: 1})
		}
	}
	if z1 > z2 {
		z1, z2 = z2, z1
	}
	for z := z1; z <= z2; z++ {
		if w.solidWithBuffer(x2, z) {
			w.Rooms = append(w.Rooms, Room{X: x2, Z: z, W: 1, H: 1})
		}
	}
}

func (w *World) PlacePickaxe(spawnX, spawnZ int) {
	// Place along the dungeon wall, away from spawn — requires exploration to find
	rng := rand.New(rand.NewSource(w.Seed + 999))
	bestX, bestZ := spawnX+1, spawnZ
	// Place 15-25 tiles from spawn, on ground near the dungeon wall
	for range 500 {
		tx := spawnX + rng.Intn(31) - 15
		tz := spawnZ + rng.Intn(31) - 15
		if w.BaseTileType(tx, tz) != TileGround {
			continue
		}
		dx, dz := tx-spawnX, tz-spawnZ
		dist := dx*dx + dz*dz
		if dist < 15*15 || dist > 25*25 {
			continue
		}
		bestX, bestZ = tx, tz
		break
	}

	w.PickaxeX = bestX
	w.PickaxeZ = bestZ
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

// RevealAroundDist reveals tiles within a given radius, blocked by solid walls
func (w *World) RevealAroundDist(tx, tz, radius int) {
	for dz := -radius; dz <= radius; dz++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dz*dz > radius*radius {
				continue
			}
			nx, nz := tx+dx, tz+dz
			if !w.HasLineOfSight(tx, tz, nx, nz) {
				continue
			}
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

// HasLineOfSight walks a line from (x0,z0) to (x1,z1), returns false if blocked by TileSolid
func (w *World) HasLineOfSight(x0, z0, x1, z1 int) bool {
	dx := x1 - x0
	dz := z1 - z0
	adx := dx
	if adx < 0 {
		adx = -adx
	}
	adz := dz
	if adz < 0 {
		adz = -adz
	}
	sx := 1
	if dx < 0 {
		sx = -1
	}
	sz := 1
	if dz < 0 {
		sz = -1
	}
	err := adx - adz
	cx, cz := x0, z0
	for {
		if cx == x1 && cz == z1 {
			return true
		}
		// Check intermediate tiles (not the source or destination)
		if cx != x0 || cz != z0 {
			if t := w.TileTypeAt(cx, cz); t == TileSolid {
				return false
			}
		}
		e2 := err * 2
		if e2 > -adz {
			err -= adz
			cx += sx
		}
		if e2 < adx {
			err += adx
			cz += sz
		}
	}
}

// RevealAround reveals tiles near a position, blocked by solid walls
func (w *World) RevealAround(tx, tz int) {
	w.RevealAroundDist(tx, tz, RevealRadius)
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

func (w *World) rollSkeletonType(rng *rand.Rand) SkeletonType {
	roll := rng.Float64()
	if roll < 0.70 {
		return SkeletonMinion
	}
	if roll < 0.95 {
		return SkeletonWarrior
	}
	// 5% elite — coin flip between rogue and mage
	if rng.Intn(2) == 0 {
		return SkeletonRogue
	}
	return SkeletonMage
}

func (w *World) placeThreats() {
	rng := rand.New(rand.NewSource(w.Seed + 777))
	for ri, r := range w.Rooms {
		// Skip corridor tiles (1x1 rooms)
		if r.W <= 1 || r.H <= 1 {
			continue
		}
		// ~50% of real rooms get enemies
		if rng.Float64() > 0.5 {
			continue
		}
		// Pack size from room area: 1 per ~8 tiles, minimum 1
		area := r.W * r.H
		count := area / 8
		if count < 1 {
			count = 1
		}
		// Scatter enemies on random tiles within the room
		for range count {
			tx := r.X + rng.Intn(r.W)
			tz := r.Z + rng.Intn(r.H)
			key := [2]int{tx, tz}
			if _, taken := w.Threats[key]; taken {
				continue
			}
			st := w.rollSkeletonType(rng)
			t := Threat{X: tx, Z: tz, Type: st, RoomIdx: ri}
			switch st {
			case SkeletonMinion:
				t.HP, t.MaxHP, t.AC = 8, 8, 11
				t.STR, t.DEX = 10, 12
				t.MoveSpeed, t.AttackDice, t.MaxRange = 4, 4, 1
			case SkeletonWarrior:
				t.HP, t.MaxHP, t.AC = 13, 13, 13
				t.STR, t.DEX = 10, 14
				t.MoveSpeed, t.AttackDice, t.MaxRange = 4, 6, 1
			case SkeletonRogue:
				t.HP, t.MaxHP, t.AC = 11, 11, 14
				t.STR, t.DEX = 8, 16
				t.MoveSpeed, t.AttackDice, t.MaxRange = 5, 6, 1
			case SkeletonMage:
				t.HP, t.MaxHP, t.AC = 9, 9, 12
				t.STR, t.DEX = 8, 14
				t.MoveSpeed, t.AttackDice, t.MaxRange = 4, 8, 5
			}
			w.Threats[key] = t
		}
	}
}

func (w *World) GetThreat(tx, tz int) (Threat, bool) {
	t, ok := w.Threats[[2]int{tx, tz}]
	return t, ok
}

func (w *World) RemoveThreat(tx, tz int) {
	delete(w.Threats, [2]int{tx, tz})
}

func (w *World) IsThreatAt(tx, tz int) bool {
	_, ok := w.Threats[[2]int{tx, tz}]
	return ok
}

// TickLeashingThreats moves disengaged threats one step toward their spawn each call.
func (w *World) TickLeashingThreats() {
	remaining := w.LeashingThreats[:0]
	for _, lt := range w.LeashingThreats {
		threat, ok := w.Threats[lt.ThreatKey]
		if !ok {
			continue // killed while leashing
		}
		if threat.X == lt.SpawnPos[0] && threat.Z == lt.SpawnPos[1] {
			continue // arrived, done
		}
		path := FindPath(w, threat.X, threat.Z, lt.SpawnPos[0], lt.SpawnPos[1])
		if path == nil || len(path) == 0 {
			continue // can't path, just drop
		}
		steps := threat.MoveSpeed
		if steps > len(path) {
			steps = len(path)
		}
		dest := path[steps-1]
		newKey := [2]int{dest[0], dest[1]}
		threat.PrevX, threat.PrevZ = threat.X, threat.Z
		threat.X, threat.Z = dest[0], dest[1]
		threat.StepProgress = 0
		threat.Moving = true
		delete(w.Threats, lt.ThreatKey)
		w.Threats[newKey] = threat
		lt.ThreatKey = newKey

		if threat.X == lt.SpawnPos[0] && threat.Z == lt.SpawnPos[1] {
			continue // just arrived
		}
		remaining = append(remaining, lt)
	}
	w.LeashingThreats = remaining
}
