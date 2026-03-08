package main

import (
	"testing"
)

// Enemy AI behavior tests: pursuit, targeting, pathfinding, leash mechanics.

// Test 1: Enemy 3 tiles away, Brynn stands still.
// Enemy has MoveSpeed 4 — should reach and attack on turn 1.
func TestAI_EnemyReachesBrynnFromDistance(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	ex, ez := brynn.X+3, brynn.Z
	for dx := 1; dx <= 3; dx++ {
		w.SetTile(brynn.X+dx, brynn.Z, TileGround)
	}
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)

	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
		HP: 50, MaxHP: 50, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	startHP := brynn.Stats.HP
	for turn := 0; turn < 20; turn++ {
		if g.Combat == nil || g.GameOver {
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			threat, ok := w.Threats[cur.ThreatKey]
			if ok {
				dist := abs(threat.X-brynn.X) + abs(threat.Z-brynn.Z)
				t.Logf("turn %d: enemy at (%d,%d) dist=%d pursuit=%d",
					turn, threat.X, threat.Z, dist, cur.PursuitLeft)
			}
			g.RunEnemyTurn(w)
			if brynn.Stats.HP < startHP {
				t.Logf("  → Brynn hit! HP %d→%d", startHP, brynn.Stats.HP)
				startHP = brynn.Stats.HP
			}
		} else {
			g.Combat.NextTurn(g, w)
		}
	}

	if brynn.Stats.HP == 28 && !g.GameOver {
		t.Error("enemy never attacked Brynn despite clear 3-tile path")
	}
}

// Test 2: Enemy truly walled off — no path at all.
// Surround enemy in a box so there's no way around.
func TestAI_EnemyLeashesWhenBlocked(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Place enemy 5 tiles away
	ex, ez := brynn.X+5, brynn.Z

	// Create a walkable cell for enemy, surrounded by solid on all sides
	w.SetTile(ex, ez, TileGround)
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		w.SetTile(ex+d[0], ez+d[1], TileSolid)
	}
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)

	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
		HP: 50, MaxHP: 50, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	leashed := false
	for turn := 0; turn < 30; turn++ {
		if g.Combat == nil {
			leashed = true
			t.Logf("turn %d: LEASHED", turn)
			break
		}
		if g.GameOver {
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			t.Logf("turn %d: pursuit=%d", turn, cur.PursuitLeft)
			g.RunEnemyTurn(w)
		} else {
			g.Combat.NextTurn(g, w)
		}
	}

	if !leashed {
		t.Error("enemy never leashed despite being boxed in")
	}
}

// Test 3: Two allies at different distances. Enemy should target nearest.
func TestAI_EnemyTargetsNearest(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)
	brynn := g.Entities[0]
	elara := g.Entities[1]

	ex, ez := brynn.X+5, brynn.Z
	for dx := 1; dx <= 5; dx++ {
		w.SetTile(brynn.X+dx, brynn.Z, TileGround)
	}
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)

	// Elara adjacent to enemy, Brynn far
	elara.X, elara.Z = ex-1, ez

	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
		HP: 50, MaxHP: 50, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	elaraStartHP := elara.Stats.HP
	brynnStartHP := brynn.Stats.HP

	for turn := 0; turn < 15; turn++ {
		if g.Combat == nil || g.GameOver {
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			g.RunEnemyTurn(w)
		} else {
			g.Combat.NextTurn(g, w)
		}
	}

	elaraDmg := elaraStartHP - elara.Stats.HP
	brynnDmg := brynnStartHP - brynn.Stats.HP
	t.Logf("Elara took %d damage, Brynn took %d damage", elaraDmg, brynnDmg)

	if brynnDmg > 0 && elaraDmg == 0 {
		t.Error("enemy attacked distant Brynn instead of adjacent Elara")
	}
}

// Test 4: Enemy can path around a small obstacle.
func TestAI_EnemyPathsAroundObstacle(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// 3-wide corridor with 1-tile wall in the middle
	for dx := 0; dx <= 4; dx++ {
		w.SetTile(brynn.X+dx, brynn.Z, TileGround)
		w.SetTile(brynn.X+dx, brynn.Z+1, TileGround)
		w.SetTile(brynn.X+dx, brynn.Z-1, TileGround)
	}
	w.SetTile(brynn.X+2, brynn.Z, TileSolid)

	ex, ez := brynn.X+4, brynn.Z
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)

	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
		HP: 50, MaxHP: 50, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	startHP := brynn.Stats.HP
	reached := false
	for turn := 0; turn < 20; turn++ {
		if g.Combat == nil || g.GameOver {
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			threat, ok := w.Threats[cur.ThreatKey]
			if ok {
				dist := abs(threat.X-brynn.X) + abs(threat.Z-brynn.Z)
				t.Logf("turn %d: enemy at (%d,%d) dist=%d", turn, threat.X, threat.Z, dist)
			}
			g.RunEnemyTurn(w)
			if brynn.Stats.HP < startHP {
				reached = true
			}
		} else {
			g.Combat.NextTurn(g, w)
		}
	}

	if !reached && !g.GameOver {
		t.Error("enemy couldn't path around 1-tile obstacle to reach Brynn")
	}
}

// Test 5: Brynn flees along corridor. Enemy chases.
// Enemy moves 4 tiles/turn, Brynn moves 1. Enemy should catch up and attack.
// If Brynn gets far enough from spawn, enemy should eventually leash.
func TestAI_PursuitDuringChase(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Long corridor
	for dx := -30; dx <= 10; dx++ {
		w.SetTile(brynn.X+dx, brynn.Z, TileGround)
	}

	ex, ez := brynn.X+2, brynn.Z
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)
	cx2, cz2 := TileToChunk(brynn.X-30, brynn.Z)
	w.EnsureChunksAround(cx2, cz2)

	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
		HP: 50, MaxHP: 50, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	attacked := false
	leashed := false
	for turn := 0; turn < 60; turn++ {
		if g.Combat == nil {
			leashed = true
			t.Logf("turn %d: combat ended (leash or victory)", turn)
			break
		}
		if g.GameOver {
			t.Logf("turn %d: Brynn died", turn)
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			threat, ok := w.Threats[cur.ThreatKey]
			if ok {
				dist := abs(threat.X-brynn.X) + abs(threat.Z-brynn.Z)
				t.Logf("turn %d: enemy (%d,%d) brynn (%d,%d) dist=%d pursuit=%d",
					turn, threat.X, threat.Z, brynn.X, brynn.Z, dist, cur.PursuitLeft)
			}
			g.RunEnemyTurn(w)
			if brynn.Stats.HP < brynn.Stats.MaxHP {
				attacked = true
			}
		} else {
			// Brynn runs 1 tile left
			nx := brynn.X - 1
			if w.IsWalkable(nx, brynn.Z) && !w.IsThreatAt(nx, brynn.Z) {
				brynn.X = nx
			}
			g.Combat.NextTurn(g, w)
		}
	}

	t.Logf("attacked=%v leashed=%v gameOver=%v", attacked, leashed, g.GameOver)
}
