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

// Find a ground cell adjacent to the first room's wall
func findStartPosition(d *Dungeon) (int, int) {
	r := d.Rooms[0]
	// Try south of the room
	z := r.Y + r.H
	x := r.X + r.W/2
	if z < d.Height && d.Cells[z][x].Type == CellGround {
		return x, z
	}
	// Try north
	z = r.Y - 1
	if z >= 0 && d.Cells[z][x].Type == CellGround {
		return x, z
	}
	// Try east
	x = r.X + r.W
	z = r.Y + r.H/2
	if x < d.Width && d.Cells[z][x].Type == CellGround {
		return x, z
	}
	// Fallback: just pick any ground cell
	for gz := 0; gz < d.Height; gz++ {
		for gx := 0; gx < d.Width; gx++ {
			if d.Cells[gz][gx].Type == CellGround {
				return gx, gz
			}
		}
	}
	return 0, 0
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

	wallDecoAModel := rl.LoadModel("../assets/models/dungeon/walls/wallDecorationA.gltf.glb")
	defer rl.UnloadModel(wallDecoAModel)
	applyShaderToModel(wallDecoAModel, shader)

	wallDecoBModel := rl.LoadModel("../assets/models/dungeon/walls/wallDecorationB.gltf.glb")
	defer rl.UnloadModel(wallDecoBModel)
	applyShaderToModel(wallDecoBModel, shader)

	// Props
	barrelModel := rl.LoadModel("../assets/models/dungeon/props/barrel.gltf.glb")
	defer rl.UnloadModel(barrelModel)
	applyShaderToModel(barrelModel, shader)

	crateModel := rl.LoadModel("../assets/models/dungeon/props/crate.gltf.glb")
	defer rl.UnloadModel(crateModel)
	applyShaderToModel(crateModel, shader)

	tableSmallModel := rl.LoadModel("../assets/models/dungeon/props/tableSmall.gltf.glb")
	defer rl.UnloadModel(tableSmallModel)
	applyShaderToModel(tableSmallModel, shader)

	tableLargeModel := rl.LoadModel("../assets/models/dungeon/props/tableLarge.gltf.glb")
	defer rl.UnloadModel(tableLargeModel)
	applyShaderToModel(tableLargeModel, shader)

	chairModel := rl.LoadModel("../assets/models/dungeon/props/chair.gltf.glb")
	defer rl.UnloadModel(chairModel)
	applyShaderToModel(chairModel, shader)

	stoolModel := rl.LoadModel("../assets/models/dungeon/props/stool.gltf.glb")
	defer rl.UnloadModel(stoolModel)
	applyShaderToModel(stoolModel, shader)

	bookcaseModel := rl.LoadModel("../assets/models/dungeon/props/bookcase.gltf.glb")
	defer rl.UnloadModel(bookcaseModel)
	applyShaderToModel(bookcaseModel, shader)

	bookcaseFilledModel := rl.LoadModel("../assets/models/dungeon/props/bookcaseFilled.gltf.glb")
	defer rl.UnloadModel(bookcaseFilledModel)
	applyShaderToModel(bookcaseFilledModel, shader)

	potsModel := rl.LoadModel("../assets/models/dungeon/props/pots.gltf.glb")
	defer rl.UnloadModel(potsModel)
	applyShaderToModel(potsModel, shader)

	bucketModel := rl.LoadModel("../assets/models/dungeon/props/bucket.gltf.glb")
	defer rl.UnloadModel(bucketModel)
	applyShaderToModel(bucketModel, shader)

	weaponRackModel := rl.LoadModel("../assets/models/dungeon/props/weaponRack.gltf.glb")
	defer rl.UnloadModel(weaponRackModel)
	applyShaderToModel(weaponRackModel, shader)

	bannerModel := rl.LoadModel("../assets/models/dungeon/props/banner.gltf.glb")
	defer rl.UnloadModel(bannerModel)
	applyShaderToModel(bannerModel, shader)

	chestCommonModel := rl.LoadModel("../assets/models/loot/chest_common.gltf.glb")
	defer rl.UnloadModel(chestCommonModel)
	applyShaderToModel(chestCommonModel, shader)

	chestRareModel := rl.LoadModel("../assets/models/loot/chest_rare.gltf.glb")
	defer rl.UnloadModel(chestRareModel)
	applyShaderToModel(chestRareModel, shader)

	torchModel := rl.LoadModel("../assets/models/dungeon/hazards/torch.gltf.glb")
	defer rl.UnloadModel(torchModel)
	applyShaderToModel(torchModel, shader)

	torchWallModel := rl.LoadModel("../assets/models/dungeon/hazards/torchWall.gltf.glb")
	defer rl.UnloadModel(torchWallModel)
	applyShaderToModel(torchWallModel, shader)

	spikesModel := rl.LoadModel("../assets/models/dungeon/hazards/tileSpikes.gltf.glb")
	defer rl.UnloadModel(spikesModel)
	applyShaderToModel(spikesModel, shader)

	knightModel := rl.LoadModel("../assets/models/characters/character_knight.gltf")
	defer rl.UnloadModel(knightModel)
	applyShaderToModel(knightModel, shader)

	// Map prop types to models
	propModels := map[PropType]rl.Model{
		PropBarrel:         barrelModel,
		PropCrate:          crateModel,
		PropTableSmall:     tableSmallModel,
		PropTableLarge:     tableLargeModel,
		PropChair:          chairModel,
		PropStool:          stoolModel,
		PropBookcase:       bookcaseModel,
		PropBookcaseFilled: bookcaseFilledModel,
		PropPots:           potsModel,
		PropBucket:         bucketModel,
		PropWeaponRack:     weaponRackModel,
		PropBanner:         bannerModel,
		PropChestCommon:    chestCommonModel,
		PropChestRare:      chestRareModel,
		PropTorch:          torchModel,
		PropSpikes:         spikesModel,
	}

	// Map wall decor types to models
	wallDecorModels := map[WallDecor]rl.Model{
		WallDecorTorch: torchWallModel,
		WallDecorDecoA: wallDecoAModel,
		WallDecorDecoB: wallDecoBModel,
	}

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

	// Player starts on ground outside the dungeon
	// Find a ground cell adjacent to the first room
	startX, startZ := findStartPosition(dungeon)
	px, pz := startX, startZ
	dungeon.RevealAround(px, pz)

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

		// Click to move (walkable = ground or floor)
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
			if wx, wz, ok := rayHitGround(ray); ok {
				gx, gz := worldToGrid(wx, wz)
				if gx >= 0 && gx < dungeon.Width && gz >= 0 && gz < dungeon.Height {
					ct := dungeon.Cells[gz][gx].Type
					if ct == CellGround || ct == CellFloor {
						px, pz = gx, gz
						dungeon.RevealAround(gx, gz)
					}
				}
			}
		}

		// Regenerate dungeon with R key
		if rl.IsKeyPressed(rl.KeyR) {
			seed = time.Now().UnixNano()
			dungeon = GenerateDungeon(seed)
			sx, sz := findStartPosition(dungeon)
			px, pz = sx, sz
			dungeon.RevealAround(px, pz)
			fmt.Printf("Regenerated dungeon with %d rooms (seed: %d)\n", len(dungeon.Rooms), seed)
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Color{R: 10, G: 10, B: 15, A: 255})
		rl.BeginMode3D(camera)

		ones := rl.Vector3{X: 1, Y: 1, Z: 1}
		wallScaleVec := rl.Vector3{X: wallScale, Y: wallScale, Z: wallScale}

		// Helper to render a wall segment
		drawWall := func(model rl.Model, pos rl.Vector3, rotation float32) {
			pos.Y = floorSurfaceY
			rl.DrawModelEx(model, pos, rl.Vector3{Y: 1}, rotation, wallScaleVec, rl.White)
		}

		for z := 0; z < dungeon.Height; z++ {
			for x := 0; x < dungeon.Width; x++ {
				cell := dungeon.Cells[z][x]
				pos := gridToWorld(x, z)
				halfTile := tileUnit * 0.5

				if cell.Type == CellGround {
					// Open ground — always visible
					rl.DrawModelEx(floorModel, pos, rl.Vector3{Y: 1}, 0, ones, rl.Color{R: 140, G: 140, B: 130, A: 255})
					continue
				}

				// CellFloor — dungeon interior

				// Walls always visible from outside (they ARE the structure)
				if cell.WallN {
					drawWall(wallModel, rl.Vector3{X: pos.X, Z: pos.Z - halfTile}, 180)
				}
				if cell.WallS {
					drawWall(wallModel, rl.Vector3{X: pos.X, Z: pos.Z + halfTile}, 0)
				}
				if cell.WallW {
					drawWall(wallModel, rl.Vector3{X: pos.X - halfTile, Z: pos.Z}, -90)
				}
				if cell.WallE {
					drawWall(wallModel, rl.Vector3{X: pos.X + halfTile, Z: pos.Z}, 90)
				}

				// Interior only rendered when revealed
				if !cell.Revealed {
					continue
				}

				// Floor
				fm := floorVariant(x, z)
				rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.White)

				// Props
				for _, prop := range cell.Props {
					m, ok := propModels[prop.Type]
					if !ok {
						continue
					}
					propPos := pos
					propPos.Y = floorSurfaceY
					if prop.Type == PropSpikes {
						propPos.Y = 0
					}
					rl.DrawModelEx(m, propPos, rl.Vector3{Y: 1}, prop.Rotation, wallScaleVec, rl.White)
				}

				// Wall decorations
				renderWallDecor := func(decor WallDecor, wallPos rl.Vector3, rotation float32) {
					m, ok := wallDecorModels[decor]
					if !ok {
						return
					}
					wallPos.Y = floorSurfaceY
					rl.DrawModelEx(m, wallPos, rl.Vector3{Y: 1}, rotation, wallScaleVec, rl.White)
				}
				if cell.WallDecorN != WallDecorNone {
					renderWallDecor(cell.WallDecorN, rl.Vector3{X: pos.X, Z: pos.Z - halfTile}, 180)
				}
				if cell.WallDecorS != WallDecorNone {
					renderWallDecor(cell.WallDecorS, rl.Vector3{X: pos.X, Z: pos.Z + halfTile}, 0)
				}
				if cell.WallDecorW != WallDecorNone {
					renderWallDecor(cell.WallDecorW, rl.Vector3{X: pos.X - halfTile, Z: pos.Z}, -90)
				}
				if cell.WallDecorE != WallDecorNone {
					renderWallDecor(cell.WallDecorE, rl.Vector3{X: pos.X + halfTile, Z: pos.Z}, 90)
				}
			}
		}

		// Player
		scaleVec := rl.Vector3{X: charScale, Y: charScale, Z: charScale}
		knightPos := gridToWorld(px, pz)
		knightPos.Y = knightYOffset
		rl.DrawModelEx(knightModel, knightPos, rl.Vector3{Y: 1}, 0, scaleVec, rl.White)

		// Tile hover highlight (ground always, floor only if revealed)
		ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
		if wx, wz, ok := rayHitGround(ray); ok {
			gx, gz := worldToGrid(wx, wz)
			if gx >= 0 && gx < dungeon.Width && gz >= 0 && gz < dungeon.Height {
				c := dungeon.Cells[gz][gx]
				if c.Type == CellGround || (c.Type == CellFloor && c.Revealed) {
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
