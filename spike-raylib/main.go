package main

import (
	"fmt"
	"math"
	"time"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

const lightingVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;

out vec3 fragPosition;
out vec2 fragTexCoord;
out vec3 fragNormal;
out vec4 fragColor;

void main() {
    fragPosition = vec3(matModel * vec4(vertexPosition, 1.0));
    fragTexCoord = vertexTexCoord;
    fragNormal = normalize(vec3(matNormal * vec4(vertexNormal, 0.0)));
    fragColor = vertexColor;
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const lightingFS = `#version 330
in vec3 fragPosition;
in vec2 fragTexCoord;
in vec3 fragNormal;
in vec4 fragColor;

uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 lightDir;
uniform vec4 lightColor;
uniform vec4 ambientColor;
uniform vec3 viewPos;

out vec4 finalColor;

void main() {
    vec4 texColor = texture(texture0, fragTexCoord);
    vec4 baseColor = texColor * colDiffuse * fragColor;

    float diff = max(dot(fragNormal, -lightDir), 0.0);
    vec4 diffuse = diff * lightColor;

    vec3 viewDir = normalize(viewPos - fragPosition);
    vec3 reflectDir = reflect(lightDir, fragNormal);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), 16.0);
    vec4 specular = spec * 0.3 * lightColor;

    finalColor = (ambientColor + diffuse + specular) * baseColor;
    finalColor.a = baseColor.a;
}
`

func tierName(t AutoTier) string {
	switch t {
	case TierRecruit:
		return "Recruit"
	case TierSoldier:
		return "Soldier"
	case TierVeteran:
		return "Veteran"
	case TierLieutenant:
		return "Lieutenant"
	}
	return "?"
}

func tierColor(t AutoTier) rl.Color {
	switch t {
	case TierRecruit:
		return rl.Color{R: 255, G: 100, B: 100, A: 255}
	case TierSoldier:
		return rl.Color{R: 255, G: 220, B: 80, A: 255}
	case TierVeteran:
		return rl.Color{R: 80, G: 220, B: 100, A: 255}
	case TierLieutenant:
		return rl.Color{R: 100, G: 150, B: 255, A: 255}
	}
	return rl.White
}

func applyShaderToModel(model rl.Model, shader rl.Shader) {
	materials := unsafe.Slice(model.Materials, model.MaterialCount)
	for i := range materials {
		materials[i].Shader = shader
	}
}

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
	floorModel := rl.LoadModel("../assets/models/dungeon/floors/tileBrickB_large.gltf.glb")
	defer rl.UnloadModel(floorModel)
	applyShaderToModel(floorModel, shader)

	floorCrackedA := rl.LoadModel("../assets/models/dungeon/floors/tileBrickB_largeCrackedA.gltf.glb")
	defer rl.UnloadModel(floorCrackedA)
	applyShaderToModel(floorCrackedA, shader)

	floorCrackedB := rl.LoadModel("../assets/models/dungeon/floors/tileBrickB_largeCrackedB.gltf.glb")
	defer rl.UnloadModel(floorCrackedB)
	applyShaderToModel(floorCrackedB, shader)

	wallModel := rl.LoadModel("../assets/models/dungeon/walls/wall.gltf.glb")
	defer rl.UnloadModel(wallModel)
	applyShaderToModel(wallModel, shader)

	pickaxeModel := rl.LoadModel("../assets/models/weapons/axe_common.gltf.glb")
	defer rl.UnloadModel(pickaxeModel)
	applyShaderToModel(pickaxeModel, shader)

	knightModel := rl.LoadModel("../assets/models/characters/character_knight.gltf")
	defer rl.UnloadModel(knightModel)
	applyShaderToModel(knightModel, shader)

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
	// Spawn player outside the dungeon wall
	spawnX := OuterRadius + DungeonWarpAmp + 5
	world.PlacePickaxe(spawnX, 0)
	game := &GameState{PlayerX: spawnX, PlayerZ: 0, SelectedEnt: -1}
	cx, cz := TileToChunk(spawnX, 0)
	world.EnsureChunksAround(cx, cz)
	world.MarkExplored(spawnX, 0)
	world.RevealAround(spawnX, 0)

	// Spawn entities near player
	game.Entities = []*Entity{
		{Name: "Brynn", X: spawnX - 2, Z: 1, Tier: TierVeteran, RevealDist: 8, Scouted: map[[2]int]bool{}},
		{Name: "Kael", X: spawnX - 1, Z: -1, Tier: TierSoldier, RevealDist: 3, Scouted: map[[2]int]bool{}},
		{Name: "Pip", X: spawnX - 2, Z: -1, Tier: TierRecruit, RevealDist: 3, Scouted: map[[2]int]bool{}},
	}
	for _, ent := range game.Entities {
		world.RevealAround(ent.X, ent.Z)
	}

	// Camera state — local
	orbitAngle := float32(math.Pi / 4)
	orbitRadius := tileUnit * 12
	localHeight := tileUnit * 10
	camTargetX := float32(0)
	camTargetZ := float32(0)

	// Map view state
	mapScale := float32(4) // pixels per tile
	mapPanX := float32(0)  // offset in pixels from player-centered
	mapPanZ := float32(0)

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

		// Toggle camera mode with Tab (not during alert — alert has its own Tab handler)
		if game.Alert == nil && rl.IsKeyPressed(rl.KeyTab) {
			if game.Camera == CameraLocal {
				game.Camera = CameraStrategic
				mapPanX = 0
				mapPanZ = 0
			} else {
				game.Camera = CameraLocal
			}
		}

		// Camera controls
		if game.Camera == CameraLocal {
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
		} else {
			// Map: right-drag to pan, scroll to zoom scale
			if rl.IsMouseButtonDown(rl.MouseButtonRight) {
				delta := rl.GetMouseDelta()
				mapPanX += delta.X
				mapPanZ += delta.Y
			}
			wheel := rl.GetMouseWheelMove()
			if wheel != 0 {
				mapScale += wheel * 0.5
				if mapScale < 1 {
					mapScale = 1
				}
				if mapScale > 16 {
					mapScale = 16
				}
			}
		}

		// Step animation
		if game.Moving {
			game.StepProgress += dt / stepInterval
			if game.StepProgress >= 1.0 {
				game.StepProgress = 1.0
				game.Moving = false
			}
		}

		// WASD movement — screen-relative based on camera angle
		if game.Camera == CameraLocal && !game.Moving {
			// Screen-space input
			var sx, sz float64
			if rl.IsKeyDown(rl.KeyW) {
				sz -= 1
			}
			if rl.IsKeyDown(rl.KeyS) {
				sz += 1
			}
			if rl.IsKeyDown(rl.KeyA) {
				sx -= 1
			}
			if rl.IsKeyDown(rl.KeyD) {
				sx += 1
			}

			if sx != 0 || sz != 0 {
				// Transform screen direction to grid direction using camera orbit angle
				// Camera forward (into screen) = toward target from camera position
				fwd := float64(orbitAngle) + math.Pi // direction camera looks
				cosA := math.Cos(fwd)
				sinA := math.Sin(fwd)
				// Screen up (W) = camera forward on ground, screen right (D) = camera right
				gridX := sx*(-sinA) - sz*cosA
				gridZ := sx*cosA - sz*sinA

				// Snap to nearest of 8 grid directions
				angle := math.Atan2(gridZ, gridX)
				sector := int(math.Round(angle/(math.Pi/4))) % 8
				dirs := [8][2]int{
					{1, 0}, {1, 1}, {0, 1}, {-1, 1},
					{-1, 0}, {-1, -1}, {0, -1}, {1, -1},
				}
				// Normalize sector to 0-7
				if sector < 0 {
					sector += 8
				}
				dx, dz := dirs[sector][0], dirs[sector][1]

				game.FacingAngle = FacingAngleFromDir(dx, dz)
				game.FacingDX, game.FacingDZ = dx, dz
				nx, nz := game.PlayerX+dx, game.PlayerZ+dz

				canMove := world.IsWalkable(nx, nz)
				if dx != 0 && dz != 0 && canMove {
					canMove = world.IsWalkable(game.PlayerX+dx, game.PlayerZ) &&
						world.IsWalkable(game.PlayerX, game.PlayerZ+dz)
				}

				// Wall slide
				if !canMove && dx != 0 && dz != 0 {
					if world.IsWalkable(game.PlayerX+dx, game.PlayerZ) {
						dz = 0
						nx, nz = game.PlayerX+dx, game.PlayerZ
						canMove = true
						game.FacingAngle = FacingAngleFromDir(dx, 0)
						game.FacingDX, game.FacingDZ = dx, 0
					} else if world.IsWalkable(game.PlayerX, game.PlayerZ+dz) {
						dx = 0
						nx, nz = game.PlayerX, game.PlayerZ+dz
						canMove = true
						game.FacingAngle = FacingAngleFromDir(0, dz)
						game.FacingDX, game.FacingDZ = 0, dz
					}
				}

				if canMove {
					game.PrevX, game.PrevZ = game.PlayerX, game.PlayerZ
					game.PlayerX, game.PlayerZ = nx, nz
					game.StepProgress = 0
					game.Moving = true
					game.TimeTicks++

					cx, cz := TileToChunk(game.PlayerX, game.PlayerZ)
					world.EnsureChunksAround(cx, cz)
					world.UnloadFarChunks(cx, cz)
					world.MarkExplored(game.PlayerX, game.PlayerZ)
					world.RevealAround(game.PlayerX, game.PlayerZ)

					if !game.HasPickaxe && game.PlayerX == world.PickaxeX && game.PlayerZ == world.PickaxeZ {
						game.HasPickaxe = true
						game.SetMessage("Picked up pickaxe! Press E near walls to break them.")
					}
				}
			}
		}

		// E to break wall in facing direction
		if game.Camera == CameraLocal && game.HasPickaxe && rl.IsKeyPressed(rl.KeyE) {
			tx, tz := game.PlayerX+game.FacingDX, game.PlayerZ+game.FacingDZ
			if t, ok := world.GetTile(tx, tz); ok && t == TileSolid {
				world.SetTile(tx, tz, TileDoorway)
				world.RevealAround(tx, tz)
				game.SetMessage("Wall broken!")
			}
		}

		// Entity step animation — synced to tick rate for smooth movement
		for _, ent := range game.Entities {
			if ent.Moving {
				ent.StepProgress += dt / tickRate
				if ent.StepProgress >= 1.0 {
					ent.StepProgress = 1.0
					ent.Moving = false
				}
			}
		}

		// Simulation tick — runs in any view, pauses on alert
		if game.Alert == nil {
			game.TickAccum += dt
			for game.TickAccum >= tickRate {
				game.TickAccum -= tickRate
				game.TimeTicks++
				if alert := game.TickEntities(world); alert != nil {
					game.Alert = alert
					game.SetMessage(alert.Message)
					break
				}
			}
		}

		// Strategic mode: click to select entity / assign task
		if game.Camera == CameraStrategic && game.Alert == nil && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			mouseX := float32(rl.GetMouseX())
			mouseY := float32(rl.GetMouseY())
			centerX := float32(screenWidth) / 2
			centerY := float32(screenHeight) / 2
			clickTX := game.PlayerX + int(math.Floor(float64((mouseX-centerX-mapPanX)/mapScale)))
			clickTZ := game.PlayerZ + int(math.Floor(float64((mouseY-centerY-mapPanZ)/mapScale)))

			// Check if clicking on an entity
			clicked := -1
			for i, ent := range game.Entities {
				if ent.X == clickTX && ent.Z == clickTZ {
					clicked = i
					break
				}
			}

			if clicked >= 0 {
				game.SelectedEnt = clicked
				game.SetMessage(fmt.Sprintf("Selected %s (%v)", game.Entities[clicked].Name, tierName(game.Entities[clicked].Tier)))
			} else if game.SelectedEnt >= 0 && world.IsWalkable(clickTX, clickTZ) {
				// Assign move task
				ent := game.Entities[game.SelectedEnt]
				path := FindPath(world, ent.X, ent.Z, clickTX, clickTZ)
				if path != nil {
					ent.Task = &Task{Type: TaskMoveTo, TargetX: clickTX, TargetZ: clickTZ, Path: path}
					game.SetMessage(fmt.Sprintf("%s moving to (%d, %d)", ent.Name, clickTX, clickTZ))
				} else {
					game.SetMessage("No path found!")
				}
			}
		}

		// X to assign explore task to selected entity
		if game.SelectedEnt >= 0 && rl.IsKeyPressed(rl.KeyX) {
			ent := game.Entities[game.SelectedEnt]
			ent.Task = &Task{Type: TaskExplore}
			game.SetMessage(fmt.Sprintf("%s is now exploring.", ent.Name))
		}

		// Dismiss alert with Space
		if game.Alert != nil && rl.IsKeyPressed(rl.KeySpace) {
			game.Alert = nil
		}

		// Alert handshake: Tab during alert snaps 3D camera to the entity
		if game.Alert != nil && rl.IsKeyPressed(rl.KeyTab) {
			ent := game.Entities[game.Alert.EntityIdx]
			pos := gridToWorld(ent.X, ent.Z)
			camTargetX = pos.X
			camTargetZ = pos.Z
			game.Camera = CameraLocal
			game.Alert = nil
		}

		// Message timer
		if game.MessageTimer > 0 {
			game.MessageTimer -= dt
		}

		// Camera follow player
		playerWorld := gridToWorld(game.PlayerX, game.PlayerZ)
		camSmooth := float32(5.0) * dt
		if camSmooth > 1 {
			camSmooth = 1
		}
		camTargetX += (playerWorld.X - camTargetX) * camSmooth
		camTargetZ += (playerWorld.Z - camTargetZ) * camSmooth

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

		if game.Camera == CameraLocal {
			drawLocal(camera, world, game, tileUnit, floorSurfaceY, knightYOffset, charScale, wallScale,
				floorVariant, gridToWorld, worldToGrid, rayHitGround,
				knightModel, wallModel, pickaxeModel, game.Entities)
		} else {
			drawMap(world, game, mapScale, mapPanX, mapPanZ, game.Entities, game.SelectedEnt)
		}

		// HUD
		modeStr := "LOCAL"
		if game.Camera == CameraStrategic {
			modeStr = "MAP"
		}
		rl.DrawText(fmt.Sprintf("Time: %d | Chunks: %d | Mode: %s",
			game.TimeTicks, len(world.Chunks), modeStr), 10, 10, 20, rl.White)
		if game.Camera == CameraLocal {
			rl.DrawText("WASD move | E break wall | Tab map | Right-drag orbit | Scroll zoom", 10, 35, 16, rl.Gray)
		} else {
			rl.DrawText("Click entity to select | Click ground to send | X explore | Right-drag pan | Tab 3D", 10, 35, 16, rl.Gray)
		}
		if game.MessageTimer > 0 {
			rl.DrawText(game.Message, 10, 60, 20, rl.Color{R: 255, G: 220, B: 100, A: 255})
		}

		// Alert overlay
		if game.Alert != nil {
			boxW, boxH := int32(500), int32(120)
			boxX := int32(screenWidth/2) - boxW/2
			boxY := int32(screenHeight/2) - boxH/2
			rl.DrawRectangle(boxX, boxY, boxW, boxH, rl.Color{R: 40, G: 10, B: 10, A: 230})
			rl.DrawRectangleLines(boxX, boxY, boxW, boxH, rl.Color{R: 255, G: 60, B: 60, A: 255})
			rl.DrawText("ALERT", boxX+boxW/2-40, boxY+12, 28, rl.Color{R: 255, G: 60, B: 60, A: 255})
			rl.DrawText(game.Alert.Message, boxX+20, boxY+50, 18, rl.White)
			rl.DrawText("[Space] Dismiss  |  [Tab] Go to 3D view", boxX+20, boxY+80, 16, rl.Gray)
		}

		rl.DrawFPS(screenWidth-90, 10)

		rl.EndDrawing()
	}
}

func drawLocal(
	camera rl.Camera3D,
	world *World,
	game *GameState,
	tileUnit, floorSurfaceY, knightYOffset, charScale, wallScale float32,
	floorVariant func(int, int) rl.Model,
	gridToWorld func(int, int) rl.Vector3,
	worldToGrid func(float32, float32) (int, int),
	rayHitGround func(rl.Ray) (float32, float32, bool),
	knightModel, wallModel, pickaxeModel rl.Model,
	entities []*Entity,
) {
	rl.BeginMode3D(camera)

	ones := rl.Vector3{X: 1, Y: 1, Z: 1}
	wallScaleVec := rl.Vector3{X: wallScale, Y: wallScale, Z: wallScale}

	for _, chunk := range world.Chunks {
		ox, oz := ChunkOrigin(chunk.CX, chunk.CZ)
		for lz := 0; lz < ChunkSize; lz++ {
			for lx := 0; lx < ChunkSize; lx++ {
				tx, tz := ox+lx, oz+lz
				tile := chunk.Tiles[lz][lx]
				pos := gridToWorld(tx, tz)

				halfTile := tileUnit * 0.5
				revealed := chunk.Revealed[lz][lx]

				drawWall := func(wallPos rl.Vector3, rotation float32) {
					wallPos.Y = floorSurfaceY
					rl.DrawModelEx(wallModel, wallPos, rl.Vector3{Y: 1}, rotation, wallScaleVec, rl.White)
				}

				switch tile {
				case TileGround:
					fm := floorVariant(tx, tz)
					rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.Color{R: 140, G: 140, B: 130, A: 255})

				case TileSolid:
					// Walls on edges facing open space (ground, floor, doorway)
					solidWall := func(nx, nz int) bool {
						n := world.TileTypeAt(nx, nz)
						if n == TileGround || n == TileDoorway {
							return true
						}
						// Show wall toward revealed floor
						if n == TileFloor && world.IsRevealed(nx, nz) {
							return true
						}
						return false
					}
					if solidWall(tx, tz-1) {
						drawWall(rl.Vector3{X: pos.X, Z: pos.Z - halfTile}, 180)
					}
					if solidWall(tx, tz+1) {
						drawWall(rl.Vector3{X: pos.X, Z: pos.Z + halfTile}, 0)
					}
					if solidWall(tx-1, tz) {
						drawWall(rl.Vector3{X: pos.X - halfTile, Z: pos.Z}, -90)
					}
					if solidWall(tx+1, tz) {
						drawWall(rl.Vector3{X: pos.X + halfTile, Z: pos.Z}, 90)
					}

			case TileFloor:
					if !revealed {
						continue
					}
					fm := floorVariant(tx, tz)
					rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.White)

					// Walls on edges facing ground (not doorways)
					needsWall := func(nx, nz int) bool {
						n := world.TileTypeAt(nx, nz)
						return n == TileGround
					}
					if needsWall(tx, tz-1) {
						drawWall(rl.Vector3{X: pos.X, Z: pos.Z - halfTile}, 180)
					}
					if needsWall(tx, tz+1) {
						drawWall(rl.Vector3{X: pos.X, Z: pos.Z + halfTile}, 0)
					}
					if needsWall(tx-1, tz) {
						drawWall(rl.Vector3{X: pos.X - halfTile, Z: pos.Z}, -90)
					}
					if needsWall(tx+1, tz) {
						drawWall(rl.Vector3{X: pos.X + halfTile, Z: pos.Z}, 90)
					}

				case TileDoorway:
					if !revealed {
						continue
					}
					fm := floorVariant(tx, tz)
					rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.White)

				case TileCore:
					// Nothing rendered — void
				}
			}
		}
	}

	// Pickaxe on ground with glow
	if !game.HasPickaxe {
		pickPos := gridToWorld(world.PickaxeX, world.PickaxeZ)
		// Pulsing glow disc on the floor
		pulse := float32(math.Sin(float64(rl.GetTime())*3.0))*0.3 + 0.7
		glowPos := pickPos
		glowPos.Y = floorSurfaceY + 0.02
		glowSize := tileUnit * 1.2 * pulse
		rl.DrawCubeV(glowPos, rl.Vector3{X: glowSize, Y: 0.05, Z: glowSize},
			rl.Color{R: 255, G: 180, B: 40, A: uint8(60 * pulse)})
		// The pickaxe itself
		pickPos.Y = floorSurfaceY
		pickScale := rl.Vector3{X: wallScale * 3, Y: wallScale * 3, Z: wallScale * 3}
		rl.DrawModelEx(pickaxeModel, pickPos, rl.Vector3{Y: 1}, 45, pickScale, rl.Color{R: 255, G: 200, B: 80, A: 255})
	}

	// Player
	scaleVec := rl.Vector3{X: charScale, Y: charScale, Z: charScale}
	var knightPos rl.Vector3
	if game.Moving {
		from := gridToWorld(game.PrevX, game.PrevZ)
		to := gridToWorld(game.PlayerX, game.PlayerZ)
		t := game.StepProgress
		bounce := float32(math.Sin(float64(t)*math.Pi)) * 0.3
		knightPos = rl.Vector3{
			X: from.X + (to.X-from.X)*t,
			Y: knightYOffset + bounce,
			Z: from.Z + (to.Z-from.Z)*t,
		}
	} else {
		knightPos = gridToWorld(game.PlayerX, game.PlayerZ)
		knightPos.Y = knightYOffset
	}
	rl.DrawModelEx(knightModel, knightPos, rl.Vector3{Y: 1}, game.FacingAngle, scaleVec, rl.White)

	// Entities
	for _, ent := range entities {
		var entPos rl.Vector3
		if ent.Moving {
			from := gridToWorld(ent.PrevX, ent.PrevZ)
			to := gridToWorld(ent.X, ent.Z)
			t := ent.StepProgress
			bounce := float32(math.Sin(float64(t)*math.Pi)) * 0.3
			entPos = rl.Vector3{
				X: from.X + (to.X-from.X)*t,
				Y: knightYOffset + bounce,
				Z: from.Z + (to.Z-from.Z)*t,
			}
		} else {
			entPos = gridToWorld(ent.X, ent.Z)
			entPos.Y = knightYOffset
		}
		rl.DrawModelEx(knightModel, entPos, rl.Vector3{Y: 1}, ent.FacingAngle, scaleVec, tierColor(ent.Tier))
	}

	// Threats (visible as red markers)
	for _, threat := range world.Threats {
		if world.IsRevealed(threat.X, threat.Z) {
			tPos := gridToWorld(threat.X, threat.Z)
			tPos.Y = floorSurfaceY + tileUnit*0.5
			size := tileUnit * 0.3
			color := rl.Color{R: 255, G: 60, B: 60, A: 200}
			if threat.Difficulty >= 2 {
				color = rl.Color{R: 200, G: 0, B: 200, A: 200}
			}
			rl.DrawCubeV(tPos, rl.Vector3{X: size, Y: size, Z: size}, color)
		}
	}

	// Hover highlight
	ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
	if wx, wz, ok := rayHitGround(ray); ok {
		gx, gz := worldToGrid(wx, wz)
		if world.IsWalkable(gx, gz) {
			hlPos := gridToWorld(gx, gz)
			hlPos.Y = floorSurfaceY + 0.05
			rl.DrawCubeV(hlPos, rl.Vector3{X: tileUnit * 0.95, Y: 0.1, Z: tileUnit * 0.95},
				rl.Color{R: 100, G: 200, B: 255, A: 60})
		}
	}

	rl.EndMode3D()
}

func drawMap(world *World, game *GameState, scale, panX, panZ float32, entities []*Entity, selectedEnt int) {
	centerX := float32(screenWidth) / 2
	centerY := float32(screenHeight) / 2

	playerOffX := centerX + panX
	playerOffY := centerY + panZ

	// Draw per-tile for each loaded chunk
	for _, chunk := range world.Chunks {
		ox, oz := ChunkOrigin(chunk.CX, chunk.CZ)
		for lz := range ChunkSize {
			for lx := range ChunkSize {
				tx, tz := ox+lx, oz+lz
				sx := playerOffX + float32(tx-game.PlayerX)*scale
				sy := playerOffY + float32(tz-game.PlayerZ)*scale

				var color rl.Color
				switch chunk.Tiles[lz][lx] {
				case TileGround:
					color = rl.Color{R: 45, G: 50, B: 40, A: 255}
				case TileSolid:
					color = rl.Color{R: 70, G: 60, B: 50, A: 255}
				case TileFloor:
					if chunk.Revealed[lz][lx] {
						color = rl.Color{R: 60, G: 65, B: 80, A: 255}
					} else {
						color = rl.Color{R: 25, G: 25, B: 35, A: 255}
					}
				case TileDoorway:
					color = rl.Color{R: 80, G: 80, B: 90, A: 255}
				case TileCore:
					color = rl.Color{R: 15, G: 15, B: 20, A: 255}
				}

				sz := int32(math.Ceil(float64(scale)))
				rl.DrawRectangle(int32(sx), int32(sy), sz, sz, color)
			}
		}
	}

	// Threat markers on map
	for _, threat := range world.Threats {
		if world.IsRevealed(threat.X, threat.Z) {
			tx := playerOffX + float32(threat.X-game.PlayerX)*scale
			tz := playerOffY + float32(threat.Z-game.PlayerZ)*scale
			color := rl.Color{R: 255, G: 60, B: 60, A: 255}
			if threat.Difficulty >= 2 {
				color = rl.Color{R: 200, G: 0, B: 200, A: 255}
			}
			rl.DrawRectangle(int32(tx), int32(tz), int32(math.Ceil(float64(scale))), int32(math.Ceil(float64(scale))), color)
		}
	}

	// Entity markers
	for i, ent := range entities {
		ex := playerOffX + float32(ent.X-game.PlayerX)*scale
		ey := playerOffY + float32(ent.Z-game.PlayerZ)*scale
		r := scale + 2
		if i == selectedEnt {
			r = scale + 4
			rl.DrawCircle(int32(ex+scale*0.5), int32(ey+scale*0.5), r+2, rl.White)
		}
		rl.DrawCircle(int32(ex+scale*0.5), int32(ey+scale*0.5), r, tierColor(ent.Tier))
		// Draw name
		rl.DrawText(ent.Name, int32(ex+scale+4), int32(ey-4), 12, tierColor(ent.Tier))
	}

	// Player marker
	rl.DrawCircle(int32(playerOffX+scale*0.5), int32(playerOffY+scale*0.5),
		scale+2, rl.Color{R: 255, G: 220, B: 80, A: 255})
}
