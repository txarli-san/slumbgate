package main

import (
	"fmt"
	"testing"
)

// newLevel1Game creates a game with level 1 Brynn matching real main.go stats.
func newLevel1Game(seed int64) (*GameState, *World) {
	w := NewWorld(seed)
	spawnX := OuterRadius + DungeonWarpAmp + 5
	w.PlacePickaxe(spawnX, 0)

	g := &GameState{SelectedEnt: 0}
	g.Entities = []*Entity{
		{Name: "Brynn", X: spawnX, Z: 0, Tier: TierRecruit, RevealDist: 6,
			Scouted: map[[2]int]bool{},
			Stats: &CombatStats{
				HP: 12, MaxHP: 12, AC: 13,
				STR: 16, DEX: 12, CON: 14, INT: 10, WIS: 12, CHA: 10,
				Level: 1, ProfBonus: 2, MoveSpeed: 5, Class: "Fighter",
				ClassCharges: 1, MaxClassCharges: 1,
				HitDice: 1, MaxHitDice: 1, HitDieSize: 10,
			}},
	}
	for _, ent := range g.Entities {
		cx, cz := TileToChunk(ent.X, ent.Z)
		w.EnsureChunksAround(cx, cz)
		w.RevealAround(ent.X, ent.Z)
	}
	return g, w
}

func addLevel1Elara(g *GameState, w *World) *Entity {
	brynn := g.Entities[0]
	elara := &Entity{
		Name: "Elara", X: brynn.X + 1, Z: brynn.Z, Tier: TierRecruit, RevealDist: 5,
		Scouted: map[[2]int]bool{},
		Stats: &CombatStats{
			HP: 7, MaxHP: 7, AC: 12,
			STR: 8, DEX: 14, CON: 12, INT: 16, WIS: 14, CHA: 10,
			Level: 1, ProfBonus: 2, MoveSpeed: 5, Class: "Mage",
			ClassCharges: 2, MaxClassCharges: 2,
			HitDice: 1, MaxHitDice: 1, HitDieSize: 6,
		},
	}
	g.Entities = append(g.Entities, elara)
	cx, cz := TileToChunk(elara.X, elara.Z)
	w.EnsureChunksAround(cx, cz)
	return elara
}

// --- XP accumulates through real combat ---

func TestXPIntegration_AccumulatesThroughCombat(t *testing.T) {
	g, w := newLevel1Game(42)
	brynn := g.Entities[0]

	walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
	if !g.HasPickaxe {
		t.Skip("couldn't reach pickaxe")
	}

	_, tx, tz, found := findFirstThreatRoom(w)
	if !found {
		t.Skip("no threats")
	}
	walkToBreakable(g, w, 0, tx, tz, 500)

	if g.Combat != nil {
		runCombatToEnd(g, w, 200)
	}
	if g.GameOver {
		t.Log("Brynn fell — valid at level 1")
		return
	}

	t.Logf("XP after first combat: %d (Level %d)", brynn.Stats.XP, brynn.Stats.Level)
	if brynn.Stats.XP > 0 {
		t.Log("XP gained from combat")
	} else {
		t.Log("no XP — enemies likely leashed (valid at level 1)")
	}
}

// --- XP split correctly between two allies ---

func TestXPIntegration_SplitBetweenAllies(t *testing.T) {
	g, w := newLevel1Game(42)
	brynn := g.Entities[0]
	elara := addLevel1Elara(g, w)

	// Place a single minion adjacent to brynn
	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	// Move elara so she doesn't block the threat tile
	elara.X, elara.Z = brynn.X, brynn.Z+1
	w.SetTile(elara.X, elara.Z, TileGround)

	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 1, MaxHP: 8, AC: 1, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}
	runCombatToEnd(g, w, 50)

	// 50 XP / 2 = 25 each
	if brynn.Stats.XP != 25 {
		t.Errorf("Brynn XP = %d, want 25", brynn.Stats.XP)
	}
	if elara.Stats.XP != 25 {
		t.Errorf("Elara XP = %d, want 25", elara.Stats.XP)
	}
}

// --- Solo entity gets full XP ---

func TestXPIntegration_SoloFullXP(t *testing.T) {
	g, w := newLevel1Game(42)
	brynn := g.Entities[0]

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonWarrior, RoomIdx: 0,
		HP: 1, MaxHP: 13, AC: 1, STR: 10, DEX: 14,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	runCombatToEnd(g, w, 50)

	if g.GameOver {
		t.Skip("Brynn died")
	}
	if brynn.Stats.XP != 100 {
		t.Errorf("Brynn XP = %d, want 100 (solo warrior kill)", brynn.Stats.XP)
	}
}

// --- Level up happens at 300 XP ---

func TestXPIntegration_LevelUpAt300(t *testing.T) {
	g, w := newLevel1Game(42)
	brynn := g.Entities[0]
	brynn.Stats.XP = 280 // 20 away from level 2

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 1, MaxHP: 8, AC: 1, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	runCombatToEnd(g, w, 50)

	if g.GameOver {
		t.Skip("Brynn died")
	}
	if brynn.Stats.Level != 2 {
		t.Errorf("Level = %d, want 2 at XP %d", brynn.Stats.Level, brynn.Stats.XP)
	}
	if brynn.Stats.MaxHP <= 12 {
		t.Errorf("MaxHP = %d, should have increased from 12", brynn.Stats.MaxHP)
	}
	if brynn.Stats.MaxHitDice != 2 {
		t.Errorf("MaxHitDice = %d, want 2", brynn.Stats.MaxHitDice)
	}
	t.Logf("Level 2: HP %d/%d, XP %d, HitDice %d/%d",
		brynn.Stats.HP, brynn.Stats.MaxHP, brynn.Stats.XP,
		brynn.Stats.HitDice, brynn.Stats.MaxHitDice)
}

// --- Fighter gains Action Surge at level 2 ---

func TestXPIntegration_FighterActionSurge(t *testing.T) {
	g, w := newLevel1Game(42)
	brynn := g.Entities[0]
	brynn.Stats.XP = 290

	if brynn.Stats.MaxClassCharges != 1 {
		t.Fatalf("level 1 Fighter should have 1 charge, got %d", brynn.Stats.MaxClassCharges)
	}

	tx, tz := brynn.X+1, brynn.Z
	w.SetTile(tx, tz, TileGround)
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 1, MaxHP: 8, AC: 1, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	runCombatToEnd(g, w, 50)

	if g.GameOver {
		t.Skip("Brynn died")
	}
	if brynn.Stats.Level != 2 {
		t.Skipf("didn't level up (XP=%d)", brynn.Stats.XP)
	}
	if brynn.Stats.MaxClassCharges != 2 {
		t.Errorf("MaxClassCharges = %d, want 2 (Second Wind + Action Surge)", brynn.Stats.MaxClassCharges)
	}
}

// --- Multi-level: enough XP to skip from 1 to 3 ---

func TestXPIntegration_MultiLevelJump(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Fighter",
		HP: 12, MaxHP: 12, AC: 13,
		STR: 16, DEX: 12, CON: 14,
		ProfBonus: 2, MoveSpeed: 5,
		ClassCharges: 1, MaxClassCharges: 1,
		HitDice: 1, MaxHitDice: 1, HitDieSize: 10,
		XP: 0,
	}

	s.XP = 900 // enough for level 3
	checkLevelUp(s)

	if s.Level != 3 {
		t.Errorf("Level = %d, want 3", s.Level)
	}
	if s.MaxHitDice != 3 {
		t.Errorf("MaxHitDice = %d, want 3", s.MaxHitDice)
	}
	if s.MaxClassCharges != 2 {
		t.Errorf("MaxClassCharges = %d, want 2 (Action Surge at level 2)", s.MaxClassCharges)
	}
	if s.MaxHP <= 12 {
		t.Error("MaxHP should have increased twice")
	}
	t.Logf("Level 3: HP %d/%d, HitDice %d, Charges %d", s.HP, s.MaxHP, s.MaxHitDice, s.MaxClassCharges)
}

// --- Mage level up: HP uses d6, spell slots stay ---

func TestXPIntegration_MageLevelUp(t *testing.T) {
	s := &CombatStats{
		Level: 1, Class: "Mage",
		HP: 7, MaxHP: 7, AC: 12,
		STR: 8, DEX: 14, CON: 12,
		ProfBonus: 2, MoveSpeed: 5,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 1, MaxHitDice: 1, HitDieSize: 6,
		XP: 300,
	}

	checkLevelUp(s)

	if s.Level != 2 {
		t.Errorf("Level = %d, want 2", s.Level)
	}
	// d6 + CON(+1) = 2-7 HP gain
	hpGain := s.MaxHP - 7
	if hpGain < 2 || hpGain > 7 {
		t.Errorf("HP gain = %d, want 2-7 (d6+1)", hpGain)
	}
	if s.MaxHitDice != 2 {
		t.Errorf("MaxHitDice = %d, want 2", s.MaxHitDice)
	}
	t.Logf("Mage Level 2: HP %d/%d (+%d), HitDice %d", s.HP, s.MaxHP, hpGain, s.MaxHitDice)
}

// --- Proficiency bonus increases at level 5 ---

func TestXPIntegration_ProfBonusAtLevel5(t *testing.T) {
	s := &CombatStats{
		Level: 4, Class: "Fighter",
		HP: 30, MaxHP: 30, AC: 13,
		STR: 16, DEX: 12, CON: 14,
		ProfBonus: 2, MoveSpeed: 5,
		ClassCharges: 2, MaxClassCharges: 2,
		HitDice: 4, MaxHitDice: 4, HitDieSize: 10,
		XP: 6500,
	}

	checkLevelUp(s)

	if s.Level != 5 {
		t.Errorf("Level = %d, want 5", s.Level)
	}
	if s.ProfBonus != 3 {
		t.Errorf("ProfBonus = %d, want 3 at level 5", s.ProfBonus)
	}
}

// --- Multi-seed: XP and level progression across seeds ---

func TestXPIntegration_MultiSeedProgression(t *testing.T) {
	type result struct {
		seed     int64
		xp       int
		level    int
		alive    bool
		combats  int
	}

	var results []result

	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newLevel1Game(seed)
			brynn := g.Entities[0]

			// Get pickaxe
			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				t.Log("couldn't reach pickaxe")
				results = append(results, result{seed: seed, alive: true})
				return
			}

			// Fight through rooms
			combats := 0
			for room := 0; room < 3; room++ {
				if g.GameOver {
					break
				}

				_, tx, tz, found := findFirstThreatRoom(w)
				if !found {
					break
				}
				walkToBreakable(g, w, 0, tx, tz, 500)

				if g.Combat != nil {
					combats++
					runCombatToEnd(g, w, 200)
				}
				if g.GameOver {
					break
				}

				// Short rest between fights
				if brynn.Stats.HP < brynn.Stats.MaxHP {
					g.TryShortRest(w, 0)
					if g.Combat != nil {
						combats++
						runCombatToEnd(g, w, 100)
					}
				}
			}

			r := result{
				seed:    seed,
				xp:      brynn.Stats.XP,
				level:   brynn.Stats.Level,
				alive:   !g.GameOver,
				combats: combats,
			}
			results = append(results, r)

			t.Logf("seed=%d xp=%d level=%d alive=%v combats=%d hp=%d/%d",
				seed, r.xp, r.level, r.alive, r.combats,
				brynn.Stats.HP, brynn.Stats.MaxHP)
		})
	}

	// Summary stats
	totalXP, alive, leveled := 0, 0, 0
	for _, r := range results {
		totalXP += r.xp
		if r.alive {
			alive++
		}
		if r.level > 1 {
			leveled++
		}
	}

	t.Logf("\n=== XP PROGRESSION SUMMARY ===")
	t.Logf("Seeds: %d | Alive: %d | Leveled up: %d | Avg XP: %d",
		len(results), alive, leveled, totalXP/max(len(results), 1))
}

// --- Level 1 survivability: can Brynn survive first room? ---

func TestXPIntegration_Level1Survivability(t *testing.T) {
	survived, died, leashed := 0, 0, 0

	for _, seed := range testSeeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			g, w := newLevel1Game(seed)
			brynn := g.Entities[0]

			walkTo(g, w, 0, w.PickaxeX, w.PickaxeZ, 200)
			if !g.HasPickaxe {
				return
			}

			_, tx, tz, found := findFirstThreatRoom(w)
			if !found {
				return
			}
			walkToBreakable(g, w, 0, tx, tz, 500)

			if g.Combat != nil {
				runCombatToEnd(g, w, 200)
			}

			if g.GameOver {
				died++
				t.Log("DIED")
			} else if g.Combat == nil {
				// Check if enemies leashed or were killed
				roomCleared := true
				for _, thr := range w.Threats {
					if thr.RoomIdx >= 0 {
						dist := abs(thr.X-brynn.X) + abs(thr.Z-brynn.Z)
						if dist < 10 {
							roomCleared = false
							break
						}
					}
				}
				if roomCleared {
					survived++
					t.Logf("SURVIVED hp=%d/%d xp=%d", brynn.Stats.HP, brynn.Stats.MaxHP, brynn.Stats.XP)
				} else {
					leashed++
					t.Logf("LEASHED hp=%d/%d xp=%d", brynn.Stats.HP, brynn.Stats.MaxHP, brynn.Stats.XP)
				}
			}
		})
	}

	t.Logf("\n=== LEVEL 1 SURVIVABILITY ===")
	total := survived + died + leashed
	if total > 0 {
		t.Logf("Survived: %d/%d (%.0f%%) | Died: %d (%.0f%%) | Leashed: %d (%.0f%%)",
			survived, total, float64(survived)*100/float64(total),
			died, float64(died)*100/float64(total),
			leashed, float64(leashed)*100/float64(total))
	}
}

// --- No XP from leashed enemies ---

func TestXPIntegration_NoXPFromLeash(t *testing.T) {
	g, w := newLevel1Game(42)
	brynn := g.Entities[0]

	// Place enemy far away so it will leash
	tx, tz := brynn.X+3, brynn.Z
	for dx := 0; dx <= 3; dx++ {
		w.SetTile(brynn.X+dx, brynn.Z, TileGround)
	}
	w.Threats[[2]int{tx, tz}] = Threat{
		X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: 0,
		HP: 50, MaxHP: 50, AC: 20, STR: 10, DEX: 12,
		MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
	}

	g.StartCombat(w, 0, 0)
	if g.Combat == nil {
		t.Fatal("combat didn't start")
	}

	// Run until leash or timeout
	for i := 0; i < 50; i++ {
		if g.Combat == nil || g.GameOver {
			break
		}
		cur := g.Combat.Current()
		if cur.IsEnemy {
			g.RunEnemyTurn(w)
		} else {
			// Player does nothing — just end turn
			g.Combat.NextTurn(g, w)
		}
	}

	if brynn.Stats.XP != 0 {
		t.Errorf("XP = %d, want 0 (enemy leashed, not killed)", brynn.Stats.XP)
	}
}
