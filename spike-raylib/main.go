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

	rl.SetShaderValue(shader, locLightDir, []float32{-0.4, -0.8, -0.3}, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, locLightColor, []float32{1.0, 0.95, 0.9, 1.0}, rl.ShaderUniformVec4)
	rl.SetShaderValue(shader, locAmbientColor, []float32{0.35, 0.35, 0.4, 1.0}, rl.ShaderUniformVec4)

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

	knightModel := rl.LoadModel("../assets/models/characters/character_knight.gltf")
	defer rl.UnloadModel(knightModel)
	applyShaderToModel(knightModel, shader)

	// Measurements
	floorBBox := rl.GetModelBoundingBox(floorModel)
	tileUnit := floorBBox.Max.X - floorBBox.Min.X
	floorSurfaceY := floorBBox.Max.Y

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
	game := &GameState{}
	world.EnsureChunksAround(0, 0)
	world.MarkExplored(0, 0)

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

		// Toggle camera mode with Tab
		if rl.IsKeyPressed(rl.KeyTab) {
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
				orbitRadius -= wheel * tileUnit * 0.5
				if orbitRadius < tileUnit*4 {
					orbitRadius = tileUnit * 4
				}
				if orbitRadius > tileUnit*25 {
					orbitRadius = tileUnit * 25
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

		// Click to move (local mode only)
		if game.Camera == CameraLocal && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
			if wx, wz, ok := rayHitGround(ray); ok {
				gx, gz := worldToGrid(wx, wz)
				path := FindPath(world, game.PlayerX, game.PlayerZ, gx, gz)
				if path != nil {
					game.Path = path
				}
			}
		}

		// Step along path
		if game.Moving {
			game.StepProgress += dt / stepInterval
			if game.StepProgress >= 1.0 {
				game.StepProgress = 1.0
				game.Moving = false
			}
		}

		if !game.Moving && len(game.Path) > 0 {
			next := game.Path[0]
			game.Path = game.Path[1:]
			dx, dz := next[0]-game.PlayerX, next[1]-game.PlayerZ
			game.PrevX, game.PrevZ = game.PlayerX, game.PlayerZ
			game.PlayerX, game.PlayerZ = next[0], next[1]
			game.FacingAngle = FacingAngleFromDir(dx, dz)
			game.StepProgress = 0
			game.Moving = true
			game.TimeTicks++

			// Chunk management on move
			cx, cz := TileToChunk(game.PlayerX, game.PlayerZ)
			world.EnsureChunksAround(cx, cz)
			world.UnloadFarChunks(cx, cz)
			world.MarkExplored(game.PlayerX, game.PlayerZ)
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
			drawLocal(camera, world, game, tileUnit, floorSurfaceY, knightYOffset, charScale,
				floorVariant, gridToWorld, worldToGrid, rayHitGround,
				knightModel)
		} else {
			drawMap(world, game, mapScale, mapPanX, mapPanZ)
		}

		// HUD
		modeStr := "LOCAL"
		if game.Camera == CameraStrategic {
			modeStr = "MAP"
		}
		rl.DrawText(fmt.Sprintf("Time: %d | Chunks: %d | Mode: %s",
			game.TimeTicks, len(world.Chunks), modeStr), 10, 10, 20, rl.White)
		rl.DrawText("Click to move | Tab to toggle map | Right-drag orbit | Scroll zoom", 10, 35, 16, rl.Gray)
		rl.DrawFPS(screenWidth-90, 10)

		rl.EndDrawing()
	}
}

func drawLocal(
	camera rl.Camera3D,
	world *World,
	game *GameState,
	tileUnit, floorSurfaceY, knightYOffset, charScale float32,
	floorVariant func(int, int) rl.Model,
	gridToWorld func(int, int) rl.Vector3,
	worldToGrid func(float32, float32) (int, int),
	rayHitGround func(rl.Ray) (float32, float32, bool),
	knightModel rl.Model,
) {
	rl.BeginMode3D(camera)

	ones := rl.Vector3{X: 1, Y: 1, Z: 1}

	// Render floor tiles for loaded chunks
	for _, chunk := range world.Chunks {
		ox, oz := ChunkOrigin(chunk.CX, chunk.CZ)
		for lz := 0; lz < ChunkSize; lz++ {
			for lx := 0; lx < ChunkSize; lx++ {
				tx, tz := ox+lx, oz+lz
				pos := gridToWorld(tx, tz)
				fm := floorVariant(tx, tz)
				rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.White)
			}
		}
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

func drawMap(world *World, game *GameState, scale, panX, panZ float32) {
	centerX := float32(screenWidth) / 2
	centerY := float32(screenHeight) / 2

	chunkPx := scale * float32(ChunkSize) // pixels per chunk

	// Player tile position as pixel offset from center
	playerOffX := centerX + panX
	playerOffY := centerY + panZ

	for _, chunk := range world.Chunks {
		ox, oz := ChunkOrigin(chunk.CX, chunk.CZ)

		// Chunk rect in screen space, relative to player
		sx := playerOffX + float32(ox-game.PlayerX)*scale
		sy := playerOffY + float32(oz-game.PlayerZ)*scale

		// Chunk fill
		color := rl.Color{R: 30, G: 30, B: 40, A: 255}
		if chunk.Explored {
			color = rl.Color{R: 60, G: 75, B: 90, A: 255}
		}
		rl.DrawRectangle(int32(sx), int32(sy), int32(chunkPx), int32(chunkPx), color)

		// Chunk border
		rl.DrawRectangleLines(int32(sx), int32(sy), int32(chunkPx), int32(chunkPx),
			rl.Color{R: 50, G: 60, B: 80, A: 255})
	}

	// Player marker
	rl.DrawCircle(int32(playerOffX+scale*0.5), int32(playerOffY+scale*0.5),
		scale*0.8+2, rl.Color{R: 255, G: 220, B: 80, A: 255})
}
