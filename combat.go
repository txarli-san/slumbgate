package main

import "math/rand"

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

func (c *Combat) NextTurn() {
	c.TurnIndex = (c.TurnIndex + 1) % len(c.Combatants)
	c.Phase = PhaseMovement
	c.ActionTaken = false
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

func rollD20() int { return rand.Intn(20) + 1 }
func rollDice(sides int) int {
	if sides <= 0 {
		return 0
	}
	return rand.Intn(sides) + 1
}
