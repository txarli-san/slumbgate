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
