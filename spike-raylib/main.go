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

// Basic lighting vertex shader (GLSL 330)
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

// Basic lighting fragment shader with one directional light + ambient
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

    // Diffuse
    float diff = max(dot(fragNormal, -lightDir), 0.0);
    vec4 diffuse = diff * lightColor;

    // Specular (subtle)
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
	rl.InitWindow(screenWidth, screenHeight, "Slumbgate - Dungeon Generation")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Load lighting shader
	shader := rl.LoadShaderFromMemory(lightingVS, lightingFS)
	defer rl.UnloadShader(shader)

	locLightDir := rl.GetShaderLocation(shader, "lightDir")
	locLightColor := rl.GetShaderLocation(shader, "lightColor")
	locAmbientColor := rl.GetShaderLocation(shader, "ambientColor")
	locViewPos := rl.GetShaderLocation(shader, "viewPos")

	lightDir := []float32{-0.4, -0.8, -0.3}
	lightColor := []float32{1.0, 0.95, 0.9, 1.0}
	ambientColor := []float32{0.35, 0.35, 0.4, 1.0}

	rl.SetShaderValue(shader, locLightDir, lightDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, locLightColor, lightColor, rl.ShaderUniformVec4)
	rl.SetShaderValue(shader, locAmbientColor, ambientColor, rl.ShaderUniformVec4)

	shader.UpdateLocation(rl.ShaderLocMatrixModel, rl.GetShaderLocation(shader, "matModel"))
	shader.UpdateLocation(rl.ShaderLocMatrixNormal, rl.GetShaderLocation(shader, "matNormal"))

	// Load models
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

	wallCornerModel := rl.LoadModel("../assets/models/dungeon/walls/wallCorner.gltf.glb")
	defer rl.UnloadModel(wallCornerModel)
	applyShaderToModel(wallCornerModel, shader)

	knightModel := rl.LoadModel("../assets/models/characters/character_knight.gltf")
	defer rl.UnloadModel(knightModel)
	applyShaderToModel(knightModel, shader)

	// Measure tile for grid unit
	floorBBox := rl.GetModelBoundingBox(floorModel)
	tileUnit := floorBBox.Max.X - floorBBox.Min.X
	floorSurfaceY := floorBBox.Max.Y
	fmt.Printf("Tile unit: %.2f, floor surface Y: %.2f\n", tileUnit, floorSurfaceY)

	// Measure wall
	wallBBox := rl.GetModelBoundingBox(wallModel)
	fmt.Printf("Wall bbox: min(%.2f, %.2f, %.2f) max(%.2f, %.2f, %.2f)\n",
		wallBBox.Min.X, wallBBox.Min.Y, wallBBox.Min.Z,
		wallBBox.Max.X, wallBBox.Max.Y, wallBBox.Max.Z)
	wallWidth := wallBBox.Max.X - wallBBox.Min.X
	wallDepth := wallBBox.Max.Z - wallBBox.Min.Z
	wallScale := tileUnit / wallWidth // scale wall to span full tile edge
	fmt.Printf("Wall width: %.2f, depth: %.2f, scale: %.2f\n", wallWidth, wallDepth, wallScale)

	// Character scaling
	charBBox := rl.GetModelBoundingBox(knightModel)
	charW := charBBox.Max.X - charBBox.Min.X
	charScale := (tileUnit * 0.6) / charW
	knightYOffset := floorSurfaceY - charBBox.Min.Y*charScale

	// Generate dungeon
	seed := time.Now().UnixNano()
	dungeon := GenerateDungeon(seed)
	fmt.Printf("Generated dungeon with %d rooms (seed: %d)\n", len(dungeon.Rooms), seed)

	// Grid world size
	gridWorld := float32(dungeon.Width) * tileUnit
	centerX := gridWorld * 0.5
	centerZ := gridWorld * 0.5

	// Place player in center of first room
	px, pz := dungeon.Rooms[0].Center()

	// Camera
	orbitAngle := float32(math.Pi / 4)
	orbitRadius := gridWorld * 0.55
	cameraHeight := gridWorld * 0.5

	camera := rl.Camera3D{
		Position:   rl.Vector3{X: centerX, Y: cameraHeight, Z: centerZ + orbitRadius},
		Target:     rl.Vector3{X: centerX, Y: 0, Z: centerZ},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       45,
		Projection: rl.CameraPerspective,
	}

	gridToWorld := func(gx, gz int) rl.Vector3 {
		return rl.Vector3{
			X: (float32(gx) + 0.5) * tileUnit,
			Y: 0,
			Z: (float32(gz) + 0.5) * tileUnit,
		}
	}

	worldToGrid := func(wx, wz float32) (int, int) {
		return int(math.Floor(float64(wx / tileUnit))), int(math.Floor(float64(wz / tileUnit)))
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

	// Floor tile variants for visual variety (seeded by position)
	floorVariant := func(x, z int) rl.Model {
		h := (x*7 + z*13 + x*z*3) % 10
		if h == 0 {
			return floorCrackedA
		}
		if h == 1 {
			return floorCrackedB
		}
		return floorModel
	}

	for !rl.WindowShouldClose() {
		// Camera orbit
		if rl.IsKeyDown(rl.KeyLeft) {
			orbitAngle -= 0.02
		}
		if rl.IsKeyDown(rl.KeyRight) {
			orbitAngle += 0.02
		}
		// Zoom
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			orbitRadius -= wheel * tileUnit * 0.5
			if orbitRadius < gridWorld*0.2 {
				orbitRadius = gridWorld * 0.2
			}
			if orbitRadius > gridWorld*1.0 {
				orbitRadius = gridWorld * 1.0
			}
		}

		camera.Position.X = centerX + orbitRadius*float32(math.Cos(float64(orbitAngle)))
		camera.Position.Z = centerZ + orbitRadius*float32(math.Sin(float64(orbitAngle)))
		camera.Position.Y = cameraHeight
		camera.Target.X = centerX
		camera.Target.Z = centerZ

		viewPos := []float32{camera.Position.X, camera.Position.Y, camera.Position.Z}
		rl.SetShaderValue(shader, locViewPos, viewPos, rl.ShaderUniformVec3)

		// Click to move
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
			if wx, wz, ok := rayHitGround(ray); ok {
				gx, gz := worldToGrid(wx, wz)
				if gx >= 0 && gx < dungeon.Width && gz >= 0 && gz < dungeon.Height {
					if dungeon.Cells[gz][gx].Type == CellFloor {
						px, pz = gx, gz
					}
				}
			}
		}

		// Regenerate dungeon with R key
		if rl.IsKeyPressed(rl.KeyR) {
			seed = time.Now().UnixNano()
			dungeon = GenerateDungeon(seed)
			px, pz = dungeon.Rooms[0].Center()
			fmt.Printf("Regenerated dungeon with %d rooms (seed: %d)\n", len(dungeon.Rooms), seed)
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Color{R: 10, G: 10, B: 15, A: 255})
		rl.BeginMode3D(camera)

		ones := rl.Vector3{X: 1, Y: 1, Z: 1}

		// Render floor tiles
		for z := 0; z < dungeon.Height; z++ {
			for x := 0; x < dungeon.Width; x++ {
				if dungeon.Cells[z][x].Type != CellFloor {
					continue
				}
				pos := gridToWorld(x, z)
				fm := floorVariant(x, z)
				rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.White)
			}
		}

		// Render walls
		wallScaleVec := rl.Vector3{X: wallScale, Y: wallScale, Z: wallScale}
		for z := 0; z < dungeon.Height; z++ {
			for x := 0; x < dungeon.Width; x++ {
				cell := dungeon.Cells[z][x]
				if cell.Type != CellFloor {
					continue
				}
				pos := gridToWorld(x, z)
				halfTile := tileUnit * 0.5

				// Wall model: centered on X (-2..2), depth on Z (-0.75..0.75), height Y (0..4)
				// Default orientation: spans along X axis, faces +Z
				// North wall: at north edge, rotated 180° to face south (into the room)
				if cell.WallN {
					wallPos := rl.Vector3{X: pos.X, Y: floorSurfaceY, Z: pos.Z - halfTile}
					rl.DrawModelEx(wallModel, wallPos, rl.Vector3{Y: 1}, 180, wallScaleVec, rl.White)
				}
				// South wall: at south edge, default orientation (faces +Z = north, into room)
				if cell.WallS {
					wallPos := rl.Vector3{X: pos.X, Y: floorSurfaceY, Z: pos.Z + halfTile}
					rl.DrawModelEx(wallModel, wallPos, rl.Vector3{Y: 1}, 0, wallScaleVec, rl.White)
				}
				// West wall: at west edge, rotated -90° (faces east, into room)
				if cell.WallW {
					wallPos := rl.Vector3{X: pos.X - halfTile, Y: floorSurfaceY, Z: pos.Z}
					rl.DrawModelEx(wallModel, wallPos, rl.Vector3{Y: 1}, -90, wallScaleVec, rl.White)
				}
				// East wall: at east edge, rotated 90° (faces west, into room)
				if cell.WallE {
					wallPos := rl.Vector3{X: pos.X + halfTile, Y: floorSurfaceY, Z: pos.Z}
					rl.DrawModelEx(wallModel, wallPos, rl.Vector3{Y: 1}, 90, wallScaleVec, rl.White)
				}
			}
		}

		// Player
		scaleVec := rl.Vector3{X: charScale, Y: charScale, Z: charScale}
		knightPos := gridToWorld(px, pz)
		knightPos.Y = knightYOffset
		rl.DrawModelEx(knightModel, knightPos, rl.Vector3{Y: 1}, 0, scaleVec, rl.White)

		// Tile hover highlight
		ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
		if wx, wz, ok := rayHitGround(ray); ok {
			gx, gz := worldToGrid(wx, wz)
			if gx >= 0 && gx < dungeon.Width && gz >= 0 && gz < dungeon.Height {
				if dungeon.Cells[gz][gx].Type == CellFloor {
					hlPos := gridToWorld(gx, gz)
					hlPos.Y = floorSurfaceY + 0.05
					rl.DrawCubeV(hlPos, rl.Vector3{X: tileUnit * 0.95, Y: 0.1, Z: tileUnit * 0.95}, rl.Color{R: 100, G: 200, B: 255, A: 60})
				}
			}
		}

		rl.EndMode3D()

		// HUD
		rl.DrawText(fmt.Sprintf("Rooms: %d | Player: (%d, %d)", len(dungeon.Rooms), px, pz), 10, 10, 20, rl.White)
		rl.DrawText("Click to move | Left/Right to orbit | Scroll to zoom | R to regenerate", 10, 35, 16, rl.Gray)
		rl.DrawFPS(screenWidth-90, 10)

		rl.EndDrawing()
	}
}
