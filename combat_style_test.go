package main

import "testing"

// --- Gladiator: +2 melee damage ---

func TestCombatStyle_Gladiator_ExtraDamage(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 3
	brynn.Stats.CombatStyle = "Gladiator"
	brynn.Stats.STR = 16 // +3 mod

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 100, MaxHP: 100, AC: 1, // AC 1 so every hit lands
		STR: 10, DEX: 10, MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)

	// Attack many times and track damage
	totalDamage := 0
	hits := 0
	for i := 0; i < 100; i++ {
		threat, ok := w.GetThreat(tx, tz)
		if !ok {
			break
		}
		before := threat.HP
		g.Combat.ActionUsed = false
		executeMeleeAttack(g, w, brynn, tx, tz)
		threat, ok = w.GetThreat(tx, tz)
		after := 0
		if ok {
			after = threat.HP
		}
		dmg := before - after
		if dmg > 0 {
			totalDamage += dmg
			hits++
		}
	}

	if hits == 0 {
		t.Fatal("no hits landed in 100 attacks")
	}

	// Average without Gladiator: 1d8 + 3 = 7.5
	// Average with Gladiator: 1d8 + 3 + 2 = 9.5
	// With 100 hits, average should be around 950. Without, around 750.
	avg := float64(totalDamage) / float64(hits)
	if avg < 8.0 {
		t.Errorf("average damage = %.1f, expected > 8.0 with Gladiator +2", avg)
	}
}

func TestCombatStyle_NoStyle_NoBonusDamage(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 1
	brynn.Stats.CombatStyle = "" // no style
	brynn.Stats.STR = 16

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 100, MaxHP: 100, AC: 1,
		STR: 10, DEX: 10, MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)

	totalDamage := 0
	hits := 0
	for i := 0; i < 100; i++ {
		threat, ok := w.GetThreat(tx, tz)
		if !ok {
			break
		}
		before := threat.HP
		g.Combat.ActionUsed = false
		executeMeleeAttack(g, w, brynn, tx, tz)
		threat, ok = w.GetThreat(tx, tz)
		after := 0
		if ok {
			after = threat.HP
		}
		dmg := before - after
		if dmg > 0 {
			totalDamage += dmg
			hits++
		}
	}

	if hits == 0 {
		t.Fatal("no hits landed")
	}

	avg := float64(totalDamage) / float64(hits)
	// Without Gladiator, average is 1d8+3 = 7.5
	if avg > 9.0 {
		t.Errorf("average damage = %.1f without style, expected < 9.0", avg)
	}
}

// --- Power Attack: -2 hit, x1.5 damage ---

func TestCombatTechnique_PowerAttack_ReducedHit(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 4
	brynn.Stats.CombatTechnique = "Power Attack"
	brynn.Stats.STR = 16
	brynn.Stats.ProfBonus = 2

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	// AC 15: without Power Attack total = d20+3+2 = d20+5, hits on 10+
	// With Power Attack: d20+3+2-2 = d20+3, hits on 12+
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonWarrior, RoomIdx: 0,
		HP: 500, MaxHP: 500, AC: 15,
		STR: 10, DEX: 10, MoveSpeed: 4, AttackDice: 6, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)

	hits := 0
	for i := 0; i < 200; i++ {
		threat, ok := w.GetThreat(tx, tz)
		if !ok {
			break
		}
		before := threat.HP
		g.Combat.ActionUsed = false
		executeMeleeAttack(g, w, brynn, tx, tz)
		threat, ok = w.GetThreat(tx, tz)
		after := 0
		if ok {
			after = threat.HP
		}
		if after < before {
			hits++
		}
	}

	// Expected hit rate with PA: ~45% (roll 12-20 = 9/20)
	// Expected without PA: ~55% (roll 10-20 = 11/20)
	hitRate := float64(hits) / 200.0
	if hitRate > 0.55 {
		t.Errorf("hit rate = %.2f, expected lower with Power Attack -2 penalty", hitRate)
	}
}

func TestCombatTechnique_PowerAttack_IncreasedDamage(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 4
	brynn.Stats.CombatTechnique = "Power Attack"
	brynn.Stats.STR = 16

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 200, MaxHP: 200, AC: 1, // always hits
		STR: 10, DEX: 10, MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)

	totalDamage := 0
	hits := 0
	for i := 0; i < 100; i++ {
		threat, ok := w.GetThreat(tx, tz)
		if !ok {
			break
		}
		before := threat.HP
		g.Combat.ActionUsed = false
		executeMeleeAttack(g, w, brynn, tx, tz)
		threat, ok = w.GetThreat(tx, tz)
		after := 0
		if ok {
			after = threat.HP
		}
		dmg := before - after
		if dmg > 0 {
			totalDamage += dmg
			hits++
		}
	}

	if hits == 0 {
		t.Fatal("no hits landed")
	}

	avg := float64(totalDamage) / float64(hits)
	// Base: 1d8+3 = 7.5, with x1.5: ~11.25
	if avg < 9.0 {
		t.Errorf("average damage = %.1f with Power Attack, expected > 9.0", avg)
	}
}

// --- Defensive Stance: AC and move already applied on choice ---

func TestCombatTechnique_DefensiveStance_StatsApplied(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Fighter", AC: 16, MoveSpeed: 5,
		PendingChoices: []string{"combat_technique"},
	}
	ApplyFighterTechnique(s, "Defensive Stance")
	if s.AC != 18 {
		t.Errorf("AC = %d, want 18", s.AC)
	}
	if s.MoveSpeed != 4 {
		t.Errorf("MoveSpeed = %d, want 4", s.MoveSpeed)
	}
}

// --- Juggernaut: AC already applied on choice ---

func TestCombatStyle_Juggernaut_ACApplied(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Fighter", AC: 16,
		PendingChoices: []string{"combat_style"},
	}
	ApplyFighterStyle(s, "Juggernaut")
	if s.AC != 18 {
		t.Errorf("AC = %d, want 18", s.AC)
	}
}

// --- Quick Strike: appears in BuildActions only with technique ---

func TestCombatTechnique_QuickStrike_InBuildActions(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 4
	brynn.Stats.CombatTechnique = "Quick Strike"

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	actions := BuildActions(g, brynn)

	found := false
	for _, a := range actions {
		if a.ID == "quick_strike" {
			found = true
			if a.Cost != CostBonus {
				t.Error("Quick Strike should be bonus action")
			}
		}
	}
	if !found {
		t.Error("Quick Strike not found in BuildActions")
	}
}

func TestCombatTechnique_NoQuickStrike_WithoutTechnique(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 4
	brynn.Stats.CombatTechnique = "" // no technique

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	actions := BuildActions(g, brynn)

	for _, a := range actions {
		if a.ID == "quick_strike" {
			t.Error("Quick Strike should not appear without the technique")
		}
	}
}

// --- Quick Strike deals half damage ---

func TestCombatTechnique_QuickStrike_HalfDamage(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 4
	brynn.Stats.CombatTechnique = "Quick Strike"
	brynn.Stats.STR = 16

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 200, MaxHP: 200, AC: 1,
		STR: 10, DEX: 10, MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)

	totalDamage := 0
	hits := 0
	for i := 0; i < 100; i++ {
		threat, ok := w.GetThreat(tx, tz)
		if !ok {
			break
		}
		before := threat.HP
		g.Combat.BonusUsed = false
		executeQuickStrike(g, w, brynn, tx, tz)
		threat, ok = w.GetThreat(tx, tz)
		after := 0
		if ok {
			after = threat.HP
		}
		dmg := before - after
		if dmg > 0 {
			totalDamage += dmg
			hits++
		}
	}

	if hits == 0 {
		t.Fatal("no hits landed")
	}

	avg := float64(totalDamage) / float64(hits)
	// Base: (1d8+3)/2 = ~3.75
	// Should be noticeably less than full melee average of 7.5
	if avg > 6.0 {
		t.Errorf("average QS damage = %.1f, expected < 6.0 (half damage)", avg)
	}
}
