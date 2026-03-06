package main

import (
	"fmt"
	"math"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
	gridSize     = 10
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
	rl.InitWindow(screenWidth, screenHeight, "Slumbgate - Raylib Spike")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Load lighting shader
	shader := rl.LoadShaderFromMemory(lightingVS, lightingFS)
	defer rl.UnloadShader(shader)

	// Get shader uniform locations
	locLightDir := rl.GetShaderLocation(shader, "lightDir")
	locLightColor := rl.GetShaderLocation(shader, "lightColor")
	locAmbientColor := rl.GetShaderLocation(shader, "ambientColor")
	locViewPos := rl.GetShaderLocation(shader, "viewPos")

	// Set light properties
	lightDir := []float32{-0.4, -0.8, -0.3} // angled down
	lightColor := []float32{1.0, 0.95, 0.9, 1.0}
	ambientColor := []float32{0.35, 0.35, 0.4, 1.0}

	rl.SetShaderValue(shader, locLightDir, lightDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, locLightColor, lightColor, rl.ShaderUniformVec4)
	rl.SetShaderValue(shader, locAmbientColor, ambientColor, rl.ShaderUniformVec4)

	// Shader needs matModel and matNormal locations set
	shader.UpdateLocation(rl.ShaderLocMatrixModel, rl.GetShaderLocation(shader, "matModel"))
	shader.UpdateLocation(rl.ShaderLocMatrixNormal, rl.GetShaderLocation(shader, "matNormal"))

	// Load models
	floorModel := rl.LoadModel("../assets/KayKit Dungeon Pack 1.0/Models/gltf/tileBrickB_large.gltf.glb")
	defer rl.UnloadModel(floorModel)
	applyShaderToModel(floorModel, shader)

	knightModel := rl.LoadModel("../assets/KayKit Dungeon Pack 1.0/Models/Characters/gltf/character_knight.gltf")
	defer rl.UnloadModel(knightModel)
	applyShaderToModel(knightModel, shader)

	mageModel := rl.LoadModel("../assets/KayKit Dungeon Pack 1.0/Models/Characters/gltf/character_mage.gltf")
	defer rl.UnloadModel(mageModel)
	applyShaderToModel(mageModel, shader)

	skullModel := rl.LoadModel("../assets/KayKit Dungeon Pack 1.0/Models/Characters/gltf/extra heads/skull.gltf.glb")
	defer rl.UnloadModel(skullModel)
	applyShaderToModel(skullModel, shader)

	// Measure floor tile to derive grid unit
	floorBBox := rl.GetModelBoundingBox(floorModel)
	tileUnit := floorBBox.Max.X - floorBBox.Min.X
	fmt.Printf("Tile unit: %.2f\n", tileUnit)

	// Character scale: fill ~60% of a tile
	charBBox := rl.GetModelBoundingBox(knightModel)
	charW := charBBox.Max.X - charBBox.Min.X
	charScale := (tileUnit * 0.6) / charW
	fmt.Printf("Character width: %.2f, scale: %.2f\n", charW, charScale)

	// Figure out where the floor surface sits
	floorSurfaceY := floorBBox.Max.Y // top of the tile mesh
	fmt.Printf("Floor surface Y: %.2f\n", floorSurfaceY)

	// Calculate Y offsets: place model bottom at floor surface level
	// offset = floorSurfaceY - (model.Min.Y * scale)
	knightYOffset := floorSurfaceY - charBBox.Min.Y*charScale
	mageBBox := rl.GetModelBoundingBox(mageModel)
	mageYOffset := floorSurfaceY - mageBBox.Min.Y*charScale
	skullBBox := rl.GetModelBoundingBox(skullModel)
	skullYOffset := floorSurfaceY - skullBBox.Min.Y*charScale
	fmt.Printf("BBox mins - knight: %.3f, mage: %.3f, skull: %.3f\n", charBBox.Min.Y, mageBBox.Min.Y, skullBBox.Min.Y)
	fmt.Printf("BBox maxs - knight: %.3f, mage: %.3f, skull: %.3f\n", charBBox.Max.Y, mageBBox.Max.Y, skullBBox.Max.Y)
	fmt.Printf("Y offsets - knight: %.2f, mage: %.2f, skull: %.2f\n", knightYOffset, mageYOffset, skullYOffset)

	// Grid world size
	gridWorld := float32(gridSize) * tileUnit
	centerX := gridWorld * 0.5
	centerZ := gridWorld * 0.5

	// Camera
	camera := rl.Camera3D{
		Position:   rl.Vector3{X: centerX, Y: gridWorld * 0.7, Z: centerZ + gridWorld*0.8},
		Target:     rl.Vector3{X: centerX, Y: 0, Z: centerZ},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       40,
		Projection: rl.CameraPerspective,
	}

	playerX, playerZ := 5, 5
	enemies := [][2]int{{2, 2}, {7, 3}, {4, 8}}
	orbitAngle := float32(math.Pi / 4)

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

	for !rl.WindowShouldClose() {
		// Camera orbit
		if rl.IsKeyDown(rl.KeyLeft) {
			orbitAngle -= 0.02
		}
		if rl.IsKeyDown(rl.KeyRight) {
			orbitAngle += 0.02
		}

		radius := gridWorld * 0.8
		camera.Position.X = centerX + radius*float32(math.Cos(float64(orbitAngle)))
		camera.Position.Z = centerZ + radius*float32(math.Sin(float64(orbitAngle)))
		camera.Target.X = centerX
		camera.Target.Z = centerZ

		// Update view position for specular
		viewPos := []float32{camera.Position.X, camera.Position.Y, camera.Position.Z}
		rl.SetShaderValue(shader, locViewPos, viewPos, rl.ShaderUniformVec3)

		// Click to move
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
			if wx, wz, ok := rayHitGround(ray); ok {
				gx, gz := worldToGrid(wx, wz)
				if gx >= 0 && gx < gridSize && gz >= 0 && gz < gridSize {
					playerX, playerZ = gx, gz
				}
			}
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Color{R: 20, G: 20, B: 30, A: 255})
		rl.BeginMode3D(camera)

		// Floor tiles at natural size with small gaps
		ones := rl.Vector3{X: 0.98, Y: 1, Z: 0.98} // slightly smaller for grid lines
		for x := 0; x < gridSize; x++ {
			for z := 0; z < gridSize; z++ {
				pos := gridToWorld(x, z)
				rl.DrawModelEx(floorModel, pos, rl.Vector3{Y: 1}, 0, ones, rl.White)
			}
		}

		// Grid lines on the floor for tile visibility
		gridLineColor := rl.Color{R: 40, G: 40, B: 50, A: 255}
		for i := 0; i <= gridSize; i++ {
			x := float32(i) * tileUnit
			rl.DrawLine3D(
				rl.Vector3{X: x, Y: 0.02, Z: 0},
				rl.Vector3{X: x, Y: 0.02, Z: gridWorld},
				gridLineColor,
			)
			z := float32(i) * tileUnit
			rl.DrawLine3D(
				rl.Vector3{X: 0, Y: 0.02, Z: z},
				rl.Vector3{X: gridWorld, Y: 0.02, Z: z},
				gridLineColor,
			)
		}

		// Player (knight)
		scaleVec := rl.Vector3{X: charScale, Y: charScale, Z: charScale}
		knightPos := gridToWorld(playerX, playerZ)
		knightPos.Y = knightYOffset
		rl.DrawModelEx(knightModel, knightPos, rl.Vector3{Y: 1}, 0, scaleVec, rl.White)

		// Enemies
		for i, e := range enemies {
			pos := gridToWorld(e[0], e[1])
			if i == 0 {
				pos.Y = mageYOffset
				rl.DrawModelEx(mageModel, pos, rl.Vector3{Y: 1}, 0, scaleVec, rl.White)
			} else {
				pos.Y = skullYOffset
				rl.DrawModelEx(skullModel, pos, rl.Vector3{Y: 1}, 0, scaleVec, rl.White)
			}
		}

		// Tile hover highlight
		ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
		if wx, wz, ok := rayHitGround(ray); ok {
			gx, gz := worldToGrid(wx, wz)
			if gx >= 0 && gx < gridSize && gz >= 0 && gz < gridSize {
				hlPos := gridToWorld(gx, gz)
				hlPos.Y = 0.05
				rl.DrawCubeV(hlPos, rl.Vector3{X: tileUnit * 0.95, Y: 0.1, Z: tileUnit * 0.95}, rl.Color{R: 100, G: 200, B: 255, A: 60})
			}
		}

		rl.EndMode3D()

		// HUD
		rl.DrawText(fmt.Sprintf("Player: (%d, %d)", playerX, playerZ), 10, 10, 20, rl.White)
		rl.DrawText("Click to move | Left/Right arrows to orbit camera", 10, 35, 16, rl.Gray)
		rl.DrawFPS(screenWidth-90, 10)

		rl.EndDrawing()
	}
}
