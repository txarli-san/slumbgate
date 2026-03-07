package main

import (
	"fmt"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Slumbgate - Infinite Dungeon")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Shader
	shader := rl.LoadShaderFromMemory(lightingVS, lightingFS)
	defer rl.UnloadShader(shader)

	locLightDir := rl.GetShaderLocation(shader, "lightDir")
	locLightColor := rl.GetShaderLocation(shader, "lightColor")
	locAmbientColor := rl.GetShaderLocation(shader, "ambientColor")
	locViewPos := rl.GetShaderLocation(shader, "viewPos")

	rl.SetShaderValue(shader, locLightDir, []float32{-0.5, -0.7, -0.4}, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, locLightColor, []float32{1.0, 0.95, 0.85, 1.0}, rl.ShaderUniformVec4)
	rl.SetShaderValue(shader, locAmbientColor, []float32{0.55, 0.55, 0.6, 1.0}, rl.ShaderUniformVec4)

	shader.UpdateLocation(rl.ShaderLocMatrixModel, rl.GetShaderLocation(shader, "matModel"))
	shader.UpdateLocation(rl.ShaderLocMatrixNormal, rl.GetShaderLocation(shader, "matNormal"))

	// Models
	floorModel := rl.LoadModel("assets/models/dungeon/floors/tileBrickB_large.gltf.glb")
	defer rl.UnloadModel(floorModel)
	applyShaderToModel(floorModel, shader)

	floorCrackedA := rl.LoadModel("assets/models/dungeon/floors/tileBrickB_largeCrackedA.gltf.glb")
	defer rl.UnloadModel(floorCrackedA)
	applyShaderToModel(floorCrackedA, shader)

	floorCrackedB := rl.LoadModel("assets/models/dungeon/floors/tileBrickB_largeCrackedB.gltf.glb")
	defer rl.UnloadModel(floorCrackedB)
	applyShaderToModel(floorCrackedB, shader)

	wallModel := rl.LoadModel("assets/models/dungeon/walls/wall.gltf.glb")
	defer rl.UnloadModel(wallModel)
	applyShaderToModel(wallModel, shader)

	pickaxeModel := rl.LoadModel("assets/models/weapons/axe_common.gltf.glb")
	defer rl.UnloadModel(pickaxeModel)
	applyShaderToModel(pickaxeModel, shader)

	knightModel := rl.LoadModel("assets/models/characters/character_knight.gltf")
	defer rl.UnloadModel(knightModel)
	applyShaderToModel(knightModel, shader)

	// Skeleton models
	skelMinion := rl.LoadModel("assets/models/enemies/Skeleton_Minion.glb")
	defer rl.UnloadModel(skelMinion)
	applyShaderToModel(skelMinion, shader)

	skelWarrior := rl.LoadModel("assets/models/enemies/Skeleton_Warrior.glb")
	defer rl.UnloadModel(skelWarrior)
	applyShaderToModel(skelWarrior, shader)

	skelRogue := rl.LoadModel("assets/models/enemies/Skeleton_Rogue.glb")
	defer rl.UnloadModel(skelRogue)
	applyShaderToModel(skelRogue, shader)

	skelMage := rl.LoadModel("assets/models/enemies/Skeleton_Mage.glb")
	defer rl.UnloadModel(skelMage)
	applyShaderToModel(skelMage, shader)

	skeletonModels := map[SkeletonType]rl.Model{
		SkeletonMinion:  skelMinion,
		SkeletonWarrior: skelWarrior,
		SkeletonRogue:   skelRogue,
		SkeletonMage:    skelMage,
	}

	// Measurements
	floorBBox := rl.GetModelBoundingBox(floorModel)
	tileUnit := floorBBox.Max.X - floorBBox.Min.X
	floorSurfaceY := floorBBox.Max.Y

	wallBBox := rl.GetModelBoundingBox(wallModel)
	wallWidth := wallBBox.Max.X - wallBBox.Min.X
	wallScale := tileUnit / wallWidth

	charBBox := rl.GetModelBoundingBox(knightModel)
	charScale := (tileUnit * 0.6) / (charBBox.Max.X - charBBox.Min.X)
	knightYOffset := floorSurfaceY - charBBox.Min.Y*charScale

	// Floor variant by position
	floorVariant := func(x, z int) rl.Model {
		h := (x*7 + z*13 + x*z*3) % 10
		if h < 0 {
			h += 10
		}
		if h == 0 {
			return floorCrackedA
		}
		if h == 1 {
			return floorCrackedB
		}
		return floorModel
	}

	// World
	seed := time.Now().UnixNano()
	world := NewWorld(seed)
	spawnX := OuterRadius + DungeonWarpAmp + 5
	world.PlacePickaxe(spawnX, 0)
	game := &GameState{SelectedEnt: 0}

	// Spawn company
	game.Entities = []*Entity{
		{Name: "Brynn", X: spawnX, Z: 0, Tier: TierVeteran, RevealDist: 8, Scouted: map[[2]int]bool{},
			Stats: &CombatStats{
				HP: 28, MaxHP: 28, AC: 16,
				STR: 16, DEX: 12, CON: 14, INT: 10, WIS: 12, CHA: 10,
				Level: 3, ProfBonus: 2, MoveSpeed: 5, Class: "Fighter", ClassCharges: 2,
			}},
		{Name: "Kael", X: spawnX - 1, Z: 1, Tier: TierSoldier, RevealDist: 3, Scouted: map[[2]int]bool{}},
		{Name: "Pip", X: spawnX - 1, Z: -1, Tier: TierRecruit, RevealDist: 3, Scouted: map[[2]int]bool{}},
	}
	for _, ent := range game.Entities {
		cx, cz := TileToChunk(ent.X, ent.Z)
		world.EnsureChunksAround(cx, cz)
		world.RevealAround(ent.X, ent.Z)
	}

	// Camera state — local
	orbitAngle := float32(math.Pi / 4)
	orbitRadius := tileUnit * 12
	localHeight := tileUnit * 10
	camTargetX := float32(0)
	camTargetZ := float32(0)

	camera := rl.Camera3D{
		Position:   rl.Vector3{X: 0, Y: localHeight, Z: orbitRadius},
		Target:     rl.Vector3{},
		Up:         rl.Vector3{Y: 1},
		Fovy:       45,
		Projection: rl.CameraPerspective,
	}

	gridToWorld := func(gx, gz int) rl.Vector3 {
		return rl.Vector3{
			X: (float32(gx) + 0.5) * tileUnit,
			Z: (float32(gz) + 0.5) * tileUnit,
		}
	}

	worldToGrid := func(wx, wz float32) (int, int) {
		gx := int(math.Floor(float64(wx / tileUnit)))
		gz := int(math.Floor(float64(wz / tileUnit)))
		return gx, gz
	}

	rayHitGround := func(ray rl.Ray) (float32, float32, bool) {
		if ray.Direction.Y == 0 {
			return 0, 0, false
		}
		t := -ray.Position.Y / ray.Direction.Y
		if t <= 0 {
			return 0, 0, false
		}
		return ray.Position.X + ray.Direction.X*t, ray.Position.Z + ray.Direction.Z*t, true
	}

	fmt.Printf("World seed: %d | Chunk size: %d\n", seed, ChunkSize)

	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// Tab to cycle selected entity
		if game.Alert == nil && rl.IsKeyPressed(rl.KeyTab) {
			if len(game.Entities) > 0 {
				game.SelectedEnt = (game.SelectedEnt + 1) % len(game.Entities)
				ent := game.Entities[game.SelectedEnt]
				game.SetMessage(fmt.Sprintf("Selected %s (%s)", ent.Name, tierName(ent.Tier)))
			}
		}

		// Camera controls — right-drag to orbit, scroll to zoom
		if rl.IsMouseButtonDown(rl.MouseButtonRight) {
			delta := rl.GetMouseDelta()
			orbitAngle += delta.X * 0.005
			localHeight -= delta.Y * tileUnit * 0.01
			if localHeight < tileUnit*3 {
				localHeight = tileUnit * 3
			}
			if localHeight > tileUnit*30 {
				localHeight = tileUnit * 30
			}
		}
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			zoomFactor := float32(1.0) - wheel*0.1
			orbitRadius *= zoomFactor
			localHeight *= zoomFactor
			if orbitRadius < tileUnit*2 {
				orbitRadius = tileUnit * 2
			}
			if orbitRadius > tileUnit*25 {
				orbitRadius = tileUnit * 25
			}
			if localHeight < tileUnit*2 {
				localHeight = tileUnit * 2
			}
			if localHeight > tileUnit*25 {
				localHeight = tileUnit * 25
			}
		}

		// Click in 3D view: select entity or assign move task (continuous mode only)
		if game.Combat == nil && game.Alert == nil && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
			if wx, wz, ok := rayHitGround(ray); ok {
				gx, gz := worldToGrid(wx, wz)

				// Check if clicking on an entity
				clicked := -1
				for i, ent := range game.Entities {
					if ent.X == gx && ent.Z == gz {
						clicked = i
						break
					}
				}

				if clicked >= 0 {
					game.SelectedEnt = clicked
					game.SetMessage(fmt.Sprintf("Selected %s (%s)", game.Entities[clicked].Name, tierName(game.Entities[clicked].Tier)))
				} else if game.SelectedEnt >= 0 {
					ent := game.Entities[game.SelectedEnt]
					if world.IsWalkable(gx, gz) {
						path := FindPath(world, ent.X, ent.Z, gx, gz)
						if path != nil {
							ent.Task = &Task{Type: TaskMoveTo, TargetX: gx, TargetZ: gz, Path: path}
							game.SetMessage(fmt.Sprintf("%s moving to (%d, %d)", ent.Name, gx, gz))
						} else {
							game.SetMessage("No path found!")
						}
					} else if game.HasPickaxe {
						if t, ok := world.GetTile(gx, gz); ok && t == TileSolid {
							path := FindPathBreakable(world, ent.X, ent.Z, gx, gz)
							if path != nil {
								ent.Task = &Task{Type: TaskMoveTo, TargetX: gx, TargetZ: gz, Path: path}
								game.SetMessage(fmt.Sprintf("%s breaching to (%d, %d)", ent.Name, gx, gz))
							} else {
								game.SetMessage("No path found!")
							}
						}
					}
				}
			}
		}

		// Entities with tasks drive the clock (continuous mode only)
		if game.Combat == nil && game.Alert == nil && game.AnyEntityBusy() {
			game.TickAccum += dt
			for game.TickAccum >= stepInterval {
				game.TickAccum -= stepInterval
				if alert := game.WorldStep(world); alert != nil {
					game.Alert = alert
					game.SetMessage(alert.Message)
					break
				}
			}
		}

		// Entity step animation
		for _, ent := range game.Entities {
			if ent.Moving {
				ent.StepProgress += dt / stepInterval
				if ent.StepProgress >= 1.0 {
					ent.StepProgress = 1.0
					ent.Moving = false
				}
			}
		}

		// Combat input
		if game.Combat != nil {
			c := game.Combat
			cur := c.Current()

			// Enemy AI turn
			if cur.IsEnemy {
				game.RunEnemyTurn(world)
			}

			// Ally turn — action hotkeys
			if !cur.IsEnemy && c.Actions != nil {
				for _, action := range c.Actions {
					var pressed bool
					switch action.Hotkey {
					case "1": pressed = rl.IsKeyPressed(rl.KeyOne)
					case "2": pressed = rl.IsKeyPressed(rl.KeyTwo)
					case "3": pressed = rl.IsKeyPressed(rl.KeyThree)
					case "4": pressed = rl.IsKeyPressed(rl.KeyFour)
					case "5": pressed = rl.IsKeyPressed(rl.KeyFive)
					case "Space": pressed = rl.IsKeyPressed(rl.KeySpace)
					}
					if pressed {
						game.TryExecuteAction(world, action)
						break
					}
				}
			}

			// Ally turn — click: target primed action, or move
			if !cur.IsEnemy && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
				if wx, wz, ok := rayHitGround(ray); ok {
					gx, gz := worldToGrid(wx, wz)
					ent := game.Entities[cur.EntityIdx]

					if c.PrimedAction != nil {
						// Execute primed action on target
						if world.IsThreatAt(gx, gz) {
							game.ExecutePrimedOnTarget(world, gx, gz)
						} else {
							c.PrimedAction = nil
							game.SetMessage("No valid target there.")
						}
					} else if world.IsWalkable(gx, gz) && !world.IsThreatAt(gx, gz) && c.MoveLeft > 0 {
						// Move to clicked tile
						path := FindPath(world, ent.X, ent.Z, gx, gz)
						if path != nil && len(path) <= c.MoveLeft {
							c.MoveLeft -= len(path)
							last := path[len(path)-1]
							dx, dz := last[0]-ent.X, last[1]-ent.Z
							if len(path) == 1 {
								dx, dz = path[0][0]-ent.X, path[0][1]-ent.Z
							}
							ent.FacingAngle = FacingAngleFromDir(dx, dz)
							ent.PrevX, ent.PrevZ = ent.X, ent.Z
							ent.X, ent.Z = last[0], last[1]
							ent.StepProgress = 0
							ent.Moving = true
						} else if path != nil {
							game.SetMessage("Too far! Need more movement.")
						}
					}
				}
			}

			// Right click cancels primed action
			if !cur.IsEnemy && rl.IsMouseButtonPressed(rl.MouseButtonRight) && c.PrimedAction != nil {
				c.PrimedAction = nil
				game.SetMessage("Action cancelled.")
			}
		}

		// Orders for selected entity: 1=Scout  2=Stop (continuous mode only)
		if game.Combat == nil && game.SelectedEnt >= 0 {
			ent := game.Entities[game.SelectedEnt]
			if rl.IsKeyPressed(rl.KeyOne) {
				ent.Task = &Task{Type: TaskExplore}
				game.SetMessage(fmt.Sprintf("%s is now scouting.", ent.Name))
			}
			if rl.IsKeyPressed(rl.KeyTwo) {
				ent.Task = nil
				game.SetMessage(fmt.Sprintf("%s holding position.", ent.Name))
			}
		}

		// Dismiss alert with Space
		if game.Alert != nil && rl.IsKeyPressed(rl.KeySpace) {
			ent := game.Entities[game.Alert.EntityIdx]
			game.SelectedEnt = game.Alert.EntityIdx
			pos := gridToWorld(ent.X, ent.Z)
			camTargetX = pos.X
			camTargetZ = pos.Z
			game.Alert = nil
		}

		// Message timer
		if game.MessageTimer > 0 {
			game.MessageTimer -= dt
		}

		// Camera follow selected entity (or first entity as fallback)
		focusIdx := game.SelectedEnt
		if focusIdx < 0 && len(game.Entities) > 0 {
			focusIdx = 0
		}
		if focusIdx >= 0 {
			ent := game.Entities[focusIdx]
			focusWorld := gridToWorld(ent.X, ent.Z)
			camSmooth := float32(5.0) * dt
			if camSmooth > 1 {
				camSmooth = 1
			}
			camTargetX += (focusWorld.X - camTargetX) * camSmooth
			camTargetZ += (focusWorld.Z - camTargetZ) * camSmooth
		}

		camera.Target = rl.Vector3{X: camTargetX, Z: camTargetZ}
		camera.Position = rl.Vector3{
			X: camTargetX + orbitRadius*float32(math.Cos(float64(orbitAngle))),
			Y: localHeight,
			Z: camTargetZ + orbitRadius*float32(math.Sin(float64(orbitAngle))),
		}
		camera.Fovy = 45
		camera.Projection = rl.CameraPerspective

		viewPos := []float32{camera.Position.X, camera.Position.Y, camera.Position.Z}
		rl.SetShaderValue(shader, locViewPos, viewPos, rl.ShaderUniformVec3)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Color{R: 10, G: 10, B: 15, A: 255})

		drawLocal(camera, world, game, tileUnit, floorSurfaceY, knightYOffset, charScale, wallScale,
			floorVariant, gridToWorld,
			knightModel, wallModel, pickaxeModel, skeletonModels, game.Entities, game.SelectedEnt)

		rl.EndDrawing()
	}
}
