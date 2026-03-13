package main

import (
	"fmt"
	"math/rand"
)

// Action economy types

type ActionCost int

const (
	CostFree     ActionCost = iota
	CostStandard            // one per turn
	CostBonus               // one per turn
	CostReaction            // one per round
)

type TargetMode int

const (
	TargetNone          TargetMode = iota // no target needed
	TargetSelf                            // self-cast
	TargetEnemyAdjacent                   // click adjacent enemy
	TargetEnemyRange                      // click enemy in range
)

type CombatAction struct {
	ID       string
	Name     string
	Cost     ActionCost
	Target   TargetMode
	Range    int
	Hotkey   string // display hint
	CanUse   func(g *GameState, ent *Entity) bool
	Execute  func(g *GameState, w *World, ent *Entity, tx, tz int)
}

// Combat state

type Combatant struct {
	IsEnemy      bool
	EntityIdx    int    // index into GameState.Entities (allies)
	ThreatKey    [2]int // key into World.Threats (enemies)
	Initiative   int
	SpawnPos     [2]int // original position, for leashing back
	PursuitLeft  int    // turns left before giving up chase (enemies only)
}

type Combat struct {
	Combatants    []Combatant
	TurnIndex     int
	MoveLeft      int
	ActionUsed    bool // standard action spent this turn
	BonusUsed     bool
	RoomIdx       int
	PrimedAction  *CombatAction // selected action awaiting target
	Actions       []*CombatAction // available actions for current entity
}

func (c *Combat) Current() *Combatant {
	return &c.Combatants[c.TurnIndex]
}

func (c *Combat) NextTurn(g *GameState, w *World) {
	c.TurnIndex = (c.TurnIndex + 1) % len(c.Combatants)
	c.ActionUsed = false
	c.BonusUsed = false
	c.PrimedAction = nil
	// Set move points for new combatant
	next := c.Current()
	if !next.IsEnemy {
		ent := g.Entities[next.EntityIdx]
		if ent.Stats != nil {
			c.MoveLeft = ent.EffectiveMoveSpeed()
		}
		c.Actions = BuildActions(g, ent)
		g.ComputeMoveRange(w, ent.X, ent.Z, c.MoveLeft)
	} else {
		if t, ok := w.Threats[next.ThreatKey]; ok {
			c.MoveLeft = t.MoveSpeed
		}
		c.Actions = nil
		g.ClearHighlights()
	}
}

// StartCombat initiates combat with an entity entering a room with threats.
func (g *GameState) StartCombat(w *World, entityIdx int, roomIdx int) {
	g.FireEvents(TriggerRoomEntered, w, EventContext{RoomIdx: roomIdx, EntityIdx: entityIdx})

	ent := g.Entities[entityIdx]
	ent.Task = nil

	var combatants []Combatant

	// Add all nearby combat-capable allies, not just the triggering entity
	for i, e := range g.Entities {
		if e.Stats == nil || e.Stats.HP <= 0 {
			continue
		}
		if i != entityIdx && !withinRange(ent.X, ent.Z, e.X, e.Z, 6) {
			continue
		}
		e.Task = nil
		dexMod := e.Stats.Mod(e.Stats.DEX)
		combatants = append(combatants, Combatant{
			IsEnemy:    false,
			EntityIdx:  i,
			Initiative: rollD20() + dexMod,
		})
	}

	for key, threat := range w.Threats {
		if threat.RoomIdx == roomIdx {
			combatants = append(combatants, Combatant{
				IsEnemy:     true,
				ThreatKey:   key,
				Initiative:  rollD20() + (threat.DEX-10)/2,
				SpawnPos:    key,
				PursuitLeft: 3,
			})
		}
	}

	// Sort by initiative descending
	for i := 0; i < len(combatants); i++ {
		for j := i + 1; j < len(combatants); j++ {
			if combatants[j].Initiative > combatants[i].Initiative {
				combatants[i], combatants[j] = combatants[j], combatants[i]
			}
		}
	}

	moveLeft := 0
	first := combatants[0]
	var actions []*CombatAction
	if !first.IsEnemy {
		ent := g.Entities[first.EntityIdx]
		if ent.Stats != nil {
			moveLeft = ent.EffectiveMoveSpeed()
		}
		actions = BuildActions(g, g.Entities[first.EntityIdx])
	} else {
		if t, ok := w.Threats[first.ThreatKey]; ok {
			moveLeft = t.MoveSpeed
		}
	}

	g.Combat = &Combat{
		Combatants: combatants,
		TurnIndex:  0,
		MoveLeft:   moveLeft,
		RoomIdx:    roomIdx,
		Actions:    actions,
	}
	g.SelectedEnt = entityIdx
	g.SetMessage("Combat started! Roll initiative!")

	if !first.IsEnemy {
		e := g.Entities[first.EntityIdx]
		g.ComputeMoveRange(w, e.X, e.Z, moveLeft)
	}
}

// mageActionDefs holds learnable spell/cantrip templates keyed by ID.
// Spells without an Execute function are not yet implemented and won't appear.
var mageActionDefs = map[string]CombatAction{
	// Cantrips (no spell slot cost)
	// Cantrips (no spell slot cost)
	"ray_of_frost": {
		ID: "ray_of_frost", Name: "Ray of Frost", Cost: CostStandard,
		Target: TargetEnemyRange, Range: 7,
		CanUse:  func(g *GameState, e *Entity) bool { return !g.Combat.ActionUsed },
		Execute: executeRayOfFrost,
	},
	"shocking_grasp": {
		ID: "shocking_grasp", Name: "Shocking Grasp", Cost: CostStandard,
		Target: TargetEnemyAdjacent, Range: 1,
		CanUse:  func(g *GameState, e *Entity) bool { return !g.Combat.ActionUsed },
		Execute: executeShockingGrasp,
	},
	// L1 Spells (cost a spell slot)
	"burning_hands": {
		ID: "burning_hands", Name: "Burning Hands", Cost: CostStandard,
		Target: TargetSelf, Range: 1,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeBurningHands,
	},
	"frost_nova": {
		ID: "frost_nova", Name: "Frost Nova", Cost: CostStandard,
		Target: TargetSelf, Range: 1,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeFrostNova,
	},
	"mind_spike": {
		ID: "mind_spike", Name: "Mind Spike", Cost: CostStandard,
		Target: TargetEnemyRange, Range: 6,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeMindSpike,
	},
	"arcane_blink": {
		ID: "arcane_blink", Name: "Arcane Blink", Cost: CostStandard,
		Target: TargetEnemyRange, Range: 5,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeArcaneBlink,
	},
	"magic_armor": {
		ID: "magic_armor", Name: "Magic Armor", Cost: CostStandard,
		Target: TargetSelf,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeMagicArmor,
	},
	"elemental_strike": {
		ID: "elemental_strike", Name: "Elemental Strike", Cost: CostStandard,
		Target: TargetEnemyRange, Range: 6,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeElementalStrike,
	},
	"feather_fall": {
		ID: "feather_fall", Name: "Feather Fall", Cost: CostStandard,
		Target: TargetSelf,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeFeatherFall,
	},
	"expeditious_retreat": {
		ID: "expeditious_retreat", Name: "Expeditious Retreat", Cost: CostStandard,
		Target: TargetSelf,
		CanUse: func(g *GameState, e *Entity) bool {
			return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
		},
		Execute: executeExpeditiousRetreat,
	},
}

// BuildActions returns the available combat actions for an entity based on class.
func BuildActions(g *GameState, ent *Entity) []*CombatAction {
	if ent.Stats == nil {
		return nil
	}

	var actions []*CombatAction

	switch ent.Stats.Class {
	case "Fighter":
		actions = append(actions,
			&CombatAction{
				ID: "melee_attack", Name: "Melee Attack", Cost: CostStandard,
				Target: TargetEnemyAdjacent, Range: 1, Hotkey: "1",
				CanUse: func(g *GameState, e *Entity) bool { return !g.Combat.ActionUsed },
				Execute: executeMeleeAttack,
			},
			&CombatAction{
				ID: "second_wind", Name: "Second Wind", Cost: CostStandard,
				Target: TargetSelf, Hotkey: "2",
				CanUse: func(g *GameState, e *Entity) bool {
					return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
				},
				Execute: executeSecondWind,
			},
			&CombatAction{
				ID: "dash", Name: "Dash", Cost: CostStandard,
				Target: TargetSelf, Hotkey: "3",
				CanUse: func(g *GameState, e *Entity) bool { return !g.Combat.ActionUsed },
				Execute: executeDash,
			},
			&CombatAction{
				ID: "shove", Name: "Shove", Cost: CostStandard,
				Target: TargetEnemyAdjacent, Range: 1, Hotkey: "4",
				CanUse: func(g *GameState, e *Entity) bool { return !g.Combat.ActionUsed },
				Execute: executeShove,
			},
			&CombatAction{
				ID: "action_surge", Name: "Action Surge", Cost: CostFree,
				Target: TargetSelf, Hotkey: "5",
				CanUse: func(g *GameState, e *Entity) bool {
					return g.Combat.ActionUsed && e.Stats.ClassCharges > 0
				},
				Execute: executeActionSurge,
			},
		)
		if ent.Stats.CombatTechnique == "Quick Strike" {
			actions = append(actions, &CombatAction{
				ID: "quick_strike", Name: "Quick Strike", Cost: CostBonus,
				Target: TargetEnemyAdjacent, Range: 1, Hotkey: "6",
				CanUse: func(g *GameState, e *Entity) bool { return !g.Combat.BonusUsed },
				Execute: executeQuickStrike,
			})
		}

	case "Mage":
		// Starting kit: Fire Bolt (cantrip) + Magic Missile (spell) always available
		actions = append(actions,
			&CombatAction{
				ID: "fire_bolt", Name: "Fire Bolt", Cost: CostStandard,
				Target: TargetEnemyRange, Range: 7, Hotkey: "1",
				CanUse: func(g *GameState, e *Entity) bool { return !g.Combat.ActionUsed },
				Execute: executeFireBolt,
			},
			&CombatAction{
				ID: "magic_missile", Name: "Magic Missile", Cost: CostStandard,
				Target: TargetEnemyRange, Range: 7, Hotkey: "2",
				CanUse: func(g *GameState, e *Entity) bool {
					return !g.Combat.ActionUsed && e.Stats.ClassCharges > 0
				},
				Execute: executeMagicMissile,
			},
		)
		// Learned cantrips (only if Execute is implemented)
		hotkey := 3
		for _, cID := range ent.Stats.KnownCantrips {
			if def, ok := mageActionDefs[cID]; ok && def.Execute != nil {
				a := def // copy
				a.Hotkey = fmt.Sprintf("%d", hotkey)
				hotkey++
				actions = append(actions, &a)
			}
		}
		// Learned spells (only if Execute is implemented)
		for _, sID := range ent.Stats.KnownSpells {
			if def, ok := mageActionDefs[sID]; ok && def.Execute != nil {
				a := def // copy
				a.Hotkey = fmt.Sprintf("%d", hotkey)
				hotkey++
				actions = append(actions, &a)
			}
		}
		actions = append(actions, &CombatAction{
			ID: "dash", Name: "Dash", Cost: CostStandard,
			Target: TargetSelf, Hotkey: fmt.Sprintf("%d", hotkey),
			CanUse: func(g *GameState, e *Entity) bool { return !g.Combat.ActionUsed },
			Execute: executeDash,
		})
	}

	// End turn is always available
	actions = append(actions, &CombatAction{
		ID: "end_turn", Name: "End Turn", Cost: CostFree,
		Target: TargetNone, Hotkey: "Space",
		CanUse:  func(g *GameState, e *Entity) bool { return true },
		Execute: func(g *GameState, w *World, e *Entity, tx, tz int) {
			g.Combat.NextTurn(g, w)
		},
	})

	return actions
}

// TryExecuteAction primes or executes an action.
// Self/None targets execute immediately. Enemy targets need a click.
func (g *GameState) TryExecuteAction(w *World, action *CombatAction) {
	c := g.Combat
	cur := c.Current()
	ent := g.Entities[cur.EntityIdx]

	if action.CanUse != nil && !action.CanUse(g, ent) {
		g.SetMessage("Can't use " + action.Name + " right now.")
		return
	}

	switch action.Target {
	case TargetNone, TargetSelf:
		action.Execute(g, w, ent, 0, 0)
	case TargetEnemyAdjacent, TargetEnemyRange:
		c.PrimedAction = action
		g.SetMessage("Select target for " + action.Name)
		g.ComputeAttackRange(w, ent.X, ent.Z, action.Range)
	}
}

// ExecutePrimedOnTarget resolves a primed action on a clicked target.
func (g *GameState) ExecutePrimedOnTarget(w *World, tx, tz int) {
	c := g.Combat
	action := c.PrimedAction
	if action == nil {
		return
	}
	cur := c.Current()
	ent := g.Entities[cur.EntityIdx]

	// Validate target
	if action.Target == TargetEnemyAdjacent && !adjacent(ent.X, ent.Z, tx, tz) {
		g.SetMessage("Target not adjacent!")
		return
	}
	if action.Target == TargetEnemyRange && !withinRange(ent.X, ent.Z, tx, tz, action.Range) {
		g.SetMessage("Target out of range!")
		return
	}
	if action.Target == TargetEnemyRange && !w.HasLineOfSight(ent.X, ent.Z, tx, tz) {
		g.SetMessage("No line of sight!")
		return
	}

	action.Execute(g, w, ent, tx, tz)
	c.PrimedAction = nil
}

// Execute functions — ported from old combat system

func executeMeleeAttack(g *GameState, w *World, ent *Entity, tx, tz int) {
	stats := ent.Stats
	threat, ok := w.GetThreat(tx, tz)
	if !ok || stats == nil {
		return
	}

	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)

	roll := rollD20Adv(false, stats.Exhaustion >= 3)
	atkMod := stats.Mod(stats.STR)
	hitBonus := ent.GearHit()
	if stats.CombatTechnique == "Power Attack" {
		hitBonus -= 2
	}
	total := roll + atkMod + stats.ProfBonus + hitBonus
	exhSuffix := ""
	if stats.Exhaustion >= 3 {
		exhSuffix = " (exhausted)"
	}

	if roll == 1 {
		g.SetMessage(fmt.Sprintf("%s attacks — nat 1! Miss!%s", ent.Name, exhSuffix))
		g.AddFloat("NAT 1!", tx, tz, 180, 180, 180, 20)
		g.Combat.ActionUsed = true
		return
	}

	if roll < 20 && total < threat.AC {
		g.SetMessage(fmt.Sprintf("%s attacks (%d+%d=%d vs AC %d) — miss!",
			ent.Name, roll, atkMod+stats.ProfBonus+hitBonus, total, threat.AC))
		g.AddFloat(fmt.Sprintf("MISS (%d)", total), tx, tz, 180, 180, 180, 18)
		g.Combat.ActionUsed = true
		return
	}

	// Hit — weapon die + STR mod + gear damage, crit doubles dice
	weaponDie := ent.WeaponDie()
	damageDice := rollDice(weaponDie)
	if roll == 20 {
		damageDice += rollDice(weaponDie)
	}
	damage := damageDice + atkMod + ent.GearDamage()
	if stats.CombatStyle == "Gladiator" {
		damage += 2
	}
	if stats.CombatTechnique == "Power Attack" {
		damage = damage * 3 / 2
	}

	threat.HP -= damage
	if threat.HP < 0 {
		threat.HP = 0
	}
	w.Threats[[2]int{tx, tz}] = threat

	if roll == 20 {
		g.SetMessage(fmt.Sprintf("%s CRITS! (%d+%d=%d) %d damage!",
			ent.Name, roll, atkMod+stats.ProfBonus, total, damage))
		g.AddFloat(fmt.Sprintf("CRIT! -%d", damage), tx, tz, 255, 220, 40, 24)
	} else {
		g.SetMessage(fmt.Sprintf("%s hits (%d+%d=%d vs AC %d) %d damage",
			ent.Name, roll, atkMod+stats.ProfBonus, total, threat.AC, damage))
		g.AddFloat(fmt.Sprintf("-%d", damage), tx, tz, 255, 255, 255, 20)
	}

	if threat.HP <= 0 {
		g.awardXP(w, threatXP(threat.Type))
		w.RemoveThreat(tx, tz)
		g.removeCombatant(w, tx, tz)
		g.SetMessage(fmt.Sprintf("%s slays the skeleton! (%d damage)", ent.Name, damage))
		g.AddFloat("SLAIN", tx, tz, 255, 60, 60, 22)
	}

	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeSecondWind(g *GameState, w *World, ent *Entity, _, _ int) {
	stats := ent.Stats
	heal := rollDice(10) + stats.Mod(stats.CON)
	if heal < 1 {
		heal = 1
	}
	stats.HP += heal
	maxHP := ent.EffectiveMaxHP()
	if stats.HP > maxHP {
		stats.HP = maxHP
	}
	stats.ClassCharges--
	g.Combat.ActionUsed = true
	g.SetMessage(fmt.Sprintf("%s uses Second Wind! Heals %d (HP: %d/%d)",
		ent.Name, heal, stats.HP, maxHP))
	g.AddFloat(fmt.Sprintf("+%d", heal), ent.X, ent.Z, 80, 255, 80, 22)
}

func executeDash(g *GameState, w *World, ent *Entity, _, _ int) {
	speed := ent.EffectiveMoveSpeed()
	g.Combat.MoveLeft += speed
	g.Combat.ActionUsed = true
	g.SetMessage(fmt.Sprintf("%s dashes! +%d movement", ent.Name, speed))
}

func executeShove(g *GameState, w *World, ent *Entity, tx, tz int) {
	threat, ok := w.GetThreat(tx, tz)
	if !ok {
		return
	}

	// Push direction: from entity toward threat
	dx, dz := tx-ent.X, tz-ent.Z
	// Normalize to unit step
	if dx > 0 { dx = 1 } else if dx < 0 { dx = -1 }
	if dz > 0 { dz = 1 } else if dz < 0 { dz = -1 }

	pushX, pushZ := tx+dx, tz+dz
	if w.IsWalkable(pushX, pushZ) && !w.IsThreatAt(pushX, pushZ) {
		// Move threat to pushed position
		delete(w.Threats, [2]int{tx, tz})
		threat.PrevX, threat.PrevZ = tx, tz
		threat.FacingAngle = FacingAngleFromDir(pushX-tx, pushZ-tz)
		threat.X, threat.Z = pushX, pushZ
		threat.StepProgress = 0
		threat.Moving = true
		newKey := [2]int{pushX, pushZ}
		w.Threats[newKey] = threat
		// Update combatant key
		for i := range g.Combat.Combatants {
			if g.Combat.Combatants[i].IsEnemy && g.Combat.Combatants[i].ThreatKey == [2]int{tx, tz} {
				g.Combat.Combatants[i].ThreatKey = newKey
				break
			}
		}
		g.SetMessage(fmt.Sprintf("%s shoves the skeleton back!", ent.Name))
	} else {
		g.SetMessage(fmt.Sprintf("%s shoves but the skeleton can't be pushed!", ent.Name))
	}

	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)
	g.Combat.ActionUsed = true
}

func executeQuickStrike(g *GameState, w *World, ent *Entity, tx, tz int) {
	stats := ent.Stats
	threat, ok := w.GetThreat(tx, tz)
	if !ok || stats == nil {
		return
	}

	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)

	roll := rollD20Adv(false, stats.Exhaustion >= 3)
	atkMod := stats.Mod(stats.STR)
	total := roll + atkMod + stats.ProfBonus + ent.GearHit()

	if roll == 1 || (roll < 20 && total < threat.AC) {
		exhStr := ""
		if stats.Exhaustion >= 3 {
			exhStr = " (exhausted)"
		}
		g.SetMessage(fmt.Sprintf("%s quick strike — miss!%s", ent.Name, exhStr))
		g.AddFloat("MISS", tx, tz, 180, 180, 180, 16)
		g.Combat.BonusUsed = true
		return
	}

	// Half damage: (weapon die + STR + gear) / 2, minimum 1
	weaponDie := ent.WeaponDie()
	damageDice := rollDice(weaponDie)
	if roll == 20 {
		damageDice += rollDice(weaponDie)
	}
	damage := (damageDice + atkMod + ent.GearDamage()) / 2
	if damage < 1 {
		damage = 1
	}

	threat.HP -= damage
	if threat.HP < 0 {
		threat.HP = 0
	}
	w.Threats[[2]int{tx, tz}] = threat

	g.SetMessage(fmt.Sprintf("%s quick strike! %d damage", ent.Name, damage))
	g.AddFloat(fmt.Sprintf("-%d", damage), tx, tz, 200, 200, 255, 16)

	if threat.HP <= 0 {
		g.awardXP(w, threatXP(threat.Type))
		w.RemoveThreat(tx, tz)
		g.removeCombatant(w, tx, tz)
		g.AddFloat("SLAIN", tx, tz, 255, 60, 60, 22)
	}

	if g.Combat != nil {
		g.Combat.BonusUsed = true
	}
}

func executeFireBolt(g *GameState, w *World, ent *Entity, tx, tz int) {
	stats := ent.Stats
	threat, ok := w.GetThreat(tx, tz)
	if !ok || stats == nil {
		return
	}

	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)

	roll := rollD20Adv(false, stats.Exhaustion >= 3)
	atkMod := stats.Mod(stats.INT)
	total := roll + atkMod + stats.ProfBonus + ent.GearHit()
	exhSuffix := ""
	if stats.Exhaustion >= 3 {
		exhSuffix = " (exhausted)"
	}

	if roll == 1 {
		g.SetMessage(fmt.Sprintf("%s casts Fire Bolt — nat 1! Miss!%s", ent.Name, exhSuffix))
		g.AddFloat("NAT 1!", tx, tz, 180, 180, 180, 20)
		g.Combat.ActionUsed = true
		return
	}

	if roll < 20 && total < threat.AC {
		g.SetMessage(fmt.Sprintf("%s casts Fire Bolt (%d+%d=%d vs AC %d) — miss!",
			ent.Name, roll, atkMod+stats.ProfBonus+ent.GearHit(), total, threat.AC))
		g.AddFloat(fmt.Sprintf("MISS (%d)", total), tx, tz, 180, 180, 180, 18)
		g.Combat.ActionUsed = true
		return
	}

	damageDice := rollDice(10)
	if roll == 20 {
		damageDice += rollDice(10)
	}
	damage := damageDice + atkMod + ent.GearDamage()

	threat.HP -= damage
	if threat.HP < 0 {
		threat.HP = 0
	}
	w.Threats[[2]int{tx, tz}] = threat

	if roll == 20 {
		g.SetMessage(fmt.Sprintf("%s Fire Bolt CRITS! %d damage!", ent.Name, damage))
		g.AddFloat(fmt.Sprintf("CRIT! -%d", damage), tx, tz, 255, 140, 20, 24)
	} else {
		g.SetMessage(fmt.Sprintf("%s Fire Bolt hits (%d+%d=%d vs AC %d) %d damage",
			ent.Name, roll, atkMod+stats.ProfBonus, total, threat.AC, damage))
		g.AddFloat(fmt.Sprintf("-%d", damage), tx, tz, 255, 140, 20, 20)
	}

	if threat.HP <= 0 {
		g.awardXP(w, threatXP(threat.Type))
		w.RemoveThreat(tx, tz)
		g.removeCombatant(w, tx, tz)
		g.SetMessage(fmt.Sprintf("%s incinerates the skeleton! (%d damage)", ent.Name, damage))
		g.AddFloat("SLAIN", tx, tz, 255, 60, 60, 22)
	}

	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeMagicMissile(g *GameState, w *World, ent *Entity, tx, tz int) {
	stats := ent.Stats
	threat, ok := w.GetThreat(tx, tz)
	if !ok || stats == nil {
		return
	}

	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)
	stats.ClassCharges--

	// 3 bolts, each 1d4+1, auto-hit
	damage := 0
	for range 3 {
		damage += rollDice(4) + 1
	}

	threat.HP -= damage
	if threat.HP < 0 {
		threat.HP = 0
	}
	w.Threats[[2]int{tx, tz}] = threat

	g.SetMessage(fmt.Sprintf("%s casts Magic Missile! 3 bolts for %d damage!", ent.Name, damage))
	g.AddFloat(fmt.Sprintf("-%d", damage), tx, tz, 150, 120, 255, 22)

	if threat.HP <= 0 {
		g.awardXP(w, threatXP(threat.Type))
		w.RemoveThreat(tx, tz)
		g.removeCombatant(w, tx, tz)
		g.SetMessage(fmt.Sprintf("%s obliterates the skeleton! (%d damage)", ent.Name, damage))
		g.AddFloat("SLAIN", tx, tz, 255, 60, 60, 22)
	}

	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

// --- Mage learnable spells ---

// spellAttackRoll: d20 + INT mod + proficiency, returns (total, roll, isCrit)
func spellAttackRoll(stats *CombatStats) (int, int, bool) {
	roll := rollD20Adv(false, stats.Exhaustion >= 3)
	mod := stats.Mod(stats.INT)
	total := roll + mod + stats.ProfBonus
	return total, roll, roll == 20
}

// spellHitOrMiss handles the common attack roll pattern. Returns true if hit.
func spellHitOrMiss(g *GameState, ent *Entity, threat Threat, tx, tz int, spellName string) (bool, int, int) {
	total, roll, _ := spellAttackRoll(ent.Stats)
	if roll == 1 {
		g.SetMessage(fmt.Sprintf("%s casts %s — nat 1! Miss!", ent.Name, spellName))
		g.AddFloat("NAT 1!", tx, tz, 180, 180, 180, 20)
		return false, roll, total
	}
	if roll < 20 && total < threat.AC {
		g.SetMessage(fmt.Sprintf("%s casts %s — miss! (%d vs AC %d)", ent.Name, spellName, total, threat.AC))
		g.AddFloat(fmt.Sprintf("MISS (%d)", total), tx, tz, 180, 180, 180, 18)
		return false, roll, total
	}
	return true, roll, total
}

// applySpellDamage deals damage, handles crit, slain, XP. Returns damage dealt.
func applySpellDamage(g *GameState, w *World, ent *Entity, tx, tz int, dieSize, numDice, roll int, spellName string, r, gr, b uint8) int {
	damage := 0
	for range numDice {
		damage += rollDice(dieSize)
	}
	if roll == 20 { // crit: double dice
		for range numDice {
			damage += rollDice(dieSize)
		}
	}
	damage += ent.Stats.Mod(ent.Stats.INT)
	if damage < 1 {
		damage = 1
	}

	threat, ok := w.GetThreat(tx, tz)
	if !ok {
		return 0
	}
	threat.HP -= damage
	if threat.HP < 0 {
		threat.HP = 0
	}
	w.Threats[[2]int{tx, tz}] = threat

	if roll == 20 {
		g.SetMessage(fmt.Sprintf("%s %s CRITS! %d damage!", ent.Name, spellName, damage))
		g.AddFloat(fmt.Sprintf("CRIT! -%d", damage), tx, tz, r, gr, b, 24)
	} else {
		g.SetMessage(fmt.Sprintf("%s %s hits! %d damage", ent.Name, spellName, damage))
		g.AddFloat(fmt.Sprintf("-%d", damage), tx, tz, r, gr, b, 20)
	}

	if threat.HP <= 0 {
		g.awardXP(w, threatXP(threat.Type))
		w.RemoveThreat(tx, tz)
		g.removeCombatant(w, tx, tz)
		g.AddFloat("SLAIN", tx, tz, 255, 60, 60, 22)
	}
	return damage
}

func executeRayOfFrost(g *GameState, w *World, ent *Entity, tx, tz int) {
	threat, ok := w.GetThreat(tx, tz)
	if !ok || ent.Stats == nil {
		return
	}
	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)

	hit, roll, _ := spellHitOrMiss(g, ent, threat, tx, tz, "Ray of Frost")
	if !hit {
		if g.Combat != nil {
			g.Combat.ActionUsed = true
		}
		return
	}
	// 1d8 + INT cold damage (cantrip, no slot cost)
	applySpellDamage(g, w, ent, tx, tz, 8, 1, roll, "Ray of Frost", 100, 180, 255)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeShockingGrasp(g *GameState, w *World, ent *Entity, tx, tz int) {
	threat, ok := w.GetThreat(tx, tz)
	if !ok || ent.Stats == nil {
		return
	}
	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)

	hit, roll, _ := spellHitOrMiss(g, ent, threat, tx, tz, "Shocking Grasp")
	if !hit {
		if g.Combat != nil {
			g.Combat.ActionUsed = true
		}
		return
	}
	// 1d8 + INT lightning damage (cantrip, melee)
	applySpellDamage(g, w, ent, tx, tz, 8, 1, roll, "Shocking Grasp", 255, 255, 80)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeBurningHands(g *GameState, w *World, ent *Entity, _, _ int) {
	if ent.Stats == nil {
		return
	}
	ent.Stats.ClassCharges--
	// 3d6 fire damage to all adjacent enemies
	damage := 0
	for range 3 {
		damage += rollDice(6)
	}
	if damage < 1 {
		damage = 1
	}

	hits := 0
	if g.Combat != nil {
		for _, cb := range g.Combat.Combatants {
			if !cb.IsEnemy {
				continue
			}
			threat, ok := w.GetThreat(cb.ThreatKey[0], cb.ThreatKey[1])
			if !ok {
				continue
			}
			if adjacent(ent.X, ent.Z, threat.X, threat.Z) {
				threat.HP -= damage
				if threat.HP < 0 {
					threat.HP = 0
				}
				w.Threats[cb.ThreatKey] = threat
				g.AddFloat(fmt.Sprintf("-%d", damage), threat.X, threat.Z, 255, 100, 20, 20)
				hits++
				if threat.HP <= 0 {
					g.awardXP(w, threatXP(threat.Type))
					w.RemoveThreat(threat.X, threat.Z)
					g.removeCombatant(w, threat.X, threat.Z)
					g.AddFloat("SLAIN", cb.ThreatKey[0], cb.ThreatKey[1], 255, 60, 60, 22)
				}
			}
		}
	}

	g.SetMessage(fmt.Sprintf("%s casts Burning Hands! %d fire damage, %d hit!", ent.Name, damage, hits))
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeFrostNova(g *GameState, w *World, ent *Entity, _, _ int) {
	if ent.Stats == nil {
		return
	}
	ent.Stats.ClassCharges--
	// 2d6 cold damage to all adjacent enemies
	damage := rollDice(6) + rollDice(6)
	if damage < 1 {
		damage = 1
	}

	hits := 0
	if g.Combat != nil {
		for _, cb := range g.Combat.Combatants {
			if !cb.IsEnemy {
				continue
			}
			threat, ok := w.GetThreat(cb.ThreatKey[0], cb.ThreatKey[1])
			if !ok {
				continue
			}
			if adjacent(ent.X, ent.Z, threat.X, threat.Z) {
				threat.HP -= damage
				if threat.HP < 0 {
					threat.HP = 0
				}
				w.Threats[cb.ThreatKey] = threat
				g.AddFloat(fmt.Sprintf("-%d", damage), threat.X, threat.Z, 100, 180, 255, 20)
				hits++
				if threat.HP <= 0 {
					g.awardXP(w, threatXP(threat.Type))
					w.RemoveThreat(threat.X, threat.Z)
					g.removeCombatant(w, threat.X, threat.Z)
					g.AddFloat("SLAIN", cb.ThreatKey[0], cb.ThreatKey[1], 255, 60, 60, 22)
				}
			}
		}
	}

	g.SetMessage(fmt.Sprintf("%s casts Frost Nova! %d cold damage, %d hit!", ent.Name, damage, hits))
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeMindSpike(g *GameState, w *World, ent *Entity, tx, tz int) {
	threat, ok := w.GetThreat(tx, tz)
	if !ok || ent.Stats == nil {
		return
	}
	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)
	ent.Stats.ClassCharges--

	hit, roll, _ := spellHitOrMiss(g, ent, threat, tx, tz, "Mind Spike")
	if !hit {
		if g.Combat != nil {
			g.Combat.ActionUsed = true
		}
		return
	}
	// 2d8 + INT psychic damage
	applySpellDamage(g, w, ent, tx, tz, 8, 2, roll, "Mind Spike", 200, 100, 255)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeArcaneBlink(g *GameState, w *World, ent *Entity, tx, tz int) {
	if ent.Stats == nil {
		return
	}
	if !w.IsWalkable(tx, tz) || w.IsThreatAt(tx, tz) {
		g.SetMessage("Can't blink there!")
		return
	}
	// Check any entity is already there
	for _, other := range g.Entities {
		if other.X == tx && other.Z == tz {
			g.SetMessage("Can't blink there — occupied!")
			return
		}
	}

	ent.Stats.ClassCharges--
	ent.X, ent.Z = tx, tz
	if g.Combat != nil {
		g.Combat.MoveLeft = 0
	}
	g.SetMessage(fmt.Sprintf("%s blinks to (%d, %d)!", ent.Name, tx, tz))
	g.AddFloat("BLINK", tx, tz, 120, 80, 255, 20)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
		g.ComputeMoveRange(w, ent.X, ent.Z, 0)
	}
}

func executeMagicArmor(g *GameState, w *World, ent *Entity, _, _ int) {
	if ent.Stats == nil {
		return
	}
	ent.Stats.ClassCharges--
	ent.Stats.AC += 2
	g.SetMessage(fmt.Sprintf("%s casts Magic Armor! +2 AC", ent.Name))
	g.AddFloat("+2 AC", ent.X, ent.Z, 120, 80, 255, 20)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeElementalStrike(g *GameState, w *World, ent *Entity, tx, tz int) {
	threat, ok := w.GetThreat(tx, tz)
	if !ok || ent.Stats == nil {
		return
	}
	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)
	ent.Stats.ClassCharges--

	hit, roll, _ := spellHitOrMiss(g, ent, threat, tx, tz, "Elemental Strike")
	if !hit {
		if g.Combat != nil {
			g.Combat.ActionUsed = true
		}
		return
	}
	// 3d8 + INT elemental damage
	applySpellDamage(g, w, ent, tx, tz, 8, 3, roll, "Elemental Strike", 255, 160, 40)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeFeatherFall(g *GameState, w *World, ent *Entity, _, _ int) {
	if ent.Stats == nil {
		return
	}
	ent.Stats.ClassCharges--
	ent.Stats.AC += 1
	g.SetMessage(fmt.Sprintf("%s casts Feather Fall! +1 AC (evasive)", ent.Name))
	g.AddFloat("Evasive!", ent.X, ent.Z, 220, 220, 255, 20)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
	}
}

func executeExpeditiousRetreat(g *GameState, w *World, ent *Entity, _, _ int) {
	if ent.Stats == nil {
		return
	}
	ent.Stats.ClassCharges--
	if g.Combat != nil {
		g.Combat.MoveLeft += 3
	}
	g.SetMessage(fmt.Sprintf("%s casts Expeditious Retreat! +3 movement", ent.Name))
	g.AddFloat("+3 Move", ent.X, ent.Z, 255, 255, 80, 20)
	if g.Combat != nil {
		g.Combat.ActionUsed = true
		g.ComputeMoveRange(w, ent.X, ent.Z, g.Combat.MoveLeft)
	}
}

func executeActionSurge(g *GameState, w *World, ent *Entity, _, _ int) {
	g.Combat.ActionUsed = false // regain standard action
	ent.Stats.ClassCharges--
	g.SetMessage(fmt.Sprintf("%s surges! Action restored!", ent.Name))
}

// removeCombatant removes a dead enemy from the initiative order.
func (g *GameState) removeCombatant(w *World, tx, tz int) {
	c := g.Combat
	key := [2]int{tx, tz}
	for i := len(c.Combatants) - 1; i >= 0; i-- {
		if c.Combatants[i].IsEnemy && c.Combatants[i].ThreatKey == key {
			c.Combatants = append(c.Combatants[:i], c.Combatants[i+1:]...)
			if c.TurnIndex >= len(c.Combatants) {
				c.TurnIndex = 0
			}
			break
		}
	}

	anyEnemy := false
	for _, cb := range c.Combatants {
		if cb.IsEnemy {
			anyEnemy = true
			break
		}
	}
	if !anyEnemy {
		roomIdx := c.RoomIdx
		g.Combat = nil
		g.ClearHighlights()
		g.SetMessage("Combat ended — victory!")
		g.FireEvents(TriggerRoomCleared, w, EventContext{RoomIdx: roomIdx})
	}
}

// RunEnemyTurn executes a simple enemy AI: move toward nearest ally, attack if adjacent.
// Enemies have a pursuit timer — if they can't attack for 3 turns, they leash back to spawn.
func (g *GameState) RunEnemyTurn(w *World) {
	c := g.Combat
	if c == nil {
		return
	}
	cur := c.Current()
	if !cur.IsEnemy {
		return
	}
	threat, ok := w.Threats[cur.ThreatKey]
	if !ok {
		// Threat gone (killed or key collision) — remove ghost combatant
		g.leashCombatant(w)
		return
	}

	// Leashing: pursuit expired — drop from combat, walk home outside initiative
	if cur.PursuitLeft <= 0 {
		w.LeashingThreats = append(w.LeashingThreats, LeashingThreat{
			ThreatKey: cur.ThreatKey,
			SpawnPos:  cur.SpawnPos,
		})
		g.leashCombatant(w)
		return
	}

	// Find nearest alive ally in combat
	bestDist := 9999
	bestIdx := -1
	for _, cb := range c.Combatants {
		if cb.IsEnemy {
			continue
		}
		if cb.EntityIdx >= len(g.Entities) {
			continue
		}
		ent := g.Entities[cb.EntityIdx]
		if ent.Stats == nil || ent.Stats.HP <= 0 {
			continue
		}
		d := abs(threat.X-ent.X) + abs(threat.Z-ent.Z)
		if d < bestDist {
			bestDist = d
			bestIdx = cb.EntityIdx
		}
	}
	if bestIdx < 0 {
		c.NextTurn(g, w)
		return
	}
	target := g.Entities[bestIdx]

	attacked := false

	inRange := threat.MaxRange <= 1 && adjacent(threat.X, threat.Z, target.X, target.Z) ||
		threat.MaxRange > 1 && withinRange(threat.X, threat.Z, target.X, target.Z, threat.MaxRange) &&
			w.HasLineOfSight(threat.X, threat.Z, target.X, target.Z)
	if inRange {
		threat.FacingAngle = FacingAngleFromDir(target.X-threat.X, target.Z-threat.Z)
		w.Threats[cur.ThreatKey] = threat
		g.ResolveEnemyAttack(w, threat, target)
		if g.GameOver || g.Combat == nil {
			return
		}
		cur.PursuitLeft = 3
		c.NextTurn(g, w)
		return
	}

	// Move toward target
	if c.MoveLeft > 0 {
		path := FindPath(w, threat.X, threat.Z, target.X, target.Z)
		if path != nil {
			steps := c.MoveLeft
			if steps > len(path) {
				steps = len(path)
			}
			for steps > 0 {
				dest := path[steps-1]
				if dest[0] == target.X && dest[1] == target.Z {
					steps--
					continue
				}
				break
			}
			if steps > 0 {
				dest := path[steps-1]
				threat.PrevX, threat.PrevZ = threat.X, threat.Z
				threat.FacingAngle = FacingAngleFromDir(dest[0]-threat.X, dest[1]-threat.Z)
				threat.X, threat.Z = dest[0], dest[1]
				threat.StepProgress = 0
				threat.Moving = true
				newKey := [2]int{dest[0], dest[1]}
				delete(w.Threats, cur.ThreatKey)
				w.Threats[newKey] = threat
				cur.ThreatKey = newKey
				c.MoveLeft -= steps
			}
		}
	}

	// Try attack after moving
	threat, ok = w.Threats[cur.ThreatKey]
	if ok {
		inRange = threat.MaxRange <= 1 && adjacent(threat.X, threat.Z, target.X, target.Z) ||
			threat.MaxRange > 1 && withinRange(threat.X, threat.Z, target.X, target.Z, threat.MaxRange) &&
				w.HasLineOfSight(threat.X, threat.Z, target.X, target.Z)
		if inRange {
			threat.FacingAngle = FacingAngleFromDir(target.X-threat.X, target.Z-threat.Z)
			w.Threats[cur.ThreatKey] = threat
			g.ResolveEnemyAttack(w, threat, target)
			attacked = true
		}
	}

	if g.GameOver || g.Combat == nil {
		return
	}

	if attacked {
		cur.PursuitLeft = 3
	} else if !w.HasLineOfSight(threat.X, threat.Z, target.X, target.Z) {
		cur.PursuitLeft -= 2 // lose interest fast without LOS
	} else {
		cur.PursuitLeft--
	}

	c.NextTurn(g, w)
}

// leashCombatant removes an enemy from initiative (they gave up chasing).
// Ends combat if no enemies remain in initiative.
func (g *GameState) leashCombatant(w *World) {
	c := g.Combat
	idx := c.TurnIndex
	c.Combatants = append(c.Combatants[:idx], c.Combatants[idx+1:]...)
	if len(c.Combatants) == 0 {
		g.Combat = nil
		g.ClearHighlights()
		g.SetMessage("Enemies lost interest.")
		return
	}
	if c.TurnIndex >= len(c.Combatants) {
		c.TurnIndex = 0
	}

	anyEnemy := false
	for _, cb := range c.Combatants {
		if cb.IsEnemy {
			anyEnemy = true
			break
		}
	}
	if !anyEnemy {
		g.Combat = nil
		g.ClearHighlights()
		g.SetMessage("Enemies lost interest.")
	} else {
		// Set up the next combatant's turn
		next := c.Current()
		if !next.IsEnemy {
			ent := g.Entities[next.EntityIdx]
			if ent.Stats != nil {
				c.MoveLeft = ent.EffectiveMoveSpeed()
			}
			c.Actions = BuildActions(g, g.Entities[next.EntityIdx])
		} else {
			if t, ok := w.Threats[next.ThreatKey]; ok {
				c.MoveLeft = t.MoveSpeed
			}
			c.Actions = nil
		}
		c.ActionUsed = false
		c.BonusUsed = false
		c.PrimedAction = nil
	}
}

// ResolveEnemyAttack handles an enemy threat attacking an ally entity.
func (g *GameState) ResolveEnemyAttack(w *World, threat Threat, target *Entity) {
	if target.Stats == nil {
		return
	}

	roll := rollD20()
	atkMod := threat.AttackMod()
	total := roll + atkMod

	if roll == 1 {
		g.SetMessage(fmt.Sprintf("Skeleton attacks %s — nat 1! Miss!", target.Name))
		g.AddFloat("NAT 1!", target.X, target.Z, 180, 180, 180, 20)
		return
	}

	ac := target.EffectiveAC()
	if roll < 20 && total < ac {
		g.SetMessage(fmt.Sprintf("Skeleton attacks %s (%d+%d=%d vs AC %d) — miss!",
			target.Name, roll, atkMod, total, ac))
		g.AddFloat(fmt.Sprintf("MISS (%d)", total), target.X, target.Z, 180, 180, 180, 18)
		return
	}

	damageDice := rollDice(threat.AttackDice)
	if roll == 20 {
		damageDice += rollDice(threat.AttackDice)
	}
	damage := damageDice + atkMod

	target.Stats.HP -= damage
	if target.Stats.HP < 0 {
		target.Stats.HP = 0
	}

	if roll == 20 {
		g.SetMessage(fmt.Sprintf("Skeleton CRITS %s! %d damage! (HP: %d/%d)",
			target.Name, damage, target.Stats.HP, target.Stats.MaxHP))
		g.AddFloat(fmt.Sprintf("CRIT! -%d", damage), target.X, target.Z, 255, 40, 40, 24)
	} else {
		g.SetMessage(fmt.Sprintf("Skeleton hits %s for %d damage (HP: %d/%d)",
			target.Name, damage, target.Stats.HP, target.Stats.MaxHP))
		g.AddFloat(fmt.Sprintf("-%d", damage), target.X, target.Z, 255, 80, 80, 20)
	}

	// Entity death
	if target.Stats.HP <= 0 {
		// Find entity index
		for i, ent := range g.Entities {
			if ent == target {
				g.KillEntity(w, i)
				break
			}
		}
	}
}

// rollD20Adv rolls a d20 with advantage/disadvantage.
// Both cancel out. Advantage: take higher. Disadvantage: take lower.
func rollD20Adv(advantage, disadvantage bool) int {
	if advantage == disadvantage {
		return rand.Intn(20) + 1
	}
	a, b := rand.Intn(20)+1, rand.Intn(20)+1
	if advantage {
		if a > b {
			return a
		}
		return b
	}
	if a < b {
		return a
	}
	return b
}

func rollD20() int { return rand.Intn(20) + 1 }
func rollDice(sides int) int {
	if sides <= 0 {
		return 0
	}
	return rand.Intn(sides) + 1
}
