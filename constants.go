package main

import "image/color"

const (
	screenWidth         = 640
	screenHeight        = 480
	tileSize            = 32
	spriteSize          = 32
	spriteScale         = float64(tileSize) / float64(spriteSize)
	mapWidth            = 10
	mapHeight           = 10
	combatLogLength     = 7
	playerBaseMovement  = 5
	enemyBaseMovement   = 4
	playerRangedRange   = 5
	hpBarHeight         = 4
	hpBarOffsetY        = 2
	sheetWidthInSprites = 32
	maxLevel            = 20
	attackBumpDuration  = 10
	deathFadeDuration   = 30

	uiPanelHeight = 50
	uiButtonSize  = 40
	uiButtonPad   = 5
)

const (
	TargetSelf          TargetType = "self"
	TargetEnemyAdjacent TargetType = "enemy_adjacent"
	TargetEnemyRange    TargetType = "enemy_range"
	TargetEmptyTile     TargetType = "empty_tile"
	TargetNone          TargetType = "none"
)

var (
	colorYellow      = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	colorWhite       = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colorBlack       = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	colorGray        = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
	colorLocked      = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	colorUIPanel     = color.NRGBA{R: 30, G: 30, B: 40, A: 240}
	colorButtonHover = color.NRGBA{R: 80, G: 80, B: 100, A: 255}
)

var visualEffectColors = map[string]color.NRGBA{
	"Fire":    {R: 255, G: 100, B: 0, A: 255},
	"Force":   {R: 220, G: 220, B: 255, A: 255},
	"Arcane":  {R: 180, G: 100, B: 255, A: 255},
	"Cold":    {R: 100, G: 200, B: 255, A: 255},
	"Psychic": {R: 255, G: 100, B: 180, A: 255},
	"Heal":    {R: 100, G: 255, B: 100, A: 255},
	"Default": {R: 255, G: 255, B: 255, A: 255},
	"None":    {R: 0, G: 0, B: 0, A: 0},
	"Bleed":   {R: 180, G: 0, B: 0, A: 255},
}

type GameState int

const (
	StateClassSelection GameState = iota
	StatePlaying
	StateGameOverScreen
)

type TurnState int

const (
	PlayerTurn TurnState = iota
	EnemyTurn
	GameOver
)

type InputMode int

const (
	InputModeMap InputMode = iota

	InputModeCharacterSheet
	InputModeRestPrompt
	InputModeLevelUp
	InputModeReactionPrompt
)

type ActionType int

const (
	ActionTypeStandard ActionType = iota
	ActionTypeBonus
	ActionTypeFree
	ActionTypeReaction
)

type ResourceType string

const (
	ResourceNone         ResourceType = "none"
	ResourceHitDice      ResourceType = "hit_dice"
	ResourceClassFeature ResourceType = "class_feature"
	ResourceSpellSlotL1  ResourceType = "spell_slot_l1"
)

type ResourceRestType int

const (
	RestTypeShort ResourceRestType = iota
	RestTypeLong
	RestTypeCombat
	RestTypeNever
)

type EnemyTurnPhase int

const (
	PhaseEnemyStartTurn EnemyTurnPhase = iota
	PhaseEnemyDecideAction
	PhaseEnemyMove
	PhaseEnemyAction
	PhaseEnemyPausedForReaction
	PhaseEnemyDone
)

const (
	ConditionSlowed             = "Slowed"
	ConditionMagicArmor         = "MagicArmor"
	ConditionFeatherFall        = "FeatherFall"
	ConditionExpeditiousRetreat = "ExpeditiousRetreat"
	ConditionShielded           = "Shielded"
	ConditionNoReactions        = "NoReactions"
	ConditionBleeding           = "Bleeding"
	ConditionStunned            = "Stunned"
)

type IntentType int

const (
	IntentMove IntentType = iota
	IntentAction
	IntentEndTurn
	IntentUIClick
	IntentCancelAction
)
