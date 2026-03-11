package main

import "testing"

// --- Fighter pending choices ---

func TestLevelUp_FighterLevel2_NoPendingChoice(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Fighter", HitDieSize: 10,
		HP: 12, MaxHP: 12, CON: 14,
		ClassCharges: 1, MaxClassCharges: 1,
		HitDice: 1, MaxHitDice: 1,
	}
	levelUp(s)
	if s.Level != 2 {
		t.Fatalf("Level = %d, want 2", s.Level)
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty at level 2", s.PendingChoices)
	}
}

func TestLevelUp_FighterLevel3_CombatStylePending(t *testing.T) {
	s := &CombatStats{
		Level: 2, Class: "Fighter", HitDieSize: 10,
		HP: 20, MaxHP: 20, CON: 14,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 2, MaxHitDice: 2,
	}
	levelUp(s)
	if s.Level != 3 {
		t.Fatalf("Level = %d, want 3", s.Level)
	}
	if len(s.PendingChoices) != 1 || s.PendingChoices[0] != "combat_style" {
		t.Errorf("PendingChoices = %v, want [combat_style]", s.PendingChoices)
	}
}

func TestLevelUp_FighterLevel4_CombatTechniquePending(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Fighter", HitDieSize: 10,
		HP: 28, MaxHP: 28, CON: 14,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 3, MaxHitDice: 3,
	}
	levelUp(s)
	if s.Level != 4 {
		t.Fatalf("Level = %d, want 4", s.Level)
	}
	if len(s.PendingChoices) != 1 || s.PendingChoices[0] != "combat_technique" {
		t.Errorf("PendingChoices = %v, want [combat_technique]", s.PendingChoices)
	}
}

// --- Mage pending choices ---

func TestLevelUp_MageLevel2_SpellPending(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Mage", HitDieSize: 6,
		HP: 7, MaxHP: 7, CON: 12,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 1, MaxHitDice: 1,
	}
	levelUp(s)
	if s.Level != 2 {
		t.Fatalf("Level = %d, want 2", s.Level)
	}
	if len(s.PendingChoices) != 1 || s.PendingChoices[0] != "spell_l1" {
		t.Errorf("PendingChoices = %v, want [spell_l1]", s.PendingChoices)
	}
}

func TestLevelUp_MageLevel3_CantripPending(t *testing.T) {
	s := &CombatStats{
		Level: 2, Class: "Mage", HitDieSize: 6,
		HP: 12, MaxHP: 12, CON: 12,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 2, MaxHitDice: 2,
	}
	levelUp(s)
	if s.Level != 3 {
		t.Fatalf("Level = %d, want 3", s.Level)
	}
	if len(s.PendingChoices) != 1 || s.PendingChoices[0] != "cantrip" {
		t.Errorf("PendingChoices = %v, want [cantrip]", s.PendingChoices)
	}
}

func TestLevelUp_MageLevel4_SpellPending(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Mage", HitDieSize: 6,
		HP: 16, MaxHP: 16, CON: 12,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 3, MaxHitDice: 3,
	}
	levelUp(s)
	if s.Level != 4 {
		t.Fatalf("Level = %d, want 4", s.Level)
	}
	if len(s.PendingChoices) != 1 || s.PendingChoices[0] != "spell_l1" {
		t.Errorf("PendingChoices = %v, want [spell_l1]", s.PendingChoices)
	}
}

// --- Multi-level jump accumulates choices ---

func TestLevelUp_FighterMultiJump_AccumulatesChoices(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Fighter", HitDieSize: 10,
		HP: 12, MaxHP: 12, CON: 14,
		ClassCharges: 1, MaxClassCharges: 1,
		HitDice: 1, MaxHitDice: 1,
		XP: 2700, // enough for level 4
	}
	checkLevelUp(s)
	if s.Level != 4 {
		t.Fatalf("Level = %d, want 4", s.Level)
	}
	// Should have combat_style (from L3) + combat_technique (from L4)
	if len(s.PendingChoices) != 2 {
		t.Fatalf("PendingChoices = %v, want 2 entries", s.PendingChoices)
	}
	if s.PendingChoices[0] != "combat_style" {
		t.Errorf("PendingChoices[0] = %s, want combat_style", s.PendingChoices[0])
	}
	if s.PendingChoices[1] != "combat_technique" {
		t.Errorf("PendingChoices[1] = %s, want combat_technique", s.PendingChoices[1])
	}
}

func TestLevelUp_MageMultiJump_AccumulatesChoices(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Mage", HitDieSize: 6,
		HP: 7, MaxHP: 7, CON: 12,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 1, MaxHitDice: 1,
		XP: 2700, // enough for level 4
	}
	checkLevelUp(s)
	if s.Level != 4 {
		t.Fatalf("Level = %d, want 4", s.Level)
	}
	// spell_l1 (L2) + cantrip (L3) + spell_l1 (L4)
	if len(s.PendingChoices) != 3 {
		t.Fatalf("PendingChoices = %v, want 3 entries", s.PendingChoices)
	}
	if s.PendingChoices[0] != "spell_l1" {
		t.Errorf("PendingChoices[0] = %s, want spell_l1", s.PendingChoices[0])
	}
	if s.PendingChoices[1] != "cantrip" {
		t.Errorf("PendingChoices[1] = %s, want cantrip", s.PendingChoices[1])
	}
	if s.PendingChoices[2] != "spell_l1" {
		t.Errorf("PendingChoices[2] = %s, want spell_l1", s.PendingChoices[2])
	}
}

// --- Non-choice levels produce no pending choices ---

func TestLevelUp_FighterLevel5_NoPendingChoice(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Fighter", HitDieSize: 10,
		HP: 36, MaxHP: 36, CON: 14,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 4, MaxHitDice: 4,
	}
	levelUp(s)
	if s.Level != 5 {
		t.Fatalf("Level = %d, want 5", s.Level)
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty at level 5", s.PendingChoices)
	}
}

func TestLevelUp_MageLevel5_NoPendingChoice(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Mage", HitDieSize: 6,
		HP: 20, MaxHP: 20, CON: 12,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 4, MaxHitDice: 4,
	}
	levelUp(s)
	if s.Level != 5 {
		t.Fatalf("Level = %d, want 5", s.Level)
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty at level 5", s.PendingChoices)
	}
}

// --- Fighter style resolution ---

func TestApply_FighterGladiator(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Fighter", AC: 16,
		PendingChoices: []string{"combat_style"},
	}
	if !ApplyFighterStyle(s, "Gladiator") {
		t.Fatal("ApplyFighterStyle returned false")
	}
	if s.CombatStyle != "Gladiator" {
		t.Errorf("CombatStyle = %s, want Gladiator", s.CombatStyle)
	}
	if s.AC != 16 {
		t.Errorf("AC = %d, want 16 (Gladiator doesn't change AC)", s.AC)
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty", s.PendingChoices)
	}
}

func TestApply_FighterRanger(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Fighter", AC: 16,
		PendingChoices: []string{"combat_style"},
	}
	if !ApplyFighterStyle(s, "Ranger") {
		t.Fatal("ApplyFighterStyle returned false")
	}
	if s.CombatStyle != "Ranger" {
		t.Errorf("CombatStyle = %s, want Ranger", s.CombatStyle)
	}
	if s.AC != 16 {
		t.Errorf("AC = %d, want 16 (Ranger doesn't change AC)", s.AC)
	}
}

func TestApply_FighterJuggernaut_AC(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Fighter", AC: 16,
		PendingChoices: []string{"combat_style"},
	}
	if !ApplyFighterStyle(s, "Juggernaut") {
		t.Fatal("ApplyFighterStyle returned false")
	}
	if s.CombatStyle != "Juggernaut" {
		t.Errorf("CombatStyle = %s, want Juggernaut", s.CombatStyle)
	}
	if s.AC != 18 {
		t.Errorf("AC = %d, want 18 (Juggernaut +2 AC)", s.AC)
	}
}

func TestApply_FighterStyle_InvalidChoice(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Fighter",
		PendingChoices: []string{"combat_style"},
	}
	if ApplyFighterStyle(s, "Berserker") {
		t.Error("should reject invalid style")
	}
	if len(s.PendingChoices) != 1 {
		t.Error("pending should not be consumed on invalid choice")
	}
}

func TestApply_FighterStyle_NoPending(t *testing.T) {
	s := &CombatStats{Level: 3, Class: "Fighter"}
	if ApplyFighterStyle(s, "Gladiator") {
		t.Error("should fail with no pending combat_style")
	}
	if s.CombatStyle != "" {
		t.Error("CombatStyle should not be set")
	}
}

// --- Fighter technique resolution ---

func TestApply_FighterPowerAttack(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Fighter", AC: 16, MoveSpeed: 5,
		PendingChoices: []string{"combat_technique"},
	}
	if !ApplyFighterTechnique(s, "Power Attack") {
		t.Fatal("ApplyFighterTechnique returned false")
	}
	if s.CombatTechnique != "Power Attack" {
		t.Errorf("CombatTechnique = %s, want Power Attack", s.CombatTechnique)
	}
	if s.AC != 16 {
		t.Errorf("AC = %d, want 16 (Power Attack doesn't change AC)", s.AC)
	}
	if s.MoveSpeed != 5 {
		t.Errorf("MoveSpeed = %d, want 5 (Power Attack doesn't change move)", s.MoveSpeed)
	}
}

func TestApply_FighterDefensiveStance(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Fighter", AC: 16, MoveSpeed: 5,
		PendingChoices: []string{"combat_technique"},
	}
	if !ApplyFighterTechnique(s, "Defensive Stance") {
		t.Fatal("ApplyFighterTechnique returned false")
	}
	if s.CombatTechnique != "Defensive Stance" {
		t.Errorf("CombatTechnique = %s, want Defensive Stance", s.CombatTechnique)
	}
	if s.AC != 18 {
		t.Errorf("AC = %d, want 18 (Defensive Stance +2 AC)", s.AC)
	}
	if s.MoveSpeed != 4 {
		t.Errorf("MoveSpeed = %d, want 4 (Defensive Stance -1 move)", s.MoveSpeed)
	}
}

func TestApply_FighterQuickStrike(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Fighter", AC: 16, MoveSpeed: 5,
		PendingChoices: []string{"combat_technique"},
	}
	if !ApplyFighterTechnique(s, "Quick Strike") {
		t.Fatal("ApplyFighterTechnique returned false")
	}
	if s.CombatTechnique != "Quick Strike" {
		t.Errorf("CombatTechnique = %s, want Quick Strike", s.CombatTechnique)
	}
	if s.AC != 16 || s.MoveSpeed != 5 {
		t.Error("Quick Strike shouldn't change AC or MoveSpeed")
	}
}

func TestApply_FighterTechnique_InvalidChoice(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Fighter",
		PendingChoices: []string{"combat_technique"},
	}
	if ApplyFighterTechnique(s, "Whirlwind") {
		t.Error("should reject invalid technique")
	}
	if len(s.PendingChoices) != 1 {
		t.Error("pending should not be consumed on invalid choice")
	}
}

func TestApply_FighterTechnique_NoPending(t *testing.T) {
	s := &CombatStats{Level: 4, Class: "Fighter"}
	if ApplyFighterTechnique(s, "Power Attack") {
		t.Error("should fail with no pending combat_technique")
	}
}

// --- Mage spell resolution ---

func TestApply_MageSpell_BurningHands(t *testing.T) {
	s := &CombatStats{
		Level: 2, Class: "Mage",
		PendingChoices: []string{"spell_l1"},
	}
	if !ApplyMageSpell(s, "burning_hands") {
		t.Fatal("ApplyMageSpell returned false")
	}
	if len(s.KnownSpells) != 1 || s.KnownSpells[0] != "burning_hands" {
		t.Errorf("KnownSpells = %v, want [burning_hands]", s.KnownSpells)
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty", s.PendingChoices)
	}
}

func TestApply_MageSpell_DuplicateRejected(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Mage",
		KnownSpells:    []string{"burning_hands"},
		PendingChoices: []string{"spell_l1"},
	}
	if ApplyMageSpell(s, "burning_hands") {
		t.Error("should reject duplicate spell")
	}
	if len(s.PendingChoices) != 1 {
		t.Error("pending should not be consumed on duplicate")
	}
}

func TestApply_MageSpell_InvalidSpell(t *testing.T) {
	s := &CombatStats{
		Level: 2, Class: "Mage",
		PendingChoices: []string{"spell_l1"},
	}
	if ApplyMageSpell(s, "fireball") {
		t.Error("should reject invalid spell")
	}
}

func TestApply_MageSpell_NoPending(t *testing.T) {
	s := &CombatStats{Level: 2, Class: "Mage"}
	if ApplyMageSpell(s, "burning_hands") {
		t.Error("should fail with no pending spell_l1")
	}
}

func TestApply_MageSpell_TwoSpellsAtL2AndL4(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Mage",
		PendingChoices: []string{"spell_l1", "spell_l1"},
	}
	if !ApplyMageSpell(s, "burning_hands") {
		t.Fatal("first spell failed")
	}
	if !ApplyMageSpell(s, "frost_nova") {
		t.Fatal("second spell failed")
	}
	if len(s.KnownSpells) != 2 {
		t.Errorf("KnownSpells = %v, want 2 spells", s.KnownSpells)
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty", s.PendingChoices)
	}
}

// --- Mage cantrip resolution ---

func TestApply_MageCantrip_RayOfFrost(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Mage",
		PendingChoices: []string{"cantrip"},
	}
	if !ApplyMageCantrip(s, "ray_of_frost") {
		t.Fatal("ApplyMageCantrip returned false")
	}
	if len(s.KnownCantrips) != 1 || s.KnownCantrips[0] != "ray_of_frost" {
		t.Errorf("KnownCantrips = %v, want [ray_of_frost]", s.KnownCantrips)
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty", s.PendingChoices)
	}
}

func TestApply_MageCantrip_DuplicateRejected(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Mage",
		KnownCantrips:  []string{"ray_of_frost"},
		PendingChoices: []string{"cantrip"},
	}
	if ApplyMageCantrip(s, "ray_of_frost") {
		t.Error("should reject duplicate cantrip")
	}
}

func TestApply_MageCantrip_InvalidCantrip(t *testing.T) {
	s := &CombatStats{
		Level: 3, Class: "Mage",
		PendingChoices: []string{"cantrip"},
	}
	if ApplyMageCantrip(s, "prestidigitation") {
		t.Error("should reject invalid cantrip")
	}
}

func TestApply_MageCantrip_NoPending(t *testing.T) {
	s := &CombatStats{Level: 3, Class: "Mage"}
	if ApplyMageCantrip(s, "ray_of_frost") {
		t.Error("should fail with no pending cantrip")
	}
}

// --- Multi-jump full resolution ---

func TestApply_FighterMultiJump_ResolveAll(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Fighter", HitDieSize: 10,
		HP: 12, MaxHP: 12, CON: 14, AC: 16, MoveSpeed: 5,
		ClassCharges: 1, MaxClassCharges: 1,
		HitDice: 1, MaxHitDice: 1,
		XP: 2700,
	}
	checkLevelUp(s)
	if len(s.PendingChoices) != 2 {
		t.Fatalf("PendingChoices = %v, want 2", s.PendingChoices)
	}
	if !ApplyFighterStyle(s, "Gladiator") {
		t.Fatal("style apply failed")
	}
	if !ApplyFighterTechnique(s, "Quick Strike") {
		t.Fatal("technique apply failed")
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty after resolving all", s.PendingChoices)
	}
}

func TestApply_MageMultiJump_ResolveAll(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Mage", HitDieSize: 6,
		HP: 7, MaxHP: 7, CON: 12,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 1, MaxHitDice: 1,
		XP: 2700,
	}
	checkLevelUp(s)
	if len(s.PendingChoices) != 3 {
		t.Fatalf("PendingChoices = %v, want 3", s.PendingChoices)
	}
	if !ApplyMageSpell(s, "burning_hands") {
		t.Fatal("first spell failed")
	}
	if !ApplyMageCantrip(s, "shocking_grasp") {
		t.Fatal("cantrip failed")
	}
	if !ApplyMageSpell(s, "frost_nova") {
		t.Fatal("second spell failed")
	}
	if len(s.PendingChoices) != 0 {
		t.Errorf("PendingChoices = %v, want empty", s.PendingChoices)
	}
	if len(s.KnownSpells) != 2 {
		t.Errorf("KnownSpells = %v, want 2", s.KnownSpells)
	}
	if len(s.KnownCantrips) != 1 {
		t.Errorf("KnownCantrips = %v, want 1", s.KnownCantrips)
	}
}

// --- Game state gate ---

func TestGate_PendingLevelUpEntity_NoneByDefault(t *testing.T) {
	g, _ := newTestGame()
	if idx := g.PendingLevelUpEntity(); idx != -1 {
		t.Errorf("PendingLevelUpEntity() = %d, want -1 with no pending", idx)
	}
}

func TestGate_PendingLevelUpEntity_DetectsFighter(t *testing.T) {
	g, _ := newTestGame()
	g.Entities[0].Stats.PendingChoices = []string{"combat_style"}
	idx := g.PendingLevelUpEntity()
	if idx != 0 {
		t.Errorf("PendingLevelUpEntity() = %d, want 0", idx)
	}
}

func TestGate_PendingLevelUpEntity_DetectsMage(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)
	g.Entities[1].Stats.PendingChoices = []string{"spell_l1"}
	idx := g.PendingLevelUpEntity()
	if idx != 1 {
		t.Errorf("PendingLevelUpEntity() = %d, want 1", idx)
	}
}

func TestGate_PendingLevelUpEntity_FirstEntityWins(t *testing.T) {
	g, w := newTestGame()
	addElara(g, w)
	g.Entities[0].Stats.PendingChoices = []string{"combat_style"}
	g.Entities[1].Stats.PendingChoices = []string{"spell_l1"}
	idx := g.PendingLevelUpEntity()
	if idx != 0 {
		t.Errorf("PendingLevelUpEntity() = %d, want 0 (first entity)", idx)
	}
}

func TestGate_PendingLevelUpEntity_ClearsAfterResolve(t *testing.T) {
	g, _ := newTestGame()
	g.Entities[0].Stats.PendingChoices = []string{"combat_style"}
	ApplyFighterStyle(g.Entities[0].Stats, "Gladiator")
	if idx := g.PendingLevelUpEntity(); idx != -1 {
		t.Errorf("PendingLevelUpEntity() = %d, want -1 after resolving", idx)
	}
}

func TestGate_PendingLevelUpEntity_FromXPKill(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Stats.Level = 2
	brynn.Stats.XP = 880 // 20 short of level 3 (900)
	brynn.Stats.HP = 20
	brynn.Stats.MaxHP = 20
	brynn.Stats.HitDice = 2
	brynn.Stats.MaxHitDice = 2

	// Place a warrior worth 100 XP — pushes past 900
	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonWarrior, RoomIdx: 0,
		HP: 1, MaxHP: 13, AC: 13, STR: 10, DEX: 14,
		MoveSpeed: 4, AttackDice: 6, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	for i := 0; i < 20 && w.IsThreatAt(tx, tz); i++ {
		executeMeleeAttack(g, w, brynn, tx, tz)
	}

	if brynn.Stats.Level != 3 {
		t.Fatalf("Level = %d, want 3", brynn.Stats.Level)
	}
	idx := g.PendingLevelUpEntity()
	if idx != 0 {
		t.Errorf("PendingLevelUpEntity() = %d, want 0 after level 3", idx)
	}
	if len(brynn.Stats.PendingChoices) != 1 || brynn.Stats.PendingChoices[0] != "combat_style" {
		t.Errorf("PendingChoices = %v, want [combat_style]", brynn.Stats.PendingChoices)
	}
}

// --- Fields exist and default to zero values ---

func TestLevelUp_NewFieldsDefaultEmpty(t *testing.T) {
	s := &CombatStats{Level: 1, Class: "Fighter"}
	if s.CombatStyle != "" {
		t.Error("CombatStyle should default to empty")
	}
	if s.CombatTechnique != "" {
		t.Error("CombatTechnique should default to empty")
	}
	if len(s.KnownSpells) != 0 {
		t.Error("KnownSpells should default to empty")
	}
	if len(s.KnownCantrips) != 0 {
		t.Error("KnownCantrips should default to empty")
	}
	if len(s.PendingChoices) != 0 {
		t.Error("PendingChoices should default to empty")
	}
}
