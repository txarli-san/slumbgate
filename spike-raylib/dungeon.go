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

type Cell struct {
	Type CellType
	// Walls on each edge (true = wall present)
	WallN, WallE, WallS, WallW bool
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

func (d *Dungeon) generateWalls() {
	for z := 0; z < d.Height; z++ {
		for x := 0; x < d.Width; x++ {
			if d.Cells[z][x].Type != CellFloor {
				continue
			}
			// North wall: if cell to the north is void or edge
			if z == 0 || d.Cells[z-1][x].Type == CellVoid {
				d.Cells[z][x].WallN = true
			}
			// South wall
			if z == d.Height-1 || d.Cells[z+1][x].Type == CellVoid {
				d.Cells[z][x].WallS = true
			}
			// West wall
			if x == 0 || d.Cells[z][x-1].Type == CellVoid {
				d.Cells[z][x].WallW = true
			}
			// East wall
			if x == d.Width-1 || d.Cells[z][x+1].Type == CellVoid {
				d.Cells[z][x].WallE = true
			}
		}
	}
}
