package main

type CameraMode int

const (
	CameraLocal    CameraMode = iota // 3D tactical view
	CameraStrategic                  // 2D top-down map
)

type GameState struct {
	PlayerX, PlayerZ int
	PrevX, PrevZ     int
	StepProgress     float32
	Moving           bool
	FacingAngle      float32
	TimeTicks        int
	Path             [][2]int
	Camera           CameraMode
}

const stepInterval = 0.15

func FacingAngleFromDir(dx, dz int) float32 {
	switch {
	case dz == -1:
		return 180
	case dz == 1:
		return 0
	case dx == -1:
		return 90
	case dx == 1:
		return -90
	}
	return 0
}

// Simple pathfinding for flat terrain — just walk straight (no obstacles yet)
func FindPath(w *World, sx, sz, gx, gz int) [][2]int {
	if !w.IsWalkable(gx, gz) {
		return nil
	}

	var path [][2]int
	x, z := sx, sz

	for x != gx || z != gz {
		if x < gx {
			x++
		} else if x > gx {
			x--
		}
		if z < gz {
			z++
		} else if z > gz {
			z--
		}
		path = append(path, [2]int{x, z})
	}
	return path
}
