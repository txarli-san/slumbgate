package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type Condition struct {
	Name     string
	Duration int
	Source   *Entity
	Data     map[string]any
}

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
	Conditions      []Condition
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

type EnemyTurnContext struct {
	Index int
	Phase EnemyTurnPhase
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
	PendingQuit               bool
	currentEnemyTurn          EnemyTurnContext
	enemiesActedThisTurn      []bool
	playerMovePending         bool
	pendingMoveStartX         int
	pendingMoveStartY         int
	pendingMoveTargetX        int
	pendingMoveTargetY        int
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
