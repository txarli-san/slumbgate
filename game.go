package main

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
)

type CameraMode int

const (
	CameraLocal    CameraMode = iota // 3D tactical view
	CameraStrategic                  // 2D top-down map
)

type AutoTier int

const (
	TierRecruit    AutoTier = iota // Full manual — can't auto-resolve anything
	TierSoldier                    // Supervised — handles weak threats
	TierVeteran                    // Autonomous — handles most threats
	TierLieutenant                 // Command node — detects before arrival
)

type TaskType int

const (
	TaskIdle    TaskType = iota
	TaskMoveTo
	TaskExplore
	TaskRest
	TaskFollow
)

type Task struct {
	Type         TaskType
	TargetX      int
	TargetZ      int
	Path         [][2]int
	PathIdx      int
	ExploreAngle float64 // current angle for perimeter patrol
	FollowIdx    int     // entity index to follow (TaskFollow)
}

type CombatStats struct {
	HP, MaxHP    int
	AC           int
	STR, DEX, CON, INT, WIS, CHA int
	Level        int
	ProfBonus    int
	MoveSpeed    int // tiles per combat turn
	Class           string // "Fighter", "Mage", etc.
	ClassCharges    int    // Second Wind / Action Surge uses
	MaxClassCharges int    // starting value, restored on rest
	HitDice         int    // remaining hit dice for short rest healing
	MaxHitDice      int    // = Level; recovered half on long rest
	HitDieSize      int    // die size: Fighter=10, Mage=6, Cleric/Rogue/Druid=8
	XP              int    // accumulated experience points
	Exhaustion      int      // 0-6; 6 = death
	CombatStyle     string   // Fighter L3: "Gladiator", "Ranger", "Juggernaut"
	CombatTechnique string   // Fighter L4: "Power Attack", "Defensive Stance", "Quick Strike"
	KnownSpells     []string // Mage: learned level 1 spells
	KnownCantrips   []string // Mage: learned cantrips
	PendingChoices  []string // level-up choices waiting to be resolved
}

// 5e SRD XP thresholds per level
func xpForLevel(level int) int {
	thresholds := map[int]int{
		2: 300, 3: 900, 4: 2700, 5: 6500,
		6: 14000, 7: 23000, 8: 34000, 9: 48000, 10: 64000,
		11: 85000, 12: 100000, 13: 120000, 14: 140000, 15: 165000,
		16: 195000, 17: 225000, 18: 265000, 19: 305000, 20: 355000,
	}
	return thresholds[level]
}

// 5e SRD XP by skeleton CR
func threatXP(stype SkeletonType) int {
	switch stype {
	case SkeletonMinion:
		return 50 // CR 1/4
	case SkeletonWarrior:
		return 100 // CR 1/2
	case SkeletonRogue:
		return 100 // CR 1/2
	case SkeletonMage:
		return 200 // CR 1
	}
	return 0
}

// awardXP splits XP among all allies in combat
func (g *GameState) awardXP(w *World, xp int) {
	if g.Combat == nil {
		return
	}
	allyCount := 0
	for _, cb := range g.Combat.Combatants {
		if !cb.IsEnemy && cb.EntityIdx < len(g.Entities) {
			allyCount++
		}
	}
	if allyCount == 0 {
		return
	}
	share := xp / allyCount
	if share < 1 {
		share = 1
	}
	for _, cb := range g.Combat.Combatants {
		if !cb.IsEnemy && cb.EntityIdx < len(g.Entities) {
			ent := g.Entities[cb.EntityIdx]
			if ent.Stats != nil {
				prevLevel := ent.Stats.Level
				ent.Stats.XP += share
				checkLevelUp(ent.Stats)
				g.AddFloat(fmt.Sprintf("+%d XP", share), ent.X, ent.Z, 255, 215, 0, 18)
				if ent.Stats.Level > prevLevel {
					g.SetMessage(fmt.Sprintf("%s reached Level %d!", ent.Name, ent.Stats.Level))
					g.AddFloat("LEVEL UP!", ent.X, ent.Z, 255, 255, 100, 24)
				}
			}
		}
	}
}

func checkLevelUp(s *CombatStats) {
	for s.Level < 20 {
		needed := xpForLevel(s.Level + 1)
		if needed == 0 || s.XP < needed {
			break
		}
		levelUp(s)
	}
}

func levelUp(s *CombatStats) {
	s.Level++
	// HP: roll hit die + CON mod (minimum 1)
	roll := rollDice(s.HitDieSize)
	gain := roll + s.Mod(s.CON)
	if gain < 1 {
		gain = 1
	}
	s.MaxHP += gain
	s.HP += gain
	// Hit dice
	s.MaxHitDice = s.Level
	s.HitDice++
	// Proficiency bonus: 5e SRD
	switch {
	case s.Level >= 17:
		s.ProfBonus = 6
	case s.Level >= 13:
		s.ProfBonus = 5
	case s.Level >= 9:
		s.ProfBonus = 4
	case s.Level >= 5:
		s.ProfBonus = 3
	default:
		s.ProfBonus = 2
	}
	// Class features and pending choices
	switch s.Class {
	case "Fighter":
		if s.Level == 2 {
			s.MaxClassCharges = 2
			s.ClassCharges = s.MaxClassCharges
		}
		if s.Level == 3 {
			s.PendingChoices = append(s.PendingChoices, "combat_style")
		}
		if s.Level == 4 {
			s.PendingChoices = append(s.PendingChoices, "combat_technique")
		}
	case "Mage":
		if s.Level == 2 {
			s.PendingChoices = append(s.PendingChoices, "spell_l1")
		}
		if s.Level == 3 {
			s.PendingChoices = append(s.PendingChoices, "cantrip")
		}
		if s.Level == 4 {
			s.PendingChoices = append(s.PendingChoices, "spell_l1")
		}
	}
}

// removePending removes the first occurrence of choiceType from PendingChoices.
func (s *CombatStats) removePending(choiceType string) bool {
	for i, c := range s.PendingChoices {
		if c == choiceType {
			s.PendingChoices = append(s.PendingChoices[:i], s.PendingChoices[i+1:]...)
			return true
		}
	}
	return false
}

// Valid combat style choices for Fighter L3
var validCombatStyles = map[string]bool{
	"Gladiator": true, "Ranger": true, "Juggernaut": true,
}

// ApplyFighterStyle resolves the Fighter L3 combat style choice.
func ApplyFighterStyle(s *CombatStats, style string) bool {
	if !validCombatStyles[style] {
		return false
	}
	if !s.removePending("combat_style") {
		return false
	}
	s.CombatStyle = style
	if style == "Juggernaut" {
		s.AC += 2
	}
	return true
}

// Valid combat technique choices for Fighter L4
var validCombatTechniques = map[string]bool{
	"Power Attack": true, "Defensive Stance": true, "Quick Strike": true,
}

// ApplyFighterTechnique resolves the Fighter L4 combat technique choice.
func ApplyFighterTechnique(s *CombatStats, technique string) bool {
	if !validCombatTechniques[technique] {
		return false
	}
	if !s.removePending("combat_technique") {
		return false
	}
	s.CombatTechnique = technique
	if technique == "Defensive Stance" {
		s.AC += 2
		s.MoveSpeed--
	}
	return true
}

// Valid L1 spells for Mage
var validSpellsL1 = map[string]bool{
	"arcane_blink": true, "burning_hands": true, "frost_nova": true, "mind_spike": true,
	"magic_armor": true, "elemental_strike": true, "feather_fall": true, "expeditious_retreat": true,
}

// ApplyMageSpell resolves a Mage spell choice (L2 or L4 level-up).
func ApplyMageSpell(s *CombatStats, spellID string) bool {
	if !validSpellsL1[spellID] {
		return false
	}
	for _, known := range s.KnownSpells {
		if known == spellID {
			return false // already known
		}
	}
	if !s.removePending("spell_l1") {
		return false
	}
	s.KnownSpells = append(s.KnownSpells, spellID)
	return true
}

// Valid cantrips for Mage
var validCantrips = map[string]bool{
	"ray_of_frost": true, "shocking_grasp": true,
}

// ApplyMageCantrip resolves the Mage L3 cantrip choice.
func ApplyMageCantrip(s *CombatStats, cantripID string) bool {
	if !validCantrips[cantripID] {
		return false
	}
	for _, known := range s.KnownCantrips {
		if known == cantripID {
			return false // already known
		}
	}
	if !s.removePending("cantrip") {
		return false
	}
	s.KnownCantrips = append(s.KnownCantrips, cantripID)
	return true
}

func (s CombatStats) Mod(stat int) int { return (stat - 10) / 2 }

// AnimState tracks which animation is playing on an entity or threat.
type AnimState struct {
	Clip  string // animation name (e.g. "Idle", "Walking_A")
	Frame int32
	Time  float32 // elapsed time in seconds
	Loop  bool
	Done  bool // true when a non-looping clip finished
}

type Entity struct {
	Name           string
	X, Z           int
	PrevX, PrevZ   int
	StepProgress   float32
	Moving         bool
	FacingAngle    float32
	Tier           AutoTier
	RevealDist     int
	Task           *Task
	Scouted        map[[2]int]bool
	Stats             *CombatStats // nil = non-combatant
	LastLongRestTick  int          // TimeTicks when last long rest completed; 0 = game start
	Anim              AnimState
	VisibleMeshes  []bool // per-mesh visibility filter for gear progression (nil = draw all)
	Equipment      map[EquipSlot]*GearItem
}

// meshNames returns human-readable labels for each mesh index per class.
// Derived from GLTF node names in the KayKit models.
func meshNames(class string) []string {
	switch class {
	case "Fighter":
		return []string{
			"0:Sword_Offhand", "1:Badge_Shield", "2:Rect_Shield", "3:Round_Shield",
			"4:Spike_Shield", "5:1H_Sword", "6:2H_Sword", "7:Helmet", "8:Cape",
			"9:ArmLeft", "10:ArmRight", "11:Body", "12:Head", "13:LegLeft", "14:LegRight",
		}
	case "Mage":
		return []string{
			"0:Spellbook", "1:Spellbook_Open", "2:1H_Wand", "3:2H_Staff",
			"4:Hat", "5:Cape",
			"6:ArmLeft", "7:ArmRight", "8:Body", "9:Head", "10:LegLeft", "11:LegRight",
		}
	}
	return nil
}

// gearForLevel returns the visible mesh mask for a class at a given level.
// Knight: 15 meshes (0-8 gear, 9-14 body). Mage: 12 meshes (0-5 gear, 6-11 body).
func gearForLevel(class string, level int) []bool {
	switch class {
	case "Fighter":
		v := make([]bool, 15)
		// Body: 9-14 (ArmL, ArmR, Body, Head, LegL, LegR)
		for i := 9; i <= 14; i++ {
			v[i] = true
		}
		if level >= 2 {
			v[5] = true // 1H_Sword
		}
		if level >= 3 {
			v[8] = true // Cape
		}
		if level >= 4 {
			v[7] = true // Helmet
			v[3] = true // Round_Shield
		}
		return v
	case "Mage":
		v := make([]bool, 12)
		// Body: 6-11 (ArmL, ArmR, Body, Head, LegL, LegR)
		for i := 6; i <= 11; i++ {
			v[i] = true
		}
		if level >= 2 {
			v[2] = true // 1H_Wand
		}
		if level >= 3 {
			v[5] = true // Cape
		}
		if level >= 4 {
			v[4] = true // Hat
			v[0] = true // Spellbook
		}
		return v
	}
	return nil // unknown class = draw all
}

// Equipment system

type EquipSlot int

const (
	SlotMainHand EquipSlot = iota
	SlotOffHand
	SlotHead
	SlotBack
)

type GearItem struct {
	Name      string
	Slot      EquipSlot
	MeshIdx   int    // which mesh to show
	Class     string // "Fighter" or "Mage"
	TwoHanded bool   // blocks OffHand
	AC        int    // AC bonus
	Hit       int    // attack roll bonus
	Damage    int    // damage bonus
	DamageDie int    // weapon die override (0 = default d8)
}

// AllGear is the master item table.
var AllGear = []GearItem{
	// Fighter
	{Name: "Longsword", Slot: SlotMainHand, MeshIdx: 5, Class: "Fighter", DamageDie: 8},
	{Name: "Greatsword", Slot: SlotMainHand, MeshIdx: 6, Class: "Fighter", TwoHanded: true, DamageDie: 12},
	{Name: "Short Sword", Slot: SlotOffHand, MeshIdx: 0, Class: "Fighter", Damage: 1},
	{Name: "Buckler", Slot: SlotOffHand, MeshIdx: 1, Class: "Fighter", AC: 1},
	{Name: "Kite Shield", Slot: SlotOffHand, MeshIdx: 2, Class: "Fighter", AC: 2},
	{Name: "Round Shield", Slot: SlotOffHand, MeshIdx: 3, Class: "Fighter", AC: 2},
	{Name: "Spike Shield", Slot: SlotOffHand, MeshIdx: 4, Class: "Fighter", AC: 1, Damage: 1},
	{Name: "Helmet", Slot: SlotHead, MeshIdx: 7, Class: "Fighter", AC: 1},
	{Name: "Cloak", Slot: SlotBack, MeshIdx: 8, Class: "Fighter"},
	// Mage
	{Name: "Wand", Slot: SlotMainHand, MeshIdx: 2, Class: "Mage", Hit: 1},
	{Name: "Staff", Slot: SlotMainHand, MeshIdx: 3, Class: "Mage", TwoHanded: true, Hit: 1, Damage: 1},
	{Name: "Spellbook", Slot: SlotOffHand, MeshIdx: 0, Class: "Mage", Damage: 1},
	{Name: "Tome of Power", Slot: SlotOffHand, MeshIdx: 1, Class: "Mage", Hit: 1},
	{Name: "Wizard Hat", Slot: SlotHead, MeshIdx: 4, Class: "Mage", AC: 1},
	{Name: "Arcane Cloak", Slot: SlotBack, MeshIdx: 5, Class: "Mage"},
}

// GearForClass returns all items available to a class.
func GearForClass(class string) []GearItem {
	var items []GearItem
	for _, g := range AllGear {
		if g.Class == class {
			items = append(items, g)
		}
	}
	return items
}

// GearAC returns total AC bonus from equipped gear.
func (e *Entity) GearAC() int {
	total := 0
	for _, item := range e.Equipment {
		total += item.AC
	}
	return total
}

// GearHit returns total attack roll bonus from equipped gear.
func (e *Entity) GearHit() int {
	total := 0
	for _, item := range e.Equipment {
		total += item.Hit
	}
	return total
}

// GearDamage returns total damage bonus from equipped gear.
func (e *Entity) GearDamage() int {
	total := 0
	for _, item := range e.Equipment {
		total += item.Damage
	}
	return total
}

// WeaponDie returns the main hand weapon die size, or 8 (longsword default).
func (e *Entity) WeaponDie() int {
	if item, ok := e.Equipment[SlotMainHand]; ok && item.DamageDie > 0 {
		return item.DamageDie
	}
	return 8
}

// EffectiveAC returns base AC + gear bonuses.
func (e *Entity) EffectiveAC() int {
	if e.Stats == nil {
		return 10
	}
	return e.Stats.AC + e.GearAC()
}

// Equip adds an item to the entity's equipment, handling 2H conflicts.
func (e *Entity) Equip(item *GearItem) {
	if e.Equipment == nil {
		e.Equipment = map[EquipSlot]*GearItem{}
	}
	// 2H weapon: also clear offhand
	if item.TwoHanded && item.Slot == SlotMainHand {
		delete(e.Equipment, SlotOffHand)
	}
	// Equipping offhand: clear 2H main hand
	if item.Slot == SlotOffHand {
		if mh, ok := e.Equipment[SlotMainHand]; ok && mh.TwoHanded {
			delete(e.Equipment, SlotMainHand)
		}
	}
	e.Equipment[item.Slot] = item
	e.RebuildVisibleMeshes()
}

// Unequip removes an item from the given slot.
func (e *Entity) Unequip(slot EquipSlot) {
	delete(e.Equipment, slot)
	e.RebuildVisibleMeshes()
}

// RebuildVisibleMeshes updates VisibleMeshes from current equipment.
func (e *Entity) RebuildVisibleMeshes() {
	if e.Stats == nil {
		return
	}
	switch e.Stats.Class {
	case "Fighter":
		v := make([]bool, 15)
		for i := 9; i <= 14; i++ {
			v[i] = true
		}
		for _, item := range e.Equipment {
			if item.MeshIdx < len(v) {
				v[item.MeshIdx] = true
			}
		}
		e.VisibleMeshes = v
	case "Mage":
		v := make([]bool, 12)
		for i := 6; i <= 11; i++ {
			v[i] = true
		}
		for _, item := range e.Equipment {
			if item.MeshIdx < len(v) {
				v[item.MeshIdx] = true
			}
		}
		e.VisibleMeshes = v
	}
}

type Alert struct {
	EntityIdx int
	ThreatX   int
	ThreatZ   int
	Message   string
}

// Event system

type EventTrigger int

const (
	TriggerRoomEntered EventTrigger = iota
	TriggerRoomCleared
)

type EventContext struct {
	RoomIdx   int
	EntityIdx int
}

type Event struct {
	ID      string
	Trigger EventTrigger
	Check   func(g *GameState, w *World, ctx EventContext) bool
	Fire    func(g *GameState, w *World, ctx EventContext)
	OneShot bool
	Fired   bool
}

type FloatingText struct {
	Text     string
	WorldX   int
	WorldZ   int
	Timer    float32 // counts down from max
	MaxTime  float32
	Color    [4]uint8 // RGBA
	FontSize int32
}

type GameState struct {
	PlayerX, PlayerZ int
	PrevX, PrevZ     int
	StepProgress     float32
	Moving           bool
	FacingAngle      float32
	FacingDX         int // grid direction player faces
	FacingDZ         int
	TimeTicks        int
	Path             [][2]int
	Camera           CameraMode
	HasPickaxe       bool
	Message          string
	MessageTimer     float32

	Entities    []*Entity
	SelectedEnt int // -1 = none
	TickAccum   float32
	Alert       *Alert
	Combat      *Combat // nil = continuous mode
	Events      []*Event
	Floats      []FloatingText
	MoveRange   map[[2]int]bool
	AttackRange map[[2]int]bool
	Day            int // increments on each long rest; starts at 1
	Debug          bool
	DebugGearPanel bool // toggle gear mesh editor
	GearCursor     int  // selected row in gear panel
	GameOver       bool // true when all entities are dead
	EquipUIOpen    bool // equipment overlay visible
	EquipCursor    int  // selected row in equipment item list
	ChestUsedBy    int  // entity index that last opened chest (-1 = none)
}

// PendingLevelUpEntity returns the index of the first entity with pending
// level-up choices, or -1 if none.
func (g *GameState) PendingLevelUpEntity() int {
	for i, ent := range g.Entities {
		if ent.Stats != nil && len(ent.Stats.PendingChoices) > 0 {
			return i
		}
	}
	return -1
}

func (g *GameState) AddFloat(text string, wx, wz int, r, gr, b uint8, size int32) {
	g.Floats = append(g.Floats, FloatingText{
		Text: text, WorldX: wx, WorldZ: wz,
		Timer: 1.5, MaxTime: 1.5,
		Color: [4]uint8{r, gr, b, 255}, FontSize: size,
	})
}

func (g *GameState) TickFloats(dt float32) {
	alive := g.Floats[:0]
	for i := range g.Floats {
		g.Floats[i].Timer -= dt
		if g.Floats[i].Timer > 0 {
			alive = append(alive, g.Floats[i])
		}
	}
	g.Floats = alive
}

func (g *GameState) ComputeMoveRange(w *World, ox, oz, maxSteps int) {
	g.MoveRange = map[[2]int]bool{}
	type node struct{ x, z, steps int }
	queue := []node{{ox, oz, 0}}
	visited := map[[2]int]bool{{ox, oz}: true}
	dirs := [8][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.steps > 0 {
			g.MoveRange[[2]int{cur.x, cur.z}] = true
		}
		if cur.steps >= maxSteps {
			continue
		}
		for _, d := range dirs {
			nx, nz := cur.x+d[0], cur.z+d[1]
			key := [2]int{nx, nz}
			if visited[key] || !w.IsWalkable(nx, nz) || w.IsThreatAt(nx, nz) {
				continue
			}
			// Prevent diagonal corner-cutting through walls
			if d[0] != 0 && d[1] != 0 {
				if !w.IsWalkable(cur.x+d[0], cur.z) || !w.IsWalkable(cur.x, cur.z+d[1]) {
					continue
				}
			}
			visited[key] = true
			queue = append(queue, node{nx, nz, cur.steps + 1})
		}
	}
}

func (g *GameState) ComputeAttackRange(w *World, ox, oz, r int) {
	g.AttackRange = map[[2]int]bool{}
	for dz := -r; dz <= r; dz++ {
		for dx := -r; dx <= r; dx++ {
			adx, adz := abs(dx), abs(dz)
			// Chebyshev distance for melee-range consistency with diagonal movement
			dist := adx
			if adz > dist {
				dist = adz
			}
			if dist <= r && dist > 0 {
				tx, tz := ox+dx, oz+dz
				if r > 1 && !w.HasLineOfSight(ox, oz, tx, tz) {
					continue
				}
				g.AttackRange[[2]int{tx, tz}] = true
			}
		}
	}
}

func (g *GameState) ClearHighlights() {
	g.MoveRange = nil
	g.AttackRange = nil
}

func (g *GameState) SetMessage(msg string) {
	g.Message = msg
	g.MessageTimer = 3.0
}

func (g *GameState) FireEvents(trigger EventTrigger, w *World, ctx EventContext) {
	for _, ev := range g.Events {
		if ev.Trigger != trigger || (ev.OneShot && ev.Fired) {
			continue
		}
		if ev.Check(g, w, ctx) {
			ev.Fire(g, w, ctx)
			ev.Fired = true
		}
	}
}

// AnyEntityBusy returns true if any entity has an active task
func (g *GameState) AnyEntityBusy() bool {
	for _, ent := range g.Entities {
		if ent.Task != nil {
			return true
		}
	}
	return false
}

const stepInterval = 0.25

const ExhaustionThresholdTicks = 960 // 16 hours without long rest → first exhaustion gain
const ExhaustionIntervalTicks = 240  // every 4 hours after threshold, another gain

// EffectiveMaxHP returns MaxHP halved at exhaustion 4+.
func (e *Entity) EffectiveMaxHP() int {
	if e.Stats == nil {
		return 1
	}
	if e.Stats.Exhaustion >= 4 {
		return e.Stats.MaxHP / 2
	}
	return e.Stats.MaxHP
}

// EffectiveMoveSpeed returns MoveSpeed, halved at exhaustion 2+, zero at 5+.
func (e *Entity) EffectiveMoveSpeed() int {
	if e.Stats == nil {
		return 0
	}
	if e.Stats.Exhaustion >= 5 {
		return 0
	}
	if e.Stats.Exhaustion >= 2 {
		return e.Stats.MoveSpeed / 2
	}
	return e.Stats.MoveSpeed
}

// CheckExhaustion gains exhaustion from time awake without long rest.
func (e *Entity) CheckExhaustion(g *GameState, w *World) {
	if e.Stats == nil || e.Stats.Exhaustion >= 6 {
		return
	}
	awake := g.TimeTicks - e.LastLongRestTick
	if awake < ExhaustionThresholdTicks {
		return
	}
	expectedLevel := 1 + (awake-ExhaustionThresholdTicks)/ExhaustionIntervalTicks
	if e.Stats.Exhaustion < expectedLevel {
		e.Stats.Exhaustion++
		g.SetMessage(fmt.Sprintf("%s gains exhaustion! (%d/6)", e.Name, e.Stats.Exhaustion))
		g.AddFloat(fmt.Sprintf("Exhaustion %d", e.Stats.Exhaustion), e.X, e.Z, 255, 160, 40, 20)
		if e.Stats.Exhaustion >= 6 {
			for i, ent := range g.Entities {
				if ent == e {
					g.KillEntity(w, i)
					return
				}
			}
		}
	}
}

func FacingAngleFromDir(dx, dz int) float32 {
	if dx == 0 && dz == 0 {
		return 0
	}
	// Model faces +Z at angle 0. Atan2 gives angle from +Z axis, clockwise positive.
	return float32(math.Atan2(float64(dx), float64(dz))) * 180 / math.Pi
}

// WorldStep advances the global clock by one tick and lets all entities act.
// Called once per player action (move, break wall, wait). Time only moves when someone acts.
func (g *GameState) WorldStep(w *World) *Alert {
	g.TimeTicks++
	// Exhaustion from time awake
	for _, ent := range g.Entities {
		ent.CheckExhaustion(g, w)
	}
	if g.GameOver {
		return nil
	}
	// HP clamping from exhaustion
	for _, ent := range g.Entities {
		if ent.Stats != nil && ent.Stats.HP > ent.EffectiveMaxHP() {
			ent.Stats.HP = ent.EffectiveMaxHP()
		}
	}
	w.TickLeashingThreats()
	return g.TickEntities(w)
}

// TickEntities advances all entities one step. Returns an alert if triggered.
func (g *GameState) TickEntities(w *World) *Alert {
	for i, ent := range g.Entities {
		if ent.Task == nil {
			continue
		}

		// Exhaustion 5+: can't move
		if ent.EffectiveMoveSpeed() == 0 {
			ent.Task = nil
			g.SetMessage(fmt.Sprintf("%s is too exhausted to move!", ent.Name))
			continue
		}

		// Exhaustion 2+: half speed — skip every other tick
		if ent.Stats != nil && ent.Stats.Exhaustion >= 2 && g.TimeTicks%2 != 0 {
			continue
		}

		// Explore: patrol dungeon perimeter (or push inward with pickaxe)
		if ent.Task.Type == TaskExplore && (ent.Task.Path == nil || ent.Task.PathIdx >= len(ent.Task.Path)) {
			if ent.Task.ExploreAngle == 0 {
				ent.Task.ExploreAngle = math.Atan2(float64(ent.Z), float64(ent.X))
			}

			if g.HasPickaxe {
				// With pickaxe: push inward at current angle, then advance
				midR := (w.OuterEdge(ent.Task.ExploreAngle) + w.InnerEdge(ent.Task.ExploreAngle)) / 2
				tx := int(math.Cos(ent.Task.ExploreAngle) * midR)
				tz := int(math.Sin(ent.Task.ExploreAngle) * midR)
				path := FindPathBreakable(w, ent.X, ent.Z, tx, tz)
				if path != nil {
					ent.Task.Path = path
					ent.Task.PathIdx = 0
				}
				// Only advance angle after committing to a breach
				ent.Task.ExploreAngle += 0.35
			} else {
				// No pickaxe: patrol outside the wall
				ent.Task.ExploreAngle += 0.35
				outerR := w.OuterEdge(ent.Task.ExploreAngle)
				patrolR := outerR + float64(ent.RevealDist)/2
				tx := int(math.Cos(ent.Task.ExploreAngle) * patrolR)
				tz := int(math.Sin(ent.Task.ExploreAngle) * patrolR)
				fx, fz, found := nearestWalkable(w, tx, tz)
				if found {
					path := FindPath(w, ent.X, ent.Z, fx, fz)
					if path != nil {
						ent.Task.Path = path
						ent.Task.PathIdx = 0
					}
				}
			}
		}

		// Follow: repath toward leader when not adjacent
		if ent.Task.Type == TaskFollow && (ent.Task.Path == nil || ent.Task.PathIdx >= len(ent.Task.Path)) {
			if ent.Task.FollowIdx >= 0 && ent.Task.FollowIdx < len(g.Entities) {
				leader := g.Entities[ent.Task.FollowIdx]
				if !adjacent(ent.X, ent.Z, leader.X, leader.Z) {
					path := FindPath(w, ent.X, ent.Z, leader.X, leader.Z)
					if path != nil && len(path) > 1 {
						// Stop one tile short — don't stand on leader
						ent.Task.Path = path[:len(path)-1]
						ent.Task.PathIdx = 0
					}
				}
			}
		}

		if ent.Task.Type != TaskMoveTo && ent.Task.Type != TaskExplore && ent.Task.Type != TaskFollow {
			continue
		}
		if ent.Task.Path == nil || ent.Task.PathIdx >= len(ent.Task.Path) {
			continue
		}

		next := ent.Task.Path[ent.Task.PathIdx]
		nx, nz := next[0], next[1]

		if !w.IsWalkable(nx, nz) {
			// Break wall if we have the pickaxe
			if g.HasPickaxe {
				if t, ok := w.GetTile(nx, nz); ok && t == TileSolid {
					w.SetTile(nx, nz, TileDoorway)
					w.RevealAround(nx, nz)
					// Wall broken — check if we just revealed threats
					if ent.Stats != nil {
						for _, threat := range w.Threats {
							if withinRange(ent.X, ent.Z, threat.X, threat.Z, 4) &&
								w.HasLineOfSight(ent.X, ent.Z, threat.X, threat.Z) {
								g.StartCombat(w, i, threat.RoomIdx)
								return nil
							}
						}
					}
					continue // spend this tick breaking, move next tick
				}
			}
			ent.Task.Path = nil // force re-plan on next tick
			continue
		}

		dx, dz := nx-ent.X, nz-ent.Z
		ent.FacingAngle = FacingAngleFromDir(dx, dz)
		ent.PrevX, ent.PrevZ = ent.X, ent.Z
		ent.X, ent.Z = nx, nz
		ent.StepProgress = 0
		ent.Moving = true
		ent.Task.PathIdx++

		cx, cz := TileToChunk(ent.X, ent.Z)
		w.EnsureChunksAround(cx, cz)
		w.UnloadFarChunks(cx, cz)
		w.RevealAroundDist(ent.X, ent.Z, ent.RevealDist)

		// Track scouted area
		if ent.Scouted != nil {
			ent.Scouted[[2]int{ent.X >> 3, ent.Z >> 3}] = true
		}

		// Pickaxe pickup
		if !g.HasPickaxe && ent.X == w.PickaxeX && ent.Z == w.PickaxeZ {
			g.HasPickaxe = true
			g.SetMessage(ent.Name + " picked up the pickaxe!")
		}

		// Exploring: check for pickaxe within perception range every step — divert if spotted
		if ent.Task != nil && ent.Task.Type == TaskExplore && !g.HasPickaxe &&
			withinRange(ent.X, ent.Z, w.PickaxeX, w.PickaxeZ, ent.RevealDist) {
			path := FindPath(w, ent.X, ent.Z, w.PickaxeX, w.PickaxeZ)
			if path != nil {
				ent.Task.Path = path
				ent.Task.PathIdx = 0
				g.SetMessage(ent.Name + " spotted something interesting!")
			}
		}

		// Check if entity can see any revealed threats (must be close + LOS)
		if ent.Stats != nil {
			for _, threat := range w.Threats {
				if withinRange(ent.X, ent.Z, threat.X, threat.Z, 4) &&
					w.HasLineOfSight(ent.X, ent.Z, threat.X, threat.Z) {
					g.StartCombat(w, i, threat.RoomIdx)
					return nil
				}
			}
		}

		// MoveTo: clear task when path is done. Explore: will re-plan next tick.
		if ent.Task != nil && ent.Task.Type == TaskMoveTo && ent.Task.PathIdx >= len(ent.Task.Path) {
			ent.Task = nil
		}
	}
	return nil
}

// TryRest initiates a rest for an entity. Inside the dungeon, there's a chance of ambush.
func (g *GameState) tryRestAmbush(w *World, entIdx int) bool {
	ent := g.Entities[entIdx]
	tile := w.TileTypeAt(ent.X, ent.Z)
	insideDungeon := tile == TileFloor || tile == TileDoorway

	if insideDungeon && rand.Intn(100) < 30 {
		count := 1 + rand.Intn(3)
		roomIdx := -1
		spawned := false
		for i := 0; i < count; i++ {
			for _, off := range [][2]int{{2, 0}, {-2, 0}, {0, 2}, {0, -2}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}} {
				tx, tz := ent.X+off[0], ent.Z+off[1]
				if w.IsWalkable(tx, tz) && !w.IsThreatAt(tx, tz) {
					key := [2]int{tx, tz}
					w.Threats[key] = Threat{
						X: tx, Z: tz, Type: SkeletonMinion, RoomIdx: roomIdx,
						HP: 8, MaxHP: 8, AC: 11, STR: 10, DEX: 12,
						MoveSpeed: 4, AttackDice: 4, MaxRange: 1,
					}
					spawned = true
					break
				}
			}
		}
		if spawned {
			g.SetMessage("Ambush! Skeletons attack during rest!")
			g.StartCombat(w, entIdx, roomIdx)
			return true
		}
	}
	return false
}

// TryRest is the legacy short rest entry point (used by keybind and existing tests).
func (g *GameState) TryRest(w *World, entIdx int) {
	g.TryShortRest(w, entIdx)
}

// TryShortRest: 1 hour. Spend a hit die to heal (roll + CON mod).
// Fighter recovers class charges. 30% ambush chance inside dungeon.
func (g *GameState) TryShortRest(w *World, entIdx int) {
	if g.tryRestAmbush(w, entIdx) {
		return
	}
	ent := g.Entities[entIdx]

	// Spend a hit die to heal
	maxHP := ent.EffectiveMaxHP()
	if ent.Stats != nil && ent.Stats.HP < maxHP && ent.Stats.HitDice > 0 {
		ent.Stats.HitDice--
		heal := rollDice(ent.Stats.HitDieSize) + ent.Stats.Mod(ent.Stats.CON)
		if heal < 1 {
			heal = 1
		}
		ent.Stats.HP += heal
		if ent.Stats.HP > maxHP {
			ent.Stats.HP = maxHP
		}
		g.SetMessage(fmt.Sprintf("%s rests — heals %d HP (%d/%d) [%d hit dice left]",
			ent.Name, heal, ent.Stats.HP, maxHP, ent.Stats.HitDice))
	} else if ent.Stats != nil && ent.Stats.HP < maxHP {
		g.SetMessage(fmt.Sprintf("%s rests — no hit dice left, no healing.", ent.Name))
	} else {
		g.SetMessage(fmt.Sprintf("%s rests — already at full health.", ent.Name))
	}

	// Short rest: Fighter recovers class charges
	if ent.Stats != nil && ent.Stats.Class == "Fighter" && ent.Stats.ClassCharges < ent.Stats.MaxClassCharges {
		ent.Stats.ClassCharges = ent.Stats.MaxClassCharges
	}

	g.TimeTicks += 60
}

// TryLongRest: 8 hours. Full HP, recover half hit dice (min 1),
// restore class charges. Higher ambush chance inside dungeon (future).
func (g *GameState) TryLongRest(w *World, entIdx int) {
	if g.tryRestAmbush(w, entIdx) {
		return
	}
	ent := g.Entities[entIdx]

	// Full HP recovery
	if ent.Stats != nil {
		ent.Stats.HP = ent.Stats.MaxHP

		// Recover half hit dice (min 1)
		recover := ent.Stats.MaxHitDice / 2
		if recover < 1 {
			recover = 1
		}
		ent.Stats.HitDice += recover
		if ent.Stats.HitDice > ent.Stats.MaxHitDice {
			ent.Stats.HitDice = ent.Stats.MaxHitDice
		}

		// Restore class charges
		ent.Stats.ClassCharges = ent.Stats.MaxClassCharges

		// Reduce exhaustion by 1
		if ent.Stats.Exhaustion > 0 {
			ent.Stats.Exhaustion--
		}

		g.SetMessage(fmt.Sprintf("%s finishes long rest — fully healed! (%d/%d) [%d hit dice]",
			ent.Name, ent.Stats.HP, ent.Stats.MaxHP, ent.Stats.HitDice))
	}

	g.TimeTicks += 480
	g.Day++
	ent.LastLongRestTick = g.TimeTicks // reset exhaustion timer from waking up
}

// KillEntity removes a dead entity from the roster and fixes all references.
// Returns true if the company is wiped (game over).
func (g *GameState) KillEntity(w *World, deadIdx int) bool {
	ent := g.Entities[deadIdx]
	g.AddFloat("FALLEN", ent.X, ent.Z, 255, 40, 40, 24)
	g.SetMessage(fmt.Sprintf("%s has fallen!", ent.Name))

	// Remove from combat initiative if active
	if g.Combat != nil {
		for i := len(g.Combat.Combatants) - 1; i >= 0; i-- {
			cb := &g.Combat.Combatants[i]
			if !cb.IsEnemy && cb.EntityIdx == deadIdx {
				g.Combat.Combatants = append(g.Combat.Combatants[:i], g.Combat.Combatants[i+1:]...)
				if g.Combat.TurnIndex >= len(g.Combat.Combatants) {
					g.Combat.TurnIndex = 0
				}
				break
			}
		}
		// Fix EntityIdx references in remaining combatants (indices shifted)
		for i := range g.Combat.Combatants {
			cb := &g.Combat.Combatants[i]
			if !cb.IsEnemy && cb.EntityIdx > deadIdx {
				cb.EntityIdx--
			}
		}
	}

	// Stop anyone following the dead entity
	for _, other := range g.Entities {
		if other.Task != nil && other.Task.Type == TaskFollow {
			if other.Task.FollowIdx == deadIdx {
				other.Task = nil
			} else if other.Task.FollowIdx > deadIdx {
				other.Task.FollowIdx--
			}
		}
	}

	// Remove from roster
	g.Entities = append(g.Entities[:deadIdx], g.Entities[deadIdx+1:]...)

	// Fix selected entity
	if g.SelectedEnt >= len(g.Entities) {
		g.SelectedEnt = len(g.Entities) - 1
	}
	if g.SelectedEnt < 0 {
		g.SelectedEnt = -1
	}

	// Check game over
	if len(g.Entities) == 0 {
		g.GameOver = true
		g.Combat = nil
		g.ClearHighlights()
		return true
	}

	// If combat has no allies left, end combat (enemies win but game continues if entities remain outside)
	if g.Combat != nil {
		anyAlly := false
		for _, cb := range g.Combat.Combatants {
			if !cb.IsEnemy {
				anyAlly = true
				break
			}
		}
		if !anyAlly {
			g.Combat = nil
			g.ClearHighlights()
		}
	}

	return false
}

func withinRange(ax, az, bx, bz, r int) bool {
	dx, dz := abs(ax-bx), abs(az-bz)
	if dx > dz {
		return dx <= r
	}
	return dz <= r
}

// adjacent returns true if two tiles are within 1 step (cardinal or diagonal)
func adjacent(ax, az, bx, bz int) bool {
	dx, dz := abs(ax-bx), abs(az-bz)
	return dx <= 1 && dz <= 1 && (dx+dz) > 0
}

// nearestClearTile spirals out from (tx,tz) to find the closest walkable tile
// with no threat and no entity.
func nearestClearTile(g *GameState, w *World, tx, tz int) (int, int, bool) {
	check := func(x, z int) bool {
		if !w.IsWalkable(x, z) {
			return false
		}
		if w.IsThreatAt(x, z) {
			return false
		}
		for _, ent := range g.Entities {
			if ent.X == x && ent.Z == z {
				return false
			}
		}
		return true
	}
	if check(tx, tz) {
		return tx, tz, true
	}
	for r := 1; r <= 8; r++ {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if abs(dx) != r && abs(dz) != r {
					continue
				}
				if check(tx+dx, tz+dz) {
					return tx + dx, tz + dz, true
				}
			}
		}
	}
	return 0, 0, false
}

// nearestWalkable spirals out from (tx,tz) to find the closest walkable tile.
func nearestWalkable(w *World, tx, tz int) (int, int, bool) {
	if w.IsWalkable(tx, tz) {
		return tx, tz, true
	}
	for r := 1; r <= 8; r++ {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if abs(dx) != r && abs(dz) != r {
					continue // only check the ring edge
				}
				if w.IsWalkable(tx+dx, tz+dz) {
					return tx + dx, tz + dz, true
				}
			}
		}
	}
	return 0, 0, false
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// A* pathfinding with cardinal movement
type astarNode struct {
	x, z, g, f int
	parent     *astarNode
}

type astarHeap []*astarNode

func (h astarHeap) Len() int            { return len(h) }
func (h astarHeap) Less(i, j int) bool   { return h[i].f < h[j].f }
func (h astarHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i] }
func (h *astarHeap) Push(x interface{})  { *h = append(*h, x.(*astarNode)) }
func (h *astarHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func findPath(w *World, sx, sz, gx, gz int, breakWalls bool) [][2]int {
	canPass := func(tx, tz int) bool {
		if w.IsWalkable(tx, tz) {
			return true
		}
		if breakWalls {
			t, ok := w.GetTile(tx, tz)
			return ok && t == TileSolid
		}
		return false
	}

	if !canPass(gx, gz) {
		return nil
	}

	type key struct{ x, z int }
	closed := map[key]bool{}
	open := &astarHeap{}
	heap.Init(open)

	heuristic := func(ax, az int) int {
		dx, dz := abs(gx-ax), abs(gz-az)
		if dx > dz {
			return dx
		}
		return dz
	}

	start := &astarNode{x: sx, z: sz, g: 0, f: heuristic(sx, sz)}
	heap.Push(open, start)

	dirs := [8][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}}

	for open.Len() > 0 {
		cur := heap.Pop(open).(*astarNode)
		if cur.x == gx && cur.z == gz {
			var path [][2]int
			for n := cur; n != nil && (n.x != sx || n.z != sz); n = n.parent {
				path = append([][2]int{{n.x, n.z}}, path...)
			}
			return path
		}

		k := key{cur.x, cur.z}
		if closed[k] {
			continue
		}
		closed[k] = true

		for _, dir := range dirs {
			nx, nz := cur.x+dir[0], cur.z+dir[1]
			if closed[key{nx, nz}] || !canPass(nx, nz) {
				continue
			}
			// Prevent diagonal corner-cutting through walls
			if dir[0] != 0 && dir[1] != 0 {
				if !w.IsWalkable(cur.x+dir[0], cur.z) || !w.IsWalkable(cur.x, cur.z+dir[1]) {
					continue
				}
			}
			ng := cur.g + 1
			// Wall tiles cost more so entities prefer existing paths
			if breakWalls {
				if t, ok := w.GetTile(nx, nz); ok && t == TileSolid {
					ng += 5
				}
			}
			nf := ng + heuristic(nx, nz)
			heap.Push(open, &astarNode{x: nx, z: nz, g: ng, f: nf, parent: cur})
		}
	}
	return nil
}

func FindPath(w *World, sx, sz, gx, gz int) [][2]int {
	return findPath(w, sx, sz, gx, gz, false)
}

func FindPathBreakable(w *World, sx, sz, gx, gz int) [][2]int {
	return findPath(w, sx, sz, gx, gz, true)
}
