package main

import (
	"flag"
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
	debugMode := flag.Bool("debug", false, "enable debug mode (no fog of war)")
	flag.Parse()

	rl.InitWindow(screenWidth, screenHeight, "Slumbgate - Infinite Dungeon")
	defer rl.CloseWindow()
	rl.SetExitKey(0) // don't quit on Esc — we use it for UI
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

	// Chest models
	chestModel := rl.LoadModel("assets/models/loot/chest_common.gltf.glb")
	defer rl.UnloadModel(chestModel)
	applyShaderToModel(chestModel, shader)
	chestTopModel := rl.LoadModel("assets/models/loot/chestTop_common.gltf.glb")
	defer rl.UnloadModel(chestTopModel)
	applyShaderToModel(chestTopModel, shader)

	// Animated character models
	knightAnim := loadAnimatedModel("assets/models/characters/animated/Knight.glb")
	defer knightAnim.Unload()
	mageAnim := loadAnimatedModel("assets/models/characters/animated/Mage.glb")
	defer mageAnim.Unload()

	knightAnim.GearBindings = gearBindings("Fighter")
	mageAnim.GearBindings = gearBindings("Mage")

	heroModels := map[string]*AnimatedModel{
		"Fighter": knightAnim,
		"Mage":    mageAnim,
	}

	// Animated skeleton models
	skelMinionAnim := loadAnimatedModel("assets/models/characters/animated/Skeleton_Minion.glb")
	defer skelMinionAnim.Unload()
	skelWarriorAnim := loadAnimatedModel("assets/models/characters/animated/Skeleton_Warrior.glb")
	defer skelWarriorAnim.Unload()
	skelMageAnim := loadAnimatedModel("assets/models/characters/animated/Skeleton_Mage.glb")
	defer skelMageAnim.Unload()

	skeletonModels := map[SkeletonType]*AnimatedModel{
		SkeletonMinion:  skelMinionAnim,
		SkeletonWarrior: skelWarriorAnim,
		SkeletonRogue:   skelWarriorAnim, // reuse warrior model for rogue
		SkeletonMage:    skelMageAnim,
	}

	// Measurements
	floorBBox := rl.GetModelBoundingBox(floorModel)
	tileUnit := floorBBox.Max.X - floorBBox.Min.X
	floorSurfaceY := floorBBox.Max.Y

	wallBBox := rl.GetModelBoundingBox(wallModel)
	wallWidth := wallBBox.Max.X - wallBBox.Min.X
	wallScale := tileUnit / wallWidth

	charBBox := rl.GetModelBoundingBox(*knightAnim.Model)
	charScale := (tileUnit * 0.6) / (charBBox.Max.X - charBBox.Min.X)
	knightYOffset := floorSurfaceY - charBBox.Min.Y*charScale

	propModels := LoadPropModels(shader, wallScale)
	defer propModels.Unload()

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

	// World + game state (initialized by initGame, can be reset on game over)
	var seed int64
	var world *World
	var spawnX int
	var game *GameState
	var mageRescueRoom int

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

	initGame := func() {
		seed = time.Now().UnixNano()
		world = NewWorld(seed)
		spawnX = OuterRadius + DungeonWarpAmp + 5
		world.PlacePickaxe(spawnX, 0)
		world.PlaceChest(spawnX, 0)
		game = &GameState{SelectedEnt: 0, Day: 1, Debug: *debugMode, ChestUsedBy: -1}
		brynn := &Entity{
			Name: "Brynn", X: spawnX, Z: 0, Tier: TierRecruit, RevealDist: 6,
			Scouted:   map[[2]int]bool{},
			Equipment: map[EquipSlot]*GearItem{},
			Stats: &CombatStats{
				HP: 12, MaxHP: 12, AC: 13,
				STR: 16, DEX: 12, CON: 14, INT: 10, WIS: 12, CHA: 10,
				Level: 1, ProfBonus: 2, MoveSpeed: 5, Class: "Fighter",
				ClassCharges: 1, MaxClassCharges: 1,
				HitDice: 1, MaxHitDice: 1, HitDieSize: 10,
			},
		}
		brynn.RebuildVisibleMeshes()
		game.Entities = []*Entity{brynn}
		for _, ent := range game.Entities {
			cx, cz := TileToChunk(ent.X, ent.Z)
			world.EnsureChunksAround(cx, cz)
			world.RevealAround(ent.X, ent.Z)
		}
		mageRescueRoom = -1
		game.Events = []*Event{
			{
				ID: "mage_rescue_enter", Trigger: TriggerRoomEntered, OneShot: true,
				Check: func(g *GameState, w *World, ctx EventContext) bool {
					return mageRescueRoom < 0
				},
				Fire: func(g *GameState, w *World, ctx EventContext) {
					mageRescueRoom = ctx.RoomIdx
					for key, t := range w.Threats {
						if t.RoomIdx == ctx.RoomIdx {
							delete(w.Threats, key)
						}
					}
					ent := g.Entities[ctx.EntityIdx]
					offsets := [2][2]int{{1, 0}, {0, 1}}
					for _, off := range offsets {
						tx, tz := ent.X+off[0], ent.Z+off[1]
						fx, fz, ok := nearestWalkable(w, tx, tz)
						if !ok {
							continue
						}
						key := [2]int{fx, fz}
						w.Threats[key] = Threat{
							X: fx, Z: fz, Type: SkeletonMinion, RoomIdx: ctx.RoomIdx,
							HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
							MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
						}
					}
				},
			},
			{
				ID: "mage_rescue_clear", Trigger: TriggerRoomCleared, OneShot: true,
				Check: func(g *GameState, w *World, ctx EventContext) bool {
					return mageRescueRoom >= 0 && ctx.RoomIdx == mageRescueRoom
				},
				Fire: func(g *GameState, w *World, ctx EventContext) {
					room := w.Rooms[ctx.RoomIdx]
					mx, mz := room.X+room.W/2, room.Z+room.H/2
					mx, mz, _ = nearestClearTile(g, w, mx, mz)
					g.Entities = append(g.Entities, &Entity{
						Name: "Elara", X: mx, Z: mz, Tier: TierRecruit, RevealDist: 5,
						Scouted: map[[2]int]bool{},
						Equipment: map[EquipSlot]*GearItem{},
						Stats: &CombatStats{
							HP: 7, MaxHP: 7, AC: 12,
							STR: 8, DEX: 14, CON: 12, INT: 16, WIS: 14, CHA: 10,
							Level: 1, ProfBonus: 2, MoveSpeed: 5, Class: "Mage",
							ClassCharges: 2, MaxClassCharges: 2,
							HitDice: 1, MaxHitDice: 1, HitDieSize: 6,
						},
					})
					g.Entities[len(g.Entities)-1].RebuildVisibleMeshes()
					g.SetMessage("Elara the Mage freed! She joins your party!")
				},
			},
		}
		fmt.Printf("World seed: %d | New expedition\n", seed)
	}
	initGame()

	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// Game over — Space to restart
		if game.GameOver {
			if rl.IsKeyPressed(rl.KeySpace) {
				initGame()
			}
			// Still render the scene + overlay
			game.TickFloats(dt)
			camera.Target = rl.Vector3{X: camTargetX, Z: camTargetZ}
			camera.Position = rl.Vector3{
				X: camTargetX + orbitRadius*float32(math.Cos(float64(orbitAngle))),
				Y: localHeight,
				Z: camTargetZ + orbitRadius*float32(math.Sin(float64(orbitAngle))),
			}
			viewPos := []float32{camera.Position.X, camera.Position.Y, camera.Position.Z}
			rl.SetShaderValue(shader, locViewPos, viewPos, rl.ShaderUniformVec3)
			rl.BeginDrawing()
			rl.ClearBackground(rl.Color{R: 10, G: 10, B: 15, A: 255})
			drawLocal(camera, world, game, dt, tileUnit, floorSurfaceY, knightYOffset, charScale, wallScale,
				floorVariant, gridToWorld,
				wallModel, pickaxeModel, chestModel, chestTopModel, heroModels, skeletonModels, propModels, game.Entities, game.SelectedEnt)
			rl.EndDrawing()
			continue
		}

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

		// Debug: L grants XP to selected entity, K toggles god mode (invincible)
		if game.Debug && game.SelectedEnt >= 0 {
			ent := game.Entities[game.SelectedEnt]
			if rl.IsKeyPressed(rl.KeyL) && ent.Stats != nil {
				prevLevel := ent.Stats.Level
				ent.Stats.XP += 500
				checkLevelUp(ent.Stats)
				game.SetMessage(fmt.Sprintf("[DEBUG] +500 XP → %d XP (Level %d)", ent.Stats.XP, ent.Stats.Level))
				if ent.Stats.Level > prevLevel {
					game.AddFloat("LEVEL UP!", ent.X, ent.Z, 255, 255, 100, 24)
				}
			}
			if rl.IsKeyPressed(rl.KeyK) && ent.Stats != nil {
				if ent.Stats.HP < ent.Stats.MaxHP {
					ent.Stats.HP = ent.Stats.MaxHP
					game.SetMessage(fmt.Sprintf("[DEBUG] %s fully healed", ent.Name))
				}
			}
			// G toggles gear debug panel
			if rl.IsKeyPressed(rl.KeyG) {
				game.DebugGearPanel = !game.DebugGearPanel
				game.GearCursor = 0
			}
			// C opens equipment UI for selected entity
			if rl.IsKeyPressed(rl.KeyC) {
				game.EquipUIOpen = !game.EquipUIOpen
				game.EquipCursor = 0
			}
			// E cycles exhaustion 0→1→...→6→0
			if rl.IsKeyPressed(rl.KeyE) && ent.Stats != nil {
				ent.Stats.Exhaustion++
				if ent.Stats.Exhaustion > 6 {
					ent.Stats.Exhaustion = 0
				}
				game.SetMessage(fmt.Sprintf("[DEBUG] %s exhaustion → %d/6", ent.Name, ent.Stats.Exhaustion))
				if ent.Stats.Exhaustion >= 6 {
					game.KillEntity(world, game.SelectedEnt)
				} else if ent.Stats.HP > ent.EffectiveMaxHP() {
					ent.Stats.HP = ent.EffectiveMaxHP()
				}
			}
			// M spawns Elara next to selected entity
			if rl.IsKeyPressed(rl.KeyM) {
				hasElara := false
				for _, e := range game.Entities {
					if e.Name == "Elara" {
						hasElara = true
						break
					}
				}
				if !hasElara {
					mx, mz, ok := nearestClearTile(game, world, ent.X+1, ent.Z)
					if ok {
						game.Entities = append(game.Entities, &Entity{
							Name: "Elara", X: mx, Z: mz, Tier: TierRecruit, RevealDist: 5,
							Scouted:       map[[2]int]bool{},
							Equipment: map[EquipSlot]*GearItem{},
							Stats: &CombatStats{
								HP: 7, MaxHP: 7, AC: 12,
								STR: 8, DEX: 14, CON: 12, INT: 16, WIS: 14, CHA: 10,
								Level: 1, ProfBonus: 2, MoveSpeed: 5, Class: "Mage",
								ClassCharges: 2, MaxClassCharges: 2,
								HitDice: 1, MaxHitDice: 1, HitDieSize: 6,
							},
						})
						game.Entities[len(game.Entities)-1].RebuildVisibleMeshes()
						game.SetMessage("[DEBUG] Elara spawned!")
					}
				} else {
					game.SetMessage("[DEBUG] Elara already in party")
				}
			}
		}

		// Gear panel input — blocks all other input when open
		gearPanelOpen := game.Debug && game.DebugGearPanel && game.SelectedEnt >= 0 &&
			game.Entities[game.SelectedEnt].VisibleMeshes != nil
		if gearPanelOpen {
			ent := game.Entities[game.SelectedEnt]
			count := len(ent.VisibleMeshes)
			if rl.IsKeyPressed(rl.KeyUp) {
				game.GearCursor--
				if game.GearCursor < 0 {
					game.GearCursor = count - 1
				}
			}
			if rl.IsKeyPressed(rl.KeyDown) {
				game.GearCursor++
				if game.GearCursor >= count {
					game.GearCursor = 0
				}
			}
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeySpace) {
				i := game.GearCursor
				ent.VisibleMeshes[i] = !ent.VisibleMeshes[i]
				names := meshNames(ent.Stats.Class)
				label := fmt.Sprintf("mesh %d", i)
				if i < len(names) {
					label = names[i]
				}
				state := "OFF"
				if ent.VisibleMeshes[i] {
					state = "ON"
				}
				game.SetMessage(fmt.Sprintf("[GEAR] %s → %s", label, state))
			}
			if rl.IsKeyPressed(rl.KeyEscape) {
				game.DebugGearPanel = false
			}
		}

		// Equipment UI input — blocks all other input while open
		if game.EquipUIOpen && game.SelectedEnt >= 0 && game.SelectedEnt < len(game.Entities) {
			ent := game.Entities[game.SelectedEnt]
			if ent.Stats != nil {
				items := GearForClass(ent.Stats.Class)
				count := len(items)
				if rl.IsKeyPressed(rl.KeyUp) {
					game.EquipCursor--
					if game.EquipCursor < 0 {
						game.EquipCursor = count - 1
					}
				}
				if rl.IsKeyPressed(rl.KeyDown) {
					game.EquipCursor++
					if game.EquipCursor >= count {
						game.EquipCursor = 0
					}
				}
				if rl.IsKeyPressed(rl.KeyEnter) {
					if game.EquipCursor < count {
						item := &AllGear[0]
						// Find the actual item in AllGear to get a stable pointer
						idx := 0
						for i := range AllGear {
							if AllGear[i].Class == ent.Stats.Class {
								if idx == game.EquipCursor {
									item = &AllGear[i]
									break
								}
								idx++
							}
						}
						// Toggle: unequip if already equipped, else equip
						if eq, ok := ent.Equipment[item.Slot]; ok && eq.Name == item.Name {
							ent.Unequip(item.Slot)
							game.SetMessage(fmt.Sprintf("Unequipped %s", item.Name))
						} else {
							ent.Equip(item)
							game.SetMessage(fmt.Sprintf("Equipped %s", item.Name))
						}
					}
				}
				if rl.IsKeyPressed(rl.KeyEscape) {
					game.EquipUIOpen = false
				}
			}
		}

		// Level-up choice input — blocks ALL other input this frame (only outside combat)
		levelUpPending := game.Combat == nil && game.PendingLevelUpEntity() >= 0
		if levelUpPending {
			pendIdx := game.PendingLevelUpEntity()
			ent := game.Entities[pendIdx]
			s := ent.Stats
			if len(s.PendingChoices) > 0 {
				choice := s.PendingChoices[0]
				switch choice {
				case "combat_style":
					if rl.IsKeyPressed(rl.KeyOne) {
						ApplyFighterStyle(s, "Gladiator")
						game.SetMessage(fmt.Sprintf("%s chose Gladiator style!", ent.Name))
					} else if rl.IsKeyPressed(rl.KeyTwo) {
						ApplyFighterStyle(s, "Ranger")
						game.SetMessage(fmt.Sprintf("%s chose Ranger style!", ent.Name))
					} else if rl.IsKeyPressed(rl.KeyThree) {
						ApplyFighterStyle(s, "Juggernaut")
						game.SetMessage(fmt.Sprintf("%s chose Juggernaut style!", ent.Name))
					}
				case "combat_technique":
					if rl.IsKeyPressed(rl.KeyOne) {
						ApplyFighterTechnique(s, "Power Attack")
						game.SetMessage(fmt.Sprintf("%s chose Power Attack!", ent.Name))
					} else if rl.IsKeyPressed(rl.KeyTwo) {
						ApplyFighterTechnique(s, "Defensive Stance")
						game.SetMessage(fmt.Sprintf("%s chose Defensive Stance!", ent.Name))
					} else if rl.IsKeyPressed(rl.KeyThree) {
						ApplyFighterTechnique(s, "Quick Strike")
						game.SetMessage(fmt.Sprintf("%s chose Quick Strike!", ent.Name))
					}
				case "spell_l1":
					var spells []string
					if s.Level <= 2 {
						spells = []string{"arcane_blink", "burning_hands", "frost_nova", "mind_spike"}
					} else {
						spells = []string{"magic_armor", "elemental_strike", "feather_fall", "expeditious_retreat"}
					}
					keys := []int32{rl.KeyOne, rl.KeyTwo, rl.KeyThree, rl.KeyFour}
					for i, key := range keys {
						if i < len(spells) && rl.IsKeyPressed(key) {
							if ApplyMageSpell(s, spells[i]) {
								name := spells[i]
								if def, ok := mageActionDefs[name]; ok {
									name = def.Name
								}
								game.SetMessage(fmt.Sprintf("%s learned %s!", ent.Name, name))
							}
						}
					}
				case "cantrip":
					cantrips := []string{"ray_of_frost", "shocking_grasp"}
					if rl.IsKeyPressed(rl.KeyOne) {
						if ApplyMageCantrip(s, cantrips[0]) {
							game.SetMessage(fmt.Sprintf("%s learned Ray of Frost!", ent.Name))
						}
					} else if rl.IsKeyPressed(rl.KeyTwo) {
						if ApplyMageCantrip(s, cantrips[1]) {
							game.SetMessage(fmt.Sprintf("%s learned Shocking Grasp!", ent.Name))
						}
					}
				}
			}
		}

		// Click in 3D view: select entity or assign move task (continuous mode only)
		if !levelUpPending && !gearPanelOpen && !game.EquipUIOpen {
		if game.Combat == nil && game.Alert == nil && game.PendingLevelUpEntity() < 0 && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
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
						} else if t, ok := world.GetTile(gx, gz); ok && t == TileCore {
							game.SetMessage("That rock is impenetrable. We need something stronger.")
						}
					} else {
						if t, ok := world.GetTile(gx, gz); ok && t == TileCore {
							game.SetMessage("That rock is impenetrable. We need something stronger.")
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

		// Chest: open equipment UI when selected entity arrives at chest tile
		if game.Combat == nil && !game.EquipUIOpen && game.SelectedEnt >= 0 && game.SelectedEnt < len(game.Entities) {
			ent := game.Entities[game.SelectedEnt]
			onChest := ent.X == world.ChestX && ent.Z == world.ChestZ && ent.Task == nil
			if onChest && game.ChestUsedBy != game.SelectedEnt {
				// New entity arrived (or same entity walked away and back)
				world.ChestOpen = true
				game.EquipUIOpen = true
				game.EquipCursor = 0
				game.ChestUsedBy = game.SelectedEnt
				game.SetMessage(fmt.Sprintf("%s opens the equipment chest!", ent.Name))
			}
			if !onChest && game.ChestUsedBy == game.SelectedEnt {
				// Entity walked away — reset so they can use it again
				game.ChestUsedBy = -1
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
		// Threat step animation
		for key, threat := range world.Threats {
			if threat.Moving {
				threat.StepProgress += dt / stepInterval
				if threat.StepProgress >= 1.0 {
					threat.StepProgress = 1.0
					threat.Moving = false
				}
				world.Threats[key] = threat
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
							if game.Combat != nil {
								ent2 := game.Entities[game.Combat.Current().EntityIdx]
								game.ComputeMoveRange(world, ent2.X, ent2.Z, game.Combat.MoveLeft)
							}
						} else {
							c.PrimedAction = nil
							game.ComputeMoveRange(world, ent.X, ent.Z, c.MoveLeft)
							game.SetMessage("No valid target there.")
						}
					} else if world.IsWalkable(gx, gz) && !world.IsThreatAt(gx, gz) && c.MoveLeft > 0 {
						// Move to clicked tile
						path := FindPath(world, ent.X, ent.Z, gx, gz)
						if path != nil && len(path) <= c.MoveLeft {
							c.MoveLeft -= len(path)
							last := path[len(path)-1]
							// Face direction of last step
							dx, dz := last[0]-ent.X, last[1]-ent.Z
							if len(path) >= 2 {
								prev := path[len(path)-2]
								dx, dz = last[0]-prev[0], last[1]-prev[1]
							}
							ent.FacingAngle = FacingAngleFromDir(dx, dz)
							ent.PrevX, ent.PrevZ = ent.X, ent.Z
							ent.X, ent.Z = last[0], last[1]
							ent.StepProgress = 0
							ent.Moving = true
							game.ComputeMoveRange(world, ent.X, ent.Z, c.MoveLeft)
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
				ent := game.Entities[cur.EntityIdx]
				game.ComputeMoveRange(world, ent.X, ent.Z, c.MoveLeft)
			}
		}

		// Orders for selected entity: 1=Scout  2=Stop (continuous mode only)
		if game.Combat == nil && game.SelectedEnt >= 0 && game.PendingLevelUpEntity() < 0 {
			ent := game.Entities[game.SelectedEnt]
			if rl.IsKeyPressed(rl.KeyOne) {
				ent.Task = &Task{Type: TaskExplore}
				game.SetMessage(fmt.Sprintf("%s is now scouting.", ent.Name))
			}
			if rl.IsKeyPressed(rl.KeyTwo) {
				ent.Task = nil
				game.SetMessage(fmt.Sprintf("%s holding position.", ent.Name))
			}
			if rl.IsKeyPressed(rl.KeyThree) {
				ent.Task = nil
				game.TryRest(world, game.SelectedEnt)
			}
			if rl.IsKeyPressed(rl.KeyFive) {
				ent.Task = nil
				game.TryLongRest(world, game.SelectedEnt)
			}
			if rl.IsKeyPressed(rl.KeyFour) {
				// Toggle: if anyone is already following selected, stop all followers
				anyFollowing := false
				for j, other := range game.Entities {
					if j != game.SelectedEnt && other.Task != nil &&
						other.Task.Type == TaskFollow && other.Task.FollowIdx == game.SelectedEnt {
						anyFollowing = true
						break
					}
				}
				if anyFollowing {
					for j, other := range game.Entities {
						if j != game.SelectedEnt && other.Task != nil &&
							other.Task.Type == TaskFollow && other.Task.FollowIdx == game.SelectedEnt {
							other.Task = nil
						}
					}
					game.SetMessage("Companions holding position.")
				} else {
					count := 0
					for j, other := range game.Entities {
						if j != game.SelectedEnt {
							other.Task = &Task{Type: TaskFollow, FollowIdx: game.SelectedEnt}
							count++
						}
					}
					game.SetMessage(fmt.Sprintf("%d companions following %s.", count, ent.Name))
				}
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
		} // end if !levelUpPending && !gearPanelOpen

		// Message timer
		if game.MessageTimer > 0 {
			game.MessageTimer -= dt
		}
		game.TickFloats(dt)

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

		drawLocal(camera, world, game, dt, tileUnit, floorSurfaceY, knightYOffset, charScale, wallScale,
			floorVariant, gridToWorld,
			wallModel, pickaxeModel, chestModel, chestTopModel, heroModels, skeletonModels, propModels, game.Entities, game.SelectedEnt)

		rl.EndDrawing()
	}
}
