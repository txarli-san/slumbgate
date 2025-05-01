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
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.selectedActionIndex--
			if g.selectedActionIndex < 0 {
				g.selectedActionIndex = len(g.availableActions) - 1
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.selectedActionIndex++
			if g.selectedActionIndex >= len(g.availableActions) {
				g.selectedActionIndex = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if g.selectedActionIndex >= 0 && g.selectedActionIndex < len(g.availableActions) {
				selectedActionDef := g.availableActions[g.selectedActionIndex]
				if !selectedActionDef.RequiresTarget {
					g.executeAction(selectedActionDef, -1, -1)
					if g.CurrentGameState == StatePlaying && selectedActionDef.ID != "wait" && g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
						g.InputMode = InputModeMap
						g.primedActionID = ""
					}
				} else {
					g.primedActionID = selectedActionDef.ID
					g.InputMode = InputModeMap
				}
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.InputMode = InputModeMap
			g.primedActionID = ""
		}
		return

	case InputModeMap:
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			g.buildAvailableActions()
			if len(g.availableActions) > 0 {
				foundLast := false
				if g.lastExecutedActionID != "" {
					for i, actionDef := range g.availableActions {
						if actionDef.ID == g.lastExecutedActionID {
							g.selectedActionIndex = i
							foundLast = true
							break
						}
					}
				}
				if !foundLast {
					g.selectedActionIndex = 0
				}
				g.InputMode = InputModeActionSelect
				g.primedActionID = ""
			} else {
				g.addCombatLog("No actions available!")
			}
			return
		}
		actionExecutedByClick := false
		if g.primedActionID != "" {
			actionDef, exists := ActionTable[g.primedActionID]
			if !exists {
				g.addCombatLog(fmt.Sprintf("Error: Unknown primed action ID '%s'. Cancelling.", g.primedActionID))
				g.primedActionID = ""
			} else {
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
					if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
						cursorX, cursorY := ebiten.CursorPosition()
						gridX := (cursorX - g.MapOffsetX) / tileSize
						gridY := (cursorY - g.MapOffsetY) / tileSize
						if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight {
							actionExecutedByClick = g.executeAction(actionDef, gridX, gridY)
							g.primedActionID = ""
						} else {
							g.addCombatLog("Clicked outside map. Targeting cancelled.")
							g.primedActionID = ""
						}
					} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
						g.addCombatLog(fmt.Sprintf("Targeting for %s cancelled.", actionDef.Name))
						g.primedActionID = ""
					}
				}
			}
		}
		if actionExecutedByClick {
			if g.CurrentGameState != StatePlaying || g.InputMode == InputModeRestPrompt || g.InputMode == InputModeLevelUp {
				return
			}
		}
		if g.primedActionID == "" && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			waitAction, exists := ActionTable["wait"]
			if exists {
				g.executeAction(waitAction, -1, -1)
			} else {
				g.addCombatLog("Error: Wait action not found!")
				g.endPlayerTurn()
			}
			return
		}
		if g.primedActionID == "" && g.Player.MovementPoints > 0 {
			moved := false
			startX, startY := g.Player.X, g.Player.Y
			targetX, targetY := startX, startY
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				targetY--
				moved = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				targetY++
				moved = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				targetX--
				moved = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				targetX++
				moved = true
			}
			if moved {
				if targetX >= 0 && targetX+g.Player.Width <= mapWidth && targetY >= 0 && targetY+g.Player.Height <= mapHeight {
					if !g.isTileFullyBlocked(targetX, targetY, g.Player.Width, g.Player.Height, -1) {
						performAoOCheck := !g.Player.IsDisengaging
						currentStartX, currentStartY := startX, startY
						currentTargetX, currentTargetY := targetX, targetY
						moveInterrupted := false

						if performAoOCheck {
							for _, enemy := range g.Enemies {
								if enemy.IsDying || enemy.HP <= 0 {
									continue
								}
								if HasCondition(&enemy.Entity, ConditionNoReactions) {
									continue
								}
								wasAdj := isAdjacentToEntity(currentStartX, currentStartY, &enemy.Entity)
								isStillAdj := isAdjacentToEntity(currentTargetX, currentTargetY, &enemy.Entity)
								if wasAdj && !isStillAdj {
									if g.Player.HP > 0 && !g.Player.IsDying {
										g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
										killedByAoO, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")

										if g.reactionPending {

											g.playerMovePending = true
											g.pendingMoveStartX = currentStartX
											g.pendingMoveStartY = currentStartY
											g.pendingMoveTargetX = currentTargetX
											g.pendingMoveTargetY = currentTargetY
											moveInterrupted = true
											return
										}
										if killedByAoO {

											moveInterrupted = true
											return
										}
									}
								}
							}
						}

						if !moveInterrupted {
							if g.Player.HP > 0 && !g.Player.IsDying {
								g.Player.X = currentTargetX
								g.Player.Y = currentTargetY
								g.Player.MovementPoints--
							}
						}
					} else {
						g.addCombatLog("Movement blocked.")
					}
				} else {
					g.addCombatLog("Cannot move outside map boundaries.")
				}
			}
		}
	}
}
