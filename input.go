package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Game) handlePlayerInput() {
	isDebugLevelUp := ebiten.IsKeyPressed(ebiten.KeyShift) && inpututil.IsKeyJustPressed(ebiten.KeyL)
	if isDebugLevelUp && g.Player != nil {
		if g.Player.Level < maxLevel {
			g.addCombatLog("DEBUG: Forcing Level Up!")
			g.levelUpPlayer()
			g.InputMode = InputModeLevelUp
			g.primedActionID = ""
			return
		} else {
			g.addCombatLog("DEBUG: Already at max level!")
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if g.InputMode == InputModeCharacterSheet {
			g.InputMode = InputModeMap
			g.primedActionID = ""
		} else if g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp && g.InputMode != InputModeReactionPrompt {
			g.InputMode = InputModeCharacterSheet
			g.primedActionID = ""
		}
		return
	}

	switch g.InputMode {
	case InputModeReactionPrompt:
		if inpututil.IsKeyJustPressed(ebiten.KeyY) {
			g.addCombatLog("Reacting with Shield!")
			shieldAction := ActionTable["shield"]
			g.executeAction(shieldAction, -1, -1)
			g.reactionPending = false
			g.InputMode = InputModeMap
			g.addCombatLog("Attack resolved (after Shield).")
		} else if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.addCombatLog("Declined Shield reaction.")
			g.Player.UsedReaction = true
			g.reactionPending = false
			g.InputMode = InputModeMap
			g.addCombatLog("Attack resolved.")
		}
		return

	case InputModeLevelUp:
		madeChoice := false
		levelUpPendingChoices := false

		if g.Player.Class == "Mage" {
			if g.Player.Level == 2 && len(g.Player.KnownSpells) < 3 {
				levelUpPendingChoices = true
				spellIndex := -1
				if inpututil.IsKeyJustPressed(ebiten.Key1) {
					spellIndex = 0
				}
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					spellIndex = 1
				}
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					spellIndex = 2
				}
				if inpututil.IsKeyJustPressed(ebiten.Key4) {
					spellIndex = 3
				}
				if spellIndex != -1 {
					level2Spells := []string{"arcane_blink", "burning_hands", "frost_nova", "mind_spike"}
					chosenSpellID := level2Spells[spellIndex]
					g.Player.KnownSpells = append(g.Player.KnownSpells, chosenSpellID)
					chosenSpellName := ActionTable[chosenSpellID].Name
					g.addCombatLog(fmt.Sprintf("Learned %s!", chosenSpellName))
					madeChoice = true
				}
			} else if g.Player.Level == 3 && len(g.Player.KnownCantrips) < 2 {
				levelUpPendingChoices = true
				cantripIndex := -1
				if inpututil.IsKeyJustPressed(ebiten.Key1) {
					cantripIndex = 0
				}
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					cantripIndex = 1
				}
				if cantripIndex != -1 {
					level3Cantrips := []string{"ray_of_frost", "shocking_grasp"}
					chosenCantripID := level3Cantrips[cantripIndex]
					g.Player.KnownCantrips = append(g.Player.KnownCantrips, chosenCantripID)
					chosenCantripName := ActionTable[chosenCantripID].Name
					g.addCombatLog(fmt.Sprintf("Learned Cantrip: %s!", chosenCantripName))
					madeChoice = true
				}
			} else if g.Player.Level == 4 && len(g.Player.KnownSpells) < 4 {
				levelUpPendingChoices = true
				spellIndex := -1
				if inpututil.IsKeyJustPressed(ebiten.Key1) {
					spellIndex = 0
				}
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					spellIndex = 1
				}
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					spellIndex = 2
				}
				if inpututil.IsKeyJustPressed(ebiten.Key4) {
					spellIndex = 3
				}
				if spellIndex != -1 {
					level4Spells := []string{"magic_armor", "elemental_strike", "feather_fall", "expeditious_retreat"}
					chosenSpellID := level4Spells[spellIndex]
					g.Player.KnownSpells = append(g.Player.KnownSpells, chosenSpellID)
					chosenSpellName := ActionTable[chosenSpellID].Name
					g.addCombatLog(fmt.Sprintf("Learned %s!", chosenSpellName))
					madeChoice = true
				}
			}
		} else if g.Player.Class == "Fighter" {
			if g.Player.Level == 3 && g.Player.CombatStyle == "" {
				levelUpPendingChoices = true
				choiceIndex := -1
				if inpututil.IsKeyJustPressed(ebiten.Key1) {
					choiceIndex = 0
				}
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					choiceIndex = 1
				}
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					choiceIndex = 2
				}
				if choiceIndex != -1 {
					styles := []string{"Gladiator", "Ranger", "Juggernaut"}
					g.Player.CombatStyle = styles[choiceIndex]
					g.addCombatLog(fmt.Sprintf("Chosen Combat Style: %s!", g.Player.CombatStyle))
					if g.Player.CombatStyle == "Juggernaut" {
						g.Player.AC += 2
						g.addCombatLog("Gained +2 AC!")
					}
					madeChoice = true
				}
			} else if g.Player.Level == 4 && g.Player.CombatTechnique == "" {
				levelUpPendingChoices = true
				choiceIndex := -1
				if inpututil.IsKeyJustPressed(ebiten.Key1) {
					choiceIndex = 0
				}
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					choiceIndex = 1
				}
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					choiceIndex = 2
				}
				if choiceIndex != -1 {
					techniques := []string{"Stunning Strike", "Defensive Stance", "Quick Strike"}
					g.Player.CombatTechnique = techniques[choiceIndex]
					g.addCombatLog(fmt.Sprintf("Chosen Combat Technique: %s!", g.Player.CombatTechnique))
					if g.Player.CombatTechnique == "Defensive Stance" {
						g.Player.AC += 2
						g.Player.MaxMovementPoints -= 1
						g.addCombatLog("Gained +2 AC, Max Movement reduced by 1!")
					}
					madeChoice = true
				}
			}
		}

		if !madeChoice && !levelUpPendingChoices && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.addCombatLog("Level up complete. Preparing for next challenge...")
			g.longRest()
			if g.CurrentWaveIndex+1 < len(g.WaveDefinitions) {
				g.startNextWave()
			} else {
				g.addCombatLog("All challenges overcome! VICTORY!")
				g.setGameOver(true)
			}
		}
		return

	case InputModeRestPrompt:
		if inpututil.IsKeyJustPressed(ebiten.KeyY) {
			g.shortRest()
			g.startNextWave()
		} else if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.addCombatLog("Skipped short rest.")
			g.startNextWave()
		}
		return

	case InputModeCharacterSheet:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyC) {
			g.InputMode = InputModeMap
		}
		return

	case InputModeActionSelect:

		return

	case InputModeMap:

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			cursorX, cursorY := ebiten.CursorPosition()
			gridX := (cursorX - g.MapOffsetX) / tileSize
			gridY := (cursorY - g.MapOffsetY) / tileSize

			if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight {
				intentData := map[string]any{
					"X": gridX,
					"Y": gridY,
				}
				if g.primedActionID != "" {
					actionDef, exists := ActionTable[g.primedActionID]
					if exists {
						canUseAction := false
						switch actionDef.ActionType {
						case ActionTypeStandard:
							canUseAction = !g.Player.ActionTaken
						case ActionTypeBonus:
							canUseAction = !g.Player.BonusActionTaken
						case ActionTypeFree, ActionTypeReaction:
							canUseAction = true
						}

						if !canUseAction {
							g.addCombatLog(fmt.Sprintf("Cannot target for %s: %s already used.", actionDef.Name, actionDef.ActionType.String()))
							g.primedActionID = ""
						} else {
							intentData["ActionID"] = g.primedActionID
							g.IntentQueue = append(g.IntentQueue, Intent{Type: IntentAction, Data: intentData})

						}
					} else {
						g.addCombatLog(fmt.Sprintf("Error: Unknown primed action ID '%s'. Cancelling.", g.primedActionID))
						g.primedActionID = ""
					}
				} else {
					g.IntentQueue = append(g.IntentQueue, Intent{Type: IntentMove, Data: intentData})
				}
			} else {
			}
		} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if g.primedActionID != "" {
				intentData := map[string]any{"ActionID": g.primedActionID}
				g.IntentQueue = append(g.IntentQueue, Intent{Type: IntentCancelAction, Data: intentData})
			}
		}

	}
}
