package main

import (
	"testing"
)

// Rest economy TDD tests. These define the contract for the hit dice system.
// Each test should FAIL until the corresponding feature is implemented.

// --- Step 1: Hit Dice ---

// Every entity has hit dice: count = Level, size = class-dependent.
// Fighter=d10, Mage=d6, Cleric=d8, Rogue=d8, Druid=d8.
func TestHitDice_ExistOnEntity(t *testing.T) {
	g, _ := newTestGame()
	brynn := g.Entities[0]

	// Brynn is a level 3 Fighter — should have 3 hit dice (d10)
	if brynn.Stats.HitDice != 3 {
		t.Errorf("Brynn HitDice = %d, want 3", brynn.Stats.HitDice)
	}
	if brynn.Stats.MaxHitDice != 3 {
		t.Errorf("Brynn MaxHitDice = %d, want 3", brynn.Stats.MaxHitDice)
	}
	if brynn.Stats.HitDieSize != 10 {
		t.Errorf("Brynn HitDieSize = %d, want 10 (Fighter)", brynn.Stats.HitDieSize)
	}
}

func TestHitDice_MageHasD6(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)
	elara := g.Entities[1]

	// Elara is a level 3 Mage — should have 3 hit dice (d6)
	if elara.Stats.HitDice != 3 {
		t.Errorf("Elara HitDice = %d, want 3", elara.Stats.HitDice)
	}
	if elara.Stats.HitDieSize != 6 {
		t.Errorf("Elara HitDieSize = %d, want 6 (Mage)", elara.Stats.HitDieSize)
	}
}

// Short rest spends a hit die to heal: roll hit die + CON mod.
// If no hit dice left, short rest heals nothing.
func TestHitDice_ShortRestSpendsHitDie(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.HP = 10 // damaged

	startDice := brynn.Stats.HitDice
	g.TryShortRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	if brynn.Stats.HitDice != startDice-1 {
		t.Errorf("HitDice after short rest = %d, want %d (spent 1)", brynn.Stats.HitDice, startDice-1)
	}
	if brynn.Stats.HP <= 10 {
		t.Errorf("HP not healed: still %d", brynn.Stats.HP)
	}
}

func TestHitDice_NoHealingWithoutHitDice(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.HP = 10
	brynn.Stats.HitDice = 0 // spent all hit dice

	g.TryShortRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	// No hit dice = no healing
	if brynn.Stats.HP != 10 {
		t.Errorf("HP changed without hit dice: got %d, want 10", brynn.Stats.HP)
	}
}

func TestHitDice_FullHPNoSpend(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	// HP already full

	startDice := brynn.Stats.HitDice
	g.TryShortRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	// At full HP, shouldn't waste a hit die
	if brynn.Stats.HitDice != startDice {
		t.Errorf("spent hit die at full HP: %d → %d", startDice, brynn.Stats.HitDice)
	}
}

// Short rest: Fighter recovers class charges (existing behavior).
func TestHitDice_FighterChargesOnShortRest(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.ClassCharges = 0

	g.TryShortRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	if brynn.Stats.ClassCharges != brynn.Stats.MaxClassCharges {
		t.Errorf("charges not restored: got %d, want %d",
			brynn.Stats.ClassCharges, brynn.Stats.MaxClassCharges)
	}
}

// Short rest costs 60 ticks (1 hour = 600 rounds of 6 seconds,
// but our tick is abstract — use 60 for now as 1hr).
func TestHitDice_ShortRestTimeCost(t *testing.T) {
	g, w := newTestGame()
	before := g.TimeTicks

	g.TryShortRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	elapsed := g.TimeTicks - before
	if elapsed != 60 {
		t.Errorf("short rest took %d ticks, want 60", elapsed)
	}
}

// --- Step 2: Long Rest ---

// Long rest: full HP, recover half hit dice (min 1), spell slots, -1 exhaustion.
// Costs 480 ticks (8 hours). Higher ambush chance inside dungeon.
func TestLongRest_FullHPRecovery(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.HP = 5

	g.TryLongRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	if brynn.Stats.HP != brynn.Stats.MaxHP {
		t.Errorf("HP after long rest = %d, want %d", brynn.Stats.HP, brynn.Stats.MaxHP)
	}
}

func TestLongRest_RecoverHalfHitDice(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.HitDice = 0 // all spent

	g.TryLongRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	// Level 3: recover 3/2 = 1 hit die (rounded down, min 1)
	want := brynn.Stats.MaxHitDice / 2
	if want < 1 {
		want = 1
	}
	if brynn.Stats.HitDice != want {
		t.Errorf("HitDice after long rest = %d, want %d", brynn.Stats.HitDice, want)
	}
}

func TestLongRest_TimeCost(t *testing.T) {
	g, w := newTestGame()
	before := g.TimeTicks

	g.TryLongRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	elapsed := g.TimeTicks - before
	if elapsed != 480 {
		t.Errorf("long rest took %d ticks, want 480", elapsed)
	}
}

func TestLongRest_FighterChargesRecover(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.ClassCharges = 0

	g.TryLongRest(w, 0)

	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	if brynn.Stats.ClassCharges != brynn.Stats.MaxClassCharges {
		t.Errorf("charges after long rest = %d, want %d",
			brynn.Stats.ClassCharges, brynn.Stats.MaxClassCharges)
	}
}
