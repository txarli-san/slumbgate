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

	// Skeleton models
	skelMinion := rl.LoadModel("../assets/models/enemies/Skeleton_Minion.glb")
	defer rl.UnloadModel(skelMinion)
	applyShaderToModel(skelMinion, shader)

	skelWarrior := rl.LoadModel("../assets/models/enemies/Skeleton_Warrior.glb")
	defer rl.UnloadModel(skelWarrior)
	applyShaderToModel(skelWarrior, shader)

	skelRogue := rl.LoadModel("../assets/models/enemies/Skeleton_Rogue.glb")
	defer rl.UnloadModel(skelRogue)
	applyShaderToModel(skelRogue, shader)

	skelMage := rl.LoadModel("../assets/models/enemies/Skeleton_Mage.glb")
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
		{Name: "Brynn", X: spawnX, Z: 0, Tier: TierVeteran, RevealDist: 8, Scouted: map[[2]int]bool{}},
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

		// Click in 3D view: select entity or assign move task
		if game.Alert == nil && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
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
				} else if game.SelectedEnt >= 0 && world.IsWalkable(gx, gz) {
					ent := game.Entities[game.SelectedEnt]
					path := FindPath(world, ent.X, ent.Z, gx, gz)
					if path != nil {
						ent.Task = &Task{Type: TaskMoveTo, TargetX: gx, TargetZ: gz, Path: path}
						game.SetMessage(fmt.Sprintf("%s moving to (%d, %d)", ent.Name, gx, gz))
					} else {
						game.SetMessage("No path found!")
					}
				}
			}
		}

		// Entities with tasks drive the clock — time flows when anyone is busy
		if game.Alert == nil && game.AnyEntityBusy() {
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

		// Entity step animation — same speed as player steps
		for _, ent := range game.Entities {
			if ent.Moving {
				ent.StepProgress += dt / stepInterval
				if ent.StepProgress >= 1.0 {
					ent.StepProgress = 1.0
					ent.Moving = false
				}
			}
		}

		// Orders for selected entity: 1=Scout  2=Stop
		if game.SelectedEnt >= 0 {
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

		// HUD
		rl.DrawText(fmt.Sprintf("Time: %d | Chunks: %d",
			game.TimeTicks, len(world.Chunks)), 10, 10, 20, rl.White)
		rl.DrawText("Click select/move | Tab cycle | 1 Scout | 2 Stop | Right-drag orbit | Scroll zoom", 10, 35, 16, rl.Gray)
		if game.MessageTimer > 0 {
			rl.DrawText(game.Message, 10, 60, 20, rl.Color{R: 255, G: 220, B: 100, A: 255})
		}

		// Entity panel — bottom left
		if game.SelectedEnt >= 0 && game.SelectedEnt < len(game.Entities) {
			ent := game.Entities[game.SelectedEnt]
			panelX := int32(10)
			panelY := int32(screenHeight - 130)
			panelW := int32(260)
			panelH := int32(120)
			rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.Color{R: 20, G: 20, B: 30, A: 210})
			rl.DrawRectangleLines(panelX, panelY, panelW, panelH, tierColor(ent.Tier))

			rl.DrawText(fmt.Sprintf("%s  [%s]", ent.Name, tierName(ent.Tier)), panelX+10, panelY+8, 20, tierColor(ent.Tier))
			rl.DrawText(fmt.Sprintf("Pos: (%d, %d)  Sight: %d", ent.X, ent.Z, ent.RevealDist), panelX+10, panelY+34, 14, rl.LightGray)

			taskStr := "Idle"
			if ent.Task != nil {
				switch ent.Task.Type {
				case TaskMoveTo:
					taskStr = fmt.Sprintf("Moving to (%d,%d)", ent.Task.TargetX, ent.Task.TargetZ)
				case TaskExplore:
					taskStr = "Scouting"
				}
			}
			rl.DrawText("Task: "+taskStr, panelX+10, panelY+54, 14, rl.White)

			rl.DrawText("[1] Scout  [2] Stop  [Click] Move to", panelX+10, panelY+80, 14, rl.Gray)
			rl.DrawText(fmt.Sprintf("< Tab (%d/%d) >", game.SelectedEnt+1, len(game.Entities)), panelX+10, panelY+98, 12, rl.DarkGray)
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
			rl.DrawText("[Space] Dismiss and focus", boxX+20, boxY+80, 16, rl.Gray)
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
	knightModel, wallModel, pickaxeModel rl.Model,
	skeletonModels map[SkeletonType]rl.Model,
	entities []*Entity,
	selectedEnt int,
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

	// Entities
	scaleVec := rl.Vector3{X: charScale, Y: charScale, Z: charScale}
	for i, ent := range entities {
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
		// Selection ring
		if i == selectedEnt {
			ringPos := entPos
			ringPos.Y = floorSurfaceY + 0.05
			rl.DrawCubeV(ringPos, rl.Vector3{X: tileUnit * 0.9, Y: 0.1, Z: tileUnit * 0.9},
				rl.Color{R: 255, G: 255, B: 255, A: 80})
		}
		rl.DrawModelEx(knightModel, entPos, rl.Vector3{Y: 1}, ent.FacingAngle, scaleVec, tierColor(ent.Tier))
	}

	// Threats (skeleton models)
	for _, threat := range world.Threats {
		if world.IsRevealed(threat.X, threat.Z) {
			tPos := gridToWorld(threat.X, threat.Z)
			tPos.Y = knightYOffset
			if model, ok := skeletonModels[threat.Type]; ok {
				rl.DrawModelEx(model, tPos, rl.Vector3{Y: 1}, 0, scaleVec, rl.White)
			}
		}
	}

	rl.EndMode3D()
}

