package main

import "testing"

func TestGearForLevel_Knight(t *testing.T) {
	// Gear: 0-8, Body: 9-14
	// L1: body only
	v := gearForLevel("Fighter", 1)
	if len(v) != 15 {
		t.Fatalf("len = %d, want 15", len(v))
	}
	for i := 0; i <= 8; i++ {
		if v[i] {
			t.Errorf("gear mesh %d should be hidden at L1", i)
		}
	}
	for i := 9; i <= 14; i++ {
		if !v[i] {
			t.Errorf("body mesh %d should be visible at L1", i)
		}
	}

	// L2: +1H_Sword (5)
	v = gearForLevel("Fighter", 2)
	if !v[5] {
		t.Error("mesh 5 (1H_Sword) should be visible at L2")
	}

	// L3: +Cape (8)
	v = gearForLevel("Fighter", 3)
	if !v[8] {
		t.Error("mesh 8 (Cape) should be visible at L3")
	}
	if !v[5] {
		t.Error("mesh 5 (1H_Sword) should still be visible at L3")
	}

	// L4: +Helmet (7) +Round_Shield (3)
	v = gearForLevel("Fighter", 4)
	if !v[7] || !v[3] {
		t.Error("mesh 7 (Helmet) and 3 (Round_Shield) should be visible at L4")
	}
	if !v[5] || !v[8] {
		t.Error("prior gear should remain visible at L4")
	}
}

func TestGearForLevel_Mage(t *testing.T) {
	// Gear: 0-5, Body: 6-11
	// L1: body only
	v := gearForLevel("Mage", 1)
	if len(v) != 12 {
		t.Fatalf("len = %d, want 12", len(v))
	}
	for i := 0; i <= 5; i++ {
		if v[i] {
			t.Errorf("gear mesh %d should be hidden at L1", i)
		}
	}
	for i := 6; i <= 11; i++ {
		if !v[i] {
			t.Errorf("body mesh %d should be visible at L1", i)
		}
	}

	// L2: +1H_Wand (2)
	v = gearForLevel("Mage", 2)
	if !v[2] {
		t.Error("mesh 2 (1H_Wand) should be visible at L2")
	}

	// L3: +Cape (5)
	v = gearForLevel("Mage", 3)
	if !v[5] || !v[2] {
		t.Error("cape and wand should be visible at L3")
	}

	// L4: +Hat (4) +Spellbook (0)
	v = gearForLevel("Mage", 4)
	if !v[4] || !v[0] {
		t.Error("hat and spellbook should be visible at L4")
	}
	if !v[2] || !v[5] {
		t.Error("prior gear should remain visible at L4")
	}
}

func TestGearForLevel_UnknownClass(t *testing.T) {
	v := gearForLevel("Cleric", 1)
	if v != nil {
		t.Error("unknown class should return nil (draw all)")
	}
}
