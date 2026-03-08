package main

import (
	"fmt"
	"testing"
)

// Scout pickaxe discovery: set Brynn to explore and measure how reliably
// she finds and picks up the pickaxe across seeds.

func TestScout_PickaxeDiscovery_SingleSeed(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Task = &Task{Type: TaskExplore}

	maxTicks := 2000
	for i := 0; i < maxTicks; i++ {
		if g.HasPickaxe || g.Combat != nil || g.GameOver {
			break
		}
		g.WorldStep(w)
	}

	if g.Combat != nil {
		t.Logf("combat triggered at tick %d before pickaxe (pos %d,%d)",
			g.TimeTicks, brynn.X, brynn.Z)
	}

	dist := abs(brynn.X-w.PickaxeX) + abs(brynn.Z-w.PickaxeZ)
	t.Logf("seed=%d ticks=%d pickaxe=%v pos=(%d,%d) pickaxeAt=(%d,%d) dist=%d",
		testSeed, g.TimeTicks, g.HasPickaxe, brynn.X, brynn.Z,
		w.PickaxeX, w.PickaxeZ, dist)

	if !g.HasPickaxe {
		t.Errorf("Brynn did not find pickaxe in %d ticks", maxTicks)
	}
}

func TestScout_PickaxeDiscovery_MultiSeed(t *testing.T) {
	found := 0
	total := len(testSeeds)

	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newSeededGame(seed)
			brynn := g.Entities[0]
			brynn.Task = &Task{Type: TaskExplore}

			maxTicks := 2000
			for i := 0; i < maxTicks; i++ {
				if g.HasPickaxe || g.Combat != nil || g.GameOver {
					break
				}
				g.WorldStep(w)
			}

			if g.Combat != nil {
				runCombatToEnd(g, w, 200)
			}

			dist := abs(brynn.X-w.PickaxeX) + abs(brynn.Z-w.PickaxeZ)
			t.Logf("ticks=%d pickaxe=%v pos=(%d,%d) dist=%d",
				g.TimeTicks, g.HasPickaxe, brynn.X, brynn.Z, dist)

			if g.HasPickaxe {
				found++
			}
		})
	}

	t.Logf("RESULTS: %d/%d seeds found pickaxe (%.0f%%)",
		found, total, float64(found)*100/float64(total))
}
