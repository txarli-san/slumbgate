package main

import (
	"testing"
)

// Invariant tests verify properties that must hold after any game action.
// These catch class-of-bug issues rather than specific scenarios.

// checkInvariants runs all invariant checks against current game state.
// Returns list of violations.
func checkInvariants(g *GameState, w *World) []string {
	var violations []string

	// --- Entity invariants ---
	for i, ent := range g.Entities {
		// No entity on unwalkable tile
		if !w.IsWalkable(ent.X, ent.Z) {
			violations = append(violations, "entity %d (%s) on unwalkable tile (%d,%d)")
			_ = i // suppress unused
		}

		if ent.Stats != nil {
			// HP never exceeds max
			if ent.Stats.HP > ent.Stats.MaxHP {
				violations = append(violations,
					"entity "+ent.Name+": HP exceeds max")
			}

			// Living entities have HP > 0
			if ent.Stats.HP <= 0 {
				violations = append(violations,
					"dead entity "+ent.Name+" still in roster")
			}

			// ClassCharges never negative
			if ent.Stats.ClassCharges < 0 {
				violations = append(violations,
					"entity "+ent.Name+": negative ClassCharges")
			}

			// ClassCharges never exceed max
			if ent.Stats.ClassCharges > ent.Stats.MaxClassCharges {
				violations = append(violations,
					"entity "+ent.Name+": ClassCharges exceed max")
			}
		}

		// Follow task points to valid entity
		if ent.Task != nil && ent.Task.Type == TaskFollow {
			if ent.Task.FollowIdx < 0 || ent.Task.FollowIdx >= len(g.Entities) {
				violations = append(violations,
					"entity "+ent.Name+": follow idx out of bounds")
			}
			if ent.Task.FollowIdx == i {
				violations = append(violations,
					"entity "+ent.Name+": following self")
			}
		}
	}

	// --- SelectedEnt invariant ---
	if len(g.Entities) > 0 {
		if g.SelectedEnt < 0 || g.SelectedEnt >= len(g.Entities) {
			violations = append(violations, "SelectedEnt out of bounds")
		}
	} else if !g.GameOver && g.SelectedEnt != -1 {
		violations = append(violations, "empty roster but SelectedEnt != -1 and not game over")
	}

	// --- Combat invariants ---
	if g.Combat != nil {
		c := g.Combat

		// TurnIndex in bounds
		if c.TurnIndex < 0 || c.TurnIndex >= len(c.Combatants) {
			violations = append(violations, "combat TurnIndex out of bounds")
		}

		// At least one combatant
		if len(c.Combatants) == 0 {
			violations = append(violations, "combat active with 0 combatants")
		}

		// All ally EntityIdx valid
		for _, cb := range c.Combatants {
			if !cb.IsEnemy {
				if cb.EntityIdx < 0 || cb.EntityIdx >= len(g.Entities) {
					violations = append(violations, "combatant EntityIdx out of bounds")
				}
			}
		}

		// At least one enemy (otherwise combat should have ended)
		hasEnemy := false
		for _, cb := range c.Combatants {
			if cb.IsEnemy {
				hasEnemy = true
				break
			}
		}
		if !hasEnemy {
			violations = append(violations, "combat active with no enemies")
		}

		// MoveLeft not negative
		if c.MoveLeft < 0 {
			violations = append(violations, "negative MoveLeft")
		}
	}

	// --- Threat invariants ---
	for key, threat := range w.Threats {
		// Threat position matches key
		if key[0] != threat.X || key[1] != threat.Z {
			violations = append(violations, "threat key/position mismatch")
		}

		// Threat HP > 0 (dead threats should be removed)
		if threat.HP <= 0 {
			violations = append(violations, "dead threat still in world")
		}

		// No threat on entity position
		for _, ent := range g.Entities {
			if ent.X == threat.X && ent.Z == threat.Z {
				violations = append(violations,
					"threat overlaps entity "+ent.Name)
			}
		}
	}

	// --- World invariants ---
	// TimeTicks never negative
	if g.TimeTicks < 0 {
		violations = append(violations, "negative TimeTicks")
	}

	return violations
}

// --- Tests that run invariants after operations ---

func TestInvariant_AfterSpawn(t *testing.T) {
	g, w := newTestGame()

	if v := checkInvariants(g, w); len(v) > 0 {
		for _, msg := range v {
			t.Error(msg)
		}
	}
}

func TestInvariant_AfterAddElara(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)

	if v := checkInvariants(g, w); len(v) > 0 {
		for _, msg := range v {
			t.Error(msg)
		}
	}
}

func TestInvariant_AfterWalkToPickaxe(t *testing.T) {
	g, w := newTestGame()
	walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)

	if g.Combat != nil {
		// Combat triggered — still check invariants
		if v := checkInvariants(g, w); len(v) > 0 {
			for _, msg := range v {
				t.Error(msg)
			}
		}
		return
	}

	if v := checkInvariants(g, w); len(v) > 0 {
		for _, msg := range v {
			t.Error(msg)
		}
	}
}

func TestInvariant_AfterRest(t *testing.T) {
	g, w := newTestGame()
	g.Entities[0].Stats.HP = 10
	g.TryRest(w, 0)

	if g.Combat != nil {
		runCombatToEnd(g, w, 100)
	}

	if !g.GameOver {
		if v := checkInvariants(g, w); len(v) > 0 {
			for _, msg := range v {
				t.Error(msg)
			}
		}
	}
}

func TestInvariant_AfterKillEntity(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)
	g.KillEntity(w, 0)

	if v := checkInvariants(g, w); len(v) > 0 {
		for _, msg := range v {
			t.Error(msg)
		}
	}
}

func TestInvariant_AfterGameOver(t *testing.T) {
	g, w := newTestGame()
	g.KillEntity(w, 0)

	if !g.GameOver {
		t.Fatal("should be game over")
	}

	// Relaxed check — game over state
	if g.Combat != nil {
		t.Error("combat should be nil after game over")
	}
	if len(g.Entities) != 0 {
		t.Error("entities should be empty after game over")
	}
}

func TestInvariant_AfterCombatVictory(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	ex, ez := brynn.X+1, brynn.Z
	w.SetTile(ex, ez, TileGround)
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)

	w.Threats[[2]int{ex, ez}] = Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
		HP: 1, MaxHP: 1, AC: 1, STR: 10, DEX: 10,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	runCombatToEnd(g, w, 50)

	if !g.GameOver {
		if v := checkInvariants(g, w); len(v) > 0 {
			for _, msg := range v {
				t.Error(msg)
			}
		}
	}
}

func TestInvariant_AfterFullSession(t *testing.T) {
	g, w := newTestGame()

	// Walk to pickaxe
	walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
	if g.Combat != nil {
		runCombatToEnd(g, w, 200)
	}
	if g.GameOver {
		return
	}
	if v := checkInvariants(g, w); len(v) > 0 {
		t.Errorf("after pickaxe: %v", v)
	}

	// Break into dungeon
	if g.HasPickaxe {
		for _, room := range w.Rooms {
			if room.W > 2 && room.H > 2 {
				tx, tz := room.X+room.W/2, room.Z+room.H/2
				walkToBreakable(g, w, 0, tx, tz, 500)
				break
			}
		}
	}
	if g.Combat != nil {
		runCombatToEnd(g, w, 200)
	}
	if g.GameOver {
		return
	}
	if v := checkInvariants(g, w); len(v) > 0 {
		t.Errorf("after dungeon breach: %v", v)
	}

	// Rest
	g.TryRest(w, 0)
	if g.Combat != nil {
		runCombatToEnd(g, w, 200)
	}
	if g.GameOver {
		return
	}
	if v := checkInvariants(g, w); len(v) > 0 {
		t.Errorf("after rest: %v", v)
	}
}

// --- Stress: many world steps ---

func TestInvariant_After1000Steps(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Set Brynn to scout
	brynn.Task = &Task{Type: TaskExplore}

	for i := 0; i < 1000; i++ {
		if g.Combat != nil || g.GameOver {
			break
		}
		g.WorldStep(w)
	}

	if g.Combat != nil {
		runCombatToEnd(g, w, 200)
	}

	if !g.GameOver {
		if v := checkInvariants(g, w); len(v) > 0 {
			for _, msg := range v {
				t.Errorf("after 1000 steps: %s", msg)
			}
		}
	}

	t.Logf("1000 steps: tick %d, pos (%d,%d), HP %d/%d, pickaxe=%v, combat=%v, gameover=%v",
		g.TimeTicks, brynn.X, brynn.Z, brynn.Stats.HP, brynn.Stats.MaxHP,
		g.HasPickaxe, g.Combat != nil, g.GameOver)
}
