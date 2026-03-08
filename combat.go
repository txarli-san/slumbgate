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
		if s := g.Entities[next.EntityIdx].Stats; s != nil {
			c.MoveLeft = s.MoveSpeed
		}
		c.Actions = BuildActions(g, g.Entities[next.EntityIdx])
	} else {
		if t, ok := w.Threats[next.ThreatKey]; ok {
			c.MoveLeft = t.MoveSpeed
		}
		c.Actions = nil
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
		if s := g.Entities[first.EntityIdx].Stats; s != nil {
			moveLeft = s.MoveSpeed
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

	roll := rollD20()
	atkMod := stats.Mod(stats.STR)
	total := roll + atkMod + stats.ProfBonus

	if roll == 1 {
		g.SetMessage(fmt.Sprintf("%s attacks — nat 1! Miss!", ent.Name))
		g.Combat.ActionUsed = true
		return
	}

	if roll < 20 && total < threat.AC {
		g.SetMessage(fmt.Sprintf("%s attacks (%d+%d=%d vs AC %d) — miss!",
			ent.Name, roll, atkMod+stats.ProfBonus, total, threat.AC))
		g.Combat.ActionUsed = true
		return
	}

	// Hit — 1d8 + STR mod, crit doubles dice
	damageDice := rollDice(8)
	if roll == 20 {
		damageDice += rollDice(8)
	}
	damage := damageDice + atkMod

	threat.HP -= damage
	w.Threats[[2]int{tx, tz}] = threat

	if roll == 20 {
		g.SetMessage(fmt.Sprintf("%s CRITS! (%d+%d=%d) %d damage!",
			ent.Name, roll, atkMod+stats.ProfBonus, total, damage))
	} else {
		g.SetMessage(fmt.Sprintf("%s hits (%d+%d=%d vs AC %d) %d damage",
			ent.Name, roll, atkMod+stats.ProfBonus, total, threat.AC, damage))
	}

	if threat.HP <= 0 {
		w.RemoveThreat(tx, tz)
		g.removeCombatant(w, tx, tz)
		g.SetMessage(fmt.Sprintf("%s slays the skeleton! (%d damage)", ent.Name, damage))
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
	if stats.HP > stats.MaxHP {
		stats.HP = stats.MaxHP
	}
	stats.ClassCharges--
	g.Combat.ActionUsed = true
	g.SetMessage(fmt.Sprintf("%s uses Second Wind! Heals %d (HP: %d/%d)",
		ent.Name, heal, stats.HP, stats.MaxHP))
}

func executeDash(g *GameState, w *World, ent *Entity, _, _ int) {
	g.Combat.MoveLeft += ent.Stats.MoveSpeed
	g.Combat.ActionUsed = true
	g.SetMessage(fmt.Sprintf("%s dashes! +%d movement", ent.Name, ent.Stats.MoveSpeed))
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
		threat.X, threat.Z = pushX, pushZ
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

	// Find nearest ally
	bestDist := 9999
	bestIdx := -1
	for i, ent := range g.Entities {
		if ent.Stats == nil || ent.Stats.HP <= 0 {
			continue
		}
		d := abs(threat.X-ent.X) + abs(threat.Z-ent.Z)
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		c.NextTurn(g, w)
		return
	}
	target := g.Entities[bestIdx]

	attacked := false

	inRange := threat.MaxRange <= 1 && adjacent(threat.X, threat.Z, target.X, target.Z) ||
		threat.MaxRange > 1 && withinRange(threat.X, threat.Z, target.X, target.Z, threat.MaxRange)
	if inRange {
		g.ResolveEnemyAttack(w, threat, target)
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
				threat.X, threat.Z = dest[0], dest[1]
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
			threat.MaxRange > 1 && withinRange(threat.X, threat.Z, target.X, target.Z, threat.MaxRange)
		if inRange {
			g.ResolveEnemyAttack(w, threat, target)
			attacked = true
		}
	}

	if attacked {
		cur.PursuitLeft = 3
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
		g.SetMessage("Enemies lost interest.")
	} else {
		// Set up the next combatant's turn
		next := c.Current()
		if !next.IsEnemy {
			if s := g.Entities[next.EntityIdx].Stats; s != nil {
				c.MoveLeft = s.MoveSpeed
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
		return
	}

	if roll < 20 && total < target.Stats.AC {
		g.SetMessage(fmt.Sprintf("Skeleton attacks %s (%d+%d=%d vs AC %d) — miss!",
			target.Name, roll, atkMod, total, target.Stats.AC))
		return
	}

	damageDice := rollDice(threat.AttackDice)
	if roll == 20 {
		damageDice += rollDice(threat.AttackDice)
	}
	damage := damageDice + atkMod

	target.Stats.HP -= damage

	if roll == 20 {
		g.SetMessage(fmt.Sprintf("Skeleton CRITS %s! %d damage! (HP: %d/%d)",
			target.Name, damage, target.Stats.HP, target.Stats.MaxHP))
	} else {
		g.SetMessage(fmt.Sprintf("Skeleton hits %s for %d damage (HP: %d/%d)",
			target.Name, damage, target.Stats.HP, target.Stats.MaxHP))
	}
}

func rollD20() int { return rand.Intn(20) + 1 }
func rollDice(sides int) int {
	if sides <= 0 {
		return 0
	}
	return rand.Intn(sides) + 1
}
