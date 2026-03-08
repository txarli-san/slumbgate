package main

import "testing"

// --- XP fields exist on CombatStats ---

func TestXP_FieldExists(t *testing.T) {
	s := &CombatStats{Level: 1}
	if s.XP != 0 {
		t.Error("new stats should have 0 XP")
	}
}

// --- XP thresholds match 5e SRD ---

func TestXP_Thresholds(t *testing.T) {
	cases := []struct {
		level     int
		threshold int
	}{
		{2, 300},
		{3, 900},
		{4, 2700},
		{5, 6500},
	}
	for _, c := range cases {
		got := xpForLevel(c.level)
		if got != c.threshold {
			t.Errorf("xpForLevel(%d) = %d, want %d", c.level, got, c.threshold)
		}
	}
}

// --- Threat XP values by skeleton type ---

func TestXP_ThreatValues(t *testing.T) {
	cases := []struct {
		stype SkeletonType
		xp    int
	}{
		{SkeletonMinion, 50},
		{SkeletonWarrior, 100},
		{SkeletonRogue, 100},
		{SkeletonMage, 200},
	}
	for _, c := range cases {
		got := threatXP(c.stype)
		if got != c.xp {
			t.Errorf("threatXP(%d) = %d, want %d", c.stype, got, c.xp)
		}
	}
}

// --- XP awarded on kill, split across combat party ---

func TestXP_AwardedOnKill(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 1
	brynn.Stats.XP = 0

	// Place a minion adjacent
	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 1, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	// Kill it with melee (1 HP so any hit kills)
	executeMeleeAttack(g, w, brynn, tx, tz)

	// Even if attack missed, keep trying
	for i := 0; i < 20 && w.IsThreatAt(tx, tz); i++ {
		executeMeleeAttack(g, w, brynn, tx, tz)
	}

	if brynn.Stats.XP != 50 {
		t.Errorf("Brynn XP = %d, want 50 after killing minion", brynn.Stats.XP)
	}
}

// --- XP split between two allies in same combat ---

func TestXP_SplitInCombat(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 1
	brynn.Stats.XP = 0

	elara := addElara(g, w)
	elara.X, elara.Z = brynn.X, brynn.Z+1
	elara.Stats.Level = 1
	elara.Stats.XP = 0

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 1, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	for i := 0; i < 20 && w.IsThreatAt(tx, tz); i++ {
		executeMeleeAttack(g, w, brynn, tx, tz)
	}

	// 50 XP / 2 allies = 25 each
	if brynn.Stats.XP != 25 {
		t.Errorf("Brynn XP = %d, want 25 (split)", brynn.Stats.XP)
	}
	if elara.Stats.XP != 25 {
		t.Errorf("Elara XP = %d, want 25 (split)", elara.Stats.XP)
	}
}

// --- Level up triggers at threshold ---

func TestXP_LevelUpAtThreshold(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 1
	brynn.Stats.XP = 280
	brynn.Stats.HP = 12
	brynn.Stats.MaxHP = 12
	brynn.Stats.HitDice = 1
	brynn.Stats.MaxHitDice = 1

	// Kill a minion worth 50 XP — pushes past 300 threshold
	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 1, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	for i := 0; i < 20 && w.IsThreatAt(tx, tz); i++ {
		executeMeleeAttack(g, w, brynn, tx, tz)
	}

	if brynn.Stats.Level != 2 {
		t.Errorf("Level = %d, want 2 after reaching 300+ XP", brynn.Stats.Level)
	}
	if brynn.Stats.MaxHP <= 12 {
		t.Error("MaxHP should increase on level up")
	}
	if brynn.Stats.MaxHitDice != 2 {
		t.Errorf("MaxHitDice = %d, want 2 at level 2", brynn.Stats.MaxHitDice)
	}
	if brynn.Stats.HitDice != 2 {
		t.Errorf("HitDice = %d, want 2 at level 2", brynn.Stats.HitDice)
	}
}

// --- Fighter gets Action Surge at level 2 ---

func TestXP_FighterActionSurgeAtLevel2(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Fighter",
		ClassCharges: 1, MaxClassCharges: 1,
	}
	levelUp(s)
	if s.Level != 2 {
		t.Errorf("Level = %d, want 2", s.Level)
	}
	// Level 2 fighter: Second Wind + Action Surge = 2 charges
	if s.MaxClassCharges != 2 {
		t.Errorf("MaxClassCharges = %d, want 2 at level 2", s.MaxClassCharges)
	}
}

// --- Level 1 Fighter has correct starting stats ---

func TestXP_Level1FighterStats(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Fighter",
		STR: 16, DEX: 12, CON: 14,
		HitDieSize: 10,
	}
	// HP = 10 + CON mod = 12
	hp := 10 + s.Mod(s.CON)
	if hp != 12 {
		t.Errorf("level 1 Fighter HP = %d, want 12", hp)
	}
	// 1 hit die at level 1
	if s.Level != 1 {
		t.Error("should be level 1")
	}
}
