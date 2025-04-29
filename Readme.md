# Slumb Gate - Dev Notes / README

## Current Status

Go/Ebiten turn-based dungeon thing. It runs. You move, enemies move, violence happens, stuff dies. Has a wave system, Fighter/Mage classes, leveling, basic rests.

## Code Structure (Current)

Standard Go layout. Files mostly do what they say:
- `main.go`: Entry point.
- `game.go`: Main loop, state, entity/wave management.
- `types.go`: Core structs (Entity, Player, Enemy, Actions...).
- `combat.go`: Attack logic, damage calculation, action handling.
- `ai.go`: Basic enemy pathfinding/attack logic.
- `input.go`: Keyboard/mouse input.
- `ui.go`: Ebiten drawing (map, sprites, HP, log, overlays).
- `data.go`: Enemy/Class/Action definitions.
- `state.go`: Turn management, game state logic (player/enemy/gameover).
- `constants.go`: Sizes, ranges, magic numbers.
- `utils.go`: Helpers (distance, modifiers).
- `main_test.go`: Some pathfinding and class feature tests.

Uses Ebiten. Assets embedded.

## What's Working / Implemented Features

- **Core Loop:** Turn-based player vs enemy waves.
- **Movement:** Grid movement, basic A* step pathfinding for enemies. Player uses arrow keys.
- **Combat:** Melee/ranged attacks, AC vs Attack Roll, Crits/Fumbles. HP bars, damage numbers.
- **Classes:**
    - Fighter: Attacks, Second Wind, Action Surge, L3/L4 Style/Technique choices.
    - Mage: Firebolt, Magic Missile, Shield + unlockable spells/cantrips, Spell Slots, Arcane Recovery.
- **Actions:** Standard/Bonus/Reaction/Free action types tracked. Dash, Disengage.
- **Enemy AI:** Minimal. Move-to-attack (melee) or maintain-range (ranged). Basic retreat logic. No special abilities used.
- **Waves:** Defined enemy groups spawn sequentially. Boss waves give level ups.
- **UI:** Basic combat log, player status display, Tab for action menu, C for character sheet.
- **State:** Game Over on death, Victory on clearing waves (theoretically), Short Rest prompts, Long Rest after level ups.
- **Effects:** Shield (+AC), Slow (from Orc Shaman), Feather Fall (dodge chance), Magic Armor (+AC), Expeditious Retreat (+Move) implemented.

## Known Issues / Bugs / Rough Edges

- **Mage Shield:** Has a weird behavior while trying to move out of mele range, need test and fix.
- **AI is Primitive:** Enemies just follow simple rules. Easily exploitable. No real tactics. Pathfinding surely breaks on weird maps.
- **Thin Content:** Only 2 classes playable. Enemy variety is mostly stat-based. Needs more meaningful differences.
- **Unbalanced:** Combat difficulty is completely un-tuned. Probably swings wildly.
- **Barebones UI:** Functional but ugly. But hey! is our ugly...
- **Error Handling:** Likely crashes on unexpected input or state. no refunds.
- **Hardcoded Data:** Waves, spawn points are just lists in code. Not flexible atm.

## Potential Next Steps / Ideas

- **Enemy Abilities:** Top priority. Implement OnHit/OnDeath behaviors. Give enemies unique actions instead of just different stats. Should make combat way less static.
- **Balance Pass:** Play it. Tweak numbers until it feels right. Earned difficulty, not just stat bloat.
- **More Content:** Add classes, enemies with distinct behaviors, more spells/items. Expand the toybox.
- **Visuals:** Maybe later. Better UI, sprites, effects. Not the main point.
- **Map Generation/Variety:** Static 10x10 grid gets old fast. Needs procedural maps or at least more layouts.

## Build / Run

It's Go.

### Needs Go installed
```bash
go run .
```

## Or build it

## WIN
```bash
GOOS=windows GOARCH=amd64 go build -o slumbgate.exe .
```
## MAC
```bash
GOOS=darwin GOARCH=arm64 go build -o slumbgate .
```
## Linux
```bash
GOOS=linux GOARCH=amd64 go build -o slumbgate .
```

## Assets
Shout-out to sethbb.itch.io/32rogues
