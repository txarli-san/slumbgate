package main

import (
	"testing"
)

// Bug: when an entity takes lethal damage, HP goes negative (e.g. HP=-7/28).
// HP should clamp to 0 on lethal damage.

func TestBug_HPClampsToZeroOnDeath(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)
	elara := g.Entities[1]

	// Set up a minimal combat so ResolveEnemyAttack works
	brynn := g.Entities[0]
	ex, ez := brynn.X+1, brynn.Z
	w.SetTile(ex, ez, TileGround)
	cx, cz := TileToChunk(ex, ez)
	w.EnsureChunksAround(cx, cz)

	threat := Threat{
		X: ex, Z: ez, Type: SkeletonMinion, RoomIdx: 0,
		HP: 100, MaxHP: 100, AC: 1, STR: 20, DEX: 10,
		MoveSpeed: 4, AttackDice: 12, MaxRange: 1,
	}
	w.Threats[[2]int{ex, ez}] = threat

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	// Elara at 1 HP — any hit deals overkill damage
	elara.Stats.HP = 1

	// Hammer ResolveEnemyAttack until Elara gets hit
	for i := 0; i < 100; i++ {
		if elara.Stats.HP <= 0 {
			break
		}
		g.ResolveEnemyAttack(w, threat, elara)
		if g.GameOver {
			break
		}
	}

	// The real check: HP should be exactly 0, not negative
	if elara.Stats.HP < 0 {
		t.Errorf("Elara HP = %d after lethal damage, want 0", elara.Stats.HP)
	}
}
