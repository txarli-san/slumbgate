package main

import (
	"fmt"
	"math/rand"
)

// Combat mode types

type CombatTurnPhase int

const (
	PhaseMovement CombatTurnPhase = iota
	PhaseAction
)

type Combatant struct {
	IsEnemy    bool
	EntityIdx  int      // index into GameState.Entities (allies)
	ThreatKey  [2]int   // key into World.Threats (enemies)
	Initiative int
}

type Combat struct {
	Combatants  []Combatant
	TurnIndex   int // whose turn in initiative order
	Phase       CombatTurnPhase
	MoveLeft    int // remaining movement for current combatant
	ActionTaken bool
	RoomIdx     int // which room this fight is in
}

func (c *Combat) Current() *Combatant {
	return &c.Combatants[c.TurnIndex]
}

func (c *Combat) NextTurn(g *GameState, w *World) {
	c.TurnIndex = (c.TurnIndex + 1) % len(c.Combatants)
	c.Phase = PhaseMovement
	c.ActionTaken = false
	// Set move points for new combatant
	next := c.Current()
	if !next.IsEnemy {
		if s := g.Entities[next.EntityIdx].Stats; s != nil {
			c.MoveLeft = s.MoveSpeed
		}
	} else {
		if t, ok := w.Threats[next.ThreatKey]; ok {
			c.MoveLeft = t.MoveSpeed
		}
	}
}

// StartCombat initiates combat with an entity entering a room with threats.
// Rolls initiative (d20 + DEX mod) for all participants.
func (g *GameState) StartCombat(w *World, entityIdx int, roomIdx int) {
	ent := g.Entities[entityIdx]
	ent.Task = nil // stop whatever they were doing

	var combatants []Combatant

	// Add the entity as ally
	dexMod := 0
	if ent.Stats != nil {
		dexMod = ent.Stats.Mod(ent.Stats.DEX)
	}
	combatants = append(combatants, Combatant{
		IsEnemy:    false,
		EntityIdx:  entityIdx,
		Initiative: rollD20() + dexMod,
	})

	// Add all threats in the same room
	for key, threat := range w.Threats {
		if threat.RoomIdx == roomIdx {
			combatants = append(combatants, Combatant{
				IsEnemy:    true,
				ThreatKey:  key,
				Initiative: rollD20() + (threat.DEX-10)/2,
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

	// Set move points for first combatant
	moveLeft := 0
	first := combatants[0]
	if !first.IsEnemy {
		if s := g.Entities[first.EntityIdx].Stats; s != nil {
			moveLeft = s.MoveSpeed
		}
	} else {
		if t, ok := w.Threats[first.ThreatKey]; ok {
			moveLeft = t.MoveSpeed
		}
	}

	g.Combat = &Combat{
		Combatants: combatants,
		TurnIndex:  0,
		Phase:      PhaseMovement,
		MoveLeft:   moveLeft,
		RoomIdx:    roomIdx,
	}
	g.SelectedEnt = entityIdx
	g.SetMessage("Combat started! Roll initiative!")
}

// ResolveCombatAttack handles an ally entity attacking a threat.
// d20 + ability mod + proficiency vs AC. Crit on nat 20, fumble on nat 1.
func (g *GameState) ResolveCombatAttack(w *World, attacker *Combatant, threat Threat, tx, tz int) {
	ent := g.Entities[attacker.EntityIdx]
	stats := ent.Stats
	if stats == nil {
		return
	}

	// Face the target
	ent.FacingAngle = FacingAngleFromDir(tx-ent.X, tz-ent.Z)

	// Attack roll: d20 + STR mod + proficiency (melee for now)
	roll := rollD20()
	atkMod := stats.Mod(stats.STR)
	total := roll + atkMod + stats.ProfBonus

	if roll == 1 {
		g.SetMessage(fmt.Sprintf("%s attacks — nat 1! Miss!", ent.Name))
		g.Combat.ActionTaken = true
		return
	}

	if roll < 20 && total < threat.AC {
		g.SetMessage(fmt.Sprintf("%s attacks (%d+%d=%d vs AC %d) — miss!",
			ent.Name, roll, atkMod+stats.ProfBonus, total, threat.AC))
		g.Combat.ActionTaken = true
		return
	}

	// Hit — roll damage: 1d8 + STR mod (longsword). Crit doubles dice.
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

	// Check if threat dies
	if threat.HP <= 0 {
		w.RemoveThreat(tx, tz)
		g.removeCombatant(attacker, tx, tz)
		g.SetMessage(fmt.Sprintf("%s slays the skeleton! (%d damage)", ent.Name, damage))
	}

	g.Combat.ActionTaken = true
}

// removeCombatant removes a dead enemy from the initiative order.
func (g *GameState) removeCombatant(_ *Combatant, tx, tz int) {
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

	// Check if all enemies are dead — end combat
	anyEnemy := false
	for _, cb := range c.Combatants {
		if cb.IsEnemy {
			anyEnemy = true
			break
		}
	}
	if !anyEnemy {
		g.Combat = nil
		g.SetMessage("Combat ended — victory!")
	}
}

// RunEnemyTurn executes a simple enemy AI: move toward nearest ally, attack if adjacent.
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
		c.NextTurn(g, w)
		return
	}

	// Find nearest ally
	bestDist := 9999
	bestIdx := -1
	for i, ent := range g.Entities {
		if ent.Stats == nil || ent.Stats.HP <= 0 {
			continue
		}
		d := abs(threat.X-ent.X) + abs(threat.Z-ent.Z) // manhattan
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

	// If adjacent, attack
	if withinRange(threat.X, threat.Z, target.X, target.Z, threat.MaxRange) && !c.ActionTaken {
		g.ResolveEnemyAttack(w, threat, target)
		c.ActionTaken = true
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
			// Don't step onto the target's tile
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
				old := [2]int{threat.X, threat.Z}
				threat.X, threat.Z = dest[0], dest[1]
				w.Threats[cur.ThreatKey] = threat
				// Update threat key if position changed
				if old != cur.ThreatKey {
					// Key is the original spawn position, doesn't change
				}
				c.MoveLeft -= steps
			}
		}
	}

	// Try attack again after moving
	threat, ok = w.Threats[cur.ThreatKey]
	if ok && !c.ActionTaken && withinRange(threat.X, threat.Z, target.X, target.Z, threat.MaxRange) {
		g.ResolveEnemyAttack(w, threat, target)
		c.ActionTaken = true
	}

	c.NextTurn(g, w)
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

	// TODO: entity death handling
}

func rollD20() int { return rand.Intn(20) + 1 }
func rollDice(sides int) int {
	if sides <= 0 {
		return 0
	}
	return rand.Intn(sides) + 1
}
