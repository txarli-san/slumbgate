package main

import (
	"math/rand"
	"testing"
)

func TestExhaustion_LongRestRecovery(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 3

	g.TryLongRest(w, 0)
	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	if brynn.Stats.Exhaustion != 2 {
		t.Errorf("exhaustion after long rest = %d, want 2", brynn.Stats.Exhaustion)
	}
}

func TestExhaustion_LongRestResetsTimer(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	g.TimeTicks = 500

	g.TryLongRest(w, 0)
	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	// LastLongRestTick should be set to post-rest time
	if brynn.LastLongRestTick != g.TimeTicks {
		t.Errorf("LastLongRestTick = %d, want %d (current TimeTicks)", brynn.LastLongRestTick, g.TimeTicks)
	}

	// Exhaustion shouldn't trigger for ExhaustionThresholdTicks after rest
	for range ExhaustionThresholdTicks - 1 {
		g.TimeTicks++
		brynn.CheckExhaustion(g, w)
	}
	if brynn.Stats.Exhaustion != 0 {
		t.Errorf("exhaustion %d before threshold, want 0", brynn.Stats.Exhaustion)
	}
}

func TestExhaustion_ShortRestNoRecovery(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 2

	g.TryShortRest(w, 0)
	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	if brynn.Stats.Exhaustion != 2 {
		t.Errorf("exhaustion after short rest = %d, want 2 (unchanged)", brynn.Stats.Exhaustion)
	}
}

// --- Effective stat methods ---

func TestExhaustion_SpeedHalved(t *testing.T) {
	g, _ := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 2

	got := brynn.EffectiveMoveSpeed()
	want := brynn.Stats.MoveSpeed / 2
	if got != want {
		t.Errorf("EffectiveMoveSpeed at exhaustion 2 = %d, want %d", got, want)
	}
}

func TestExhaustion_SpeedZero(t *testing.T) {
	g, _ := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 5

	if brynn.EffectiveMoveSpeed() != 0 {
		t.Errorf("EffectiveMoveSpeed at exhaustion 5 = %d, want 0", brynn.EffectiveMoveSpeed())
	}
}

func TestExhaustion_MaxHPHalved(t *testing.T) {
	g, _ := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 4

	got := brynn.EffectiveMaxHP()
	want := brynn.Stats.MaxHP / 2
	if got != want {
		t.Errorf("EffectiveMaxHP at exhaustion 4 = %d, want %d", got, want)
	}
}

// --- Integration: death ---

func TestExhaustion_DeathFromTimeAwake(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 5

	// Push time so CheckExhaustion triggers level 6
	g.TimeTicks = brynn.LastLongRestTick + ExhaustionThresholdTicks + ExhaustionIntervalTicks*5
	brynn.CheckExhaustion(g, w)

	// Brynn should be gone
	for _, ent := range g.Entities {
		if ent.Name == "Brynn" {
			t.Error("Brynn still alive at exhaustion 6")
		}
	}
}

func TestExhaustion_DeathFromDirectSet(t *testing.T) {
	// Simulates the debug key path: set exhaustion to 6 directly → entity dies
	g, w := newTestGame()
	addElara(g, w) // so game doesn't end
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 6

	g.KillEntity(w, 0)

	for _, ent := range g.Entities {
		if ent.Name == "Brynn" {
			t.Error("Brynn still alive after KillEntity at exhaustion 6")
		}
	}
	if g.GameOver {
		t.Error("game over with Elara still alive")
	}
}

// --- Integration: continuous mode movement ---

func TestExhaustion_SpeedZeroCancelsTask(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 5
	brynn.Task = &Task{Type: TaskExplore}

	g.TickEntities(w)

	if brynn.Task != nil {
		t.Error("task not cancelled at exhaustion 5")
	}
}

func TestExhaustion_HalfSpeedContinuous(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Give brynn a straight-line path
	startX := brynn.X
	path := make([][2]int, 20)
	for i := range path {
		path[i] = [2]int{startX + i + 1, brynn.Z}
		// Make sure tiles are walkable
		w.SetTile(path[i][0], path[i][1], TileGround)
	}

	// Normal speed: count tiles moved over 10 ticks
	brynn.Task = &Task{Type: TaskMoveTo, Path: path, TargetX: path[len(path)-1][0], TargetZ: brynn.Z}
	normalMoves := 0
	for range 10 {
		prevX := brynn.X
		g.TimeTicks++
		g.TickEntities(w)
		if brynn.X != prevX {
			normalMoves++
		}
	}

	// Reset
	brynn.X = startX
	brynn.Stats.Exhaustion = 2
	brynn.Task = &Task{Type: TaskMoveTo, Path: path, TargetX: path[len(path)-1][0], TargetZ: brynn.Z}
	exhaustedMoves := 0
	for range 10 {
		prevX := brynn.X
		g.TimeTicks++
		g.TickEntities(w)
		if brynn.X != prevX {
			exhaustedMoves++
		}
	}

	// Exhausted entity should move roughly half as many tiles
	if exhaustedMoves >= normalMoves {
		t.Errorf("exhaustion 2: moved %d tiles in 10 ticks, normal moved %d — should be fewer",
			exhaustedMoves, normalMoves)
	}
	if exhaustedMoves == 0 {
		t.Error("exhaustion 2: moved 0 tiles, should still move (just slower)")
	}
}

// --- Integration: HP clamping ---

func TestExhaustion_HPClampedOnWorldStep(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.HP = brynn.Stats.MaxHP // full health
	brynn.Stats.Exhaustion = 4

	g.WorldStep(w)

	maxHP := brynn.EffectiveMaxHP()
	if brynn.Stats.HP > maxHP {
		t.Errorf("HP %d exceeds EffectiveMaxHP %d after WorldStep", brynn.Stats.HP, maxHP)
	}
}

func TestExhaustion_ShortRestCapsAtEffectiveMax(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Exhaustion = 4
	effectiveMax := brynn.EffectiveMaxHP()
	brynn.Stats.HP = effectiveMax - 1

	g.TryShortRest(w, 0)
	if g.Combat != nil {
		t.Skip("ambushed during rest")
	}

	if brynn.Stats.HP > effectiveMax {
		t.Errorf("HP %d exceeds EffectiveMaxHP %d after short rest", brynn.Stats.HP, effectiveMax)
	}
}

// --- Integration: time-based exhaustion gain ---

func TestExhaustion_TimeAwakeGain(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]

	// Just before threshold: no exhaustion
	g.TimeTicks = ExhaustionThresholdTicks - 1
	brynn.CheckExhaustion(g, w)
	if brynn.Stats.Exhaustion != 0 {
		t.Fatalf("exhaustion %d before threshold", brynn.Stats.Exhaustion)
	}

	// At threshold: gain 1
	g.TimeTicks = ExhaustionThresholdTicks
	brynn.CheckExhaustion(g, w)
	if brynn.Stats.Exhaustion != 1 {
		t.Errorf("exhaustion at threshold = %d, want 1", brynn.Stats.Exhaustion)
	}

	// One interval later: gain 2
	g.TimeTicks = ExhaustionThresholdTicks + ExhaustionIntervalTicks
	brynn.CheckExhaustion(g, w)
	if brynn.Stats.Exhaustion != 2 {
		t.Errorf("exhaustion after 1 interval = %d, want 2", brynn.Stats.Exhaustion)
	}

	// Doesn't double-gain on same tick
	brynn.CheckExhaustion(g, w)
	if brynn.Stats.Exhaustion != 2 {
		t.Errorf("double-called same tick: exhaustion = %d, want 2", brynn.Stats.Exhaustion)
	}
}

func TestExhaustion_WorldStepGainsExhaustion(t *testing.T) {
	// Exhaustion should be gained through the real WorldStep path, not just direct CheckExhaustion
	g, w := newTestGame()
	brynn := g.Entities[0]
	g.TimeTicks = ExhaustionThresholdTicks - 1 // WorldStep will increment to threshold

	g.WorldStep(w)

	if brynn.Stats.Exhaustion != 1 {
		t.Errorf("exhaustion after WorldStep at threshold = %d, want 1", brynn.Stats.Exhaustion)
	}
}

// --- rollD20Adv ---

func TestExhaustion_DisadvantageRolls(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// Simulate disadvantage: min of two rolls
	sumDisadv := 0
	n := 1000
	for range n {
		a, b := rng.Intn(20)+1, rng.Intn(20)+1
		if a < b {
			sumDisadv += a
		} else {
			sumDisadv += b
		}
	}

	// Normal: single roll
	sumNormal := 0
	for range n {
		sumNormal += rng.Intn(20) + 1
	}

	avgDisadv := float64(sumDisadv) / float64(n)
	avgNormal := float64(sumNormal) / float64(n)

	if avgDisadv >= avgNormal {
		t.Errorf("disadvantage avg %.1f not lower than normal avg %.1f", avgDisadv, avgNormal)
	}

	// Verify rollD20Adv range
	for range 100 {
		r := rollD20Adv(false, false)
		if r < 1 || r > 20 {
			t.Fatalf("rollD20Adv(false,false) = %d, out of range", r)
		}
	}
	for range 100 {
		r := rollD20Adv(true, true)
		if r < 1 || r > 20 {
			t.Fatalf("rollD20Adv(true,true) = %d, out of range", r)
		}
	}
}
