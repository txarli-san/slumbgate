package main

import "testing"

// --- Player ranged attacks require LOS ---

func TestRangedLOS_PlayerBlockedByWall(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)
	elara.Stats.Level = 3
	elara.Stats.KnownSpells = []string{"magic_missile"}

	// Place wall between Elara and enemy
	wallX := elara.X + 2
	w.SetTile(wallX, elara.Z, TileSolid)

	// Place enemy behind wall
	tx, tz := elara.X+4, elara.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 1, 0)

	// Prime a ranged attack
	actions := BuildActions(g, elara)
	var fireBolt *CombatAction
	for i := range actions {
		if actions[i].ID == "fire_bolt" {
			fireBolt = actions[i]
			break
		}
	}
	if fireBolt == nil {
		t.Fatal("fire bolt not found")
	}

	g.Combat.PrimedAction = fireBolt
	hpBefore := 8

	// Try to execute on target behind wall
	g.ExecutePrimedOnTarget(w, tx, tz)

	threat, ok := w.GetThreat(tx, tz)
	if !ok {
		t.Fatal("threat disappeared")
	}
	if threat.HP != hpBefore {
		t.Error("ranged attack should not hit through wall")
	}
}

func TestRangedLOS_PlayerClearShot(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)
	elara.Stats.Level = 3
	elara.Stats.KnownSpells = []string{"mind_spike"}

	// Clear path to enemy — no wall
	tx, tz := elara.X+3, elara.Z
	for x := elara.X + 1; x <= tx; x++ {
		w.SetTile(x, elara.Z, TileGround)
	}
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 50, MaxHP: 50, AC: 1, STR: 10, DEX: 10,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 1, 0)

	actions := BuildActions(g, elara)
	var mindSpike *CombatAction
	for i := range actions {
		if actions[i].ID == "mind_spike" {
			mindSpike = actions[i]
			break
		}
	}
	if mindSpike == nil {
		t.Fatal("mind spike not found")
	}

	g.Combat.PrimedAction = mindSpike

	// Fire at target with clear LOS
	g.ExecutePrimedOnTarget(w, tx, tz)

	threat, ok := w.GetThreat(tx, tz)
	if ok && threat.HP == 50 {
		t.Error("ranged attack with clear LOS should deal damage")
	}
}

// --- Enemy ranged attacks require LOS ---

func TestRangedLOS_EnemyBlockedByWall(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	startHP := brynn.Stats.HP

	// Place ranged enemy 4 tiles away
	tx, tz := brynn.X+4, brynn.Z
	for x := brynn.X + 1; x <= tx; x++ {
		w.SetTile(x, brynn.Z, TileGround)
	}
	// Put wall between them
	w.SetTile(brynn.X+2, brynn.Z, TileSolid)

	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMage, RoomIdx: 0,
		HP: 20, MaxHP: 20, AC: 13, STR: 10, DEX: 16,
		MoveSpeed: 4, AttackDice: 8, MaxRange: 5,
	}

	g.StartCombat(w, 0, 0)

	// Run enemy turn — should NOT be able to shoot through wall
	for i := 0; i < 10; i++ {
		if g.Combat == nil {
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			g.RunEnemyTurn(w)
			break
		}
		g.Combat.NextTurn(g, w)
	}

	if brynn.Stats.HP < startHP {
		// Enemy might have moved closer and attacked melee — check threat position
		threat, ok := w.GetThreat(tx, tz)
		if ok && threat.X == tx && threat.Z == tz {
			t.Error("enemy ranged attack hit through wall without moving")
		}
	}
}

func TestRangedLOS_EnemyClearShot(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Place ranged enemy 3 tiles away, clear path
	tx, tz := brynn.X+3, brynn.Z
	for x := brynn.X + 1; x <= tx; x++ {
		w.SetTile(x, brynn.Z, TileGround)
	}

	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMage, RoomIdx: 0,
		HP: 20, MaxHP: 20, AC: 13, STR: 10, DEX: 16,
		MoveSpeed: 4, AttackDice: 8, MaxRange: 5,
	}

	g.StartCombat(w, 0, 0)

	// Run many enemy turns to get at least one hit (AC can cause misses)
	hits := 0
	for i := 0; i < 50; i++ {
		if g.Combat == nil || g.GameOver {
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			hpBefore := brynn.Stats.HP
			g.RunEnemyTurn(w)
			if brynn.Stats.HP < hpBefore {
				hits++
			}
		} else {
			// Player just ends turn
			g.Combat.NextTurn(g, w)
		}
	}

	if hits == 0 && !g.GameOver {
		t.Error("ranged enemy with clear LOS should land at least one hit in 50 turns")
	}
}

// --- Attack range highlight respects LOS ---

func TestRangedLOS_HighlightFiltered(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Place wall 2 tiles to the right
	w.SetTile(brynn.X+2, brynn.Z, TileSolid)

	// Compute attack range with r=5 (ranged)
	g.ComputeAttackRange(w, brynn.X, brynn.Z, 5)

	// Tile behind wall should NOT be highlighted
	behindWall := [2]int{brynn.X + 3, brynn.Z}
	if g.AttackRange[behindWall] {
		t.Error("tile behind wall should not be in attack range highlight")
	}

	// Tile in clear direction should be highlighted
	clearTile := [2]int{brynn.X, brynn.Z + 3}
	if !g.AttackRange[clearTile] {
		t.Error("tile with clear LOS should be in attack range highlight")
	}
}

// --- Melee still works without LOS check (adjacent) ---

func TestRangedLOS_MeleeUnaffected(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Compute melee range (r=1)
	g.ComputeAttackRange(w, brynn.X, brynn.Z, 1)

	// All 8 adjacent tiles should be highlighted
	count := 0
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dz == 0 {
				continue
			}
			if g.AttackRange[[2]int{brynn.X + dx, brynn.Z + dz}] {
				count++
			}
		}
	}
	if count != 8 {
		t.Errorf("melee range should highlight all 8 adjacent tiles, got %d", count)
	}
}
