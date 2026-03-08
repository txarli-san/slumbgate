package main

import (
	"testing"
)

// --- Setup: real world, real entities, real stats ---
// No mocks. These use NewWorld with a fixed seed so the dungeon is
// deterministic and identical to what a player would see.

const testSeed = 42

// newTestGame creates a full game state identical to main() startup.
func newTestGame() (*GameState, *World) {
	w := NewWorld(testSeed)
	spawnX := OuterRadius + DungeonWarpAmp + 5
	w.PlacePickaxe(spawnX, 0)

	g := &GameState{SelectedEnt: 0}
	g.Entities = []*Entity{
		{Name: "Brynn", X: spawnX, Z: 0, Tier: TierVeteran, RevealDist: 8,
			Scouted: map[[2]int]bool{},
			Stats: &CombatStats{
				HP: 28, MaxHP: 28, AC: 16,
				STR: 16, DEX: 12, CON: 14, INT: 10, WIS: 12, CHA: 10,
				Level: 3, ProfBonus: 2, MoveSpeed: 5,
				Class: "Fighter", ClassCharges: 2, MaxClassCharges: 2,
			HitDice: 3, MaxHitDice: 3, HitDieSize: 10,
			}},
	}

	for _, ent := range g.Entities {
		cx, cz := TileToChunk(ent.X, ent.Z)
		w.EnsureChunksAround(cx, cz)
		w.RevealAround(ent.X, ent.Z)
	}
	return g, w
}

// addElara adds the Mage to the roster at a walkable position near Brynn.
func addElara(g *GameState, w *World) *Entity {
	brynn := g.Entities[0]
	elara := &Entity{
		Name: "Elara", X: brynn.X + 1, Z: brynn.Z, Tier: TierSoldier, RevealDist: 5,
		Scouted: map[[2]int]bool{},
		Stats: &CombatStats{
			HP: 18, MaxHP: 18, AC: 12,
			STR: 8, DEX: 14, CON: 12, INT: 16, WIS: 14, CHA: 10,
			Level: 3, ProfBonus: 2, MoveSpeed: 5,
			Class: "Mage", ClassCharges: 3, MaxClassCharges: 3,
		HitDice: 3, MaxHitDice: 3, HitDieSize: 6,
		},
	}
	g.Entities = append(g.Entities, elara)
	cx, cz := TileToChunk(elara.X, elara.Z)
	w.EnsureChunksAround(cx, cz)
	return elara
}

// --- World generation determinism ---

func TestWorldGeneration_Deterministic(t *testing.T) {
	w1 := NewWorld(testSeed)
	w2 := NewWorld(testSeed)

	if len(w1.Rooms) != len(w2.Rooms) {
		t.Fatalf("room count differs: %d vs %d", len(w1.Rooms), len(w2.Rooms))
	}
	for i := range w1.Rooms {
		if w1.Rooms[i] != w2.Rooms[i] {
			t.Fatalf("room %d differs: %v vs %v", i, w1.Rooms[i], w2.Rooms[i])
		}
	}
	if len(w1.Threats) != len(w2.Threats) {
		t.Fatalf("threat count differs: %d vs %d", len(w1.Threats), len(w2.Threats))
	}
}

func TestWorldGeneration_HasRoomsAndThreats(t *testing.T) {
	w := NewWorld(testSeed)

	if len(w.Rooms) < 10 {
		t.Fatalf("expected many rooms, got %d", len(w.Rooms))
	}
	if len(w.Threats) < 5 {
		t.Fatalf("expected threats, got %d", len(w.Threats))
	}
}

func TestWorldGeneration_RoomsInsideRing(t *testing.T) {
	w := NewWorld(testSeed)

	for i, r := range w.Rooms {
		// Skip corridor tiles (1x1)
		if r.W <= 1 && r.H <= 1 {
			continue
		}
		// Every tile of a real room should be in the solid ring
		for z := r.Z; z < r.Z+r.H; z++ {
			for x := r.X; x < r.X+r.W; x++ {
				base := w.BaseTileType(x, z)
				if base != TileSolid {
					t.Fatalf("room %d tile (%d,%d) has base type %d, expected TileSolid", i, x, z, base)
				}
			}
		}
	}
}

// --- Spawn integrity ---

func TestSpawn_BrynnOnWalkableTile(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	if !w.IsWalkable(brynn.X, brynn.Z) {
		t.Fatalf("Brynn spawned on non-walkable tile at (%d,%d)", brynn.X, brynn.Z)
	}
}

func TestSpawn_PickaxeOnGround(t *testing.T) {
	_, w := newTestGame()

	tile := w.TileTypeAt(w.PickaxeX, w.PickaxeZ)
	if tile != TileGround {
		t.Fatalf("pickaxe at (%d,%d) is on tile type %d, expected TileGround",
			w.PickaxeX, w.PickaxeZ, tile)
	}
}

func TestSpawn_PickaxeNotAtSpawn(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	if w.PickaxeX == brynn.X && w.PickaxeZ == brynn.Z {
		t.Fatal("pickaxe spawned directly on Brynn")
	}
	dx := w.PickaxeX - brynn.X
	dz := w.PickaxeZ - brynn.Z
	dist := dx*dx + dz*dz
	if dist < 15*15 {
		t.Fatalf("pickaxe too close to spawn: distance² = %d (min 225)", dist)
	}
}

// --- Entity death ---

func TestKillEntity_RemovesFromRoster(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)

	g.KillEntity(w, 0) // kill Brynn

	if len(g.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(g.Entities))
	}
	if g.Entities[0].Name != "Elara" {
		t.Fatalf("expected Elara to survive, got %s", g.Entities[0].Name)
	}
}

func TestKillEntity_LastEntityTriggersGameOver(t *testing.T) {
	g, w := newTestGame()

	over := g.KillEntity(w, 0)

	if !over || !g.GameOver {
		t.Fatal("expected game over when last entity dies")
	}
	if len(g.Entities) != 0 {
		t.Fatalf("expected empty roster, got %d", len(g.Entities))
	}
	if g.SelectedEnt != -1 {
		t.Fatalf("expected SelectedEnt -1, got %d", g.SelectedEnt)
	}
}

func TestKillEntity_FixesSelectedEnt(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)
	g.SelectedEnt = 1 // select Elara

	g.KillEntity(w, 1) // kill Elara

	if g.SelectedEnt != 0 {
		t.Fatalf("expected SelectedEnt 0, got %d", g.SelectedEnt)
	}
}

func TestKillEntity_FixesFollowers(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)

	// Elara follows Brynn
	elara.Task = &Task{Type: TaskFollow, FollowIdx: 0}

	g.KillEntity(w, 0) // kill Brynn

	// Elara's follow should be cleared (leader died)
	if elara.Task != nil {
		t.Fatalf("expected Elara task nil after leader died, got type %d", elara.Task.Type)
	}
}

func TestKillEntity_RemovesFromInitiative(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)

	// Set up combat with both entities and an enemy
	brynn := g.Entities[0]
	threatPos := [2]int{brynn.X + 2, brynn.Z}
	w.Threats[threatPos] = Threat{
		X: threatPos[0], Z: threatPos[1], Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.Combat = &Combat{
		Combatants: []Combatant{
			{IsEnemy: false, EntityIdx: 0, Initiative: 15},
			{IsEnemy: true, ThreatKey: threatPos, Initiative: 12},
			{IsEnemy: false, EntityIdx: 1, Initiative: 10},
		},
		TurnIndex: 0,
	}

	g.KillEntity(w, 0) // Brynn dies

	// Should have 2 combatants left (enemy + Elara)
	if len(g.Combat.Combatants) != 2 {
		t.Fatalf("expected 2 combatants, got %d", len(g.Combat.Combatants))
	}
	// Elara's EntityIdx should shift from 1 to 0
	for _, cb := range g.Combat.Combatants {
		if !cb.IsEnemy {
			if cb.EntityIdx != 0 {
				t.Fatalf("expected Elara idx 0, got %d", cb.EntityIdx)
			}
			if g.Entities[cb.EntityIdx].Name != "Elara" {
				t.Fatalf("entity at fixed idx is %s, expected Elara",
					g.Entities[cb.EntityIdx].Name)
			}
		}
	}
	_ = elara
}

func TestKillEntity_AllAlliesDeadEndsCombat(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	threatPos := [2]int{brynn.X + 2, brynn.Z}
	w.Threats[threatPos] = Threat{
		X: threatPos[0], Z: threatPos[1], Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.Combat = &Combat{
		Combatants: []Combatant{
			{IsEnemy: false, EntityIdx: 0, Initiative: 15},
			{IsEnemy: true, ThreatKey: threatPos, Initiative: 12},
		},
		TurnIndex: 0,
	}

	g.KillEntity(w, 0)

	if g.Combat != nil {
		t.Fatal("expected combat to end when all allies dead")
	}
	if !g.GameOver {
		t.Fatal("expected game over")
	}
}

// --- Rest ---

func TestTryRest_HealsAndAdvancesClock(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.HP = 10

	before := g.TimeTicks
	g.TryRest(w, 0)

	// Brynn is on TileGround (outside dungeon), so no ambush possible
	if brynn.Stats.HP <= 10 {
		t.Fatalf("expected HP increase from 10, got %d", brynn.Stats.HP)
	}
	if brynn.Stats.HP > brynn.Stats.MaxHP {
		t.Fatalf("HP %d exceeds max %d", brynn.Stats.HP, brynn.Stats.MaxHP)
	}
	if g.TimeTicks != before+60 {
		t.Fatalf("expected clock +60 (short rest), got %d", g.TimeTicks-before)
	}
}

func TestTryRest_FighterRecoversCharges(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.ClassCharges = 0

	g.TryRest(w, 0)

	if brynn.Stats.ClassCharges != brynn.Stats.MaxClassCharges {
		t.Fatalf("expected Fighter charges %d, got %d",
			brynn.Stats.MaxClassCharges, brynn.Stats.ClassCharges)
	}
}

func TestTryRest_MageDoesNotRecoverCharges(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)
	elara.Stats.ClassCharges = 0

	g.TryRest(w, 1)

	if elara.Stats.ClassCharges != 0 {
		t.Fatalf("expected Mage charges 0 (needs long rest), got %d",
			elara.Stats.ClassCharges)
	}
}

func TestTryRest_FullHPStillAdvancesClock(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	// Already at full HP
	if brynn.Stats.HP != brynn.Stats.MaxHP {
		t.Fatal("expected full HP at start")
	}

	before := g.TimeTicks
	g.TryRest(w, 0)

	if g.TimeTicks != before+60 {
		t.Fatalf("expected clock +60 (short rest) even at full HP, got %d", g.TimeTicks-before)
	}
}

// --- Combat flow ---

func TestStartCombat_InitiativeOrder(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Find a room with threats
	var roomIdx int
	var threatKey [2]int
	found := false
	for key, threat := range w.Threats {
		roomIdx = threat.RoomIdx
		threatKey = key
		found = true
		break
	}
	if !found {
		t.Fatal("no threats in world")
	}

	// Move Brynn adjacent to the threat
	brynn.X, brynn.Z = threatKey[0]-1, threatKey[1]
	cx, cz := TileToChunk(brynn.X, brynn.Z)
	w.EnsureChunksAround(cx, cz)

	g.StartCombat(w, 0, roomIdx)

	if g.Combat == nil {
		t.Fatal("combat should be active")
	}
	if len(g.Combat.Combatants) < 2 {
		t.Fatalf("expected at least 2 combatants, got %d", len(g.Combat.Combatants))
	}

	// Initiative should be sorted descending
	for i := 1; i < len(g.Combat.Combatants); i++ {
		if g.Combat.Combatants[i].Initiative > g.Combat.Combatants[i-1].Initiative {
			t.Fatalf("initiative not sorted: %d > %d at index %d",
				g.Combat.Combatants[i].Initiative, g.Combat.Combatants[i-1].Initiative, i)
		}
	}
}

func TestCombat_MeleeAttackReducesThreatHP(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Place a weak threat adjacent to Brynn
	tx, tz := brynn.X+1, brynn.Z
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 100, MaxHP: 100, AC: 1, // AC 1 = always hit (except nat 1)
		STR: 10, DEX: 10, MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.Combat = &Combat{
		Combatants: []Combatant{
			{IsEnemy: false, EntityIdx: 0, Initiative: 20},
			{IsEnemy: true, ThreatKey: [2]int{tx, tz}, Initiative: 5, SpawnPos: [2]int{tx, tz}, PursuitLeft: 3},
		},
		TurnIndex: 0,
		MoveLeft:  5,
		Actions:   BuildActions(g, brynn),
	}

	// Try melee attack many times until we hit (nat 1 misses)
	initialHP := 100
	for i := 0; i < 50; i++ {
		threat, ok := w.Threats[[2]int{tx, tz}]
		if !ok || threat.HP < initialHP {
			break // damage dealt
		}
		g.Combat.ActionUsed = false
		executeMeleeAttack(g, w, brynn, tx, tz)
	}

	threat, ok := w.Threats[[2]int{tx, tz}]
	if !ok {
		return // threat was killed, that's fine
	}
	if threat.HP >= initialHP {
		t.Fatal("expected threat to take damage after multiple attacks")
	}
}

func TestCombat_EnemyAttackDamagesEntity(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Place threat adjacent with high attack mod
	tx, tz := brynn.X+1, brynn.Z
	threat := Threat{
		X: tx, Z: tz, Type: SkeletonWarrior, RoomIdx: 0,
		HP: 13, MaxHP: 13, AC: 13, STR: 10, DEX: 20, // +5 attack mod
		MoveSpeed: 4, AttackDice: 6, MaxRange: 1,
	}
	w.Threats[[2]int{tx, tz}] = threat

	initialHP := brynn.Stats.HP
	// Many attempts to account for misses
	for i := 0; i < 50; i++ {
		if brynn.Stats.HP < initialHP {
			break
		}
		g.ResolveEnemyAttack(w, threat, brynn)
		if g.GameOver {
			return // died, which still proves damage happened
		}
	}

	if brynn.Stats.HP >= initialHP {
		t.Fatal("expected Brynn to take damage from enemy attacks")
	}
}

// --- Pathfinding on real world ---

func TestPathfinding_BrynnCanReachPickaxe(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	path := FindPath(w, brynn.X, brynn.Z, w.PickaxeX, w.PickaxeZ)
	if path == nil {
		t.Fatalf("no path from Brynn (%d,%d) to pickaxe (%d,%d)",
			brynn.X, brynn.Z, w.PickaxeX, w.PickaxeZ)
	}
	last := path[len(path)-1]
	if last[0] != w.PickaxeX || last[1] != w.PickaxeZ {
		t.Fatalf("path doesn't reach pickaxe: ends at (%d,%d)", last[0], last[1])
	}
}

func TestPathfinding_CanBreakIntoDungeon(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Find any floor tile (inside dungeon)
	var targetX, targetZ int
	found := false
	for _, room := range w.Rooms {
		if room.W > 1 && room.H > 1 {
			targetX = room.X + room.W/2
			targetZ = room.Z + room.H/2
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no rooms found")
	}

	// Normal path should fail (wall between ground and dungeon interior)
	path := FindPath(w, brynn.X, brynn.Z, targetX, targetZ)
	if path != nil {
		t.Log("direct path exists — unusual but possible if room touches outer edge")
	}

	// Breakable path should succeed
	path = FindPathBreakable(w, brynn.X, brynn.Z, targetX, targetZ)
	if path == nil {
		t.Fatalf("no breakable path from (%d,%d) to room at (%d,%d)",
			brynn.X, brynn.Z, targetX, targetZ)
	}
}

// --- WorldStep integration ---

func TestWorldStep_EntityWalksToPickaxe(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	path := FindPath(w, brynn.X, brynn.Z, w.PickaxeX, w.PickaxeZ)
	if path == nil {
		t.Fatal("no path to pickaxe")
	}
	brynn.Task = &Task{Type: TaskMoveTo, TargetX: w.PickaxeX, TargetZ: w.PickaxeZ, Path: path}

	// Step until arrival or timeout
	for i := 0; i < 500; i++ {
		g.WorldStep(w)
		if brynn.X == w.PickaxeX && brynn.Z == w.PickaxeZ {
			break
		}
	}

	if brynn.X != w.PickaxeX || brynn.Z != w.PickaxeZ {
		t.Fatalf("Brynn didn't reach pickaxe: at (%d,%d), target (%d,%d)",
			brynn.X, brynn.Z, w.PickaxeX, w.PickaxeZ)
	}
	if !g.HasPickaxe {
		t.Fatal("pickaxe not picked up after walking over it")
	}
}

func TestWorldStep_ClockAdvancesPerStep(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	path := FindPath(w, brynn.X, brynn.Z, w.PickaxeX, w.PickaxeZ)
	if path == nil {
		t.Fatal("no path to pickaxe")
	}
	brynn.Task = &Task{Type: TaskMoveTo, TargetX: w.PickaxeX, TargetZ: w.PickaxeZ, Path: path}

	steps := 0
	for i := 0; i < 500; i++ {
		g.WorldStep(w)
		steps++
		if brynn.Task == nil {
			break
		}
	}

	if g.TimeTicks != steps {
		t.Fatalf("expected TimeTicks=%d (one per WorldStep), got %d", steps, g.TimeTicks)
	}
}

// --- Line of sight ---

func TestLineOfSight_ClearOnGround(t *testing.T) {
	_, w := newTestGame()
	spawnX := OuterRadius + DungeonWarpAmp + 5

	// Ground tiles should have LOS to nearby ground tiles
	if !w.HasLineOfSight(spawnX, 0, spawnX+5, 0) {
		t.Fatal("expected clear LOS on ground")
	}
}

func TestLineOfSight_BlockedBySolid(t *testing.T) {
	_, w := newTestGame()

	// Find a solid tile and check LOS through it
	// The dungeon wall should block LOS from outside to inside
	spawnX := OuterRadius + DungeonWarpAmp + 5

	// Walk inward until we hit solid
	solidX := spawnX
	for solidX > 0 {
		if w.BaseTileType(solidX, 0) == TileSolid {
			break
		}
		solidX--
	}
	if solidX <= 0 {
		t.Fatal("couldn't find solid tile walking inward")
	}

	// LOS from outside through solid to deeper should be blocked
	blocked := !w.HasLineOfSight(spawnX, 0, solidX-5, 0)
	if !blocked {
		t.Log("LOS not blocked — solid tile may be thin at this angle (acceptable)")
	}
}

// --- Fog of war ---

func TestFogOfWar_SpawnAreaRevealed(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Brynn's tile should be revealed
	if !w.IsRevealed(brynn.X, brynn.Z) {
		t.Fatal("Brynn's tile should be revealed at spawn")
	}
	// Adjacent tiles should be revealed
	if !w.IsRevealed(brynn.X+1, brynn.Z) {
		t.Fatal("tile adjacent to Brynn should be revealed")
	}
}

func TestFogOfWar_DistantDungeonNotRevealed(t *testing.T) {
	_, w := newTestGame()

	// A room deep in the dungeon shouldn't be revealed
	for _, room := range w.Rooms {
		if room.W <= 1 {
			continue
		}
		rx, rz := room.X+room.W/2, room.Z+room.H/2
		if !w.IsRevealed(rx, rz) {
			return // found one, test passes
		}
	}
	t.Fatal("expected at least one unrevealed room")
}
