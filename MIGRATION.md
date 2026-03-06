# Ebiten → Raylib Migration Plan

## Goal

Replace the Ebiten 2D renderer with Raylib-go 3D renderer using KayKit Dungeon Pack models.
Keep all game logic intact. Clean separation between logic and rendering.

## Current Coupling Points

Three things tie game logic to Ebiten:

1. **Entity struct has rendering fields** — `Sprite *ebiten.Image`, `DrawOpts`, `CurrentAlpha`, `AttackBumpTimer`
2. **Game struct mixes concerns** — holds both game state (`Enemies`, `CombatLog`) and rendering state (`FloatingTexts`, `IconFont`, `ActionButtons`)
3. **game.go Init** loads sprites alongside game state setup

## Architecture

Two layers, one-directional data flow:

```
Input (Raylib) → Intents → Game Logic → Game State → Renderer (Raylib)
```

### Game Logic Layer (zero rendering imports)

- Owns all state: player, enemies, map, waves, combat log, turns, conditions
- Entities identify their visual by a **string model ID** (e.g. `"knight"`, `"goblin_scout"`) — not a pointer to any rendering object
- Visual feedback expressed as **events**, not rendering state. Instead of `AttackBumpTimer = 10`, logic emits `{Type: "attack_bump", EntityID: 3}`. Renderer decides what that looks like.
- Exposes: `ProcessIntent(intent)`, `Tick()`, and readable state fields
- `FloatingTexts`, `CurrentAlpha`, `AttackBumpTimer` move out to renderer

### Renderer Layer (Raylib-specific)

- Owns window, camera, loaded models, shader, fonts, animations
- **Model registry**: maps model IDs → loaded `rl.Model` + Y-offset + scale
- Each frame: reads game state, renders the scene
- Captures mouse/keyboard → creates Intents → pushes to game logic
- Owns all visual-only state: floating text animations, death fades, attack bumps, camera
- Draws HUD, combat log, menus, tooltips by reading game state

### Main Loop

```go
for !rl.WindowShouldClose() {
    renderer.HandleInput(gameState)
    gameState.ProcessIntents()
    gameState.Tick()
    renderer.Update()  // animate bumps, fades, camera
    renderer.Draw(gameState)
}
```

## What Stays, What Moves

| Thing | Currently | After |
|---|---|---|
| `Entity.Sprite` | `*ebiten.Image` | `ModelID string` |
| `Entity.DrawOpts` | `ebiten.DrawImageOptions` | removed |
| `Entity.AttackBumpTimer` | on Entity | renderer visual state |
| `Entity.IsDying` / `CurrentAlpha` | on Entity | `IsDying` stays (logic needs it), alpha/fade → renderer |
| `FloatingTexts` | on Game | renderer |
| `CombatLog` | on Game | stays (game state), renderer reads it |
| `ActionButtons` / `IconFont` | on Game | renderer |
| `UIButton` type | in types.go | renderer |
| Intent system | in input.go | stays, input capture swaps Ebiten → Raylib |
| `Game.Update()` | mixes tick + animation | splits: `Tick()` (logic) + renderer animation update |

## Files Impact

### Untouched (pure game logic)
- `combat.go` — attack resolution, spells, actions
- `ai.go` — pathfinding, enemy AI
- `state.go` — turn flow, wave completion, rest
- `data.go` — enemy/class/action definitions (add ModelID fields)
- `constants.go` — enums, conditions, action types
- `utils.go` — math helpers
- `main_test.go` — all tests

### Modified
- `types.go` — remove rendering fields from Entity/Game, add ModelID strings
- `game.go` — extract rendering state out, split Update into logic Tick vs render
- `input.go` — swap Ebiten input capture for Raylib, keep Intent creation

### Replaced
- `ui.go` — fully rewritten as Raylib 3D renderer
- `main.go` — Raylib window init, main loop

## Entity Definition Changes

```go
// Before
type Entity struct {
    X, Y, Width, Height int
    HP, MaxHP, AC       int
    // ... stats ...
    Sprite              *ebiten.Image      // remove
    DrawOpts            ebiten.DrawImageOptions // remove
    AttackBumpTimer     int                // move to renderer
    CurrentAlpha        float64            // move to renderer
    IsDying             bool               // stays
    Conditions          []Condition
}

// After
type Entity struct {
    X, Y, Width, Height int
    HP, MaxHP, AC       int
    // ... stats ...
    ModelID             string             // "knight", "goblin_scout", etc.
    IsDying             bool
    Conditions          []Condition
}
```

## Model Registry (Renderer)

```go
type ModelEntry struct {
    Model   rl.Model
    Scale   float32
    YOffset float32
}

type Renderer struct {
    Models      map[string]ModelEntry
    Camera      rl.Camera3D
    Shader      rl.Shader
    TileUnit    float32
    // visual-only state per entity
    BumpTimers  map[int]int
    FadeAlphas  map[int]float64
    FloatingTexts []*FloatingText
}
```

## Migration Steps

1. Split Entity/Game structs — move rendering fields out
2. Add ModelID strings to entity and enemy definitions
3. Build renderer around the Raylib spike (spike-raylib/)
4. Wire Raylib input capture to existing Intent system
5. Connect game logic to renderer in main loop
6. Port HUD/UI (combat log, status display, menus)
7. Remove Ebiten dependency
