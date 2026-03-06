package main

import "math/rand"

const (
	gridWidth  = 20
	gridHeight = 20

	minRoomSize = 3
	maxRoomSize = 7
	maxRooms    = 12
)

type CellType int

const (
	CellGround CellType = iota // open ground (walkable, outside)
	CellFloor                  // dungeon interior floor
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
	Rotation float32
}

type WallDecor int

const (
	WallDecorNone WallDecor = iota
	WallDecorTorch
	WallDecorDecoA
	WallDecorDecoB
)

type Cell struct {
	Type CellType
	// Walls on each edge
	WallN, WallE, WallS, WallW bool
	// Wall decorations
	WallDecorN, WallDecorE, WallDecorS, WallDecorW WallDecor
	// Props on this tile
	Props []Prop
	// Fog of war — interior only revealed when explored
	Revealed bool
}

type Room struct {
	X, Y, W, H int
}

type Dungeon struct {
	Width, Height int
	Cells         [][]Cell
	Rooms         []Room
	// Breakable wall: one outer wall that can be destroyed with pickaxe
	BreakableX, BreakableZ int
	BreakableDir           int // 0=N, 1=E, 2=S, 3=W
	// Pickaxe location on ground
	PickaxeX, PickaxeZ int
}

func NewDungeon() *Dungeon {
	d := &Dungeon{
		Width:  gridWidth,
		Height: gridHeight,
		Cells:  make([][]Cell, gridHeight),
	}
	for z := range d.Cells {
		d.Cells[z] = make([]Cell, gridWidth)
		for x := range d.Cells[z] {
			d.Cells[z][x].Type = CellGround
			d.Cells[z][x].Revealed = true // ground always visible
		}
	}
	return d
}

func GenerateDungeon(seed int64) *Dungeon {
	rng := rand.New(rand.NewSource(seed))
	d := NewDungeon()

	// Place rooms
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

	// Connect rooms with corridors
	for i := 1; i < len(d.Rooms); i++ {
		cx1, cy1 := d.Rooms[i-1].Center()
		cx2, cy2 := d.Rooms[i].Center()
		if rng.Intn(2) == 0 {
			d.carveCorridorH(cx1, cx2, cy1)
			d.carveCorridorV(cy1, cy2, cx2)
		} else {
			d.carveCorridorV(cy1, cy2, cx1)
			d.carveCorridorH(cx1, cx2, cy2)
		}
	}

	// Walls between floor and ground
	d.generateWalls()

	// Props in rooms
	d.placeProps(rng)

	// Wall decor
	d.placeWallDecor(rng)

	// Pick one outer wall as the breakable entry point
	d.placeBreakableWall(rng)

	// Place pickaxe on a random ground cell
	d.placePickaxe(rng)

	return d
}

func (r Room) Center() (int, int) {
	return r.X + r.W/2, r.Y + r.H/2
}

func (d *Dungeon) roomOverlaps(r Room) bool {
	for _, existing := range d.Rooms {
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
			d.Cells[z][x].Revealed = false // interior starts hidden
		}
	}
}

func (d *Dungeon) carveCorridorH(x1, x2, z int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		d.Cells[z][x].Type = CellFloor
		d.Cells[z][x].Revealed = false
	}
}

func (d *Dungeon) carveCorridorV(z1, z2, x int) {
	if z1 > z2 {
		z1, z2 = z2, z1
	}
	for z := z1; z <= z2; z++ {
		d.Cells[z][x].Type = CellFloor
		d.Cells[z][x].Revealed = false
	}
}

func (d *Dungeon) generateWalls() {
	for z := 0; z < d.Height; z++ {
		for x := 0; x < d.Width; x++ {
			if d.Cells[z][x].Type != CellFloor {
				continue
			}
			// Wall where floor meets ground (or grid edge)
			if z == 0 || d.Cells[z-1][x].Type == CellGround {
				d.Cells[z][x].WallN = true
			}
			if z == d.Height-1 || d.Cells[z+1][x].Type == CellGround {
				d.Cells[z][x].WallS = true
			}
			if x == 0 || d.Cells[z][x-1].Type == CellGround {
				d.Cells[z][x].WallW = true
			}
			if x == d.Width-1 || d.Cells[z][x+1].Type == CellGround {
				d.Cells[z][x].WallE = true
			}
		}
	}
}

// RevealAround reveals nearby cells, but only reveals interior (floor) cells
// when the player is inside the dungeon (standing on floor)
func (d *Dungeon) RevealAround(cx, cz int) {
	onFloor := d.Cells[cz][cx].Type == CellFloor
	for dz := -2; dz <= 2; dz++ {
		for dx := -2; dx <= 2; dx++ {
			x, z := cx+dx, cz+dz
			if x >= 0 && x < d.Width && z >= 0 && z < d.Height {
				c := &d.Cells[z][x]
				if c.Type == CellFloor && !onFloor {
					continue // don't reveal interior from outside
				}
				c.Revealed = true
			}
		}
	}
}

func (d *Dungeon) isAgainstWall(x, z int) bool {
	c := d.Cells[z][x]
	return c.WallN || c.WallS || c.WallE || c.WallW
}

func (d *Dungeon) placeProps(rng *rand.Rand) {
	for _, room := range d.Rooms {
		area := room.W * room.H
		propCount := area/10 + 1

		if rng.Float32() < 0.3 {
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
				Type: chestType, Rotation: rotation,
			})
		}

		if rng.Float32() < 0.15 {
			sx := room.X + rng.Intn(room.W)
			sz := room.Y + rng.Intn(room.H)
			if len(d.Cells[sz][sx].Props) == 0 {
				d.Cells[sz][sx].Props = append(d.Cells[sz][sx].Props, Prop{Type: PropSpikes})
			}
		}

		for i := 0; i < propCount; i++ {
			px := room.X + rng.Intn(room.W)
			pz := room.Y + rng.Intn(room.H)
			if len(d.Cells[pz][px].Props) > 0 {
				continue
			}
			rotation := float32(rng.Intn(4)) * 90

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
				scatterProps := []PropType{PropBarrel, PropCrate, PropPots, PropBucket, PropStool}
				prop := scatterProps[rng.Intn(len(scatterProps))]
				d.Cells[pz][px].Props = append(d.Cells[pz][px].Props, Prop{Type: prop, Rotation: rotation})
			}
		}

		if area >= 25 && rng.Float32() < 0.5 {
			tx := room.X + 1 + rng.Intn(room.W-2)
			tz := room.Y + 1 + rng.Intn(room.H-2)
			if len(d.Cells[tz][tx].Props) == 0 {
				d.Cells[tz][tx].Props = append(d.Cells[tz][tx].Props, Prop{
					Type: PropTableLarge, Rotation: float32(rng.Intn(2)) * 90,
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

func (d *Dungeon) placeBreakableWall(rng *rand.Rand) {
	// Collect all outer wall segments (walls that face ground)
	type wallSeg struct {
		x, z, dir int // dir: 0=N, 1=E, 2=S, 3=W
	}
	var candidates []wallSeg

	for z := 0; z < d.Height; z++ {
		for x := 0; x < d.Width; x++ {
			c := d.Cells[z][x]
			if c.Type != CellFloor {
				continue
			}
			if c.WallN {
				candidates = append(candidates, wallSeg{x, z, 0})
			}
			if c.WallE {
				candidates = append(candidates, wallSeg{x, z, 1})
			}
			if c.WallS {
				candidates = append(candidates, wallSeg{x, z, 2})
			}
			if c.WallW {
				candidates = append(candidates, wallSeg{x, z, 3})
			}
		}
	}

	if len(candidates) > 0 {
		pick := candidates[rng.Intn(len(candidates))]
		d.BreakableX = pick.x
		d.BreakableZ = pick.z
		d.BreakableDir = pick.dir
	}
}

func (d *Dungeon) placePickaxe(rng *rand.Rand) {
	// Place on a random ground cell
	for range 100 {
		x := rng.Intn(d.Width)
		z := rng.Intn(d.Height)
		if d.Cells[z][x].Type == CellGround {
			d.PickaxeX = x
			d.PickaxeZ = z
			return
		}
	}
}

// BreakWall removes the breakable wall segment, allowing passage
func (d *Dungeon) BreakWall() {
	c := &d.Cells[d.BreakableZ][d.BreakableX]
	switch d.BreakableDir {
	case 0:
		c.WallN = false
	case 1:
		c.WallE = false
	case 2:
		c.WallS = false
	case 3:
		c.WallW = false
	}
}

func (d *Dungeon) placeWallDecor(rng *rand.Rand) {
	decorTypes := []WallDecor{WallDecorTorch, WallDecorDecoA, WallDecorDecoB}

	for z := 0; z < d.Height; z++ {
		for x := 0; x < d.Width; x++ {
			c := &d.Cells[z][x]
			if c.Type != CellFloor {
				continue
			}
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
