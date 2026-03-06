package main

import "math/rand"

const (
	dungeonWidth  = 20
	dungeonHeight = 20

	minRoomSize = 3
	maxRoomSize = 7
	maxRooms    = 12
)

type CellType int

const (
	CellVoid  CellType = iota // nothing — black/empty
	CellFloor                 // walkable floor
)

type PropType int

const (
	PropNone PropType = iota
	PropBarrel
	PropCrate
	PropTableSmall
	PropTableLarge
	PropChair
	PropStool
	PropBookcase
	PropBookcaseFilled
	PropPots
	PropBucket
	PropWeaponRack
	PropChestCommon
	PropChestRare
	PropTorch
	PropBanner
	PropSpikes
)

type Prop struct {
	Type     PropType
	Rotation float32 // Y-axis rotation in degrees
}

type WallDecor int

const (
	WallDecorNone WallDecor = iota
	WallDecorTorch
	WallDecorBanner
	WallDecorDecoA
	WallDecorDecoB
)

type Cell struct {
	Type CellType
	// Walls on each edge (true = wall present)
	WallN, WallE, WallS, WallW bool
	// Wall decorations per side
	WallDecorN, WallDecorE, WallDecorS, WallDecorW WallDecor
	// Props on this tile
	Props []Prop
}

type Room struct {
	X, Y, W, H int
}

type Dungeon struct {
	Width, Height int
	Cells         [][]Cell
	Rooms         []Room
}

func NewDungeon() *Dungeon {
	d := &Dungeon{
		Width:  dungeonWidth,
		Height: dungeonHeight,
		Cells:  make([][]Cell, dungeonHeight),
	}
	for z := range d.Cells {
		d.Cells[z] = make([]Cell, dungeonWidth)
	}
	return d
}

func GenerateDungeon(seed int64) *Dungeon {
	rng := rand.New(rand.NewSource(seed))
	d := NewDungeon()

	// Phase 1: Place rooms
	for i := 0; i < maxRooms*3; i++ {
		if len(d.Rooms) >= maxRooms {
			break
		}
		w := minRoomSize + rng.Intn(maxRoomSize-minRoomSize+1)
		h := minRoomSize + rng.Intn(maxRoomSize-minRoomSize+1)
		x := 1 + rng.Intn(d.Width-w-2)
		y := 1 + rng.Intn(d.Height-h-2)

		room := Room{X: x, Y: y, W: w, H: h}
		if d.roomOverlaps(room) {
			continue
		}
		d.carveRoom(room)
		d.Rooms = append(d.Rooms, room)
	}

	// Phase 2: Connect rooms with corridors
	for i := 1; i < len(d.Rooms); i++ {
		cx1, cy1 := d.Rooms[i-1].Center()
		cx2, cy2 := d.Rooms[i].Center()

		// L-shaped corridor: horizontal then vertical (or vice versa)
		if rng.Intn(2) == 0 {
			d.carveCorridorH(cx1, cx2, cy1)
			d.carveCorridorV(cy1, cy2, cx2)
		} else {
			d.carveCorridorV(cy1, cy2, cx1)
			d.carveCorridorH(cx1, cx2, cy2)
		}
	}

	// Phase 3: Add walls around all floor cells
	d.generateWalls()

	// Phase 4: Scatter props in rooms
	d.placeProps(rng)

	// Phase 5: Wall decorations (torches, banners)
	d.placeWallDecor(rng)

	return d
}

func (r Room) Center() (int, int) {
	return r.X + r.W/2, r.Y + r.H/2
}

func (d *Dungeon) roomOverlaps(r Room) bool {
	for _, existing := range d.Rooms {
		// 1-cell padding between rooms
		if r.X-1 < existing.X+existing.W+1 &&
			r.X+r.W+1 > existing.X-1 &&
			r.Y-1 < existing.Y+existing.H+1 &&
			r.Y+r.H+1 > existing.Y-1 {
			return true
		}
	}
	return false
}

func (d *Dungeon) carveRoom(r Room) {
	for z := r.Y; z < r.Y+r.H; z++ {
		for x := r.X; x < r.X+r.W; x++ {
			d.Cells[z][x].Type = CellFloor
		}
	}
}

func (d *Dungeon) carveCorridorH(x1, x2, z int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		d.Cells[z][x].Type = CellFloor
	}
}

func (d *Dungeon) carveCorridorV(z1, z2, x int) {
	if z1 > z2 {
		z1, z2 = z2, z1
	}
	for z := z1; z <= z2; z++ {
		d.Cells[z][x].Type = CellFloor
	}
}

// isInRoom returns true if (x,z) is inside any room (not a corridor)
func (d *Dungeon) isInRoom(x, z int) *Room {
	for i := range d.Rooms {
		r := &d.Rooms[i]
		if x >= r.X && x < r.X+r.W && z >= r.Y && z < r.Y+r.H {
			return r
		}
	}
	return nil
}

// isAgainstWall returns true if cell has at least one wall
func (d *Dungeon) isAgainstWall(x, z int) bool {
	c := d.Cells[z][x]
	return c.WallN || c.WallS || c.WallE || c.WallW
}

func (d *Dungeon) generateWalls() {
	for z := 0; z < d.Height; z++ {
		for x := 0; x < d.Width; x++ {
			if d.Cells[z][x].Type != CellFloor {
				continue
			}
			if z == 0 || d.Cells[z-1][x].Type == CellVoid {
				d.Cells[z][x].WallN = true
			}
			if z == d.Height-1 || d.Cells[z+1][x].Type == CellVoid {
				d.Cells[z][x].WallS = true
			}
			if x == 0 || d.Cells[z][x-1].Type == CellVoid {
				d.Cells[z][x].WallW = true
			}
			if x == d.Width-1 || d.Cells[z][x+1].Type == CellVoid {
				d.Cells[z][x].WallE = true
			}
		}
	}
}

func (d *Dungeon) placeProps(rng *rand.Rand) {
	for _, room := range d.Rooms {
		area := room.W * room.H

		// Conservative prop count: 1 per ~10 tiles, +1
		propCount := area/10 + 1

		// Place a chest in ~30% of rooms, against a wall
		if rng.Float32() < 0.3 {
			// Try to find a wall-adjacent spot
			cx, cz := d.randomWallSpot(room, rng)
			chestType := PropChestCommon
			if rng.Float32() < 0.15 {
				chestType = PropChestRare
			}
			c := d.Cells[cz][cx]
			rotation := float32(0)
			if c.WallN {
				rotation = 180
			} else if c.WallS {
				rotation = 0
			} else if c.WallW {
				rotation = -90
			} else if c.WallE {
				rotation = 90
			}
			d.Cells[cz][cx].Props = append(d.Cells[cz][cx].Props, Prop{
				Type:     chestType,
				Rotation: rotation,
			})
		}

		// Spike traps in ~15% of rooms, just 1 tile
		if rng.Float32() < 0.15 {
			sx := room.X + rng.Intn(room.W)
			sz := room.Y + rng.Intn(room.H)
			if len(d.Cells[sz][sx].Props) == 0 {
				d.Cells[sz][sx].Props = append(d.Cells[sz][sx].Props, Prop{Type: PropSpikes})
			}
		}

		// Scatter a few props
		for i := 0; i < propCount; i++ {
			px := room.X + rng.Intn(room.W)
			pz := room.Y + rng.Intn(room.H)
			if len(d.Cells[pz][px].Props) > 0 {
				continue
			}

			rotation := float32(rng.Intn(4)) * 90

			// Wall-adjacent: bookcases or weapon racks (less frequent)
			if d.isAgainstWall(px, pz) && rng.Float32() < 0.35 {
				wallProps := []PropType{PropBookcaseFilled, PropWeaponRack}
				prop := wallProps[rng.Intn(len(wallProps))]
				c := d.Cells[pz][px]
				if c.WallN {
					rotation = 180
				} else if c.WallS {
					rotation = 0
				} else if c.WallW {
					rotation = -90
				} else if c.WallE {
					rotation = 90
				}
				d.Cells[pz][px].Props = append(d.Cells[pz][px].Props, Prop{Type: prop, Rotation: rotation})
			} else {
				scatterProps := []PropType{
					PropBarrel, PropCrate, PropPots, PropBucket, PropStool,
				}
				prop := scatterProps[rng.Intn(len(scatterProps))]
				d.Cells[pz][px].Props = append(d.Cells[pz][px].Props, Prop{Type: prop, Rotation: rotation})
			}
		}

		// Big rooms get a table
		if area >= 25 && rng.Float32() < 0.5 {
			tx := room.X + 1 + rng.Intn(room.W-2)
			tz := room.Y + 1 + rng.Intn(room.H-2)
			if len(d.Cells[tz][tx].Props) == 0 {
				d.Cells[tz][tx].Props = append(d.Cells[tz][tx].Props, Prop{
					Type:     PropTableLarge,
					Rotation: float32(rng.Intn(2)) * 90,
				})
			}
		}
	}
}

func (d *Dungeon) randomWallSpot(room Room, rng *rand.Rand) (int, int) {
	for range 20 {
		x := room.X + rng.Intn(room.W)
		z := room.Y + rng.Intn(room.H)
		if d.isAgainstWall(x, z) && len(d.Cells[z][x].Props) == 0 {
			return x, z
		}
	}
	return room.Center()
}

func (d *Dungeon) placeWallDecor(rng *rand.Rand) {
	decorTypes := []WallDecor{WallDecorTorch, WallDecorDecoA, WallDecorDecoB}

	for z := 0; z < d.Height; z++ {
		for x := 0; x < d.Width; x++ {
			c := &d.Cells[z][x]
			if c.Type != CellFloor {
				continue
			}
			// ~15% chance per wall segment — sparse, not every wall
			if c.WallN && rng.Float32() < 0.15 {
				c.WallDecorN = decorTypes[rng.Intn(len(decorTypes))]
			}
			if c.WallS && rng.Float32() < 0.15 {
				c.WallDecorS = decorTypes[rng.Intn(len(decorTypes))]
			}
			if c.WallW && rng.Float32() < 0.15 {
				c.WallDecorW = decorTypes[rng.Intn(len(decorTypes))]
			}
			if c.WallE && rng.Float32() < 0.15 {
				c.WallDecorE = decorTypes[rng.Intn(len(decorTypes))]
			}
		}
	}
}
