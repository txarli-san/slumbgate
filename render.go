package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"math"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
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
		return rl.Color{R: 180, G: 200, B: 255, A: 255}
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

// AnimatedModel holds a model with its animation set and name→index lookup.
// The Model pointer is heap-allocated to prevent C pointer invalidation.
type AnimatedModel struct {
	Model  *rl.Model
	Anims  []rl.ModelAnimation
	Index  map[string]int // animation name → index
}

func loadAnimatedModel(path string) *AnimatedModel {
	model := rl.LoadModel(path)
	// Fix raylib bug: UpdateModelAnimation crashes if any mesh lacks boneWeights/boneIds.
	// Allocate zeroed arrays via C so raylib can free them normally.
	meshes := unsafe.Slice(model.Meshes, model.MeshCount)
	for i := range meshes {
		n := meshes[i].VertexCount
		if meshes[i].BoneWeights == nil {
			meshes[i].BoneWeights = (*float32)(C.calloc(C.size_t(n*4), C.size_t(unsafe.Sizeof(float32(0)))))
		}
		if meshes[i].BoneIds == nil {
			meshes[i].BoneIds = (*int32)(C.calloc(C.size_t(n*4), C.size_t(unsafe.Sizeof(int32(0)))))
		}
	}
	m := &AnimatedModel{
		Model: &model,
		Anims: rl.LoadModelAnimations(path),
		Index: map[string]int{},
	}
	for i, a := range m.Anims {
		name := ""
		for _, b := range a.Name {
			if b == 0 {
				break
			}
			name += string(rune(b))
		}
		m.Index[name] = i
	}
	return m
}

func (m *AnimatedModel) Unload() {
	rl.UnloadModel(*m.Model)
	rl.UnloadModelAnimations(m.Anims)
}

// UpdateAnim advances the animation by dt seconds and updates the model mesh.
// Must be called per-entity before drawing (update-then-draw pattern).
func (m *AnimatedModel) UpdateAnim(anim *AnimState, dt float32) {
	idx, ok := m.Index[anim.Clip]
	if !ok || len(m.Anims) == 0 {
		return
	}
	a := m.Anims[idx]
	duration := float32(a.FrameCount) / 60.0 // authored at ~60fps
	anim.Time += dt
	if anim.Loop {
		for anim.Time >= duration {
			anim.Time -= duration
		}
	} else if anim.Time >= duration {
		anim.Time = duration
		anim.Done = true
	}
	anim.Frame = int32(anim.Time / duration * float32(a.FrameCount))
	if anim.Frame >= a.FrameCount {
		anim.Frame = a.FrameCount - 1
	}
	rl.UpdateModelAnimation(*m.Model, a, anim.Frame)
}

func drawLocal(
	camera rl.Camera3D,
	world *World,
	game *GameState,
	dt float32,
	tileUnit, floorSurfaceY, knightYOffset, charScale, wallScale float32,
	floorVariant func(int, int) rl.Model,
	gridToWorld func(int, int) rl.Vector3,
	wallModel, pickaxeModel rl.Model,
	heroModels map[string]*AnimatedModel, // "Fighter" → Knight, "Mage" → Mage
	skeletonModels map[SkeletonType]*AnimatedModel,
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
				revealed := chunk.Revealed[lz][lx] || game.Debug

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
						if n == TileFloor && (world.IsRevealed(nx, nz) || game.Debug) {
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

					// Walls on edges facing non-dungeon tiles
					needsWall := func(nx, nz int) bool {
						n := world.TileTypeAt(nx, nz)
						return n == TileGround || n == TileCore
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

					// Walls on edges facing non-dungeon tiles
					needsDoorWall := func(nx, nz int) bool {
						n := world.TileTypeAt(nx, nz)
						return n == TileGround || n == TileCore
					}
					if needsDoorWall(tx, tz-1) {
						drawWall(rl.Vector3{X: pos.X, Z: pos.Z - halfTile}, 180)
					}
					if needsDoorWall(tx, tz+1) {
						drawWall(rl.Vector3{X: pos.X, Z: pos.Z + halfTile}, 0)
					}
					if needsDoorWall(tx-1, tz) {
						drawWall(rl.Vector3{X: pos.X - halfTile, Z: pos.Z}, -90)
					}
					if needsDoorWall(tx+1, tz) {
						drawWall(rl.Vector3{X: pos.X + halfTile, Z: pos.Z}, 90)
					}

				case TileCore:
					// Nothing rendered — void
				}
			}
		}
	}

	// Range highlights
	for tile := range game.MoveRange {
		pos := gridToWorld(tile[0], tile[1])
		pos.Y = floorSurfaceY + 0.03
		rl.DrawCubeV(pos, rl.Vector3{X: tileUnit * 0.9, Y: 0.04, Z: tileUnit * 0.9},
			rl.Color{R: 40, G: 180, B: 255, A: 50})
	}
	for tile := range game.AttackRange {
		pos := gridToWorld(tile[0], tile[1])
		pos.Y = floorSurfaceY + 0.03
		rl.DrawCubeV(pos, rl.Vector3{X: tileUnit * 0.9, Y: 0.04, Z: tileUnit * 0.9},
			rl.Color{R: 255, G: 60, B: 60, A: 50})
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

	// Entities (update-then-draw: animate shared model, draw, repeat per entity)
	scaleVec := rl.Vector3{X: charScale, Y: charScale, Z: charScale}
	for i, ent := range entities {
		var entPos rl.Vector3
		if ent.Moving {
			from := gridToWorld(ent.PrevX, ent.PrevZ)
			to := gridToWorld(ent.X, ent.Z)
			t := ent.StepProgress
			entPos = rl.Vector3{
				X: from.X + (to.X-from.X)*t,
				Y: knightYOffset,
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
		// Derive animation from state
		wantClip := "Idle"
		wantLoop := true
		if ent.Moving {
			wantClip = "Walking_A"
		}
		if ent.Anim.Clip != wantClip {
			ent.Anim = AnimState{Clip: wantClip, Loop: wantLoop}
		}
		// Pick model by class
		class := "Fighter"
		if ent.Stats != nil && ent.Stats.Class != "" {
			class = ent.Stats.Class
		}
		if am, ok := heroModels[class]; ok {
			am.UpdateAnim(&ent.Anim, dt)
			rl.DrawModelEx(*am.Model, entPos, rl.Vector3{Y: 1}, ent.FacingAngle, scaleVec, tierColor(ent.Tier))
		}
	}

	// Threats (skeleton models — same update-then-draw pattern)
	for key, threat := range world.Threats {
		if world.IsRevealed(threat.X, threat.Z) || game.Debug {
			var tPos rl.Vector3
			if threat.Moving {
				from := gridToWorld(threat.PrevX, threat.PrevZ)
				to := gridToWorld(threat.X, threat.Z)
				t := threat.StepProgress
				tPos = rl.Vector3{
					X: from.X + (to.X-from.X)*t,
					Y: knightYOffset,
					Z: from.Z + (to.Z-from.Z)*t,
				}
			} else {
				tPos = gridToWorld(threat.X, threat.Z)
				tPos.Y = knightYOffset
			}
			if am, ok := skeletonModels[threat.Type]; ok {
				wantClip := "Idle_Combat"
				if threat.Moving {
					wantClip = "Walking_A"
				}
				if threat.Anim.Clip != wantClip {
					threat.Anim = AnimState{Clip: wantClip, Loop: true}
				}
				am.UpdateAnim(&threat.Anim, dt)
				world.Threats[key] = threat
				rl.DrawModelEx(*am.Model, tPos, rl.Vector3{Y: 1}, 0, scaleVec, rl.White)
			}
		}
	}

	rl.EndMode3D()

	// HUD
	if game.Combat != nil {
		rl.DrawText("COMBAT", 10, 10, 24, rl.Color{R: 255, G: 60, B: 60, A: 255})
		rl.DrawText("Click tile to move | Click enemy to attack | Space end turn", 10, 38, 16, rl.Gray)
	} else {
		rl.DrawText(fmt.Sprintf("Time: %d | Chunks: %d",
			game.TimeTicks, len(world.Chunks)), 10, 10, 20, rl.White)
		rl.DrawText("Click select/move | Tab cycle | 1 Scout | 2 Stop | Right-drag orbit | Scroll zoom", 10, 35, 16, rl.Gray)
	}
	if game.MessageTimer > 0 {
		rl.DrawText(game.Message, 10, 60, 20, rl.Color{R: 255, G: 220, B: 100, A: 255})
	}
	if game.Debug {
		rl.DrawText("DEBUG", 10, 85, 20, rl.Color{R: 255, G: 100, B: 255, A: 255})
	}

	// Entity panel — bottom left
	if game.SelectedEnt >= 0 && game.SelectedEnt < len(game.Entities) {
		ent := game.Entities[game.SelectedEnt]
		panelX := int32(10)
		panelY := int32(screenHeight - 140)
		panelW := int32(260)
		panelH := int32(130)
		rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.Color{R: 20, G: 20, B: 30, A: 210})
		rl.DrawRectangleLines(panelX, panelY, panelW, panelH, tierColor(ent.Tier))

		rl.DrawText(fmt.Sprintf("%s  [%s]", ent.Name, tierName(ent.Tier)), panelX+10, panelY+8, 20, tierColor(ent.Tier))
		if ent.Stats != nil {
			s := ent.Stats
			rl.DrawText(fmt.Sprintf("Lv %d %s  HP %d/%d", s.Level, s.Class, s.HP, s.MaxHP),
				panelX+10, panelY+34, 14, rl.LightGray)
			nextXP := xpForLevel(s.Level + 1)
			if nextXP > 0 {
				rl.DrawText(fmt.Sprintf("XP: %d / %d", s.XP, nextXP),
					panelX+10, panelY+52, 14, rl.Color{R: 255, G: 215, B: 0, A: 255})
			} else {
				rl.DrawText(fmt.Sprintf("XP: %d (MAX)", s.XP),
					panelX+10, panelY+52, 14, rl.Color{R: 255, G: 215, B: 0, A: 255})
			}
		} else {
			rl.DrawText(fmt.Sprintf("Pos: (%d, %d)  Sight: %d", ent.X, ent.Z, ent.RevealDist), panelX+10, panelY+34, 14, rl.LightGray)
		}

		taskStr := "Idle"
		if ent.Task != nil {
			switch ent.Task.Type {
			case TaskMoveTo:
				taskStr = fmt.Sprintf("Moving to (%d,%d)", ent.Task.TargetX, ent.Task.TargetZ)
			case TaskExplore:
				taskStr = "Scouting"
			case TaskFollow:
				if ent.Task.FollowIdx >= 0 && ent.Task.FollowIdx < len(entities) {
					taskStr = fmt.Sprintf("Following %s", entities[ent.Task.FollowIdx].Name)
				}
			}
		}
		rl.DrawText("Task: "+taskStr, panelX+10, panelY+72, 14, rl.White)

		rl.DrawText("[1] Scout [2] Stop [3] Rest [4] Follow", panelX+10, panelY+92, 14, rl.Gray)
		rl.DrawText(fmt.Sprintf("< Tab (%d/%d) >", game.SelectedEnt+1, len(game.Entities)), panelX+10, panelY+110, 12, rl.DarkGray)
	}

	// Combat UI
	if game.Combat != nil {
		c := game.Combat
		// Initiative order — right side, below minimap
		panelX := int32(screenWidth - 220)
		panelY := int32(220)
		panelW := int32(210)
		panelH := int32(30 + int32(len(c.Combatants))*22)
		rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.Color{R: 20, G: 20, B: 30, A: 210})
		rl.DrawRectangleLines(panelX, panelY, panelW, panelH, rl.Color{R: 255, G: 60, B: 60, A: 255})
		rl.DrawText("Initiative", panelX+10, panelY+6, 16, rl.White)

		for ci, cb := range c.Combatants {
			y := panelY + 26 + int32(ci)*22
			color := rl.LightGray
			name := ""
			if cb.IsEnemy {
				if t, ok := world.Threats[cb.ThreatKey]; ok {
					switch t.Type {
					case SkeletonMinion:
						name = "Sk. Minion"
					case SkeletonWarrior:
						name = "Sk. Warrior"
					case SkeletonRogue:
						name = "Sk. Rogue"
					case SkeletonMage:
						name = "Sk. Mage"
					}
					name = fmt.Sprintf("%s (%d/%d)", name, t.HP, t.MaxHP)
				}
				color = rl.Color{R: 255, G: 100, B: 100, A: 255}
			} else {
				ent := game.Entities[cb.EntityIdx]
				name = ent.Name
				if ent.Stats != nil {
					name = fmt.Sprintf("%s (%d/%d)", name, ent.Stats.HP, ent.Stats.MaxHP)
				}
				color = tierColor(ent.Tier)
			}
			if ci == c.TurnIndex {
				rl.DrawText(">", panelX+4, y, 14, rl.Yellow)
			}
			rl.DrawText(fmt.Sprintf("%2d  %s", cb.Initiative, name), panelX+16, y, 14, color)
		}

		// Active turn info + action bar — bottom
		cur := c.Current()
		if !cur.IsEnemy {
			ent := game.Entities[cur.EntityIdx]

			// Action bar background
			barH := int32(52)
			barY := int32(screenHeight) - barH
			rl.DrawRectangle(0, barY, screenWidth, barH, rl.Color{R: 20, G: 20, B: 30, A: 230})

			// Status line
			info := fmt.Sprintf("%s | HP: %d/%d | Move: %d",
				ent.Name, ent.Stats.HP, ent.Stats.MaxHP, c.MoveLeft)
			rl.DrawText(info, 10, barY+4, 16, rl.Yellow)

			// Action buttons
			btnX := int32(10)
			btnY := barY + 24
			for _, action := range c.Actions {
				canUse := action.CanUse != nil && action.CanUse(game, ent)
				label := fmt.Sprintf("[%s] %s", action.Hotkey, action.Name)
				tw := rl.MeasureText(label, 14)
				padW := tw + 12

				btnColor := rl.Color{R: 40, G: 40, B: 55, A: 255}
				textColor := rl.LightGray
				if !canUse {
					textColor = rl.Color{R: 80, G: 80, B: 80, A: 255}
				} else if c.PrimedAction != nil && c.PrimedAction.ID == action.ID {
					btnColor = rl.Color{R: 80, G: 60, B: 20, A: 255}
					textColor = rl.Yellow
				}

				rl.DrawRectangle(btnX, btnY, padW, 22, btnColor)
				rl.DrawRectangleLines(btnX, btnY, padW, 22, rl.Color{R: 80, G: 80, B: 100, A: 255})
				rl.DrawText(label, btnX+6, btnY+4, 14, textColor)

				btnX += padW + 4
			}

			if c.PrimedAction != nil {
				msg := fmt.Sprintf(">> Click target for %s | Right-click cancel", c.PrimedAction.Name)
				tw := rl.MeasureText(msg, 16)
				rl.DrawText(msg, (screenWidth-tw)/2, barY-24, 16, rl.Yellow)
			}
		} else {
			rl.DrawText("Enemy turn...", (screenWidth-130)/2, screenHeight-40, 20, rl.Color{R: 255, G: 100, B: 100, A: 255})
		}
	}

	// Floating combat text
	for _, ft := range game.Floats {
		worldPos := gridToWorld(ft.WorldX, ft.WorldZ)
		progress := 1.0 - ft.Timer/ft.MaxTime
		worldPos.Y = floorSurfaceY + 2.0 + float32(progress)*3.0
		screenPos := rl.GetWorldToScreen(worldPos, camera)
		alpha := ft.Timer / ft.MaxTime
		if alpha > 1 {
			alpha = 1
		}
		col := rl.Color{R: ft.Color[0], G: ft.Color[1], B: ft.Color[2], A: uint8(alpha * 255)}
		tw := rl.MeasureText(ft.Text, ft.FontSize)
		rl.DrawText(ft.Text, int32(screenPos.X)-tw/2, int32(screenPos.Y), ft.FontSize, col)
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

	// Level-up choice overlay (only outside combat)
	if game.Combat == nil && game.PendingLevelUpEntity() >= 0 {
		pendIdx := game.PendingLevelUpEntity()
		ent := game.Entities[pendIdx]
		s := ent.Stats
		if len(s.PendingChoices) > 0 {
			boxW, boxH := int32(420), int32(200)
			boxX := int32(screenWidth/2) - boxW/2
			boxY := int32(screenHeight/2) - boxH/2
			rl.DrawRectangle(boxX, boxY, boxW, boxH, rl.Color{R: 15, G: 20, B: 35, A: 235})
			rl.DrawRectangleLines(boxX, boxY, boxW, boxH, rl.Color{R: 255, G: 215, B: 0, A: 255})

			title := fmt.Sprintf("%s reached Level %d!", ent.Name, s.Level)
			tw := rl.MeasureText(title, 22)
			rl.DrawText(title, boxX+boxW/2-tw/2, boxY+12, 22, rl.Color{R: 255, G: 215, B: 0, A: 255})

			choice := s.PendingChoices[0]
			lineY := boxY + 48

			switch choice {
			case "combat_style":
				rl.DrawText("Choose Combat Style:", boxX+20, lineY, 16, rl.White)
				lineY += 28
				rl.DrawText("[1] Gladiator (+2 melee damage)", boxX+30, lineY, 16, rl.LightGray)
				lineY += 22
				rl.DrawText("[2] Ranger (+2 ranged hit)", boxX+30, lineY, 16, rl.LightGray)
				lineY += 22
				rl.DrawText("[3] Juggernaut (+2 AC)", boxX+30, lineY, 16, rl.LightGray)

			case "combat_technique":
				rl.DrawText("Choose Combat Technique:", boxX+20, lineY, 16, rl.White)
				lineY += 28
				rl.DrawText("[1] Power Attack (-2 hit, +50% dmg)", boxX+30, lineY, 16, rl.LightGray)
				lineY += 22
				rl.DrawText("[2] Defensive Stance (+2 AC, -1 move)", boxX+30, lineY, 16, rl.LightGray)
				lineY += 22
				rl.DrawText("[3] Quick Strike (bonus atk, half dmg)", boxX+30, lineY, 16, rl.LightGray)

			case "spell_l1":
				rl.DrawText("Choose a Level 1 Spell:", boxX+20, lineY, 16, rl.White)
				lineY += 28
				var spells [][2]string
				if s.Level <= 2 {
					spells = [][2]string{
						{"Arcane Blink", "teleport within move range"},
						{"Burning Hands", "AoE adjacent, fire"},
						{"Frost Nova", "AoE adjacent, cold"},
						{"Mind Spike", "ranged, psychic"},
					}
				} else {
					spells = [][2]string{
						{"Magic Armor", "+2 AC for duration"},
						{"Elemental Strike", "ranged, elemental"},
						{"Feather Fall", "fall protection"},
						{"Expeditious Retreat", "+3 move for duration"},
					}
				}
				for i, sp := range spells {
					rl.DrawText(fmt.Sprintf("[%d] %s (%s)", i+1, sp[0], sp[1]),
						boxX+30, lineY, 16, rl.LightGray)
					lineY += 22
				}

			case "cantrip":
				rl.DrawText("Choose a Cantrip:", boxX+20, lineY, 16, rl.White)
				lineY += 28
				rl.DrawText("[1] Ray of Frost (ranged, cold + slow)", boxX+30, lineY, 16, rl.LightGray)
				lineY += 22
				rl.DrawText("[2] Shocking Grasp (melee, lightning)", boxX+30, lineY, 16, rl.LightGray)
			}
		}
	}

	// Minimap — top right, tile-level
	{
		mmSize := int32(200)
		mmX := int32(screenWidth) - mmSize - 10
		mmY := int32(10)
		mmScale := int32(2) // pixels per tile
		tilesVisible := int(mmSize / mmScale)
		half := tilesVisible / 2

		// Center on selected entity
		cx, cz := 0, 0
		if game.SelectedEnt >= 0 && game.SelectedEnt < len(entities) {
			cx = entities[game.SelectedEnt].X
			cz = entities[game.SelectedEnt].Z
		}

		// Background
		rl.DrawRectangle(mmX-1, mmY-1, mmSize+2, mmSize+2, rl.Color{R: 10, G: 10, B: 15, A: 230})

		for dz := 0; dz < tilesVisible; dz++ {
			for dx := 0; dx < tilesVisible; dx++ {
				tx := cx - half + dx
				tz := cz - half + dz
				t := world.TileTypeAt(tx, tz)
				vis := world.IsRevealed(tx, tz) || game.Debug

				var col rl.Color
				switch t {
				case TileGround:
					col = rl.Color{R: 50, G: 50, B: 40, A: 255}
				case TileSolid:
					if !vis {
						continue
					}
					col = rl.Color{R: 80, G: 80, B: 90, A: 255}
				case TileFloor:
					if !vis {
						continue
					}
					col = rl.Color{R: 140, G: 130, B: 110, A: 255}
				case TileDoorway:
					if !vis {
						continue
					}
					col = rl.Color{R: 120, G: 110, B: 90, A: 255}
				case TileCore:
					col = rl.Color{R: 20, G: 15, B: 30, A: 255}
				default:
					continue
				}

				px := mmX + int32(dx)*mmScale
				py := mmY + int32(dz)*mmScale
				rl.DrawRectangle(px, py, mmScale, mmScale, col)
			}
		}

		// Threats
		for _, threat := range world.Threats {
			if !world.IsRevealed(threat.X, threat.Z) && !game.Debug {
				continue
			}
			dx := threat.X - cx + half
			dz := threat.Z - cz + half
			if dx >= 0 && dx < tilesVisible && dz >= 0 && dz < tilesVisible {
				px := mmX + int32(dx)*mmScale
				py := mmY + int32(dz)*mmScale
				rl.DrawRectangle(px, py, mmScale, mmScale, rl.Color{R: 255, G: 60, B: 60, A: 255})
			}
		}

		// Entities
		for _, ent := range entities {
			dx := ent.X - cx + half
			dz := ent.Z - cz + half
			if dx >= 0 && dx < tilesVisible && dz >= 0 && dz < tilesVisible {
				px := mmX + int32(dx)*mmScale
				py := mmY + int32(dz)*mmScale
				rl.DrawRectangle(px, py, mmScale, mmScale, rl.Color{R: 80, G: 255, B: 80, A: 255})
			}
		}

		// Border
		rl.DrawRectangleLines(mmX-1, mmY-1, mmSize+2, mmSize+2, rl.Color{R: 80, G: 80, B: 100, A: 255})
	}

	// Game over overlay
	if game.GameOver {
		rl.DrawRectangle(0, 0, screenWidth, screenHeight, rl.Color{R: 0, G: 0, B: 0, A: 180})
		title := "YOUR COMPANY HAS FALLEN"
		tw := rl.MeasureText(title, 36)
		rl.DrawText(title, (screenWidth-tw)/2, screenHeight/2-40, 36, rl.Color{R: 255, G: 60, B: 60, A: 255})
		sub := "The dungeon claims another expedition."
		sw := rl.MeasureText(sub, 20)
		rl.DrawText(sub, (screenWidth-sw)/2, screenHeight/2+10, 20, rl.Color{R: 200, G: 200, B: 200, A: 255})
		restart := "[Space] New Expedition"
		rw := rl.MeasureText(restart, 20)
		rl.DrawText(restart, (screenWidth-rw)/2, screenHeight/2+50, 20, rl.Gray)
	}

	rl.DrawFPS(screenWidth-90, screenHeight-25)
}
