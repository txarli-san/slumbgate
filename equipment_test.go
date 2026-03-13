package main

import "testing"

func TestEquip_GearAC(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 13, Class: "Fighter"},
	}

	if ent.GearAC() != 0 {
		t.Fatalf("expected 0 gear AC, got %d", ent.GearAC())
	}

	// Equip round shield (+2 AC)
	ent.Equip(&AllGear[5]) // Round Shield
	if ent.GearAC() != 2 {
		t.Fatalf("expected 2 gear AC with Round Shield, got %d", ent.GearAC())
	}
	if ent.EffectiveAC() != 15 {
		t.Fatalf("expected effective AC 15, got %d", ent.EffectiveAC())
	}

	// Add helmet (+1 AC)
	ent.Equip(&AllGear[7]) // Helmet
	if ent.GearAC() != 3 {
		t.Fatalf("expected 3 gear AC with shield+helmet, got %d", ent.GearAC())
	}
}

func TestEquip_GearHitAndDamage(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 12, Class: "Mage"},
	}

	// Equip wand (+1 hit)
	ent.Equip(&AllGear[9]) // Wand
	if ent.GearHit() != 1 {
		t.Fatalf("expected 1 gear hit with Wand, got %d", ent.GearHit())
	}

	// Add spellbook (+1 damage)
	ent.Equip(&AllGear[11]) // Spellbook
	if ent.GearDamage() != 1 {
		t.Fatalf("expected 1 gear damage with Spellbook, got %d", ent.GearDamage())
	}
}

func TestEquip_WeaponDie(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 13, Class: "Fighter"},
	}

	// Default (no weapon): d8
	if ent.WeaponDie() != 8 {
		t.Fatalf("expected default weapon die 8, got %d", ent.WeaponDie())
	}

	// Equip longsword (d8)
	ent.Equip(&AllGear[0]) // Longsword
	if ent.WeaponDie() != 8 {
		t.Fatalf("expected d8 with longsword, got %d", ent.WeaponDie())
	}

	// Equip greatsword (d12)
	ent.Equip(&AllGear[1]) // Greatsword
	if ent.WeaponDie() != 12 {
		t.Fatalf("expected d12 with greatsword, got %d", ent.WeaponDie())
	}
}

func TestEquip_TwoHandedClearsOffhand(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 13, Class: "Fighter"},
	}

	// Equip shield first
	ent.Equip(&AllGear[5]) // Round Shield → OffHand
	if _, ok := ent.Equipment[SlotOffHand]; !ok {
		t.Fatal("shield should be in offhand")
	}

	// Equip greatsword (2H) — should clear offhand
	ent.Equip(&AllGear[1]) // Greatsword → MainHand, 2H
	if _, ok := ent.Equipment[SlotOffHand]; ok {
		t.Fatal("offhand should be cleared when equipping 2H weapon")
	}
	if _, ok := ent.Equipment[SlotMainHand]; !ok {
		t.Fatal("greatsword should be in mainhand")
	}
}

func TestEquip_OffhandClears2H(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 13, Class: "Fighter"},
	}

	// Equip greatsword (2H)
	ent.Equip(&AllGear[1])
	if ent.Equipment[SlotMainHand].Name != "Greatsword" {
		t.Fatal("should have greatsword")
	}

	// Equip shield → should clear 2H main hand
	ent.Equip(&AllGear[5]) // Round Shield
	if _, ok := ent.Equipment[SlotMainHand]; ok {
		t.Fatal("2H weapon should be cleared when equipping offhand")
	}
	if ent.Equipment[SlotOffHand].Name != "Round Shield" {
		t.Fatal("shield should be equipped")
	}
}

func TestEquip_Unequip(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 13, Class: "Fighter"},
	}

	ent.Equip(&AllGear[7]) // Helmet
	if ent.GearAC() != 1 {
		t.Fatal("helmet should give +1 AC")
	}

	ent.Unequip(SlotHead)
	if ent.GearAC() != 0 {
		t.Fatal("AC should be 0 after unequipping helmet")
	}
}

func TestEquip_RebuildVisibleMeshes(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 13, Class: "Fighter"},
	}
	ent.RebuildVisibleMeshes()

	// Body meshes 9-14 should be visible, gear 0-8 hidden
	for i := 0; i <= 8; i++ {
		if ent.VisibleMeshes[i] {
			t.Errorf("gear mesh %d should be hidden with no equipment", i)
		}
	}
	for i := 9; i <= 14; i++ {
		if !ent.VisibleMeshes[i] {
			t.Errorf("body mesh %d should be visible", i)
		}
	}

	// Equip longsword (mesh 5)
	ent.Equip(&AllGear[0])
	if !ent.VisibleMeshes[5] {
		t.Error("mesh 5 (longsword) should be visible after equipping")
	}

	// Unequip
	ent.Unequip(SlotMainHand)
	if ent.VisibleMeshes[5] {
		t.Error("mesh 5 should be hidden after unequipping")
	}
}

func TestEquip_GearForClass(t *testing.T) {
	fighter := GearForClass("Fighter")
	if len(fighter) != 9 {
		t.Fatalf("expected 9 fighter items, got %d", len(fighter))
	}

	mage := GearForClass("Mage")
	if len(mage) != 6 {
		t.Fatalf("expected 6 mage items, got %d", len(mage))
	}

	cleric := GearForClass("Cleric")
	if len(cleric) != 0 {
		t.Fatalf("expected 0 cleric items, got %d", len(cleric))
	}
}

func TestEquip_EffectiveACInCombat(t *testing.T) {
	g, w := newTestGame()
	brynn := g.Entities[0]
	brynn.Equipment = map[EquipSlot]*GearItem{}
	baseAC := brynn.Stats.AC

	// Equip round shield (+2 AC)
	brynn.Equip(&AllGear[5])

	// Place a weak enemy
	tx, tz := brynn.X+1, brynn.Z
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 100, MaxHP: 100, AC: 1,
		STR: 10, DEX: 10, MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	// Verify effective AC
	if brynn.EffectiveAC() != baseAC+2 {
		t.Fatalf("expected effective AC %d, got %d", baseAC+2, brynn.EffectiveAC())
	}
}

func TestEquip_MageStaff2H(t *testing.T) {
	ent := &Entity{
		Equipment: map[EquipSlot]*GearItem{},
		Stats:     &CombatStats{AC: 12, Class: "Mage"},
	}

	// Equip spellbook (offhand)
	ent.Equip(&AllGear[11])
	if _, ok := ent.Equipment[SlotOffHand]; !ok {
		t.Fatal("spellbook should be in offhand")
	}

	// Equip staff (2H) — should clear offhand
	ent.Equip(&AllGear[10]) // Staff
	if _, ok := ent.Equipment[SlotOffHand]; ok {
		t.Fatal("offhand should be cleared for 2H staff")
	}
	if ent.GearHit() != 1 {
		t.Fatalf("expected staff hit +1, got %d", ent.GearHit())
	}
	if ent.GearDamage() != 1 {
		t.Fatalf("expected staff damage +1, got %d", ent.GearDamage())
	}
}

func TestPlaceChest(t *testing.T) {
	w := NewWorld(testSeed)
	spawnX := OuterRadius + DungeonWarpAmp + 5
	w.PlaceChest(spawnX, 0)

	// Chest should be on ground
	tile := w.BaseTileType(w.ChestX, w.ChestZ)
	if tile != TileGround {
		t.Fatalf("chest at (%d,%d) on tile type %d, expected TileGround", w.ChestX, w.ChestZ, tile)
	}

	// Should be 3-5 tiles from spawn
	dx := w.ChestX - spawnX
	dz := w.ChestZ - 0
	dist2 := dx*dx + dz*dz
	if dist2 < 3*3 || dist2 > 5*5+1 {
		t.Fatalf("chest distance² from spawn = %d, expected 9-25", dist2)
	}
}
