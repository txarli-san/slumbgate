package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

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
)

var (
	colorYellow = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	colorWhite  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colorBlack  = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	colorGray   = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
	colorLocked = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
)

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

func (ts TurnState) String() string {
	switch ts {
	case PlayerTurn:
		return "Player Turn"
	case EnemyTurn:
		return "Enemy Turn"
	case GameOver:
		return "Game Over"
	default:
		return "Unknown"
	}
}

type InputMode int

const (
	InputModeMap InputMode = iota
	InputModeActionSelect
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

func (at ActionType) String() string {
	switch at {
	case ActionTypeStandard:
		return "Action"
	case ActionTypeBonus:
		return "Bonus Action"
	case ActionTypeFree:
		return "Free Action"
	case ActionTypeReaction:
		return "Reaction"
	default:
		return "Unknown Action Type"
	}
}

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

type Entity struct {
	X               int
	Y               int
	Width           int
	Height          int
	HP              int
	MaxHP           int
	AC              int
	Strength        int
	Dexterity       int
	Constitution    int
	Intelligence    int
	Wisdom          int
	Charisma        int
	Sprite          *ebiten.Image
	DrawOpts        ebiten.DrawImageOptions
	Name            string
	AttackBumpTimer int
	IsDying         bool
	CurrentAlpha    float64
}

type Player struct {
	Entity
	Level                int
	Class                string
	ProficiencyBonus     int
	MovementPoints       int
	MaxMovementPoints    int
	ActionTaken          bool
	BonusActionTaken     bool
	IsDisengaging        bool
	ClassResources       map[string]int
	HitDice              int
	MaxHitDice           int
	SpellSlotsL1         int
	MaxSpellSlotsL1      int
	KnownSpells          []string
	KnownCantrips        []string
	UsedReaction         bool
	UsedArcaneRecovery   bool
	ACBonusUntilNextTurn int
	LastSpellCastID      string
	CombatStyle          string
	CombatTechnique      string
	IsSlowed             bool

	SlowDuration               int
	MagicArmorDuration         int
	ExpeditiousRetreatDuration int
	FeatherFallDuration        int
	NextElementType            string
}

type Enemy struct {
	Entity
	MovementPoints    int
	MaxMovementPoints int
	ActionAvailable   bool
	AttackType        string
	MaxRange          int
}

type ClassAction struct {
	ActionID      string
	Name          string
	RequiredLevel int
	ActionType    ActionType
	ResourceType  ResourceType
	ResourceCost  int
	UsesPerRest   int
	RefreshesOn   ResourceRestType
	Description   string
}

type ClassDefinition struct {
	Name             string
	HitDieSize       int
	PrimaryAbility   string
	SavingThrowProfs []string
	ClassActions     map[string]*ClassAction
	SpriteSheetX     int
	SpriteSheetY     int
}

type SelectableClass struct {
	Name        string
	IsAvailable bool
}

type Game struct {
	Player                    *Player
	Enemies                   []*Enemy
	GameMap                   [mapWidth][mapHeight]int
	TileImage                 *ebiten.Image
	CurrentTurn               TurnState
	CurrentWaveIndex          int
	CombatLog                 []string
	MapOffsetX                int
	MapOffsetY                int
	RangeOverlayTile          *ebiten.Image
	InputMode                 InputMode
	rogueSheet                *ebiten.Image
	monsterSheet              *ebiten.Image
	availableActions          []*ActionDefinition
	selectedActionIndex       int
	primedActionID            string
	lastExecutedActionID      string
	WaveDefinitions           []WaveDefinition
	FloatingTexts             []*FloatingText
	reactionPending           bool
	reactionAttackerID        int
	pendingLevelUpSpellChoice int
	CurrentGameState          GameState
	selectableClasses         []SelectableClass
	classSelectionIndex       int
	isVictory                 bool
}

type ActionExecuteFunc func(g *Game, targetX, targetY int) bool

type TargetType string

type FloatingText struct {
	Text      string
	X, Y      float64
	Life      int
	MaxLife   int
	Color     color.Color
	VelocityY float64
}

const (
	TargetSelf          TargetType = "self"
	TargetEnemyAdjacent TargetType = "enemy_adjacent"
	TargetEnemyRange    TargetType = "enemy_range"
	TargetEmptyTile     TargetType = "empty_tile"
	TargetNone          TargetType = "none"
)

type ActionDefinition struct {
	ID               string
	Name             string
	ActionType       ActionType
	ResourceType     ResourceType
	ResourceCost     int
	Targeting        TargetType
	Range            int
	RequiresTarget   bool
	Execute          ActionExecuteFunc
	VisualEffectType string
}

type EnemyDefinition struct {
	Name             string
	SpriteSheetX     int
	SpriteSheetY     int
	BaseHP           int
	AC               int
	Str              int
	Dex              int
	Con              int
	Int              int
	Wis              int
	Cha              int
	Move             int
	Width            int
	Height           int
	AttackType       string
	MaxRange         int
	AttackDiceNum    int
	AttackDiceSize   int
	AttackAbilityMod string
}

type EnemySpawnInfo struct {
	TypeName      string
	SpawnPointIdx int
}

type WaveDefinition struct {
	EnemiesToSpawn []EnemySpawnInfo
	IsBossWave     bool
}

var visualEffectColors = map[string]color.NRGBA{
	"Fire":    {R: 255, G: 100, B: 0, A: 255},
	"Force":   {R: 220, G: 220, B: 255, A: 255},
	"Arcane":  {R: 180, G: 100, B: 255, A: 255},
	"Cold":    {R: 100, G: 200, B: 255, A: 255},
	"Psychic": {R: 255, G: 100, B: 180, A: 255},
	"Heal":    {R: 100, G: 255, B: 100, A: 255},
	"Default": {R: 255, G: 255, B: 255, A: 255},
	"None":    {R: 0, G: 0, B: 0, A: 0},
}

var EnemyDefinitions = map[string]EnemyDefinition{
	"Small Slime": {
		Name: "Small Slime", SpriteSheetX: 0, SpriteSheetY: 2,
		BaseHP: 6, AC: 10, Str: 12, Dex: 10, Con: 11, Int: 8, Wis: 8, Cha: 8, Move: 3,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 0, AttackDiceSize: 0, AttackAbilityMod: "STR",
	},
	"Tough Slime": {
		Name: "Tough Slime", SpriteSheetX: 1, SpriteSheetY: 2,
		BaseHP: 7, AC: 10, Str: 12, Dex: 10, Con: 11, Int: 8, Wis: 8, Cha: 8, Move: 3,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 0, AttackDiceSize: 0, AttackAbilityMod: "STR",
	},
	"Goo Spitter": {
		Name: "Goo Spitter", SpriteSheetX: 2, SpriteSheetY: 2,
		BaseHP: 5, AC: 11, Str: 8, Dex: 14, Con: 10, Int: 8, Wis: 8, Cha: 8, Move: 4,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 4,
		AttackDiceNum: 0, AttackDiceSize: 0, AttackAbilityMod: "DEX",
	},
	"Big Slime Boss": {
		Name: "Big Slime Boss", SpriteSheetX: 1, SpriteSheetY: 2,
		BaseHP: 25, AC: 12, Str: 14, Dex: 8, Con: 15, Int: 6, Wis: 6, Cha: 6, Move: 2,
		Width: 2, Height: 2, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 0, AttackDiceSize: 0, AttackAbilityMod: "STR",
	},
	"Melee Skeleton": {
		Name: "Melee Skeleton", SpriteSheetX: 0, SpriteSheetY: 4,
		BaseHP: 13, AC: 13, Str: 10, Dex: 14, Con: 15, Int: 6, Wis: 8, Cha: 5, Move: enemyBaseMovement,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 1, AttackDiceSize: 6, AttackAbilityMod: "DEX",
	},
	"Ranged Skeleton": {
		Name: "Ranged Skeleton", SpriteSheetX: 1, SpriteSheetY: 4,
		BaseHP: 11, AC: 13, Str: 8, Dex: 16, Con: 13, Int: 6, Wis: 8, Cha: 5, Move: enemyBaseMovement,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 5,
		AttackDiceNum: 1, AttackDiceSize: 6, AttackAbilityMod: "DEX",
	},
	"Goblin Scout": {
		Name: "Goblin Scout", SpriteSheetX: 2, SpriteSheetY: 0,
		BaseHP: 6, AC: 9, Str: 10, Dex: 15, Con: 10, Int: 10, Wis: 10, Cha: 8, Move: 6,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 1, AttackDiceSize: 4, AttackAbilityMod: "DEX",
	},
	"Goblin Archer": {
		Name: "Goblin Archer", SpriteSheetX: 5, SpriteSheetY: 0,
		BaseHP: 4, AC: 10, Str: 8, Dex: 13, Con: 11, Int: 10, Wis: 12, Cha: 8, Move: 5,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 6,
		AttackDiceNum: 1, AttackDiceSize: 6, AttackAbilityMod: "DEX",
	},
	"Goblin Brute": {
		Name: "Goblin Brute", SpriteSheetX: 7, SpriteSheetY: 0,
		BaseHP: 8, AC: 12, Str: 14, Dex: 12, Con: 14, Int: 8, Wis: 8, Cha: 8, Move: 4,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 1, AttackDiceSize: 8, AttackAbilityMod: "STR",
	},
	"Goblin Chieftain": {
		Name: "Goblin Chieftain", SpriteSheetX: 4, SpriteSheetY: 0,
		BaseHP: 30, AC: 15, Str: 17, Dex: 14, Con: 16, Int: 12, Wis: 13, Cha: 14, Move: 4,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 2, AttackDiceSize: 6, AttackAbilityMod: "STR",
	},
	"Goblin Warlock": {
		Name: "Goblin Warlock", SpriteSheetX: 6, SpriteSheetY: 0,
		BaseHP: 10, AC: 11, Str: 8, Dex: 12, Con: 10, Int: 14, Wis: 12, Cha: 12, Move: 4,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 5,
		AttackDiceNum: 1, AttackDiceSize: 6, AttackAbilityMod: "INT",
	},
	"Spore Mushroom": {
		Name: "Spore Mushroom", SpriteSheetX: 0, SpriteSheetY: 10,
		BaseHP: 12, AC: 11, Str: 6, Dex: 8, Con: 14, Int: 2, Wis: 10, Cha: 4, Move: 2,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 3,
		AttackDiceNum: 1, AttackDiceSize: 4, AttackAbilityMod: "WIS",
	},
	"Elder Spore Mushroom": {
		Name: "Elder Spore Mushroom", SpriteSheetX: 1, SpriteSheetY: 10,
		BaseHP: 40, AC: 13, Str: 8, Dex: 6, Con: 18, Int: 4, Wis: 12, Cha: 6, Move: 1,
		Width: 2, Height: 2, AttackType: "ranged", MaxRange: 4,
		AttackDiceNum: 1, AttackDiceSize: 8, AttackAbilityMod: "CON",
	},
	"Centaur": {
		Name: "Centaur", SpriteSheetX: 3, SpriteSheetY: 7,
		BaseHP: 18, AC: 14, Str: 14, Dex: 16, Con: 14, Int: 10, Wis: 12, Cha: 10, Move: 7,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 5,
		AttackDiceNum: 1, AttackDiceSize: 8, AttackAbilityMod: "DEX",
	},
	"Druid": {
		Name: "Druid", SpriteSheetX: 6, SpriteSheetY: 7,
		BaseHP: 14, AC: 12, Str: 8, Dex: 12, Con: 12, Int: 10, Wis: 16, Cha: 12, Move: 4,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 4,
		AttackDiceNum: 1, AttackDiceSize: 6, AttackAbilityMod: "WIS",
	},
	"Faun": {
		Name: "Faun", SpriteSheetX: 0, SpriteSheetY: 7,
		BaseHP: 12, AC: 13, Str: 10, Dex: 14, Con: 12, Int: 12, Wis: 14, Cha: 14, Move: 5,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 3,
		AttackDiceNum: 1, AttackDiceSize: 6, AttackAbilityMod: "WIS",
	},
	"Dire Wolf": {
		Name: "Dire Wolf", SpriteSheetX: 5, SpriteSheetY: 6,
		BaseHP: 20, AC: 13, Str: 16, Dex: 15, Con: 14, Int: 3, Wis: 12, Cha: 6, Move: 7,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 1, AttackDiceSize: 8, AttackAbilityMod: "STR",
	},
	"Naga": {
		Name: "Naga", SpriteSheetX: 4, SpriteSheetY: 7,
		BaseHP: 22, AC: 15, Str: 14, Dex: 16, Con: 14, Int: 12, Wis: 12, Cha: 14, Move: 4,
		Width: 2, Height: 2, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 1, AttackDiceSize: 8, AttackAbilityMod: "STR",
	},
	"Tauren": {
		Name: "Tauren", SpriteSheetX: 7, SpriteSheetY: 7,
		BaseHP: 45, AC: 15, Str: 18, Dex: 10, Con: 18, Int: 8, Wis: 12, Cha: 10, Move: 4,
		Width: 2, Height: 2, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 2, AttackDiceSize: 8, AttackAbilityMod: "STR",
	},
	"Orc Warrior": {
		Name: "Orc Warrior", SpriteSheetX: 3, SpriteSheetY: 0,
		BaseHP: 18, AC: 15, Str: 18, Dex: 10, Con: 18, Int: 8, Wis: 12, Cha: 10, Move: 4,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 2, AttackDiceSize: 8, AttackAbilityMod: "STR",
	},
	"Orc Shaman": {
		Name: "Orc Shaman", SpriteSheetX: 1, SpriteSheetY: 0,
		BaseHP: 15, AC: 12, Str: 8, Dex: 12, Con: 13, Int: 10, Wis: 15, Cha: 10, Move: 4,
		Width: 1, Height: 1, AttackType: "ranged", MaxRange: 5,
		AttackDiceNum: 1, AttackDiceSize: 8, AttackAbilityMod: "WIS",
	},
	"Orc Chieftain": {
		Name: "Orc Chieftain", SpriteSheetX: 4, SpriteSheetY: 0,
		BaseHP: 18, AC: 15, Str: 18, Dex: 10, Con: 18, Int: 8, Wis: 12, Cha: 10, Move: 4,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 2, AttackDiceSize: 8, AttackAbilityMod: "STR",
	},
	"Orc Brute": {
		Name: "Orc Brute", SpriteSheetX: 0, SpriteSheetY: 0,
		BaseHP: 18, AC: 15, Str: 18, Dex: 10, Con: 18, Int: 8, Wis: 12, Cha: 10, Move: 4,
		Width: 1, Height: 1, AttackType: "melee", MaxRange: 1,
		AttackDiceNum: 2, AttackDiceSize: 8, AttackAbilityMod: "STR",
	},
}

var ClassDefinitions = map[string]*ClassDefinition{
	"Fighter": {
		Name:             "Fighter",
		HitDieSize:       10,
		PrimaryAbility:   "STR",
		SavingThrowProfs: []string{"STR", "CON"},
		SpriteSheetX:     1,
		SpriteSheetY:     1,
		ClassActions: map[string]*ClassAction{
			"second_wind": {
				ActionID:      "second_wind",
				Name:          "Second Wind",
				RequiredLevel: 2,
				ActionType:    ActionTypeStandard,
				ResourceType:  ResourceClassFeature,
				ResourceCost:  1,
				UsesPerRest:   1,
				RefreshesOn:   RestTypeNever,
				Description:   "Use Action: Heal using 1 Hit Die (d10 + CON).",
			},
			"action_surge": {
				ActionID:      "action_surge",
				Name:          "Action Surge",
				RequiredLevel: 2,
				ActionType:    ActionTypeFree,
				ResourceType:  ResourceClassFeature,
				ResourceCost:  1,
				UsesPerRest:   1,
				RefreshesOn:   RestTypeNever,
				Description:   "Gain an additional standard action this turn.",
			},
		},
	},
	"Mage": {
		Name:             "Mage",
		HitDieSize:       6,
		PrimaryAbility:   "INT",
		SavingThrowProfs: []string{"INT", "WIS"},
		SpriteSheetX:     0,
		SpriteSheetY:     4,
		ClassActions: map[string]*ClassAction{
			"arcane_recovery": {
				ActionID:      "arcane_recovery",
				Name:          "Arcane Recovery",
				RequiredLevel: 2,
				ActionType:    ActionTypeFree,
				ResourceType:  ResourceClassFeature,
				ResourceCost:  1,
				UsesPerRest:   1,
				RefreshesOn:   RestTypeLong,
				Description:   "Once per long rest cycle, recover spell slots during a short rest.",
			},
		},
	},
}

//go:embed assets/*
var assetsFS embed.FS

var ActionTable map[string]*ActionDefinition

func init() {
	ActionTable = map[string]*ActionDefinition{
		"melee_attack": {
			ID:               "melee_attack",
			Name:             "Melee Attack",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetEnemyAdjacent,
			Range:            1,
			RequiresTarget:   true,
			Execute:          executeMeleeAttack,
			VisualEffectType: "Default",
		},
		"ranged_attack": {
			ID:               "ranged_attack",
			Name:             "Ranged Attack",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetEnemyRange,
			Range:            playerRangedRange,
			RequiresTarget:   true,
			Execute:          executeRangedAttack,
			VisualEffectType: "Default",
		},
		"quick_strike": {
			ID:               "quick_strike",
			Name:             "Quick Strike",
			ActionType:       ActionTypeBonus,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetEnemyAdjacent,
			Range:            1,
			RequiresTarget:   true,
			Execute:          executeQuickStrike,
			VisualEffectType: "Default",
		},
		"dash": {
			ID:               "dash",
			Name:             "Dash",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeDash,
			VisualEffectType: "Default",
		},
		"disengage": {
			ID:               "disengage",
			Name:             "Disengage",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeDisengage,
			VisualEffectType: "Default",
		},
		"wait": {
			ID:               "wait",
			Name:             "Wait (End Turn)",
			ActionType:       ActionTypeFree,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetNone,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeWait,
			VisualEffectType: "None",
		},
		"second_wind": {
			ID:               "second_wind",
			Name:             "Second Wind",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceClassFeature,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeSecondWind,
			VisualEffectType: "Heal",
		},
		"action_surge": {
			ID:               "action_surge",
			Name:             "Action Surge",
			ActionType:       ActionTypeFree,
			ResourceType:     ResourceClassFeature,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeActionSurge,
			VisualEffectType: "Default",
		},
		"firebolt": {
			ID:               "firebolt",
			Name:             "Firebolt",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetEnemyRange,
			Range:            playerRangedRange + 2,
			RequiresTarget:   true,
			Execute:          executeFirebolt,
			VisualEffectType: "Fire",
		},
		"arcane_recovery": {
			ID:               "arcane_recovery",
			Name:             "Arcane Recovery",
			ActionType:       ActionTypeFree,
			ResourceType:     ResourceClassFeature,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          nil,
			VisualEffectType: "None",
		},
		"magic_missile": {
			ID:               "magic_missile",
			Name:             "Magic Missile",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetEnemyRange,
			Range:            playerRangedRange + 2,
			RequiresTarget:   true,
			Execute:          executeMagicMissile,
			VisualEffectType: "Force",
		},
		"shield": {
			ID:               "shield",
			Name:             "Shield",
			ActionType:       ActionTypeReaction,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeShield,
			VisualEffectType: "Arcane",
		},
		"arcane_blink": {
			ID:               "arcane_blink",
			Name:             "Arcane Blink",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetEmptyTile,
			Range:            playerBaseMovement,
			RequiresTarget:   true,
			Execute:          executeArcaneBlink,
			VisualEffectType: "Arcane",
		},
		"burning_hands": {
			ID:               "burning_hands",
			Name:             "Burning Hands",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            1,
			RequiresTarget:   false,
			Execute:          executeBurningHands,
			VisualEffectType: "Fire",
		},
		"frost_nova": {
			ID:               "frost_nova",
			Name:             "Frost Nova",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            1,
			RequiresTarget:   false,
			Execute:          executeFrostNova,
			VisualEffectType: "Cold",
		},
		"mind_spike": {
			ID:               "mind_spike",
			Name:             "Mind Spike",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetEnemyRange,
			Range:            playerRangedRange,
			RequiresTarget:   true,
			Execute:          executeMindSpike,
			VisualEffectType: "Psychic",
		},
		"ray_of_frost": {
			ID:               "ray_of_frost",
			Name:             "Ray of Frost",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetEnemyRange,
			Range:            playerRangedRange + 1, // 6 range?
			RequiresTarget:   true,
			Execute:          executeRayOfFrost,
			VisualEffectType: "Cold",
		},
		"shocking_grasp": {
			ID:               "shocking_grasp",
			Name:             "Shocking Grasp",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceNone,
			ResourceCost:     0,
			Targeting:        TargetEnemyAdjacent,
			Range:            1,
			RequiresTarget:   true,
			Execute:          executeShockingGrasp,
			VisualEffectType: "Default",
		},

		"magic_armor": {
			ID:               "magic_armor",
			Name:             "Magic Armor",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeMagicArmor,
			VisualEffectType: "Arcane",
		},
		"elemental_strike": {
			ID:               "elemental_strike",
			Name:             "Elemental Strike",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetEnemyRange,
			Range:            playerRangedRange,
			RequiresTarget:   true,
			Execute:          executeElementalStrike,
			VisualEffectType: "Default",
		},
		"feather_fall": {
			ID:               "feather_fall",
			Name:             "Feather Fall",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeFeatherFall,
			VisualEffectType: "Default",
		},
		"expeditious_retreat": {
			ID:               "expeditious_retreat",
			Name:             "Expeditious Retreat",
			ActionType:       ActionTypeStandard,
			ResourceType:     ResourceSpellSlotL1,
			ResourceCost:     1,
			Targeting:        TargetSelf,
			Range:            0,
			RequiresTarget:   false,
			Execute:          executeExpeditiousRetreat,
			VisualEffectType: "Default",
		},
	}
}

func loadImage(path string) (*ebiten.Image, error) {
	file, err := assetsFS.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded image %s: %w", path, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode embedded image %s: %w", path, err)
	}
	return ebiten.NewImageFromImage(img), nil
}

func getSpriteFromSheet(sheet *ebiten.Image, sx, sy int) *ebiten.Image {
	if sheet == nil {
		log.Println("Warning: Sprite sheet not loaded.")
		img := ebiten.NewImage(spriteSize, spriteSize)
		img.Fill(colorWhite)
		return img
	}
	x := sx * spriteSize
	y := sy * spriteSize
	rect := image.Rect(x, y, x+spriteSize, y+spriteSize)
	bounds := sheet.Bounds()
	if !rect.In(bounds) {
		log.Printf("Warning: Sprite index (%d, %d) out of sheet bounds (%v).", sx, sy, bounds)
		img := ebiten.NewImage(spriteSize, spriteSize)
		img.Fill(colorWhite)
		return img
	}
	return sheet.SubImage(rect).(*ebiten.Image)
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

func NewGameInitial() *Game {
	g := &Game{}
	g.CurrentGameState = StateClassSelection
	g.selectableClasses = []SelectableClass{
		{Name: "Fighter", IsAvailable: true},
		{Name: "Mage", IsAvailable: true},
		{Name: "Druid", IsAvailable: false},
		{Name: "Cleric", IsAvailable: false},
		{Name: "Rogue", IsAvailable: false},
	}
	g.classSelectionIndex = 0

	for i, class := range g.selectableClasses {
		if class.IsAvailable {
			g.classSelectionIndex = i
			break
		}
	}
	return g
}

func (g *Game) InitializeGameplay(playerClassName string) {
	g.Enemies = make([]*Enemy, 0)
	g.CombatLog = make([]string, 0, combatLogLength)
	g.MapOffsetX = (screenWidth - (mapWidth * tileSize)) / 2
	g.MapOffsetY = (screenHeight - (mapHeight * tileSize)) / 2
	g.InputMode = InputModeMap
	g.availableActions = make([]*ActionDefinition, 0)
	g.lastExecutedActionID = ""
	g.CurrentWaveIndex = -1
	g.FloatingTexts = make([]*FloatingText, 0)
	g.isVictory = false

	g.WaveDefinitions = []WaveDefinition{
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Scout", SpawnPointIdx: 0}, {TypeName: "Goblin Scout", SpawnPointIdx: 1}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Scout", SpawnPointIdx: 2}, {TypeName: "Goblin Scout", SpawnPointIdx: 3}, {TypeName: "Goblin Archer", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Archer", SpawnPointIdx: 2}, {TypeName: "Goblin Archer", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Chieftain", SpawnPointIdx: 4}}, IsBossWave: true},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Scout", SpawnPointIdx: 0}, {TypeName: "Goblin Scout", SpawnPointIdx: 1}, {TypeName: "Goblin Archer", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Archer", SpawnPointIdx: 4}, {TypeName: "Goblin Archer", SpawnPointIdx: 2}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Warlock", SpawnPointIdx: 4}, {TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Scout", SpawnPointIdx: 2}, {TypeName: "Goblin Scout", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goblin Chieftain", SpawnPointIdx: 4}, {TypeName: "Goblin Brute", SpawnPointIdx: 0}, {TypeName: "Goblin Brute", SpawnPointIdx: 1}, {TypeName: "Goblin Warlock", SpawnPointIdx: 3}}, IsBossWave: true},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Small Slime", SpawnPointIdx: 0}, {TypeName: "Small Slime", SpawnPointIdx: 1}, {TypeName: "Spore Mushroom", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Tough Slime", SpawnPointIdx: 0}, {TypeName: "Tough Slime", SpawnPointIdx: 1}, {TypeName: "Spore Mushroom", SpawnPointIdx: 2}, {TypeName: "Spore Mushroom", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Goo Spitter", SpawnPointIdx: 4}, {TypeName: "Tough Slime", SpawnPointIdx: 0}, {TypeName: "Tough Slime", SpawnPointIdx: 1}, {TypeName: "Small Slime", SpawnPointIdx: 2}, {TypeName: "Small Slime", SpawnPointIdx: 3}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Elder Spore Mushroom", SpawnPointIdx: 4}, {TypeName: "Goo Spitter", SpawnPointIdx: 2}, {TypeName: "Goo Spitter", SpawnPointIdx: 3}, {TypeName: "Tough Slime", SpawnPointIdx: 0}, {TypeName: "Tough Slime", SpawnPointIdx: 1}}, IsBossWave: true},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Centaur", SpawnPointIdx: 2}, {TypeName: "Centaur", SpawnPointIdx: 3}, {TypeName: "Druid", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Dire Wolf", SpawnPointIdx: 0}, {TypeName: "Faun", SpawnPointIdx: 2}, {TypeName: "Faun", SpawnPointIdx: 3}, {TypeName: "Druid", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Centaur", SpawnPointIdx: 2}, {TypeName: "Centaur", SpawnPointIdx: 3}, {TypeName: "Naga", SpawnPointIdx: 1}, {TypeName: "Druid", SpawnPointIdx: 4}}, IsBossWave: false},
		{EnemiesToSpawn: []EnemySpawnInfo{{TypeName: "Tauren", SpawnPointIdx: 4}, {TypeName: "Druid", SpawnPointIdx: 2}, {TypeName: "Druid", SpawnPointIdx: 3}, {TypeName: "Dire Wolf", SpawnPointIdx: 0}, {TypeName: "Dire Wolf", SpawnPointIdx: 1}}, IsBossWave: true},
	}

	var err error
	rogueSheetPath := "assets/rogues.png"
	monsterSheetPath := "assets/monsters.png"
	tilePath := "assets/tiles.png"

	g.rogueSheet, err = loadImage(rogueSheetPath)
	if err != nil {
		log.Printf("Error loading rogue sheet: %v. Using fallback.", err)
		g.rogueSheet = ebiten.NewImage(spriteSize*sheetWidthInSprites, spriteSize*10)
		g.rogueSheet.Fill(color.NRGBA{R: 50, G: 50, B: 50, A: 255})
	}
	g.monsterSheet, err = loadImage(monsterSheetPath)
	if err != nil {
		log.Printf("Error loading monster sheet: %v. Using fallback.", err)
		g.monsterSheet = ebiten.NewImage(spriteSize*sheetWidthInSprites, spriteSize*15)
		g.monsterSheet.Fill(color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	}
	tileSheet, err := loadImage(tilePath)
	if err != nil {
		log.Printf("Error loading tile sheet: %v. Creating fallback tile.", err)
		g.TileImage = ebiten.NewImage(tileSize, tileSize)
		vector.DrawFilledRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), color.NRGBA{R: 50, G: 50, B: 50, A: 255}, false)
		vector.StrokeRect(g.TileImage, 0, 0, float32(tileSize), float32(tileSize), 1, color.NRGBA{R: 80, G: 80, B: 80, A: 255}, false)
	} else {
		floorSprite := getSpriteFromSheet(tileSheet, 0, 0)
		g.TileImage = floorSprite
	}

	g.RangeOverlayTile = ebiten.NewImage(tileSize, tileSize)
	overlayColor := color.NRGBA{R: 0, G: 100, B: 200, A: 80}
	vector.DrawFilledRect(g.RangeOverlayTile, 0, 0, float32(tileSize), float32(tileSize), overlayColor, false)

	classDef, classExists := ClassDefinitions[playerClassName]
	if !classExists {
		log.Fatalf("FATAL: Player class '%s' not found in ClassDefinitions!", playerClassName)
	}

	playerSprite := getSpriteFromSheet(g.rogueSheet, classDef.SpriteSheetX, classDef.SpriteSheetY)
	playerStr, playerDex, playerCon := 10, 10, 10
	playerInt, playerWis, playerCha := 10, 10, 10
	startLevel := 1
	knownSpells := make([]string, 0)
	knownCantrips := make([]string, 0)
	maxSlotsL1 := 0

	if playerClassName == "Fighter" {
		playerStr, playerDex, playerCon = 16, 14, 14
		playerInt, playerWis, playerCha = 8, 8, 10
	} else if playerClassName == "Mage" {
		playerStr, playerDex, playerCon = 8, 14, 14
		playerInt, playerWis, playerCha = 16, 10, 8
		knownCantrips = append(knownCantrips, "firebolt")
		knownSpells = append(knownSpells, "magic_missile", "shield")
		maxSlotsL1 = 2
	}

	playerConMod := getModifier(playerCon)
	playerDexMod := getModifier(playerDex)
	playerMaxHP := classDef.HitDieSize + playerConMod
	playerAC := 10 + playerDexMod

	g.Player = &Player{
		Entity: Entity{
			X: mapWidth / 2, Y: mapHeight / 2, Width: 1, Height: 1,
			HP: playerMaxHP, MaxHP: playerMaxHP, AC: playerAC,
			Strength: playerStr, Dexterity: playerDex, Constitution: playerCon,
			Intelligence: playerInt, Wisdom: playerWis, Charisma: playerCha,
			Sprite: playerSprite, Name: "Player", CurrentAlpha: 1.0,
		},
		Level:                startLevel,
		Class:                playerClassName,
		ProficiencyBonus:     calculateProficiencyBonus(startLevel),
		MaxMovementPoints:    playerBaseMovement,
		ClassResources:       make(map[string]int),
		MaxHitDice:           startLevel,
		HitDice:              startLevel,
		MaxSpellSlotsL1:      maxSlotsL1,
		SpellSlotsL1:         maxSlotsL1,
		KnownSpells:          knownSpells,
		KnownCantrips:        knownCantrips,
		UsedReaction:         false,
		UsedArcaneRecovery:   false,
		ACBonusUntilNextTurn: 0,
		LastSpellCastID:      "",
		CombatStyle:          "",
		CombatTechnique:      "",
		IsSlowed:             false,
		SlowDuration:         0,
		NextElementType:      "Fire",
	}

	g.initializePlayerResources()

	g.CurrentTurn = PlayerTurn
	g.SpawnNextWave()
	g.startPlayerTurn()
	g.CurrentGameState = StatePlaying
}

func (g *Game) initializePlayerResources() {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}
	for actionID, classAction := range classDef.ClassActions {
		if classAction.ResourceType == ResourceClassFeature {
			if classAction.RequiredLevel <= g.Player.Level {
				if classAction.UsesPerRest > 0 {
					g.Player.ClassResources[actionID] = classAction.UsesPerRest
				} else {
					g.Player.ClassResources[actionID] = 0
				}
			} else {
				g.Player.ClassResources[actionID] = 0
			}
		}
	}
	if g.Player.Class == "Mage" {
		g.Player.MaxSpellSlotsL1 = 0
		if g.Player.Level >= 1 {
			g.Player.MaxSpellSlotsL1 = 2
		}
		if g.Player.Level >= 2 {
			g.Player.MaxSpellSlotsL1 = 3
		}

	}
}

func (g *Game) refreshLevelUpResources() {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}

	refreshedSomething := false
	logMsg := "Level Up! Resources refreshed: "

	for actionID, classAction := range classDef.ClassActions {
		if classAction.RefreshesOn == RestTypeNever {
			if g.Player.Level >= classAction.RequiredLevel {
				currentUses := 0
				if val, ok := g.Player.ClassResources[actionID]; ok {
					currentUses = val
				}
				maxUses := classAction.UsesPerRest
				if currentUses < maxUses || g.Player.Level == classAction.RequiredLevel {
					g.Player.ClassResources[actionID] = maxUses
					logMsg += fmt.Sprintf("%s, ", classAction.Name)
					refreshedSomething = true
				}
			}
		}
	}

	if g.Player.Class == "Mage" {
		oldMaxSlots := g.Player.MaxSpellSlotsL1
		newMaxSlots := 0
		if g.Player.Level >= 1 {
			newMaxSlots = 2
		}
		if g.Player.Level >= 3 {
			newMaxSlots = 3
		}

		if newMaxSlots > oldMaxSlots {
			g.Player.MaxSpellSlotsL1 = newMaxSlots
			g.Player.SpellSlotsL1 = newMaxSlots
			logMsg += "Spell Slots, "
			refreshedSomething = true
		} else {
			g.Player.SpellSlotsL1 = g.Player.MaxSpellSlotsL1
		}
	}

	if refreshedSomething {
		if len(logMsg) > 2 {
			logMsg = logMsg[:len(logMsg)-2] + "."
		}
		g.addCombatLog(logMsg)
	}
}

func (g *Game) resetPlayerResources(restType ResourceRestType) {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return
	}
	logMsg := ""
	if restType == RestTypeShort {
		logMsg = "Short Rest complete. Resources refreshed: "
	} else if restType == RestTypeLong {
		logMsg = "Long Rest complete. Resources refreshed: "
	}

	refreshedSomething := false
	for actionID, classAction := range classDef.ClassActions {
		if classAction.RefreshesOn == RestTypeNever {
			continue
		}

		shouldReset := false
		if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
			if classAction.RefreshesOn == restType {
				shouldReset = true
			}
			if restType == RestTypeLong && classAction.RefreshesOn == RestTypeShort {
				shouldReset = true
			}
		}

		if shouldReset && g.Player.Level >= classAction.RequiredLevel {
			currentUses := g.Player.ClassResources[actionID]
			maxUses := classAction.UsesPerRest

			if currentUses < maxUses {
				g.Player.ClassResources[actionID] = maxUses
				logMsg += fmt.Sprintf("%s, ", classAction.Name)
				refreshedSomething = true
			}
		}
	}

	if g.Player.Class == "Mage" && restType == RestTypeLong {
		if g.Player.SpellSlotsL1 < g.Player.MaxSpellSlotsL1 {
			g.Player.SpellSlotsL1 = g.Player.MaxSpellSlotsL1
			logMsg += "Spell Slots, "
			refreshedSomething = true
		}
		if g.Player.UsedArcaneRecovery {
			g.Player.UsedArcaneRecovery = false
			logMsg += "Arcane Recovery Use, "
			refreshedSomething = true
		}
	}

	if refreshedSomething {
		if len(logMsg) > 2 {
			logMsg = logMsg[:len(logMsg)-2] + "."
		}
		g.addCombatLog(logMsg)
	}

	if restType == RestTypeLong {
		diceToRecover := max(1, g.Player.MaxHitDice/2)
		initialDice := g.Player.HitDice
		g.Player.HitDice = min(g.Player.MaxHitDice, g.Player.HitDice+diceToRecover)
		recoveredCount := g.Player.HitDice - initialDice
		if recoveredCount > 0 {
			g.addCombatLog(fmt.Sprintf("Recovered %d Hit Dice.", recoveredCount))
		}
	}
}

func (g *Game) shortRest() {
	g.addCombatLog("Player takes a Short Rest...")
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		g.addCombatLog("Cannot determine class for Hit Dice.")
		return
	}

	if g.Player.HitDice > 0 {
		conMod := getModifier(g.Player.Constitution)
		healRoll := rand.Intn(classDef.HitDieSize) + 1
		healAmount := max(1, healRoll+conMod)
		actualHeal := min(healAmount, g.Player.MaxHP-g.Player.HP)
		actualHeal = max(0, actualHeal)

		if actualHeal > 0 {
			g.Player.HP += actualHeal
			g.addCombatLog(fmt.Sprintf("Spent 1 Hit Die (d%d), recovered %d HP (Rolled %d%+d).", classDef.HitDieSize, actualHeal, healRoll, conMod))
		} else {
			g.addCombatLog("Spent 1 Hit Die, but already at full HP.")
		}
		g.Player.HitDice--
	} else {
		g.addCombatLog("No Hit Dice left to spend.")
	}

	if g.Player.Class == "Mage" && g.Player.Level >= 2 && !g.Player.UsedArcaneRecovery {
		slotsToRecoverFloat := math.Ceil(float64(g.Player.Level) / 2.0)
		slotsToRecover := int(slotsToRecoverFloat)

		if slotsToRecover > 0 && g.Player.SpellSlotsL1 < g.Player.MaxSpellSlotsL1 {
			recovered := min(slotsToRecover, g.Player.MaxSpellSlotsL1-g.Player.SpellSlotsL1)
			if recovered > 0 {
				g.Player.SpellSlotsL1 += recovered
				g.Player.UsedArcaneRecovery = true
				g.addCombatLog(fmt.Sprintf("Used Arcane Recovery! Recovered %d L1 spell slot(s).", recovered))
			} else {
				g.addCombatLog("Arcane Recovery: Already at max spell slots.")
			}
		} else if slotsToRecover > 0 {
			g.addCombatLog("Arcane Recovery: Already at max spell slots.")
		}
	}
}

func (g *Game) longRest() {
	g.addCombatLog("Player takes a Long Rest...")
	g.Player.HP = g.Player.MaxHP
	g.resetPlayerResources(RestTypeLong)
	g.addCombatLog("HP fully restored.")
}

func (g *Game) levelUpPlayer() {
	if g.Player.Level >= maxLevel {
		g.addCombatLog("Already at max level!")
		return
	}

	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		log.Printf("Error: Cannot find class definition %s for level up.", g.Player.Class)
		return
	}

	g.Player.Level++
	g.addCombatLog(fmt.Sprintf("LEVEL UP! Reached Level %d!", g.Player.Level))

	conMod := getModifier(g.Player.Constitution)
	hpRoll := rand.Intn(classDef.HitDieSize) + 1
	hpIncrease := max(1, hpRoll+conMod)
	g.Player.MaxHP += hpIncrease
	g.Player.HP += hpIncrease
	g.addCombatLog(fmt.Sprintf("Max HP increased by %d (Rolled %d%+d).", hpIncrease, hpRoll, conMod))

	g.Player.MaxHitDice++
	g.Player.HitDice++
	g.addCombatLog("Gained 1 Hit Die.")

	newProfBonus := calculateProficiencyBonus(g.Player.Level)
	if newProfBonus > g.Player.ProficiencyBonus {
		g.Player.ProficiencyBonus = newProfBonus
		g.addCombatLog(fmt.Sprintf("Proficiency Bonus increased to +%d.", newProfBonus))
	}

	g.refreshLevelUpResources()
}

func (g *Game) startPlayerTurn() {
	currentACBonus := 0
	currentMoveBonus := 0

	if g.Player.IsSlowed {
		g.Player.SlowDuration--
		if g.Player.SlowDuration <= 0 {
			g.Player.IsSlowed = false
			g.addCombatLog("Slow effect wears off.")
		}
	}
	if g.Player.MagicArmorDuration > 0 {
		g.Player.MagicArmorDuration--
		if g.Player.MagicArmorDuration <= 0 {
			g.addCombatLog("Magic Armor fades.")
		} else {
			currentACBonus += 2
		}
	}
	if g.Player.ExpeditiousRetreatDuration > 0 {
		g.Player.ExpeditiousRetreatDuration--
		if g.Player.ExpeditiousRetreatDuration <= 0 {
			g.addCombatLog("Expeditious Retreat ends.")
		} else {
			currentMoveBonus += 3
		}
	}
	if g.Player.FeatherFallDuration > 0 {
		g.Player.FeatherFallDuration--
		if g.Player.FeatherFallDuration <= 0 {
			g.addCombatLog("Feather Fall ends.")
		}
	}

	g.CurrentTurn = PlayerTurn
	g.Player.MovementPoints = g.Player.MaxMovementPoints + currentMoveBonus
	if g.Player.IsSlowed {
		g.Player.MovementPoints = max(1, g.Player.MovementPoints/2)
	}

	g.Player.ActionTaken = false
	g.Player.BonusActionTaken = false
	g.Player.IsDisengaging = false
	g.Player.UsedReaction = false
	g.Player.ACBonusUntilNextTurn = currentACBonus
	g.InputMode = InputModeMap
	g.primedActionID = ""
}

func (g *Game) setGameOver(victory bool) {
	g.CurrentTurn = GameOver
	g.CurrentGameState = StateGameOverScreen
	g.isVictory = victory
}

func (g *Game) handleWaveCompletion() {
	currentWaveDef := g.WaveDefinitions[g.CurrentWaveIndex]
	isFinalWave := g.CurrentWaveIndex+1 >= len(g.WaveDefinitions)

	if currentWaveDef.IsBossWave && g.Player.Level < maxLevel {
		g.addCombatLog("Final Boss Defeated!")
		g.levelUpPlayer()
		if g.Player.Class == "Mage" && g.Player.Level == 2 && len(g.Player.KnownSpells) == 0 {
			g.addCombatLog("Choose a new Level 1 spell!")
			g.pendingLevelUpSpellChoice = 0
		}
		g.InputMode = InputModeLevelUp
	} else if !isFinalWave {
		g.addCombatLog(fmt.Sprintf("Wave Cleared! Spend 1 Hit Die (of %d) to heal? [Y/N]", g.Player.HitDice))
		g.InputMode = InputModeRestPrompt
	} else {
		g.addCombatLog("All challenges overcome! VICTORY!")
		g.longRest()
		g.setGameOver(true)
	}
}

func (g *Game) startNextWave() {
	g.SpawnNextWave()
	if len(g.Enemies) > 0 {
		g.startPlayerTurn()
	} else {
		g.addCombatLog("Error spawning next wave or wave empty.")
		g.setGameOver(false)
	}
	g.InputMode = InputModeMap
}

func (g *Game) endPlayerTurn() {
	g.cleanupDeadEnemies()

	if len(g.Enemies) == 0 {
		g.handleWaveCompletion()
	} else {
		g.startEnemyTurn()
	}

	g.primedActionID = ""
	if g.CurrentGameState == StatePlaying && g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
		g.InputMode = InputModeMap
	}
}

func (g *Game) startEnemyTurn() {
	g.CurrentTurn = EnemyTurn
}

func (g *Game) endEnemyTurn() {
	g.cleanupDeadEnemies()

	if g.Player.IsDying && g.Player.CurrentAlpha <= 0 {
		if g.CurrentGameState != StateGameOverScreen {
			g.addCombatLog("Player has faded away! Game Over.")
			g.setGameOver(false)
		}
		return
	}

	if g.CurrentTurn == GameOver || g.CurrentGameState == StateGameOverScreen {
		return
	}
	if len(g.Enemies) == 0 {
		if g.Player.IsDying {
			if g.CurrentGameState != StateGameOverScreen {
				g.setGameOver(false)
			}
			return
		}
		g.handleWaveCompletion()
	} else {
		if !g.Player.IsDying {
			g.startPlayerTurn()
		} else {
			if g.CurrentGameState != StateGameOverScreen {
				g.setGameOver(false)
			}
		}
	}
}

func (g *Game) SpawnNextWave() {
	g.CurrentWaveIndex++

	if g.CurrentWaveIndex >= len(g.WaveDefinitions) {
		log.Printf("Attempted to spawn wave index %d, but only %d waves are defined.", g.CurrentWaveIndex, len(g.WaveDefinitions))
		g.addCombatLog("No more waves defined.")
		if g.CurrentGameState != StateGameOverScreen {
			g.setGameOver(true)
		}
		return
	}

	waveDef := g.WaveDefinitions[g.CurrentWaveIndex]
	g.addCombatLog(fmt.Sprintf("--- Starting Wave %d ---", g.CurrentWaveIndex+1))
	g.Enemies = make([]*Enemy, 0)

	spawnPoints := []image.Point{
		{X: 2, Y: 2},
		{X: mapWidth - 3, Y: mapHeight - 3},
		{X: 2, Y: mapHeight - 3},
		{X: mapWidth - 3, Y: 2},
		{X: mapWidth / 2, Y: 1},
	}

	for _, spawnInfo := range waveDef.EnemiesToSpawn {
		enemyDef, exists := EnemyDefinitions[spawnInfo.TypeName]
		if !exists {
			log.Printf("Warning: Enemy type '%s' not found in definitions.", spawnInfo.TypeName)
			continue
		}

		spawnX, spawnY := -1, -1
		if spawnInfo.SpawnPointIdx >= 0 && spawnInfo.SpawnPointIdx < len(spawnPoints) {
			spawnPoint := spawnPoints[spawnInfo.SpawnPointIdx]
			spawnX = spawnPoint.X
			spawnY = spawnPoint.Y
			if enemyDef.Width > 1 {
				spawnX -= (enemyDef.Width - 1) / 2
			}
			if enemyDef.Height > 1 {
				spawnY -= (enemyDef.Height - 1) / 2
			}
			if spawnX < 0 {
				spawnX = 0
			}
			if spawnY < 0 {
				spawnY = 0
			}
			if spawnX+enemyDef.Width > mapWidth {
				spawnX = mapWidth - enemyDef.Width
			}
			if spawnY+enemyDef.Height > mapHeight {
				spawnY = mapHeight - enemyDef.Height
			}

		} else {
			log.Printf("Warning: Invalid spawn point index %d for enemy '%s'. Skipping.", spawnInfo.SpawnPointIdx, enemyDef.Name)
			continue
		}

		if g.isTileFullyBlocked(spawnX, spawnY, enemyDef.Width, enemyDef.Height, -1) {
			log.Printf("Warning: Spawn point %d (%d,%d) for %s is blocked. Trying fallback.", spawnInfo.SpawnPointIdx, spawnX, spawnY, enemyDef.Name)
			foundAlt := false
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					altX, altY := spawnX+dx, spawnY+dy
					if altX >= 0 && altX+enemyDef.Width <= mapWidth && altY >= 0 && altY+enemyDef.Height <= mapHeight {
						if !g.isTileFullyBlocked(altX, altY, enemyDef.Width, enemyDef.Height, -1) {
							spawnX, spawnY = altX, altY
							foundAlt = true
							break
						}
					}
				}
				if foundAlt {
					break
				}
			}
			if !foundAlt {
				log.Printf("Error: Could not find alternative spawn location for %s near (%d,%d). Skipping.", enemyDef.Name, spawnX, spawnY)
				continue
			}
		}

		sprite := getSpriteFromSheet(g.monsterSheet, enemyDef.SpriteSheetX, enemyDef.SpriteSheetY)
		g.spawnEnemyFromDef(spawnX, spawnY, enemyDef, sprite)
	}
}

func (g *Game) spawnEnemyFromDef(x, y int, def EnemyDefinition, sprite *ebiten.Image) {
	conMod := getModifier(def.Con)
	maxHp := def.BaseHP + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, Width: def.Width, Height: def.Height,
			HP: maxHp, MaxHP: maxHp, AC: def.AC,
			Strength: def.Str, Dexterity: def.Dex, Constitution: def.Con,
			Intelligence: def.Int, Wisdom: def.Wis, Charisma: def.Cha,
			Sprite: sprite, Name: def.Name,
			CurrentAlpha: 1.0,
		},
		MaxMovementPoints: def.Move,
		AttackType:        def.AttackType,
		MaxRange:          def.MaxRange,
	}
	g.Enemies = append(g.Enemies, enemy)
	g.addCombatLog(fmt.Sprintf("%s appears!", enemy.Name))
}

func (g *Game) spawnEnemy(x, y int, name string, baseHp, ac, str, dex, con, intel, wis, cha, move, w, h int, sprite *ebiten.Image) {
	conMod := getModifier(con)
	maxHp := baseHp + conMod
	if maxHp < 1 {
		maxHp = 1
	}
	enemy := &Enemy{
		Entity: Entity{
			X: x, Y: y, Width: w, Height: h,
			HP: maxHp, MaxHP: maxHp, AC: ac,
			Strength: str, Dexterity: dex, Constitution: con,
			Intelligence: intel, Wisdom: wis, Charisma: cha,
			Sprite: sprite, Name: name,
			CurrentAlpha: 1.0,
		},
		MaxMovementPoints: move,
	}
	g.Enemies = append(g.Enemies, enemy)
}

func (g *Game) addCombatLog(msg string) {
	if g.CombatLog == nil {
		g.CombatLog = make([]string, 0, combatLogLength)
	}
	g.CombatLog = append(g.CombatLog, msg)
	if len(g.CombatLog) > combatLogLength {
		g.CombatLog = g.CombatLog[len(g.CombatLog)-combatLogLength:]
	}
}

func (g *Game) isTileBlocked(checkX, checkY, movingEnemyIndex int) bool {
	if g.Player != nil && !g.Player.IsDying && g.Player.HP > 0 && checkX >= g.Player.X && checkX < g.Player.X+g.Player.Width &&
		checkY >= g.Player.Y && checkY < g.Player.Y+g.Player.Height {
		return true
	}
	for i, enemy := range g.Enemies {
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}
		if i == movingEnemyIndex {
			continue
		}
		if checkX >= enemy.X && checkX < enemy.X+enemy.Width &&
			checkY >= enemy.Y && checkY < enemy.Y+enemy.Height {
			return true
		}
	}
	return false
}

func (g *Game) isTileFullyBlocked(checkX, checkY, entityWidth, entityHeight, movingEnemyIndex int) bool {
	for w := 0; w < entityWidth; w++ {
		for h := 0; h < entityHeight; h++ {
			tileX, tileY := checkX+w, checkY+h
			if tileX < 0 || tileX >= mapWidth || tileY < 0 || tileY >= mapHeight {
				return true
			}
			if g.isTileBlocked(tileX, tileY, movingEnemyIndex) {
				return true
			}
		}
	}
	return false
}

func (g *Game) getEnemyAt(x, y int) *Enemy {
	for i, enemy := range g.Enemies {
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}
		if x >= enemy.X && x < enemy.X+enemy.Width &&
			y >= enemy.Y && y < enemy.Y+enemy.Height {
			return g.Enemies[i]
		}
	}
	return nil
}

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

func (g *Game) resolveAttack(attacker *Entity, defender *Entity, attackerProfBonus int, attackType string) (bool, bool) {
	if attacker == nil || defender == nil || attacker.HP <= 0 || defender.HP <= 0 || defender.IsDying {
		return false, false
	}

	isPlayerAttacking := g.Player != nil && attacker == &g.Player.Entity
	isPlayerDefending := g.Player != nil && defender == &g.Player.Entity

	var attackAbilityMod int
	var abilityName string
	var enemyDef *EnemyDefinition

	attackPenalty := 0
	damageMultiplier := 1.0

	if isPlayerDefending && g.Player.FeatherFallDuration > 0 {
		if rand.Intn(2) == 0 {
			g.addCombatLog("Player nimbly dodges (Feather Fall)!")
			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
				Text: "Dodge!", X: float64(g.MapOffsetX + defender.X*tileSize + (defender.Width*tileSize)/2), Y: float64(g.MapOffsetY+defender.Y*tileSize) - 15, Life: 30, MaxLife: 30, Color: colorWhite, VelocityY: -0.5,
			})
			return false, false
		}
	}

	if isPlayerAttacking {
		if g.Player.CombatTechnique == "Power Attack" {
			attackPenalty = -2
			damageMultiplier = 1.5
		}
		switch attackType {
		case "ranged":
			attackAbilityMod = getModifier(attacker.Dexterity)
			abilityName = "DEX"
			if g.Player.CombatStyle == "Ranger" {
				attackerProfBonus += 2
			}
		default:
			attackAbilityMod = getModifier(attacker.Strength)
			abilityName = "STR"
		}
	} else {
		foundDef := false
		var tempDef EnemyDefinition
		for _, def := range EnemyDefinitions {
			if def.Name == attacker.Name {
				tempDef = def
				enemyDef = &tempDef
				foundDef = true
				break
			}
		}
		if foundDef {
			switch enemyDef.AttackAbilityMod {
			case "STR":
				attackAbilityMod = getModifier(attacker.Strength)
				abilityName = "STR"
			case "DEX":
				attackAbilityMod = getModifier(attacker.Dexterity)
				abilityName = "DEX"
			case "INT":
				attackAbilityMod = getModifier(attacker.Intelligence)
				abilityName = "INT"
			case "WIS":
				attackAbilityMod = getModifier(attacker.Wisdom)
				abilityName = "WIS"
			case "CHA":
				attackAbilityMod = getModifier(attacker.Charisma)
				abilityName = "CHA"
			default:
				if enemyDef.AttackType == "ranged" {
					attackAbilityMod = getModifier(attacker.Dexterity)
					abilityName = "DEX"
				} else {
					attackAbilityMod = getModifier(attacker.Strength)
					abilityName = "STR"
				}
			}
		} else {
			if attackType == "ranged" {
				attackAbilityMod = getModifier(attacker.Dexterity)
				abilityName = "DEX"
			} else {
				attackAbilityMod = getModifier(attacker.Strength)
				abilityName = "STR"
			}
		}
	}

	roll := rand.Intn(20) + 1
	effectiveAC := defender.AC
	if isPlayerDefending {
		effectiveAC += g.Player.ACBonusUntilNextTurn
	}

	attackRoll := roll + attackerProfBonus + attackAbilityMod + attackPenalty
	isCrit := roll == 20
	isFumble := roll == 1
	hit := !isFumble && (isCrit || attackRoll >= effectiveAC)

	modString := fmt.Sprintf("%+d", attackAbilityMod)
	profString := ""
	if attackerProfBonus != 0 {
		profString = fmt.Sprintf("+%d", attackerProfBonus)
	}
	penaltyString := ""
	if attackPenalty != 0 {
		penaltyString = fmt.Sprintf("%d", attackPenalty)
	}
	rollString := fmt.Sprintf("%d%s%s%s(%s)=%d", roll, profString, modString, penaltyString, abilityName, attackRoll)
	attackVerb := "attacks"
	if attackType == "ranged" {
		attackVerb = "shoots"
		if !isPlayerAttacking && attacker.Name == "Orc Shaman" {
			attackVerb = "casts Bone Chill at"
		}
	}
	logMsg := fmt.Sprintf("%s %s %s(AC%d). Roll: %s.", attacker.Name, attackVerb, defender.Name, effectiveAC, rollString)
	if isPlayerDefending && g.Player.ACBonusUntilNextTurn > 0 {
		logMsg = fmt.Sprintf("%s %s %s(AC%d+%d Buff=%d). Roll: %s.", attacker.Name, attackVerb, defender.Name, defender.AC, g.Player.ACBonusUntilNextTurn, effectiveAC, rollString)
	}

	textSpawnX := float64(g.MapOffsetX + defender.X*tileSize + (defender.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + defender.Y*tileSize)
	textLifetime := 60
	killed := false
	attackLanded := false

	if isPlayerDefending && hit && !g.Player.UsedReaction {
		playerKnowsShield := false
		for _, known := range g.Player.KnownSpells {
			if known == "shield" {
				playerKnowsShield = true
				break
			}
		}
		if !playerKnowsShield && g.Player.Class == "Mage" {
			playerKnowsShield = true
		}
		if playerKnowsShield && g.Player.SpellSlotsL1 > 0 {
			g.addCombatLog("Hit detected! Use Shield reaction? [Y/N]")
			g.reactionPending = true
			g.InputMode = InputModeReactionPrompt
			g.addCombatLog(logMsg + " (Pending Reaction...)")
			return false, false
		}
	}

	if isFumble {
		logMsg += " FUMBLE! Miss!"
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "FUMBLE!", X: textSpawnX, Y: textSpawnY - 15, Life: textLifetime / 2, MaxLife: textLifetime / 2, Color: color.NRGBA{R: 150, G: 0, B: 0, A: 255}, VelocityY: -0.5})
	} else if hit {
		attackLanded = true
		attacker.AttackBumpTimer = attackBumpDuration
		var damage int
		damageRoll1, damageRoll2 := 0, 0
		damageLog := ""
		baseDamage, styleDamageBonus := 0, 0
		if isCrit {
			logMsg += " CRITICAL!"
		}

		if isPlayerAttacking { // Player DMG Calculation
			switch attackType {
			case "ranged":
				damageRoll1 = rand.Intn(6) + 1
				if isCrit {
					damageRoll2 = rand.Intn(6) + 1
				}
				baseDamage = damageRoll1 + damageRoll2 + attackAbilityMod
				if isCrit {
					damageLog = fmt.Sprintf(" (2d6[%d,%d]%+d)", damageRoll1, damageRoll2, attackAbilityMod)
				} else {
					damageLog = fmt.Sprintf(" (1d6[%d]%+d)", damageRoll1, attackAbilityMod)
				}
			default:
				damageRoll1 = rand.Intn(8) + 1
				if isCrit {
					damageRoll2 = rand.Intn(8) + 1
				}
				if g.Player.CombatStyle == "Gladiator" {
					styleDamageBonus = 2
				}
				baseDamage = damageRoll1 + damageRoll2 + attackAbilityMod + styleDamageBonus
				if isCrit {
					damageLog = fmt.Sprintf(" (2d8[%d,%d]%+d", damageRoll1, damageRoll2, attackAbilityMod)
				} else {
					damageLog = fmt.Sprintf(" (1d8[%d]%+d", damageRoll1, attackAbilityMod)
				}
				if styleDamageBonus > 0 {
					damageLog += fmt.Sprintf("+%d Style", styleDamageBonus)
				}
				damageLog += ")"
			}
		} else { // Enemy DMG Calculation
			if enemyDef != nil && enemyDef.AttackDiceNum > 0 && enemyDef.AttackDiceSize > 0 {
				totalDiceRoll := 0
				diceRolls := []int{}
				numDiceToRoll := enemyDef.AttackDiceNum
				if isCrit {
					numDiceToRoll *= 2
				}
				for i := 0; i < numDiceToRoll; i++ {
					roll := rand.Intn(enemyDef.AttackDiceSize) + 1
					totalDiceRoll += roll
					diceRolls = append(diceRolls, roll)
				}
				baseDamage = totalDiceRoll + attackAbilityMod
				rollsStr := ""
				for i, r := range diceRolls {
					rollsStr += fmt.Sprintf("%d", r)
					if i < len(diceRolls)-1 {
						rollsStr += ","
					}
				}
				damageLog = fmt.Sprintf(" (%dd%d[%s]%+d)", numDiceToRoll, enemyDef.AttackDiceSize, rollsStr, attackAbilityMod)
			} else {
				baseDamage = max(1, attackAbilityMod)
				if isCrit {
					baseDamage = max(1, baseDamage*2)
				}
				damageLog = fmt.Sprintf(" (%+d)", baseDamage)
			}
		}

		damage = int(float64(max(1, baseDamage)) * damageMultiplier)
		actualDamage := min(damage, defender.HP)
		defender.HP -= actualDamage
		logMsg += fmt.Sprintf(" Hit! Deals %d%s dmg.", actualDamage, damageLog)
		if damageMultiplier != 1.0 {
			logMsg += fmt.Sprintf(" (%.1fx Power Attack)", damageMultiplier)
		}

		if isPlayerDefending && !isPlayerAttacking && attacker.Name == "Orc Shaman" {
			g.Player.IsSlowed = true
			g.Player.SlowDuration = 2
			logMsg += " Player is Slowed!"
			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Slowed!", X: textSpawnX, Y: textSpawnY + 15, Life: textLifetime, MaxLife: textLifetime, Color: visualEffectColors["Cold"], VelocityY: -0.5})
		}

		ftColor := visualEffectColors["Default"]
		if !isPlayerAttacking && attacker.Name == "Orc Shaman" {
			ftColor = visualEffectColors["Cold"]
		}
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: textLifetime, MaxLife: textLifetime, Color: ftColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: textLifetime / 2, MaxLife: textLifetime / 2, Color: hitTextColor, VelocityY: -0.5})

		if defender.HP <= 0 {
			logMsg += fmt.Sprintf(" %s dies!", defender.Name)
			defender.IsDying = true
			killed = true
			if isPlayerDefending {
				g.setGameOver(false)
			}
		}
	} else if !isFumble {
		logMsg += " Miss!"
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: textLifetime / 2, MaxLife: textLifetime / 2, Color: colorGray, VelocityY: -0.5})
	}
	g.addCombatLog(logMsg)
	return killed, attackLanded
}

func (g *Game) executeAction(actionDef *ActionDefinition, targetX, targetY int) bool {
	if actionDef == nil {
		g.addCombatLog("Error: Tried to execute nil action.")
		return false
	}

	classDef := ClassDefinitions[g.Player.Class]
	var classAction *ClassAction
	if classDef != nil {
		classAction = classDef.ClassActions[actionDef.ID]
	}

	if actionDef.ResourceType == ResourceSpellSlotL1 {
		if g.Player.SpellSlotsL1 < actionDef.ResourceCost {
			g.addCombatLog(fmt.Sprintf("Not enough L1 spell slots for %s (%d needed).", actionDef.Name, actionDef.ResourceCost))
			return false
		}
	} else if actionDef.ResourceType == ResourceClassFeature {
		if classAction != nil && classAction.UsesPerRest > 0 {
			if _, ok := g.Player.ClassResources[actionDef.ID]; !ok {
				g.addCombatLog(fmt.Sprintf("Resource %s not initialized for player.", actionDef.ID))
				return false
			}
			if g.Player.ClassResources[actionDef.ID] < classAction.ResourceCost {
				g.addCombatLog(fmt.Sprintf("Not enough uses left for %s.", actionDef.Name))
				return false
			}
		}
	}

	switch actionDef.ActionType {
	case ActionTypeStandard:
		if g.Player.ActionTaken {
			g.addCombatLog(fmt.Sprintf("Cannot use %s: Action already used.", actionDef.Name))
			return false
		}
	case ActionTypeBonus:
		if g.Player.BonusActionTaken {
			g.addCombatLog(fmt.Sprintf("Cannot use %s: Bonus Action already used.", actionDef.Name))
			return false
		}
	case ActionTypeReaction:
		if g.Player.UsedReaction {
			g.addCombatLog(fmt.Sprintf("Cannot use %s: Reaction already used.", actionDef.Name))
			return false
		}
	case ActionTypeFree:

	}

	success := actionDef.Execute(g, targetX, targetY)

	if success {
		g.lastExecutedActionID = actionDef.ID

		if actionDef.ResourceType == ResourceSpellSlotL1 {
			g.Player.SpellSlotsL1 -= actionDef.ResourceCost
		} else if actionDef.ResourceType == ResourceClassFeature {
			if classAction != nil && classAction.UsesPerRest > 0 {
				g.Player.ClassResources[actionDef.ID] -= classAction.ResourceCost
			}
		}

		switch actionDef.ActionType {
		case ActionTypeStandard:
			g.Player.ActionTaken = true
		case ActionTypeBonus:
			g.Player.BonusActionTaken = true
		case ActionTypeReaction:
			g.Player.UsedReaction = true
		case ActionTypeFree:

		}

		if actionDef.ID == "action_surge" {
			g.Player.ActionTaken = false
		}

		if actionDef.ID == "wait" {
			g.endPlayerTurn()
		}
	}

	return success
}

func executeMeleeAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil && isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "melee")
		if g.reactionPending {
			return false
		}
		return true
	}
	g.addCombatLog("Invalid target for melee attack (or not adjacent).")
	return false
}

func executeRangedAttack(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy != nil {
		dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
		if dist <= playerRangedRange {
			g.resolveAttack(&g.Player.Entity, &targetEnemy.Entity, g.Player.ProficiencyBonus, "ranged")
			if g.reactionPending {
				return false
			}
			return true
		}
		g.addCombatLog(fmt.Sprintf("Target %s out of range (%d > %d).", targetEnemy.Name, dist, playerRangedRange))
		return false
	}
	g.addCombatLog("No valid target selected at cursor for ranged attack.")
	return false
}

func executeDash(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Dash!")
	g.Player.MovementPoints += g.Player.MaxMovementPoints
	return true
}

func executeDisengage(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Disengage!")
	g.Player.IsDisengaging = true
	return true
}

func executeWait(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player waits, ending turn.")
	return true
}

func executeSecondWind(g *Game, targetX, targetY int) bool {
	classDef := ClassDefinitions[g.Player.Class]
	if classDef == nil {
		return false
	}
	classAction := classDef.ClassActions["second_wind"]
	if classAction == nil {
		return false
	}

	conMod := getModifier(g.Player.Constitution)
	healRoll := rand.Intn(classDef.HitDieSize) + 1
	healAmount := max(1, healRoll+conMod)
	actualHeal := min(healAmount, g.Player.MaxHP-g.Player.HP)
	actualHeal = max(0, actualHeal)

	if actualHeal > 0 {
		g.Player.HP += actualHeal
		g.addCombatLog(fmt.Sprintf("Used Second Wind! Healed %d HP (Rolled %d%+d).", actualHeal, healRoll, conMod))
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("+%d", actualHeal),
			X:         float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2),
			Y:         float64(g.MapOffsetY + g.Player.Y*tileSize),
			Life:      60,
			MaxLife:   60,
			Color:     visualEffectColors["Heal"],
			VelocityY: -0.5,
		})
	} else {
		g.addCombatLog("Used Second Wind, but already at full HP.")
	}

	return true
}

func executeActionSurge(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Player uses Action Surge! Gains an extra action.")
	return true
}

func executeQuickStrike(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil || !isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		g.addCombatLog("Invalid target for Quick Strike (or not adjacent).")
		return false
	}

	attackAbilityMod := getModifier(g.Player.Strength)
	roll := rand.Intn(20) + 1
	isCrit := roll == 20
	isFumble := roll == 1
	attackRoll := roll + g.Player.ProficiencyBonus + attackAbilityMod
	effectiveAC := targetEnemy.AC

	rollString := fmt.Sprintf("%d+%d+%d(STR)=%d", roll, g.Player.ProficiencyBonus, attackAbilityMod, attackRoll)
	logMsg := fmt.Sprintf("Player Quick Strikes %s(AC%d). Roll: %s.", targetEnemy.Name, effectiveAC, rollString)
	hit := !isFumble && (isCrit || attackRoll >= effectiveAC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(8) + 1
		}
		baseDamage := damageRoll1 + damageRoll2 + attackAbilityMod
		finalDamage := max(1, baseDamage/2)
		actualDamage := min(finalDamage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (Half of 2d8[%d,%d]%+d) dmg.", actualDamage, damageRoll1, damageRoll2, attackAbilityMod)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (Half of 1d8[%d]%+d) dmg.", actualDamage, damageRoll1, attackAbilityMod)
		}
		logMsg += dmgLog

		ftColor := visualEffectColors["Default"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text: fmt.Sprintf("-%d", actualDamage),
			X:    textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: ftColor, VelocityY: -0.5,
		})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text: hitText,
			X:    textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5,
		})

		if targetEnemy.HP <= 0 {
			logMsg += fmt.Sprintf(" %s dies!", targetEnemy.Name)
			targetEnemy.IsDying = true
		}
	} else {
		logMsg += " Miss!"
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text: "Miss!",
			X:    textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5,
		})
	}

	g.addCombatLog(logMsg)
	return true
}

func executeFirebolt(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil {
		g.addCombatLog("Invalid target for Firebolt.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)

	actionRange := ActionTable["firebolt"].Range

	if dist > actionRange {
		g.addCombatLog("Target out of range for Firebolt.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Firebolt at %s!", targetEnemy.Name))
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, getModifier(g.Player.Intelligence), g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(10) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(10) + 1
		}
		damage := damageRoll1 + damageRoll2
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (2d10[%d,%d]) fire damage.", actualDamage, damageRoll1, damageRoll2)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (1d10[%d]) fire damage.", actualDamage, damageRoll1)
		}
		g.addCombatLog(attackLog + dmgLog)

		ftColor := visualEffectColors["Fire"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}

		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("-%d", actualDamage),
			X:         textSpawnX,
			Y:         textSpawnY,
			Life:      60,
			MaxLife:   60,
			Color:     ftColor,
			VelocityY: -0.5,
		})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      hitText,
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     hitTextColor,
			VelocityY: -0.5,
		})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      "Miss!",
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     colorGray,
			VelocityY: -0.5,
		})
	}

	return true
}

func executeMagicMissile(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil {
		g.addCombatLog("Invalid target for Magic Missile.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)

	actionRange := ActionTable["magic_missile"].Range

	if dist > actionRange {
		g.addCombatLog("Target out of range for Magic Missile.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Magic Missile at %s!", targetEnemy.Name))
	totalDamage := 0
	damageLog := " (["
	boltCount := 3

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	for i := 0; i < boltCount; i++ {
		boltDamage := rand.Intn(4) + 1 + 1
		totalDamage += boltDamage
		damageLog += fmt.Sprintf("%d", boltDamage)
		if i < boltCount-1 {
			damageLog += ","
		}

		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("-%d", boltDamage),
			X:         textSpawnX + float64(rand.Intn(10)-5),
			Y:         textSpawnY + float64(rand.Intn(10)-5),
			Life:      30 + i*5,
			MaxLife:   30 + i*5,
			Color:     visualEffectColors["Force"],
			VelocityY: -0.3 - float64(i)*0.05,
		})
	}
	damageLog += "])"

	actualDamage := min(totalDamage, targetEnemy.HP)
	targetEnemy.HP -= actualDamage
	g.addCombatLog(fmt.Sprintf(" Hits automatically! Deals %d total%s force damage.", actualDamage, damageLog))

	if targetEnemy.HP <= 0 {
		g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
		targetEnemy.IsDying = true
	}

	return true
}

func executeShield(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Used Reaction: Shield!")
	g.Player.ACBonusUntilNextTurn = 5
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text:      "+5 AC!",
		X:         float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2),
		Y:         float64(g.MapOffsetY+g.Player.Y*tileSize) - 15,
		Life:      60,
		MaxLife:   60,
		Color:     visualEffectColors["Arcane"],
		VelocityY: -0.5,
	})
	return true
}

func executeArcaneBlink(g *Game, targetX, targetY int) bool {
	if g.isTileFullyBlocked(targetX, targetY, g.Player.Width, g.Player.Height, -1) {
		g.addCombatLog("Cannot Arcane Blink into blocked tile.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetX, targetY)
	maxRange := playerBaseMovement
	if dist > maxRange {
		g.addCombatLog(fmt.Sprintf("Target tile too far for Arcane Blink (Max %d).", maxRange))
		return false
	}

	g.addCombatLog(fmt.Sprintf("Used Arcane Blink to (%d, %d)!", targetX, targetY))
	g.Player.X = targetX
	g.Player.Y = targetY
	g.Player.MovementPoints = 0
	return true
}

func executeBurningHands(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Burning Hands!")
	damageRoll1 := rand.Intn(6) + 1
	damageRoll2 := rand.Intn(6) + 1
	damageRoll3 := rand.Intn(6) + 1
	damage := damageRoll1 + damageRoll2 + damageRoll3
	damage = max(1, damage)
	g.addCombatLog(fmt.Sprintf("Deals %d (3d6[%d,%d,%d]) fire damage in a cone.", damage, damageRoll1, damageRoll2, damageRoll3))

	targetsHit := 0
	for i := len(g.Enemies) - 1; i >= 0; i-- {
		enemy := g.Enemies[i]
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}

		if isAdjacentSimple(g.Player.X, g.Player.Y, enemy.X, enemy.Y) {
			actualDamage := min(damage, enemy.HP)
			enemy.HP -= actualDamage
			g.addCombatLog(fmt.Sprintf(" Hits %s for %d damage!", enemy.Name, actualDamage))
			targetsHit++

			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
				Text:      fmt.Sprintf("-%d", actualDamage),
				X:         float64(g.MapOffsetX + enemy.X*tileSize + (enemy.Width*tileSize)/2),
				Y:         float64(g.MapOffsetY + enemy.Y*tileSize),
				Life:      60,
				MaxLife:   60,
				Color:     visualEffectColors["Fire"],
				VelocityY: -0.5,
			})

			if enemy.HP <= 0 {
				g.addCombatLog(fmt.Sprintf(" %s burns away!", enemy.Name))
				enemy.IsDying = true
			}
		}
	}

	if targetsHit == 0 {
		g.addCombatLog(" No targets hit.")
	}

	return true
}

func executeFrostNova(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Frost Nova!")
	damageRoll1 := rand.Intn(6) + 1
	damageRoll2 := rand.Intn(6) + 1
	damage := damageRoll1 + damageRoll2
	damage = max(1, damage)
	g.addCombatLog(fmt.Sprintf("Deals %d (2d6[%d,%d]) cold damage to adjacent enemies.", damage, damageRoll1, damageRoll2))

	targetsHit := 0
	for i := len(g.Enemies) - 1; i >= 0; i-- {
		enemy := g.Enemies[i]
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}

		if isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity) {
			actualDamage := min(damage, enemy.HP)
			enemy.HP -= actualDamage
			g.addCombatLog(fmt.Sprintf(" Hits %s for %d damage!", enemy.Name, actualDamage))
			targetsHit++

			g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
				Text:      fmt.Sprintf("-%d", actualDamage),
				X:         float64(g.MapOffsetX + enemy.X*tileSize + (enemy.Width*tileSize)/2),
				Y:         float64(g.MapOffsetY + enemy.Y*tileSize),
				Life:      60,
				MaxLife:   60,
				Color:     visualEffectColors["Cold"],
				VelocityY: -0.5,
			})

			if enemy.HP <= 0 {
				g.addCombatLog(fmt.Sprintf(" %s freezes and shatters!", enemy.Name))
				enemy.IsDying = true
			}
		}
	}

	if targetsHit == 0 {
		g.addCombatLog(" No targets hit.")
	}

	return true
}

func executeMindSpike(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil {
		g.addCombatLog("Invalid target for Mind Spike.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
	maxRange := playerRangedRange
	if dist > maxRange {
		g.addCombatLog("Target out of range for Mind Spike.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Mind Spike at %s!", targetEnemy.Name))
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, getModifier(g.Player.Intelligence), g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := rand.Intn(8) + 1
		damageRoll3 := 0
		damageRoll4 := 0
		if isCrit {
			damageRoll3 = rand.Intn(8) + 1
			damageRoll4 = rand.Intn(8) + 1
		}
		damage := damageRoll1 + damageRoll2 + damageRoll3 + damageRoll4
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (4d8[%d,%d,%d,%d]) psychic damage.", actualDamage, damageRoll1, damageRoll2, damageRoll3, damageRoll4)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (2d8[%d,%d]) psychic damage.", actualDamage, damageRoll1, damageRoll2)
		}
		g.addCombatLog(attackLog + dmgLog)

		ftColor := visualEffectColors["Psychic"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      fmt.Sprintf("-%d", actualDamage),
			X:         textSpawnX,
			Y:         textSpawnY,
			Life:      60,
			MaxLife:   60,
			Color:     ftColor,
			VelocityY: -0.5,
		})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      hitText,
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     hitTextColor,
			VelocityY: -0.5,
		})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s's mind is shattered!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
			Text:      "Miss!",
			X:         textSpawnX,
			Y:         textSpawnY - 15,
			Life:      30,
			MaxLife:   30,
			Color:     colorGray,
			VelocityY: -0.5,
		})
	}
	return true
}

func executeRayOfFrost(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	actionDef := ActionTable["ray_of_frost"]
	if targetEnemy == nil {
		g.addCombatLog("Invalid target.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
	if dist > actionDef.Range {
		g.addCombatLog("Target out of range.")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Ray of Frost at %s!", targetEnemy.Name))
	// casterAbilityMod := getModifier(g.Player.Wisdom)
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, getModifier(g.Player.Intelligence), g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(8) + 1
		}
		damage := damageRoll1 + damageRoll2
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (2d8[%d,%d]) cold dmg.", actualDamage, damageRoll1, damageRoll2)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (1d8[%d]) cold dmg.", actualDamage, damageRoll1)
		}
		logMsg := attackLog + dmgLog
		g.addCombatLog(logMsg)

		ftColor := visualEffectColors["Cold"]
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: ftColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5})
	}
	return true
}

func executeShockingGrasp(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	if targetEnemy == nil || !isAdjacentToEntity(g.Player.X, g.Player.Y, &targetEnemy.Entity) {
		g.addCombatLog("Invalid target (or not adjacent).")
		return false
	}

	g.addCombatLog(fmt.Sprintf("Casting Shocking Grasp on %s!", targetEnemy.Name))
	casterAbilityMod := getModifier(g.Player.Intelligence)
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, casterAbilityMod, g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)

	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		damageRoll1 := rand.Intn(8) + 1
		damageRoll2 := 0
		if isCrit {
			damageRoll2 = rand.Intn(8) + 1
		}
		damage := damageRoll1 + damageRoll2
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (2d8[%d,%d]) lightning dmg.", actualDamage, damageRoll1, damageRoll2)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (1d8[%d]) lightning dmg.", actualDamage, damageRoll1)
		}
		logMsg := attackLog + dmgLog

		g.addCombatLog(logMsg)

		ftColor := colorYellow
		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			ftColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: ftColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5})
	}
	return true
}

func executeMagicArmor(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Magic Armor!")
	g.Player.MagicArmorDuration = 3
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text: "+2 AC!", X: float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2), Y: float64(g.MapOffsetY+g.Player.Y*tileSize) - 15, Life: 60, MaxLife: 60, Color: visualEffectColors["Arcane"], VelocityY: -0.5,
	})
	return true
}

func executeElementalStrike(g *Game, targetX, targetY int) bool {
	targetEnemy := g.getEnemyAt(targetX, targetY)
	actionDef := ActionTable["elemental_strike"]
	if targetEnemy == nil {
		g.addCombatLog("Invalid target.")
		return false
	}
	dist := distance(g.Player.X, g.Player.Y, targetEnemy.X, targetEnemy.Y)
	if dist > actionDef.Range {
		g.addCombatLog("Target out of range.")
		return false
	}

	currentElement := g.Player.NextElementType
	g.addCombatLog(fmt.Sprintf("Casting Elemental Strike (%s) at %s!", currentElement, targetEnemy.Name))

	casterAbilityMod := getModifier(g.Player.Intelligence)
	attackTotal, roll, isCrit := spellAttackRoll(g)
	hit := isCrit || attackTotal >= targetEnemy.AC
	attackLog := fmt.Sprintf(" (Roll %d + INT %d + Prof %d = %d vs AC %d)",
		roll, casterAbilityMod, g.Player.ProficiencyBonus, attackTotal, targetEnemy.AC)

	textSpawnX := float64(g.MapOffsetX + targetEnemy.X*tileSize + (targetEnemy.Width*tileSize)/2)
	textSpawnY := float64(g.MapOffsetY + targetEnemy.Y*tileSize)
	elementColor := visualEffectColors["Default"]
	if currentElement == "Fire" {
		elementColor = visualEffectColors["Fire"]
	}
	if currentElement == "Cold" {
		elementColor = visualEffectColors["Cold"]
	}
	if currentElement == "Lightning" {
		elementColor = colorYellow
	}
	if hit {
		g.Player.AttackBumpTimer = attackBumpDuration
		d1, d2, d3 := rand.Intn(8)+1, rand.Intn(8)+1, rand.Intn(8)+1
		critD1, critD2, critD3 := 0, 0, 0
		if isCrit {
			critD1, critD2, critD3 = rand.Intn(8)+1, rand.Intn(8)+1, rand.Intn(8)+1
		}
		damage := d1 + d2 + d3 + critD1 + critD2 + critD3
		damage = max(1, damage)
		actualDamage := min(damage, targetEnemy.HP)
		targetEnemy.HP -= actualDamage

		dmgLog := ""
		if isCrit {
			dmgLog = fmt.Sprintf(" CRIT! Deals %d (6d8[%d,%d,%d,%d,%d,%d]) %s dmg.", actualDamage, d1, d2, d3, critD1, critD2, critD3, currentElement)
		} else {
			dmgLog = fmt.Sprintf(" Hit! Deals %d (3d8[%d,%d,%d]) %s dmg.", actualDamage, d1, d2, d3, currentElement)
		}
		g.addCombatLog(attackLog + dmgLog)

		hitTextColor := colorWhite
		hitText := "Hit!"
		if isCrit {
			elementColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
			hitTextColor = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
			hitText = "CRITICAL!"
		}
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: fmt.Sprintf("-%d", actualDamage), X: textSpawnX, Y: textSpawnY, Life: 60, MaxLife: 60, Color: elementColor, VelocityY: -0.5})
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: hitText, X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: hitTextColor, VelocityY: -0.5})

		if targetEnemy.HP <= 0 {
			g.addCombatLog(fmt.Sprintf("%s dies!", targetEnemy.Name))
			targetEnemy.IsDying = true
		}
	} else {
		g.addCombatLog(attackLog + " Miss!")
		g.FloatingTexts = append(g.FloatingTexts, &FloatingText{Text: "Miss!", X: textSpawnX, Y: textSpawnY - 15, Life: 30, MaxLife: 30, Color: colorGray, VelocityY: -0.5})
	}

	if currentElement == "Fire" {
		g.Player.NextElementType = "Cold"
	}
	if currentElement == "Cold" {
		g.Player.NextElementType = "Lightning"
	}
	if currentElement == "Lightning" {
		g.Player.NextElementType = "Fire"
	}

	return true
}

func executeFeatherFall(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Feather Fall!")
	g.Player.FeatherFallDuration = 2
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text: "Evasive!", X: float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2), Y: float64(g.MapOffsetY+g.Player.Y*tileSize) - 15, Life: 60, MaxLife: 60, Color: colorWhite, VelocityY: -0.5,
	})
	return true
}

func executeExpeditiousRetreat(g *Game, targetX, targetY int) bool {
	g.addCombatLog("Casting Expeditious Retreat!")
	g.Player.ExpeditiousRetreatDuration = 2
	g.FloatingTexts = append(g.FloatingTexts, &FloatingText{
		Text: "Speedy!", X: float64(g.MapOffsetX + g.Player.X*tileSize + (g.Player.Width*tileSize)/2), Y: float64(g.MapOffsetY+g.Player.Y*tileSize) - 15, Life: 60, MaxLife: 60, Color: colorYellow, VelocityY: -0.5,
	})
	return true
}

func (g *Game) handlePlayerInput() {
	// --- DEBUG COMMANDS START ---
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
	// --- DEBUG COMMANDS END ---

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
				} // Arcane Blink
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					spellIndex = 1
				} // Burning Hands
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					spellIndex = 2
				} // Frost Nova
				if inpututil.IsKeyJustPressed(ebiten.Key4) {
					spellIndex = 3
				} // Mind Spike
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
				} // Ray of Frost
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					cantripIndex = 1
				} // Shocking Grasp
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
				} // Magic Armor
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					spellIndex = 1
				} // Elemental Strike
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					spellIndex = 2
				} // Feather Fall
				if inpututil.IsKeyJustPressed(ebiten.Key4) {
					spellIndex = 3
				} // Expeditious Retreat
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
				} // Gladiator
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					choiceIndex = 1
				} // Ranger
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					choiceIndex = 2
				} // Juggernaut
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
				} // Power Attack
				if inpututil.IsKeyJustPressed(ebiten.Key2) {
					choiceIndex = 1
				} // Defensive Stance
				if inpututil.IsKeyJustPressed(ebiten.Key3) {
					choiceIndex = 2
				} // Quick Strike
				if choiceIndex != -1 {
					techniques := []string{"Power Attack", "Defensive Stance", "Quick Strike"}
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
						if performAoOCheck {
							for _, enemy := range g.Enemies {
								if enemy.IsDying || enemy.HP <= 0 {
									continue
								}
								wasAdj := isAdjacentToEntity(startX, startY, &enemy.Entity)
								isStillAdj := isAdjacentToEntity(targetX, targetY, &enemy.Entity)
								if wasAdj && !isStillAdj {
									if g.Player.HP > 0 && !g.Player.IsDying && !g.Player.UsedReaction {
										g.addCombatLog(fmt.Sprintf("%s makes an Opportunity Attack!", enemy.Name))
										killedByAoO, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
										if g.reactionPending {
											return
										}
										if killedByAoO {
											g.Player.MovementPoints--
											return
										}
									}
								}
							}
						}
						if g.Player.HP > 0 && !g.Player.IsDying {
							g.Player.X = targetX
							g.Player.Y = targetY
							g.Player.MovementPoints--
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

func (g *Game) buildAvailableActions() {
	g.availableActions = []*ActionDefinition{}
	possibleActions := make(map[string]bool)

	alwaysAvailable := map[string]bool{"wait": true}
	if g.Player.Class == "Mage" {
	} else {
		alwaysAvailable["melee_attack"] = true
		alwaysAvailable["ranged_attack"] = true
		alwaysAvailable["dash"] = true
		alwaysAvailable["disengage"] = true
	}

	for id := range alwaysAvailable {
		if _, exists := ActionTable[id]; exists {
			possibleActions[id] = true
		}
	}
	for _, cantripID := range g.Player.KnownCantrips {
		if _, exists := ActionTable[cantripID]; exists {
			possibleActions[cantripID] = true
		}
	}
	for _, spellID := range g.Player.KnownSpells {
		if _, exists := ActionTable[spellID]; exists {
			possibleActions[spellID] = true
		}
	}

	classDef, exists := ClassDefinitions[g.Player.Class]
	if exists {
		for actionID, classAction := range classDef.ClassActions {
			if g.Player.Level >= classAction.RequiredLevel {
				if _, actionDefExists := ActionTable[actionID]; actionDefExists {
					possibleActions[actionID] = true
				} else {
					log.Printf("Warning: ClassAction '%s' defined for class '%s' but not found in main ActionTable.", actionID, g.Player.Class)
				}
			}
		}
	}

	if g.Player.CombatTechnique == "Quick Strike" {
		if _, exists := ActionTable["quick_strike"]; exists {
			possibleActions["quick_strike"] = true
		}
	}

	tempAvailableActions := []*ActionDefinition{}
	for id := range possibleActions {
		actionDef := ActionTable[id]
		isAvailable := true

		switch actionDef.ActionType {
		case ActionTypeStandard:
			if g.Player.ActionTaken {
				isAvailable = false
			}
		case ActionTypeBonus:
			if g.Player.BonusActionTaken {
				isAvailable = false
			}
		case ActionTypeReaction:
			isAvailable = false
		case ActionTypeFree:
		}
		if !isAvailable {
			continue
		}

		if actionDef.ResourceType == ResourceSpellSlotL1 {
			if g.Player.SpellSlotsL1 < actionDef.ResourceCost {
				isAvailable = false
			}
		} else if actionDef.ResourceType == ResourceClassFeature {
			if classDef != nil {
				if classAction, ok := classDef.ClassActions[id]; ok {
					if classAction.UsesPerRest > 0 {
						if _, resOk := g.Player.ClassResources[id]; !resOk {
							isAvailable = false
						} else if g.Player.ClassResources[id] < classAction.ResourceCost {
							isAvailable = false
						}
					}
				}
			}
		}

		if !isAvailable {
			continue
		}
		if actionDef.ID == "disengage" && !isPlayerAdjacentToEnemy(g) {
			isAvailable = false
		}
		if isAvailable {
			tempAvailableActions = append(tempAvailableActions, actionDef)
		}
	}

	sort.Slice(tempAvailableActions, func(i, j int) bool {
		if tempAvailableActions[i].ActionType != tempAvailableActions[j].ActionType {
			return tempAvailableActions[i].ActionType < tempAvailableActions[j].ActionType
		}
		return tempAvailableActions[i].Name < tempAvailableActions[j].Name
	})

	g.availableActions = tempAvailableActions
}

func (g *Game) findPathStep(startX, startY, targetX, targetY, entityWidth, entityHeight, movingEnemyIndex int) (int, int, bool) {
	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	bestMoveX, bestMoveY := startX, startY

	possibleMoves := []image.Point{}
	for _, offset := range moveOffsets {
		nextX, nextY := startX+offset.X, startY+offset.Y
		if !g.isTileFullyBlocked(nextX, nextY, entityWidth, entityHeight, movingEnemyIndex) {
			possibleMoves = append(possibleMoves, image.Point{X: nextX, Y: nextY})
		}
	}

	if len(possibleMoves) == 0 {
		return startX, startY, false
	}

	minDist := math.MaxInt32
	for _, move := range possibleMoves {
		dist := distance(move.X, move.Y, targetX, targetY)
		if dist < minDist {
			minDist = dist
		}
	}

	currentDist := distance(startX, startY, targetX, targetY)
	if currentDist <= 1 && minDist > currentDist {
		return startX, startY, false
	}

	bestMoves := []image.Point{}
	for _, move := range possibleMoves {
		if distance(move.X, move.Y, targetX, targetY) == minDist {
			bestMoves = append(bestMoves, move)
		}
	}

	if len(bestMoves) > 0 {
		dx := targetX - startX
		dy := targetY - startY
		preferredMoveFound := false
		if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
			for _, move := range bestMoves {
				if move.X != startX {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}
		if !preferredMoveFound && math.Abs(float64(dy)) > math.Abs(float64(dx)) {
			for _, move := range bestMoves {
				if move.Y != startY {
					bestMoveX, bestMoveY = move.X, move.Y
					preferredMoveFound = true
					break
				}
			}
		}
		if !preferredMoveFound {
			if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
				for _, move := range bestMoves {
					if move.Y != startY {
						bestMoveX, bestMoveY = move.X, move.Y
						preferredMoveFound = true
						break
					}
				}
			}
		}

		if !preferredMoveFound && len(bestMoves) > 0 {
			bestMoveX, bestMoveY = bestMoves[0].X, bestMoves[0].Y
		}

		if bestMoveX != startX || bestMoveY != startY {
			return bestMoveX, bestMoveY, true
		}
	}

	return startX, startY, false
}

func (g *Game) findRetreatStep(startX, startY, targetX, targetY, entityWidth, entityHeight, movingEnemyIndex int) (int, int, bool) {
	moveOffsets := []image.Point{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	currentDist := distance(startX, startY, targetX, targetY)
	bestMoveX, bestMoveY := startX, startY
	maxDist := currentDist

	possibleMoves := []image.Point{}
	for _, offset := range moveOffsets {
		nextX, nextY := startX+offset.X, startY+offset.Y
		if !g.isTileFullyBlocked(nextX, nextY, entityWidth, entityHeight, movingEnemyIndex) {
			possibleMoves = append(possibleMoves, image.Point{X: nextX, Y: nextY})
		}
	}

	if len(possibleMoves) == 0 {
		return startX, startY, false
	}

	bestMoves := []image.Point{}
	for _, move := range possibleMoves {
		dist := distance(move.X, move.Y, targetX, targetY)
		if dist > maxDist {
			maxDist = dist
		}
	}

	if maxDist == currentDist {

		maxDist = -1
		for _, move := range possibleMoves {
			dist := distance(move.X, move.Y, targetX, targetY)
			if dist > maxDist {
				maxDist = dist
			}
		}
	}

	for _, move := range possibleMoves {
		if distance(move.X, move.Y, targetX, targetY) == maxDist {
			bestMoves = append(bestMoves, move)
		}
	}

	if len(bestMoves) > 0 {

		bestMoveX, bestMoveY = bestMoves[0].X, bestMoves[0].Y
		if bestMoveX != startX || bestMoveY != startY {
			return bestMoveX, bestMoveY, true
		}
	}

	return startX, startY, false
}

func (g *Game) handleEnemyTurns() {
	if g.Player == nil || g.Player.IsDying || g.Player.HP <= 0 {
		return
	}

	for i, enemy := range g.Enemies {
		if enemy.IsDying || enemy.HP <= 0 {
			continue
		}

		enemy.MovementPoints = enemy.MaxMovementPoints
		enemy.ActionAvailable = true
		actedThisTurn := false

		distToPlayer := distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
		isAdj := isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

		if enemy.AttackType == "ranged" {
			if !isAdj && distToPlayer <= enemy.MaxRange && enemy.ActionAvailable {
				killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
				if g.reactionPending {
					return
				}
				enemy.ActionAvailable = false
				actedThisTurn = true
				if killedPlayer {
					return
				}
			}
		} else {
			if isAdj && enemy.ActionAvailable {
				killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
				if g.reactionPending {
					return
				}
				enemy.ActionAvailable = false
				actedThisTurn = true
				if killedPlayer {
					return
				}
			}
		}

		for enemy.MovementPoints > 0 {
			if g.Player.IsDying || g.Player.HP <= 0 {
				break
			}

			distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
			isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)
			movedThisStep := false

			if enemy.AttackType == "ranged" {
				if isAdj {
					nextX, nextY, foundMove := g.findRetreatStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						movedThisStep = true
					} else {
						break
					}
				} else if distToPlayer > enemy.MaxRange {
					nextX, nextY, foundMove := g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						movedThisStep = true
					} else {
						break
					}
				} else {
					break
				}
			} else {
				if !isAdj {
					nextX, nextY, foundMove := g.findPathStep(enemy.X, enemy.Y, g.Player.X, g.Player.Y, enemy.Width, enemy.Height, i)
					if foundMove {
						enemy.X, enemy.Y = nextX, nextY
						enemy.MovementPoints--
						movedThisStep = true
					} else {
						break
					}
				} else {
					break
				}
			}
			if !movedThisStep {
				break
			}
		}

		if !actedThisTurn && enemy.ActionAvailable {
			if g.Player.IsDying || g.Player.HP <= 0 {
				continue
			}

			distToPlayer = distance(enemy.X, enemy.Y, g.Player.X, g.Player.Y)
			isAdj = isAdjacentToEntity(g.Player.X, g.Player.Y, &enemy.Entity)

			if enemy.AttackType == "ranged" {
				if !isAdj && distToPlayer <= enemy.MaxRange {
					killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "ranged")
					if g.reactionPending {
						return
					}
					enemy.ActionAvailable = false
					if killedPlayer {
						return
					}
				}
			} else {
				if isAdj {
					killedPlayer, _ := g.resolveAttack(&enemy.Entity, &g.Player.Entity, 0, "melee")
					if g.reactionPending {
						return
					}
					enemy.ActionAvailable = false
					if killedPlayer {
						return
					}
				}
			}
		}
		enemy.ActionAvailable = false

	}
}

func (g *Game) cleanupDeadEnemies() {
	initialCount := len(g.Enemies)
	aliveEnemies := make([]*Enemy, 0, len(g.Enemies))
	for _, enemy := range g.Enemies {
		if !enemy.IsDying || enemy.CurrentAlpha > 0 {
			aliveEnemies = append(aliveEnemies, enemy)
		}
	}
	if len(aliveEnemies) != initialCount {
		g.Enemies = aliveEnemies
	}
}

func (g *Game) Update() error {
	activeTexts := make([]*FloatingText, 0, len(g.FloatingTexts))
	for _, ft := range g.FloatingTexts {
		ft.Life--
		if ft.Life > 0 {
			ft.Y += ft.VelocityY
			activeTexts = append(activeTexts, ft)
		}
	}
	g.FloatingTexts = activeTexts

	if g.Player != nil {
		if g.Player.AttackBumpTimer > 0 {
			g.Player.AttackBumpTimer--
		}
		if g.Player.IsDying && g.Player.CurrentAlpha > 0 {
			g.Player.CurrentAlpha -= 1.0 / float64(deathFadeDuration)
			if g.Player.CurrentAlpha < 0 {
				g.Player.CurrentAlpha = 0
			}
		}
	}

	for _, enemy := range g.Enemies {
		if enemy.AttackBumpTimer > 0 {
			enemy.AttackBumpTimer--
		}
		if enemy.IsDying && enemy.CurrentAlpha > 0 {
			enemy.CurrentAlpha -= 1.0 / float64(deathFadeDuration)
			if enemy.CurrentAlpha < 0 {
				enemy.CurrentAlpha = 0
			}
		}
	}

	switch g.CurrentGameState {
	case StateClassSelection:
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.classSelectionIndex--
			if g.classSelectionIndex < 0 {
				g.classSelectionIndex = len(g.selectableClasses) - 1
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.classSelectionIndex++
			if g.classSelectionIndex >= len(g.selectableClasses) {
				g.classSelectionIndex = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			selected := g.selectableClasses[g.classSelectionIndex]
			if selected.IsAvailable {
				g.InitializeGameplay(selected.Name)
			}
		}
	case StatePlaying:
		g.UpdatePlaying()
	case StateGameOverScreen:
		break
	}

	return nil
}

func (g *Game) UpdatePlaying() {
	if g.reactionPending {
		g.handlePlayerInput()
		return
	}

	if g.CurrentTurn == GameOver || g.CurrentGameState == StateGameOverScreen {
		return
	}

	if g.InputMode == InputModeLevelUp {
		g.handlePlayerInput()
		return
	}
	if g.InputMode == InputModeRestPrompt {
		g.handlePlayerInput()
		return
	}

	if g.CurrentTurn == PlayerTurn || g.InputMode == InputModeActionSelect || g.InputMode == InputModeCharacterSheet {
		g.handlePlayerInput()
	}

	if g.CurrentGameState != StatePlaying || g.InputMode == InputModeLevelUp || g.InputMode == InputModeRestPrompt {
		return
	}

	if g.CurrentTurn == EnemyTurn {
		g.handleEnemyTurns()
		if g.reactionPending {
			return
		}
		if g.Player.IsDying && g.Player.CurrentAlpha <= 0 && g.CurrentGameState != StateGameOverScreen {
			g.setGameOver(false)
			return
		}
		if g.InputMode != InputModeRestPrompt && g.InputMode != InputModeLevelUp {
			g.endEnemyTurn()
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.CurrentGameState {
	case StateClassSelection:
		g.DrawClassSelection(screen)
	case StatePlaying:
		g.DrawPlaying(screen)
	case StateGameOverScreen:
		g.DrawPlaying(screen)
		g.DrawGameOver(screen)
	}
}

func (g *Game) DrawClassSelection(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 10, G: 10, B: 20, A: 255})
	title := "Select Your Class"
	titleFont := basicfont.Face7x13
	titleBounds := text.BoundString(titleFont, title)
	titleX := (screenWidth - titleBounds.Dx()) / 2
	titleY := screenHeight / 4
	text.Draw(screen, title, titleFont, titleX, titleY, colorWhite)

	itemStartY := titleY + 40
	itemLineHeight := 20
	itemX := screenWidth / 3

	for i, class := range g.selectableClasses {
		lineText := class.Name
		lineColor := colorGray

		if !class.IsAvailable {
			lineText += " (Locked)"
			lineColor = colorLocked
		}

		if i == g.classSelectionIndex {
			lineText = "> " + lineText
			if class.IsAvailable {
				lineColor = colorWhite
			} else {
				lineColor = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
			}
		}

		text.Draw(screen, lineText, titleFont, itemX, itemStartY+(i*itemLineHeight), lineColor)
	}
	helpText := "Up/Down to navigate, Enter to select"
	helpBounds := text.BoundString(titleFont, helpText)
	helpX := (screenWidth - helpBounds.Dx()) / 2
	helpY := screenHeight - 40
	text.Draw(screen, helpText, titleFont, helpX, helpY, colorGray)
}

func (g *Game) DrawPlaying(screen *ebiten.Image) {
	mapOffsetX, mapOffsetY := g.MapOffsetX, g.MapOffsetY
	tileOpts := &ebiten.DrawImageOptions{}
	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			screenX := float64(mapOffsetX + x*tileSize)
			screenY := float64(mapOffsetY + y*tileSize)
			tileOpts.GeoM.Reset()
			tileOpts.GeoM.Translate(screenX, screenY)
			if g.TileImage != nil {
				screen.DrawImage(g.TileImage, tileOpts)
			}
		}
	}
	if g.CurrentTurn == PlayerTurn && g.InputMode == InputModeMap {
		cursorX, cursorY := ebiten.CursorPosition()
		gridX := (cursorX - g.MapOffsetX) / tileSize
		gridY := (cursorY - g.MapOffsetY) / tileSize
		if gridX >= 0 && gridX < mapWidth && gridY >= 0 && gridY < mapHeight {
			hoverScreenX := float32(mapOffsetX + gridX*tileSize)
			hoverScreenY := float32(mapOffsetY + gridY*tileSize)
			hoverColor := color.NRGBA{R: 255, G: 255, B: 255, A: 100}
			vector.StrokeRect(screen, hoverScreenX, hoverScreenY, float32(tileSize), float32(tileSize), 1, hoverColor, false)
		}
	}
	if g.CurrentTurn == PlayerTurn && g.primedActionID != "" && g.RangeOverlayTile != nil {
		actionDef, exists := ActionTable[g.primedActionID]
		if exists && actionDef.RequiresTarget && actionDef.Range > 0 && g.Player != nil {
			overlayOpts := &ebiten.DrawImageOptions{}
			originX, originY := g.Player.X, g.Player.Y
			for x := 0; x < mapWidth; x++ {
				for y := 0; y < mapHeight; y++ {
					dist := distance(originX, originY, x, y)
					isInRange := false
					if actionDef.Targeting == TargetEmptyTile {
						isInRange = dist > 0 && dist <= actionDef.Range && !g.isTileFullyBlocked(x, y, 1, 1, -1)
					} else {
						isInRange = dist > 0 && dist <= actionDef.Range
					}
					if isInRange {
						screenX := float64(mapOffsetX + x*tileSize)
						screenY := float64(mapOffsetY + y*tileSize)
						overlayOpts.GeoM.Reset()
						overlayOpts.GeoM.Translate(screenX, screenY)
						screen.DrawImage(g.RangeOverlayTile, overlayOpts)
					}
				}
			}
		}
	}

	entitiesToDraw := make([]*Entity, 0, len(g.Enemies)+1)
	for _, enemy := range g.Enemies {
		if !enemy.IsDying || enemy.CurrentAlpha > 0 {
			entitiesToDraw = append(entitiesToDraw, &enemy.Entity)
		}
	}
	if g.Player != nil && (!g.Player.IsDying || g.Player.CurrentAlpha > 0) {
		entitiesToDraw = append(entitiesToDraw, &g.Player.Entity)
	}
	sort.Slice(entitiesToDraw, func(i, j int) bool {
		if entitiesToDraw[i].Y != entitiesToDraw[j].Y {
			return entitiesToDraw[i].Y < entitiesToDraw[j].Y
		}
		return entitiesToDraw[i].X < entitiesToDraw[j].X
	})
	entityOpts := &ebiten.DrawImageOptions{}
	for _, entity := range entitiesToDraw {
		if entity.Sprite == nil {
			continue
		}
		entityScreenX := float64(mapOffsetX + entity.X*tileSize)
		entityScreenY := float64(mapOffsetY + entity.Y*tileSize)
		bumpOffsetX := 0.0
		if entity.AttackBumpTimer > 0 {
			progress := float64(attackBumpDuration-entity.AttackBumpTimer) / float64(attackBumpDuration)
			bumpOffsetX = math.Sin(progress*math.Pi) * 4.0
			entityScreenX += bumpOffsetX
		}
		entityOpts.GeoM.Reset()
		if entity.Width > 1 || entity.Height > 1 {
			entityOpts.GeoM.Scale(float64(entity.Width), float64(entity.Height))
		}
		entityOpts.GeoM.Translate(entityScreenX, entityScreenY)
		entityOpts.ColorM.Reset()
		if entity.CurrentAlpha < 1.0 {
			entityOpts.ColorM.Scale(1, 1, 1, entity.CurrentAlpha)
		}
		screen.DrawImage(entity.Sprite, entityOpts)
		if entity.CurrentAlpha > 0 && entity.MaxHP > 0 {
			hpBarBaseX := float64(mapOffsetX+entity.X*tileSize) + bumpOffsetX
			hpBarBaseY := float64(mapOffsetY + (entity.Y+entity.Height)*tileSize)
			hpBarX, hpBarY := float32(hpBarBaseX), float32(hpBarBaseY+hpBarOffsetY)
			hpBarWidth := float32(tileSize * entity.Width)
			hpRatio := float32(max(0, entity.HP)) / float32(entity.MaxHP)
			hpRatio = maxF(0.0, minF(1.0, hpRatio))
			var hpColor color.NRGBA
			if hpRatio > 0.6 {
				hpColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
			} else if hpRatio > 0.3 {
				hpColor = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
			} else {
				hpColor = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
			}
			hpBgColor := color.NRGBA{R: 50, G: 50, B: 50, A: 255}
			hpBarAlpha := uint8(entity.CurrentAlpha * 255)
			hpColor.A, hpBgColor.A = hpBarAlpha, hpBarAlpha
			outlineColor := color.NRGBA{R: 0, G: 0, B: 0, A: hpBarAlpha}
			vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, hpBgColor, false)
			vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarWidth*hpRatio, hpBarHeight, hpColor, false)
			vector.StrokeRect(screen, hpBarX, hpBarY, hpBarWidth, hpBarHeight, 1, outlineColor, false)
		}
	}

	uiStartY, uiLineHeight, statusStartY := 10, 15, 10
	if g.Player != nil {
		waveText := fmt.Sprintf("Wave: %d / %d", g.CurrentWaveIndex+1, len(g.WaveDefinitions))
		waveTextWidth := text.BoundString(basicfont.Face7x13, waveText).Dx()
		ebitenutil.DebugPrintAt(screen, waveText, screenWidth-waveTextWidth-10, statusStartY)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Turn: %s", g.CurrentTurn.String()), 10, uiStartY)
		playerHpVal := max(0, g.Player.HP)
		effectiveAC := g.Player.AC + g.Player.ACBonusUntilNextTurn
		acString := fmt.Sprintf("%d", g.Player.AC)
		if g.Player.ACBonusUntilNextTurn > 0 {
			acString = fmt.Sprintf("%d (+%d Buff)", effectiveAC, g.Player.ACBonusUntilNextTurn)
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP: %d/%d AC: %s", playerHpVal, g.Player.MaxHP, acString), 10, uiStartY+uiLineHeight*1)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Move: %d/%d", g.Player.MovementPoints, g.Player.MaxMovementPoints), 10, uiStartY+uiLineHeight*2)
		if g.Player.Class == "Mage" {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("L1 Slots: %d/%d", g.Player.SpellSlotsL1, g.Player.MaxSpellSlotsL1), 10, uiStartY+uiLineHeight*3)
		} else {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Hit Dice: %d/%d", g.Player.HitDice, g.Player.MaxHitDice), 10, uiStartY+uiLineHeight*3)
		}
		actionStatusText := "Action: Available"
		if g.Player.ActionTaken {
			actionStatusText = "Action: Used"
		}
		ebitenutil.DebugPrintAt(screen, actionStatusText, 10, uiStartY+uiLineHeight*4)
		bonusActionStatusText := "Bonus Action: Available"
		if g.Player.BonusActionTaken {
			bonusActionStatusText = "Bonus Action: Used"
		}
		ebitenutil.DebugPrintAt(screen, bonusActionStatusText, 10, uiStartY+uiLineHeight*5)
		reactionStatusText := "Reaction: Available"
		if g.Player.UsedReaction {
			reactionStatusText = "Reaction: Used"
		}
		ebitenutil.DebugPrintAt(screen, reactionStatusText, 10, uiStartY+uiLineHeight*6)
		primedActionText := "Primed: None"
		if g.primedActionID != "" {
			if actionDef, exists := ActionTable[g.primedActionID]; exists {
				primedActionText = fmt.Sprintf("Primed: %s", actionDef.Name)
			} else {
				primedActionText = fmt.Sprintf("Primed: ??? (%s)", g.primedActionID)
			}
		}
		ebitenutil.DebugPrintAt(screen, primedActionText, 10, uiStartY+uiLineHeight*7)
		statusText := ""
		if g.Player.IsDisengaging {
			statusText += "Disengaging "
		}
		if g.Player.IsSlowed {
			statusText += "Slowed "
		}
		if g.Player.MagicArmorDuration > 0 {
			statusText += "MagicArmor "
		}
		if g.Player.ExpeditiousRetreatDuration > 0 {
			statusText += "Speedy "
		}
		if g.Player.FeatherFallDuration > 0 {
			statusText += "Evasive "
		}
		if statusText != "" {
			ebitenutil.DebugPrintAt(screen, "Status: "+statusText, 10, uiStartY+uiLineHeight*8)
		}
	}
	if g.InputMode == InputModeActionSelect {
		menuW, menuH := screenWidth/2, screenHeight/2+20
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 20, B: 30, A: 220}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, colorWhite, false)
		title := "Select Action ([Tab] / [Esc] to Cancel)"
		titleX, titleY := menuX+10, menuY+15
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, colorWhite)
		itemStartY, itemLineHeight := titleY+25, 18
		for i, actionDef := range g.availableActions {
			actionText := actionDef.Name
			resourceText := ""
			if actionDef.ResourceType == ResourceSpellSlotL1 {
				resourceText = fmt.Sprintf(" (Cost: %d L1 Slot)", actionDef.ResourceCost)
			} else if actionDef.ResourceType == ResourceClassFeature {
				classDef := ClassDefinitions[g.Player.Class]
				if classDef != nil {
					if classAction, ok := classDef.ClassActions[actionDef.ID]; ok {
						if classAction.UsesPerRest > 0 {
							uses := 0
							if val, resOk := g.Player.ClassResources[actionDef.ID]; resOk {
								uses = val
							}
							resourceText = fmt.Sprintf(" (%d/%d)", uses, classAction.UsesPerRest)
						}
					}
				}
			}
			actionText += resourceText
			var itemColor color.Color = colorGray
			if i == g.selectedActionIndex {
				actionText = "> " + actionText
				itemColor = colorWhite
			}
			itemX, itemY := menuX+15, itemStartY+(i*itemLineHeight)
			text.Draw(screen, actionText, basicfont.Face7x13, itemX, itemY, itemColor)
		}
	}
	if g.InputMode == InputModeCharacterSheet && g.Player != nil {
		menuW, menuH := screenWidth/2+40, screenHeight/2+60
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 30, G: 20, B: 20, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, colorWhite, false)
		title := fmt.Sprintf("%s - Level %d %s ([C] / [Esc] to Close)", g.Player.Name, g.Player.Level, g.Player.Class)
		titleX, titleY := menuX+10, menuY+15
		text.Draw(screen, title, basicfont.Face7x13, titleX, titleY, colorWhite)
		infoStartY, infoLineHeight, col1X, col2X := titleY+25, 14, menuX+15, menuX+menuW/2
		lineNum := 0
		text.Draw(screen, fmt.Sprintf("HP: %d / %d", max(0, g.Player.HP), g.Player.MaxHP), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("AC: %d", g.Player.AC), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("Movement: %d", g.Player.MaxMovementPoints), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("Prof Bonus: +%d", g.Player.ProficiencyBonus), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		classHitDieSize := 0
		if cd, ok := ClassDefinitions[g.Player.Class]; ok {
			classHitDieSize = cd.HitDieSize
		}
		text.Draw(screen, fmt.Sprintf("Hit Dice: %d / %d (d%d)", g.Player.HitDice, g.Player.MaxHitDice, classHitDieSize), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		if g.Player.Class == "Mage" {
			text.Draw(screen, fmt.Sprintf("L1 Slots: %d / %d", g.Player.SpellSlotsL1, g.Player.MaxSpellSlotsL1), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
			lineNum++
		}
		if g.Player.CombatStyle != "" {
			text.Draw(screen, fmt.Sprintf("Style: %s", g.Player.CombatStyle), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
			lineNum++
		}
		if g.Player.CombatTechnique != "" {
			text.Draw(screen, fmt.Sprintf("Technique: %s", g.Player.CombatTechnique), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
			lineNum++
		}
		lineNum = max(lineNum, 8)
		if g.Player.Class == "Mage" {
			intMod := getModifier(g.Player.Intelligence)
			dc := spellSaveDC(g)
			text.Draw(screen, fmt.Sprintf("Spell Atk: +%d", g.Player.ProficiencyBonus+intMod), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
			lineNum++
			text.Draw(screen, fmt.Sprintf("Spell Save DC: %d", dc), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
			lineNum++
		} else {
			meleeMod := getModifier(g.Player.Strength)
			rangedMod := getModifier(g.Player.Dexterity)
			text.Draw(screen, fmt.Sprintf("Melee Atk: +%d (1d8%+d)", g.Player.ProficiencyBonus+meleeMod, meleeMod), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
			lineNum++
			text.Draw(screen, fmt.Sprintf("Ranged Atk: +%d (1d6%+d)", g.Player.ProficiencyBonus+rangedMod, rangedMod), basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorWhite)
			lineNum++
		}
		lineNum = max(lineNum, 11)
		text.Draw(screen, "Class Features:", basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorGray)
		lineNum++
		classDef := ClassDefinitions[g.Player.Class]
		if classDef != nil {
			featureIDs := make([]string, 0, len(classDef.ClassActions))
			for id := range classDef.ClassActions {
				featureIDs = append(featureIDs, id)
			}
			sort.Slice(featureIDs, func(i, j int) bool {
				ca1, ca2 := classDef.ClassActions[featureIDs[i]], classDef.ClassActions[featureIDs[j]]
				if ca1.RequiredLevel != ca2.RequiredLevel {
					return ca1.RequiredLevel < ca2.RequiredLevel
				}
				return ca1.Name < ca2.Name
			})
			for _, id := range featureIDs {
				classAction := classDef.ClassActions[id]
				if g.Player.Level >= classAction.RequiredLevel {
					featureText := fmt.Sprintf(" L%d: %s", classAction.RequiredLevel, classAction.Name)
					if classAction.ResourceType == ResourceClassFeature && classAction.UsesPerRest > 0 {
						uses := 0
						if val, ok := g.Player.ClassResources[id]; ok {
							uses = val
						}
						featureText += fmt.Sprintf(" (%d/%d)", uses, classAction.UsesPerRest)
						if classAction.RefreshesOn == RestTypeLong {
							featureText += " (LR)"
						} else if classAction.RefreshesOn == RestTypeShort {
							featureText += " (SR)"
						} else if classAction.RefreshesOn == RestTypeNever {
							featureText += " (Per Lvl)"
						}
					}
					text.Draw(screen, featureText, basicfont.Face7x13, col1X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
					lineNum++
				}
			}
		}
		if len(g.Player.KnownCantrips) > 0 {
			text.Draw(screen, "Known Cantrips:", basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorGray)
			lineNum++
			for _, cantripID := range g.Player.KnownCantrips {
				spellName := "??"
				if spellDef, exists := ActionTable[cantripID]; exists {
					spellName = spellDef.Name
				}
				text.Draw(screen, fmt.Sprintf(" - %s", spellName), basicfont.Face7x13, col1X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
				lineNum++
			}
		}
		if len(g.Player.KnownSpells) > 0 {
			text.Draw(screen, "Known Spells:", basicfont.Face7x13, col1X, infoStartY+(lineNum*infoLineHeight), colorGray)
			lineNum++
			for _, spellID := range g.Player.KnownSpells {
				spellName := "??"
				if spellDef, exists := ActionTable[spellID]; exists {
					spellName = spellDef.Name
				}
				text.Draw(screen, fmt.Sprintf(" - %s", spellName), basicfont.Face7x13, col1X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
				lineNum++
			}
		}
		lineNum = 0
		text.Draw(screen, "Attributes:", basicfont.Face7x13, col2X, infoStartY+(lineNum*infoLineHeight), colorGray)
		lineNum++
		text.Draw(screen, fmt.Sprintf("STR: %d (%+d)", g.Player.Strength, getModifier(g.Player.Strength)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("DEX: %d (%+d)", g.Player.Dexterity, getModifier(g.Player.Dexterity)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("CON: %d (%+d)", g.Player.Constitution, getModifier(g.Player.Constitution)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("INT: %d (%+d)", g.Player.Intelligence, getModifier(g.Player.Intelligence)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("WIS: %d (%+d)", g.Player.Wisdom, getModifier(g.Player.Wisdom)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
		text.Draw(screen, fmt.Sprintf("CHA: %d (%+d)", g.Player.Charisma, getModifier(g.Player.Charisma)), basicfont.Face7x13, col2X+5, infoStartY+(lineNum*infoLineHeight), colorWhite)
		lineNum++
	}

	logLineHeight := 13
	logStartY := screenHeight - (combatLogLength * logLineHeight) - 10
	logX := 10
	if g.CombatLog != nil {
		for i, msg := range g.CombatLog {
			isRestPrompt, isLevelUpMsg, isReactionPrompt := false, false, false
			if g.Player != nil {
				isRestPrompt = g.InputMode == InputModeRestPrompt && i == len(g.CombatLog)-1 && msg == fmt.Sprintf("Wave Cleared! Spend 1 Hit Die (of %d) to heal? [Y/N]", g.Player.HitDice)
				levelUpMsgPattern := "LEVEL UP! Reached Level"
				isLevelUpMsg = g.InputMode == InputModeLevelUp && i == len(g.CombatLog)-1 && (strings.HasPrefix(msg, levelUpMsgPattern) || strings.HasPrefix(msg, "Learned") || strings.HasPrefix(msg, "Chosen"))
			}
			isReactionPrompt = g.InputMode == InputModeReactionPrompt && i == len(g.CombatLog)-1
			var msgColor color.Color = colorWhite
			if isRestPrompt || isLevelUpMsg || isReactionPrompt {
				msgColor = colorYellow
			}
			text.Draw(screen, msg, basicfont.Face7x13, logX, logStartY+(i*logLineHeight), msgColor)
		}
	}

	for _, ft := range g.FloatingTexts {
		alpha := uint8(255 * (float64(ft.Life) / float64(ft.MaxLife)))
		nrgbaColor := color.NRGBAModel.Convert(ft.Color).(color.NRGBA)
		finalColor := nrgbaColor
		finalColor.A = alpha
		shadowColor := color.NRGBA{R: 0, G: 0, B: 0, A: alpha / 2}
		bounds := text.BoundString(basicfont.Face7x13, ft.Text)
		textX, textY := int(ft.X)-bounds.Dx()/2, int(ft.Y)
		text.Draw(screen, ft.Text, basicfont.Face7x13, textX+1, textY+1, shadowColor)
		text.Draw(screen, ft.Text, basicfont.Face7x13, textX, textY, finalColor)
	}

	if g.InputMode == InputModeLevelUp && g.Player != nil {
		menuW, menuH := screenWidth/2, screenHeight/3+20
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 20, G: 30, B: 20, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, colorWhite, false)

		levelUpTitle := fmt.Sprintf("Level %d Reached!", g.Player.Level)
		titleBounds := text.BoundString(basicfont.Face7x13, levelUpTitle)
		titleX := menuX + (menuW-titleBounds.Dx())/2
		titleY := menuY + 15
		text.Draw(screen, levelUpTitle, basicfont.Face7x13, titleX, titleY, colorWhite)
		lineStartY := titleY + 25
		lineNum := 0
		lineSpacing := 15
		choiceX := titleX

		choicePending := false
		if g.Player.Class == "Mage" {
			if g.Player.Level == 2 && len(g.Player.KnownSpells) < 3 {
				choicePending = true
				spellChoiceText := "Choose L1 Spell:"
				text.Draw(screen, spellChoiceText, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				level2Spells := []string{"arcane_blink", "burning_hands", "frost_nova", "mind_spike"}
				for i, spellID := range level2Spells {
					spellName := ActionTable[spellID].Name
					choiceLine := fmt.Sprintf("[%d] %s", i+1, spellName)
					text.Draw(screen, choiceLine, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			} else if g.Player.Level == 3 && len(g.Player.KnownCantrips) < 2 {
				choicePending = true
				cantripChoiceText := "Choose Cantrip:"
				text.Draw(screen, cantripChoiceText, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				level3Cantrips := []string{"ray_of_frost", "shocking_grasp"}
				for i, cantripID := range level3Cantrips {
					cantripName := ActionTable[cantripID].Name
					choiceLine := fmt.Sprintf("[%d] %s", i+1, cantripName)
					text.Draw(screen, choiceLine, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			} else if g.Player.Level == 4 && len(g.Player.KnownSpells) < 4 {
				choicePending = true
				spellChoiceText := "Choose L1 Spell:"
				text.Draw(screen, spellChoiceText, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				level4Spells := []string{"magic_armor", "elemental_strike", "feather_fall", "expeditious_retreat"}
				for i, spellID := range level4Spells {
					spellName := ActionTable[spellID].Name
					choiceLine := fmt.Sprintf("[%d] %s", i+1, spellName)
					text.Draw(screen, choiceLine, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			}
		} else if g.Player.Class == "Fighter" {
			if g.Player.Level == 3 && g.Player.CombatStyle == "" {
				choicePending = true
				styleChoiceText := "Choose Combat Style:"
				text.Draw(screen, styleChoiceText, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				styles := []string{"Gladiator (+2 Melee Dmg)", "Ranger (+2 Ranged Hit)", "Juggernaut (+2 AC)"}
				for i, styleDesc := range styles {
					choiceLine := fmt.Sprintf("[%d] %s", i+1, styleDesc)
					text.Draw(screen, choiceLine, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			} else if g.Player.Level == 4 && g.Player.CombatTechnique == "" {
				choicePending = true
				techChoiceText := "Choose Combat Technique:"
				text.Draw(screen, techChoiceText, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorYellow)
				lineNum++
				techniques := []string{"Power Attack (-2 Hit/+50% Dmg)", "Defensive Stance (+2 AC/-1 Move)", "Quick Strike (Bonus Atk/Half Dmg)"}
				for i, techDesc := range techniques {
					choiceLine := fmt.Sprintf("[%d] %s", i+1, techDesc)
					text.Draw(screen, choiceLine, basicfont.Face7x13, choiceX, lineStartY+(lineNum*lineSpacing), colorWhite)
					lineNum++
				}
			}
		}

		if !choicePending {
			summaryText := "Check Character Sheet [C] for details."
			summaryBounds := text.BoundString(basicfont.Face7x13, summaryText)
			summaryX := menuX + (menuW-summaryBounds.Dx())/2
			summaryY := titleY + 25
			text.Draw(screen, summaryText, basicfont.Face7x13, summaryX, summaryY, colorGray)
			continueMsg := "Press [Enter] to Continue"
			continueBounds := text.BoundString(basicfont.Face7x13, continueMsg)
			continueX := menuX + (menuW-continueBounds.Dx())/2
			continueY := menuY + menuH - 30
			text.Draw(screen, continueMsg, basicfont.Face7x13, continueX, continueY, colorWhite)
		}
	}

	if g.InputMode == InputModeReactionPrompt {
		menuW, menuH := screenWidth/3, screenHeight/6
		menuX, menuY := (screenWidth-menuW)/2, (screenHeight-menuH)/2
		vector.DrawFilledRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), color.NRGBA{R: 30, G: 30, B: 50, A: 230}, false)
		vector.StrokeRect(screen, float32(menuX), float32(menuY), float32(menuW), float32(menuH), 2, colorWhite, false)
		promptText := "Use Shield Reaction? [Y/N]"
		promptBounds := text.BoundString(basicfont.Face7x13, promptText)
		promptX := menuX + (menuW-promptBounds.Dx())/2
		promptY := menuY + (menuH-promptBounds.Dy())/2
		text.Draw(screen, promptText, basicfont.Face7x13, promptX, promptY, colorYellow)
	}
}

func (g *Game) DrawGameOver(screen *ebiten.Image) {
	gameOverMsg := "GAME OVER"
	if g.isVictory {
		gameOverMsg = "VICTORY!"
	}

	msgFont := basicfont.Face7x13
	bounds := text.BoundString(msgFont, gameOverMsg)
	msgX := (screenWidth - bounds.Dx()) / 2
	msgY := (screenHeight - bounds.Dy()) / 2

	overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	vector.DrawFilledRect(screen, 0, 0, float32(screenWidth), float32(screenHeight), overlayColor, false)

	text.Draw(screen, gameOverMsg, msgFont, msgX+1, msgY+1, colorBlack)
	text.Draw(screen, gameOverMsg, msgFont, msgX, msgY, colorWhite)
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

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	rand.Seed(time.Now().UnixNano())

	game := NewGameInitial()
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Slumb Gate")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
