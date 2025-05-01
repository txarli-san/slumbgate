package main

import (
	"math"
	"math/rand"
)

func distance(x1, y1, x2, y2 int) int {
	return int(math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2)))
}

func isAdjacentSimple(x1, y1, x2, y2 int) bool {
	return distance(x1, y1, x2, y2) == 1
}

func isAdjacentToEntity(px, py int, entity *Entity) bool {
	if entity == nil || entity.IsDying || entity.HP <= 0 {
		return false
	}
	for ex := entity.X; ex < entity.X+entity.Width; ex++ {
		for ey := entity.Y; ey < entity.Y+entity.Height; ey++ {
			if isAdjacentSimple(px, py, ex, ey) {
				return true
			}
		}
	}
	return false
}

func getModifier(score int) int { return (score - 10) / 2 }

func calculateProficiencyBonus(level int) int {
	if level < 5 {
		return 2
	}
	if level < 9 {
		return 3
	}
	if level < 13 {
		return 4
	}
	if level < 17 {
		return 5
	}
	return 6
}

func spellAttackRoll(g *Game) (int, int, bool) {
	mod := getModifier(g.Player.Intelligence)
	roll := rand.Intn(20) + 1
	isCrit := roll == 20
	isFumble := roll == 1
	total := roll + mod + g.Player.ProficiencyBonus
	return total, roll, isCrit && !isFumble
}

func spellSaveDC(g *Game) int {
	mod := getModifier(g.Player.Intelligence)
	return 8 + mod + g.Player.ProficiencyBonus
}

func isPlayerAdjacentToEnemy(g *Game) bool {
	if g.Player == nil {
		return false
	}
	for _, enemy := range g.Enemies {
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}
		if isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity) {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func ApplyCondition(e *Entity, c Condition) {
	if e == nil {
		return
	}
	found := false
	for i := range e.Conditions {
		if e.Conditions[i].Name == c.Name {
			e.Conditions[i] = c
			found = true
			break
		}
	}
	if !found {
		e.Conditions = append(e.Conditions, c)
	}
}

func RemoveCondition(e *Entity, name string) {
	if e == nil {
		return
	}
	nextConditions := make([]Condition, 0, len(e.Conditions))
	for _, cond := range e.Conditions {
		if cond.Name != name {
			nextConditions = append(nextConditions, cond)
		}
	}
	e.Conditions = nextConditions
}

func HasCondition(e *Entity, name string) bool {
	if e == nil {
		return false
	}
	for _, cond := range e.Conditions {
		if cond.Name == name {
			return true
		}
	}
	return false
}

func GetCondition(e *Entity, name string) *Condition {
	if e == nil {
		return nil
	}
	for i := range e.Conditions {
		if e.Conditions[i].Name == name {
			return &e.Conditions[i]
		}
	}
	return nil
}

func TickConditions(e *Entity) {
	if e == nil {
		return
	}
	nextConditions := make([]Condition, 0, len(e.Conditions))
	for _, cond := range e.Conditions {

		if cond.Duration > 0 {
			cond.Duration--
		}

		if cond.Duration != 0 {
			nextConditions = append(nextConditions, cond)
		}
	}
	e.Conditions = nextConditions
}

func GetEffectiveAC(e *Entity) int {
	if e == nil {
		return 0
	}
	baseAC := e.AC
	bonusAC := 0
	for _, cond := range e.Conditions {
		if value, ok := cond.Data["ACBonus"].(int); ok {
			bonusAC += value
		}
	}
	return baseAC + bonusAC
}
