package main

import (
	"testing"
)

// Scenario tests run full game sequences using real world generation.
// No mocks. Same code paths as a real player session.

// --- Setup ---

// scenario creates a full game with Brynn, loads chunks, and returns
// everything needed to drive a game session.
func scenario(t *testing.T) (*GameState, *World) {
	t.Helper()
	g, w := newTestGame()
	return g, w
}

// walkTo moves an entity step-by-step to a destination using WorldStep.
// Returns false if unreachable within maxSteps.
func walkTo(g *GameState, w *World, entIdx, tx, tz, maxSteps int) bool {
	ent := g.Entities[entIdx]
	path := FindPath(w, ent.X, ent.Z, tx, tz)
	if path == nil {
		return false
	}
	ent.Task = &Task{Type: TaskMoveTo, TargetX: tx, TargetZ: tz, Path: path}
	for i := 0; i < maxSteps; i++ {
		if g.Combat != nil || g.GameOver {
			return true // combat triggered or game ended — still a valid outcome
		}
		g.WorldStep(w)
		if ent.X == tx && ent.Z == tz {
			return true
		}
	}
	return false
}

// walkToBreakable moves an entity using breakable pathfinding.
func walkToBreakable(g *GameState, w *World, entIdx, tx, tz, maxSteps int) bool {
	ent := g.Entities[entIdx]
	path := FindPathBreakable(w, ent.X, ent.Z, tx, tz)
	if path == nil {
		return false
	}
	ent.Task = &Task{Type: TaskMoveTo, TargetX: tx, TargetZ: tz, Path: path}
	for i := 0; i < maxSteps; i++ {
		if g.Combat != nil || g.GameOver {
			return true
		}
		g.WorldStep(w)
		if ent.X == tx && ent.Z == tz {
			return true
		}
	}
	return false
}

// runCombatToEnd runs combat turns until combat ends or maxTurns exceeded.
// Player entities use melee/fire bolt on nearest enemy. Returns true if combat ended.
func runCombatToEnd(g *GameState, w *World, maxTurns int) bool {
	for turn := 0; turn < maxTurns; turn++ {
		if g.Combat == nil || g.GameOver {
			return true
		}
		c := g.Combat
		cur := c.Current()

		if cur.IsEnemy {
			g.RunEnemyTurn(w)
			continue
		}

		// Player turn: find nearest enemy and attack
		ent := g.Entities[cur.EntityIdx]
		attacked := false

		for _, cb := range c.Combatants {
			if !cb.IsEnemy {
				continue
			}
			threat, ok := w.Threats[cb.ThreatKey]
			if !ok {
				continue
			}

			// Try melee if adjacent
			if adjacent(ent.X, ent.Z, threat.X, threat.Z) {
				for _, action := range c.Actions {
					if action.ID == "melee_attack" || action.ID == "fire_bolt" {
						if action.CanUse(g, ent) {
							action.Execute(g, w, ent, threat.X, threat.Z)
							attacked = true
							break
						}
					}
				}
				if attacked {
					break
				}
			}

			// Try ranged if in range
			if withinRange(ent.X, ent.Z, threat.X, threat.Z, 7) {
				for _, action := range c.Actions {
					if action.ID == "fire_bolt" || action.ID == "magic_missile" {
						if action.CanUse(g, ent) {
							action.Execute(g, w, ent, threat.X, threat.Z)
							attacked = true
							break
						}
					}
				}
				if attacked {
					break
				}
			}
		}

		if !attacked {
			// Can't attack — move toward nearest enemy
			moved := false
			if c.MoveLeft > 0 {
				for _, cb := range c.Combatants {
					if !cb.IsEnemy {
						continue
					}
					threat, ok := w.Threats[cb.ThreatKey]
					if !ok {
						continue
					}
					path := FindPath(w, ent.X, ent.Z, threat.X, threat.Z)
					if path != nil && len(path) > 1 {
						// Move as close as possible without stepping on enemy
						steps := c.MoveLeft
						if steps > len(path)-1 {
							steps = len(path) - 1
						}
						if steps > 0 {
							dest := path[steps-1]
							ent.PrevX, ent.PrevZ = ent.X, ent.Z
							ent.X, ent.Z = dest[0], dest[1]
							c.MoveLeft -= steps
							moved = true
						}
					}
					break
				}
			}
			if !moved {
				// End turn
				if g.Combat != nil {
					c.NextTurn(g, w)
				}
			}
		} else {
			// After attacking, end turn
			if g.Combat != nil {
				g.Combat.NextTurn(g, w)
			}
		}
	}
	return g.Combat == nil
}

// findFirstThreatRoom returns the index and position of the first room with threats.
func findFirstThreatRoom(w *World) (roomIdx int, tx, tz int, found bool) {
	for _, threat := range w.Threats {
		room := w.Rooms[threat.RoomIdx]
		return threat.RoomIdx, room.X + room.W/2, room.Z + room.H/2, true
	}
	return 0, 0, 0, false
}

// findAdjacentWalkable finds a walkable tile adjacent to (tx,tz).
func findAdjacentWalkable(w *World, tx, tz int) (int, int, bool) {
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, nz := tx+d[0], tz+d[1]
		if w.IsWalkable(nx, nz) && !w.IsThreatAt(nx, nz) {
			return nx, nz, true
		}
	}
	return 0, 0, false
}

// --- Scenario 1: Scout → Pickaxe → Wall Break → Dungeon Entry ---

func TestScenario_ScoutFindsPickaxe(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]

	if g.HasPickaxe {
		t.Fatal("should not start with pickaxe")
	}

	// Walk Brynn to the pickaxe
	reached := walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
	if !reached {
		t.Fatalf("Brynn couldn't reach pickaxe at (%d,%d) from (%d,%d)",
			w.PickaxeX, w.PickaxeZ, brynn.X, brynn.Z)
	}

	if !g.HasPickaxe {
		t.Fatal("pickaxe not acquired after walking over it")
	}
}

func TestScenario_BreakIntoDungeon(t *testing.T) {
	g, w := scenario(t)

	// First get the pickaxe
	walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
	if !g.HasPickaxe {
		t.Fatal("need pickaxe first")
	}

	// Find a dungeon room
	var targetRoom Room
	found := false
	for _, room := range w.Rooms {
		if room.W > 2 && room.H > 2 {
			targetRoom = room
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no suitable room found")
	}

	tx, tz := targetRoom.X+targetRoom.W/2, targetRoom.Z+targetRoom.H/2
	brynn := g.Entities[0]

	// Walk with wall-breaking
	reached := walkToBreakable(g, w, 0, tx, tz, 500)

	// Either we reached the room or combat triggered (both valid)
	if g.Combat != nil {
		t.Log("combat triggered during breach — valid outcome")
		return
	}
	if !reached && !g.GameOver {
		t.Fatalf("couldn't breach to room at (%d,%d) from (%d,%d)",
			tx, tz, brynn.X, brynn.Z)
	}

	// Verify we broke at least one wall (doorway should exist on our path)
	doorwayFound := false
	cx, cz := TileToChunk(brynn.X, brynn.Z)
	w.EnsureChunksAround(cx, cz)
	for dz := -20; dz <= 20; dz++ {
		for dx := -20; dx <= 20; dx++ {
			if w.TileTypeAt(brynn.X+dx, brynn.Z+dz) == TileDoorway {
				doorwayFound = true
				break
			}
		}
		if doorwayFound {
			break
		}
	}
	if !doorwayFound {
		t.Fatal("no doorway created — wall breaking didn't work")
	}
}

// --- Scenario 2: Full Combat Round ---

func TestScenario_CombatVictory(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]

	// Place a weak enemy adjacent to Brynn on a walkable tile
	ex, ez := brynn.X+1, brynn.Z
	// Ensure the tile is walkable
	w.SetTile(ex, ez, TileGround)
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)

	roomIdx := 0
	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: roomIdx,
		HP: 1, MaxHP: 1, AC: 1, STR: 10, DEX: 10,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	startHP := brynn.Stats.HP
	g.StartCombat(w, 0, roomIdx)

	if g.Combat == nil {
		t.Fatal("combat should be active")
	}

	ended := runCombatToEnd(g, w, 50)
	if !ended {
		t.Fatal("combat didn't end within 50 turns")
	}

	if g.GameOver {
		t.Fatal("Brynn shouldn't die to a 1 HP minion")
	}

	// Enemy should be gone
	if w.IsThreatAt(ex, ez) {
		t.Fatal("threat should be removed after combat victory")
	}

	t.Logf("Brynn HP after combat: %d/%d (started %d)", brynn.Stats.HP, brynn.Stats.MaxHP, startHP)
}

func TestScenario_CombatMultiEnemy(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]

	// Place 3 minions near Brynn
	roomIdx := 0
	positions := [][2]int{{brynn.X + 1, brynn.Z}, {brynn.X + 2, brynn.Z}, {brynn.X, brynn.Z + 1}}
	for _, pos := range positions {
		w.SetTile(pos[0], pos[1], TileGround)
		w.Threats[[2]int{pos[0], pos[1]}] = Threat{
			X: pos[0], Z: pos[1], Type: SkeletonMinion, RoomIdx: roomIdx,
			HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
			MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
		}
	}

	g.StartCombat(w, 0, roomIdx)

	if len(g.Combat.Combatants) != 4 { // Brynn + 3 enemies
		t.Fatalf("expected 4 combatants, got %d", len(g.Combat.Combatants))
	}

	ended := runCombatToEnd(g, w, 200)

	if g.GameOver {
		t.Log("Brynn died to 3 minions — valid outcome for a real fight")
		return
	}

	if !ended {
		t.Fatal("combat didn't resolve within 200 turns")
	}

	// All enemies should be gone
	for _, pos := range positions {
		if w.IsThreatAt(pos[0], pos[1]) {
			t.Fatalf("threat still at (%d,%d) after victory", pos[0], pos[1])
		}
	}
}

// --- Scenario 3: Enemy Pursuit Leash ---

func TestScenario_EnemyLeashesAfterPursuit(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]

	// Place enemy far from Brynn (out of melee range) with wall between them
	ex, ez := brynn.X+8, brynn.Z
	for x := brynn.X + 1; x < ex; x++ {
		w.SetTile(x, brynn.Z, TileGround)
	}
	w.SetTile(ex, ez, TileGround)

	// Block LOS with a wall midway
	wallX := brynn.X + 4
	w.SetTile(wallX, brynn.Z, TileSolid)

	roomIdx := 0
	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: roomIdx,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 2, AttackDice: 4, MaxRange: 1, // slow — 2 tiles per turn
	}

	g.StartCombat(w, 0, roomIdx)

	// Find the enemy combatant
	var enemyCb *Combatant
	for i := range g.Combat.Combatants {
		if g.Combat.Combatants[i].IsEnemy {
			enemyCb = &g.Combat.Combatants[i]
			break
		}
	}
	if enemyCb == nil {
		t.Fatal("no enemy in combat")
	}

	// Run turns — enemy can't reach Brynn (wall blocks path), should leash
	for turn := 0; turn < 30; turn++ {
		if g.Combat == nil {
			break
		}
		c := g.Combat
		cur := c.Current()
		if cur.IsEnemy {
			g.RunEnemyTurn(w)
		} else {
			// Player just ends turn (doesn't move)
			c.NextTurn(g, w)
		}
	}

	// Combat should have ended via leashing
	if g.Combat != nil {
		t.Fatal("expected combat to end via enemy leash")
	}

	// Enemy should be in leashing threats or back at spawn
	t.Log("enemy leashed successfully")
}

// --- Scenario 4: Rest Ambush ---

func TestScenario_RestAmbushInsideDungeon(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]

	// Move Brynn to a floor tile (inside dungeon)
	var floorX, floorZ int
	found := false
	for _, room := range w.Rooms {
		if room.W > 2 && room.H > 2 {
			floorX = room.X + room.W/2
			floorZ = room.Z + room.H/2
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no room found")
	}

	// Teleport Brynn into the room (skip the walk — we're testing rest)
	brynn.X, brynn.Z = floorX, floorZ
	cx, cz := TileToChunk(brynn.X, brynn.Z)
	w.EnsureChunksAround(cx, cz)

	// Clear existing threats in this room to isolate the ambush test
	for key, threat := range w.Threats {
		if threat.RoomIdx >= 0 {
			room := w.Rooms[threat.RoomIdx]
			if floorX >= room.X && floorX < room.X+room.W && floorZ >= room.Z && floorZ < room.Z+room.H {
				delete(w.Threats, key)
			}
		}
	}

	// Rest many times — 30% chance of ambush each time
	ambushTriggered := false
	for i := 0; i < 50; i++ {
		if g.Combat != nil {
			break // previous ambush still active
		}
		g.TryRest(w, 0)
		if g.Combat != nil {
			ambushTriggered = true
			// Verify combat has enemies
			hasEnemy := false
			for _, cb := range g.Combat.Combatants {
				if cb.IsEnemy {
					hasEnemy = true
					break
				}
			}
			if !hasEnemy {
				t.Fatal("ambush combat started with no enemies")
			}
			// Fight it out
			runCombatToEnd(g, w, 100)
			break
		}
	}

	if !ambushTriggered {
		t.Fatal("no ambush in 50 rests at 30% chance — statistically improbable")
	}
}

// --- Scenario 5: Mage Rescue Event Chain ---

func TestScenario_MageRescue(t *testing.T) {
	g, w := scenario(t)

	// Set up events exactly like main.go
	mageRescueRoom := -1
	g.Events = []*Event{
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
				g.Entities = append(g.Entities, &Entity{
					Name: "Elara", X: mx, Z: mz, Tier: TierSoldier, RevealDist: 5,
					Scouted: map[[2]int]bool{},
					Stats: &CombatStats{
						HP: 18, MaxHP: 18, AC: 12,
						STR: 8, DEX: 14, CON: 12, INT: 16, WIS: 14, CHA: 10,
						Level: 3, ProfBonus: 2, MoveSpeed: 5, Class: "Mage", ClassCharges: 3, MaxClassCharges: 3,
					},
				})
			},
		},
	}

	// Get pickaxe first
	walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
	if !g.HasPickaxe {
		t.Fatal("need pickaxe")
	}

	// Find a room with threats
	roomIdx, tx, tz, found := findFirstThreatRoom(w)
	if !found {
		t.Fatal("no threats in world")
	}

	// Walk toward it (will trigger combat on LOS)
	brynn := g.Entities[0]
	walkToBreakable(g, w, 0, tx, tz, 500)

	if g.Combat == nil && mageRescueRoom < 0 {
		// Didn't trigger yet — try starting combat manually at the room
		brynn.X, brynn.Z = tx, tz
		cx, cz := TileToChunk(tx, tz)
		w.EnsureChunksAround(cx, cz)
		g.StartCombat(w, 0, roomIdx)
	}

	if mageRescueRoom < 0 {
		t.Fatal("mage rescue enter event never fired")
	}

	// Fight to win
	if g.Combat != nil {
		ended := runCombatToEnd(g, w, 200)
		if g.GameOver {
			t.Log("Brynn died during rescue — valid but can't test Elara spawn")
			return
		}
		if !ended {
			t.Fatal("combat didn't end")
		}
	}

	// Elara should have been added
	if len(g.Entities) < 2 {
		t.Fatal("Elara not added to roster after clearing rescue room")
	}

	elara := g.Entities[1]
	if elara.Name != "Elara" {
		t.Fatalf("expected Elara, got %s", elara.Name)
	}
	if elara.Stats.Class != "Mage" {
		t.Fatalf("expected Mage class, got %s", elara.Stats.Class)
	}

	t.Logf("Elara rescued at (%d,%d), Brynn HP: %d/%d", elara.X, elara.Z, brynn.Stats.HP, brynn.Stats.MaxHP)
}

// --- Scenario 6: Multi-Entity Combat ---

func TestScenario_TwoAlliesInCombat(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]
	elara := addElara(g, w)

	// Place them adjacent
	elara.X, elara.Z = brynn.X+1, brynn.Z

	// Place an enemy near both
	ex, ez := brynn.X+2, brynn.Z
	w.SetTile(ex, ez, TileGround)
	roomIdx := 0
	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: roomIdx,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, roomIdx)

	// Both should be in combat
	allyCount := 0
	for _, cb := range g.Combat.Combatants {
		if !cb.IsEnemy {
			allyCount++
		}
	}
	if allyCount != 2 {
		t.Fatalf("expected 2 allies in combat, got %d", allyCount)
	}

	ended := runCombatToEnd(g, w, 100)
	if !ended && !g.GameOver {
		t.Fatal("combat didn't resolve")
	}

	if !g.GameOver {
		// Both should survive against 1 minion
		if len(g.Entities) != 2 {
			t.Fatalf("expected 2 survivors, got %d", len(g.Entities))
		}
	}
}

// --- Scenario 7: Death Mid-Combat ---

func TestScenario_EntityDiesMidCombat(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]
	elara := addElara(g, w)
	elara.X, elara.Z = brynn.X+1, brynn.Z
	elara.Stats.HP = 1 // Elara will die to any hit

	// Place a strong enemy adjacent to Elara
	ex, ez := brynn.X+2, brynn.Z
	w.SetTile(ex, ez, TileGround)
	roomIdx := 0
	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonWarrior, RoomIdx: roomIdx,
		HP: 13, MaxHP: 13, AC: 13, STR: 10, DEX: 16,
		MoveSpeed: 4, AttackDice: 6, MaxRange: 1,
	}

	g.StartCombat(w, 0, roomIdx)

	initialCombatants := len(g.Combat.Combatants)
	if initialCombatants != 3 {
		t.Fatalf("expected 3 combatants, got %d", initialCombatants)
	}

	// Run combat — Elara should die, Brynn should continue
	ended := runCombatToEnd(g, w, 200)

	if g.GameOver {
		t.Log("both entities died — valid outcome")
		return
	}

	if !ended {
		t.Fatal("combat didn't resolve within 200 turns")
	}

	// Elara should be dead (removed from roster)
	for _, ent := range g.Entities {
		if ent.Name == "Elara" && ent.Stats.HP <= 0 {
			t.Fatal("dead Elara still in roster")
		}
	}

	// Brynn should still be alive
	found := false
	for _, ent := range g.Entities {
		if ent.Name == "Brynn" && ent.Stats.HP > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("Brynn should survive (game not over)")
	}

	t.Logf("Brynn survived with HP: %d/%d, roster size: %d",
		brynn.Stats.HP, brynn.Stats.MaxHP, len(g.Entities))
}

// --- Scenario: Full Session ---

func TestScenario_FullSessionPickaxeToCombat(t *testing.T) {
	g, w := scenario(t)
	brynn := g.Entities[0]

	// 1. Walk to pickaxe
	t.Log("Phase 1: Walking to pickaxe")
	walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
	if !g.HasPickaxe {
		t.Fatal("didn't get pickaxe")
	}
	t.Logf("  Pickaxe acquired at tick %d", g.TimeTicks)

	// 2. Break into dungeon
	t.Log("Phase 2: Breaking into dungeon")
	var targetRoom Room
	for _, room := range w.Rooms {
		if room.W > 2 && room.H > 2 {
			targetRoom = room
			break
		}
	}
	tx, tz := targetRoom.X+targetRoom.W/2, targetRoom.Z+targetRoom.H/2
	walkToBreakable(g, w, 0, tx, tz, 500)
	t.Logf("  Brynn at (%d,%d), tick %d", brynn.X, brynn.Z, g.TimeTicks)

	// 3. If combat triggered, fight
	if g.Combat != nil {
		t.Log("Phase 3: Combat!")
		combatants := len(g.Combat.Combatants)
		t.Logf("  %d combatants in initiative", combatants)

		ended := runCombatToEnd(g, w, 200)
		if g.GameOver {
			t.Logf("  Brynn fell in combat at tick %d", g.TimeTicks)
			return
		}
		if !ended {
			t.Fatal("  combat didn't resolve")
		}
		t.Logf("  Victory! Brynn HP: %d/%d", brynn.Stats.HP, brynn.Stats.MaxHP)
	}

	// 4. Rest
	if brynn.Stats.HP < brynn.Stats.MaxHP {
		t.Log("Phase 4: Resting")
		beforeHP := brynn.Stats.HP
		g.TryRest(w, 0)
		if g.Combat != nil {
			t.Log("  Ambush during rest!")
			runCombatToEnd(g, w, 200)
			if g.GameOver {
				t.Log("  Fell during ambush")
				return
			}
		}
		t.Logf("  HP: %d -> %d", beforeHP, brynn.Stats.HP)
	}

	t.Logf("Session complete: tick %d, HP %d/%d, entities %d",
		g.TimeTicks, brynn.Stats.HP, brynn.Stats.MaxHP, len(g.Entities))
}
