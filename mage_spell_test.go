package main

import "testing"

// --- Level 1 Mage: only starting kit + dash + end turn ---

func TestMageSpells_Level1_StartingKit(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)
	elara.Stats.Level = 1
	elara.Stats.KnownSpells = nil
	elara.Stats.KnownCantrips = nil

	tx, tz := elara.X+1, elara.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}
	g.StartCombat(w, 1, 0)
	actions := BuildActions(g, elara)

	ids := map[string]bool{}
	for _, a := range actions {
		ids[a.ID] = true
	}

	if !ids["fire_bolt"] {
		t.Error("missing Fire Bolt (starting cantrip)")
	}
	if !ids["magic_missile"] {
		t.Error("missing Magic Missile (starting spell)")
	}
	if !ids["dash"] {
		t.Error("missing Dash")
	}
	if !ids["end_turn"] {
		t.Error("missing End Turn")
	}
	// Should have exactly 4 actions at level 1
	if len(actions) != 4 {
		names := make([]string, len(actions))
		for i, a := range actions {
			names[i] = a.ID
		}
		t.Errorf("expected 4 actions, got %d: %v", len(actions), names)
	}
}

// --- Learned spells appear in action bar ---

func TestMageSpells_LearnedSpellsAppear(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)
	elara.Stats.Level = 2
	elara.Stats.KnownSpells = []string{"burning_hands"}
	elara.Stats.KnownCantrips = []string{"ray_of_frost"}

	tx, tz := elara.X+1, elara.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}
	g.StartCombat(w, 1, 0)
	actions := BuildActions(g, elara)

	ids := map[string]bool{}
	for _, a := range actions {
		ids[a.ID] = true
	}
	if !ids["ray_of_frost"] {
		t.Error("ray_of_frost should appear after learning")
	}
	if !ids["burning_hands"] {
		t.Error("burning_hands should appear after learning")
	}
}

// --- Registry contains all learnable spells ---

func TestMageSpells_RegistryComplete(t *testing.T) {
	// All spells from validSpellsL1 should be in the registry
	for spellID := range validSpellsL1 {
		if _, ok := mageActionDefs[spellID]; !ok {
			t.Errorf("spell %s missing from mageActionDefs", spellID)
		}
	}
	// All cantrips from validCantrips should be in the registry
	for cantripID := range validCantrips {
		if _, ok := mageActionDefs[cantripID]; !ok {
			t.Errorf("cantrip %s missing from mageActionDefs", cantripID)
		}
	}
}

// --- Hotkeys assigned sequentially ---

func TestMageSpells_HotkeySequence(t *testing.T) {
	g, w := newTestGame()
	elara := addElara(g, w)
	elara.Stats.Level = 1
	elara.Stats.KnownSpells = nil
	elara.Stats.KnownCantrips = nil

	tx, tz := elara.X+1, elara.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}
	g.StartCombat(w, 1, 0)
	actions := BuildActions(g, elara)

	// Fire Bolt = 1, Magic Missile = 2, Dash = 3
	expected := map[string]string{
		"fire_bolt":      "1",
		"magic_missile":  "2",
		"dash":           "3",
	}
	for _, a := range actions {
		if exp, ok := expected[a.ID]; ok {
			if a.Hotkey != exp {
				t.Errorf("%s hotkey = %s, want %s", a.ID, a.Hotkey, exp)
			}
		}
	}
}
