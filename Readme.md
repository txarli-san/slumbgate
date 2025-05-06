# Slumb Gate - Dev Notes / README

![slumb-gate](https://github.com/user-attachments/assets/4a652a0b-5798-47b8-907c-4afd68fdeb77)

## Current Status

Go/Ebiten turn-based dungeon thing. It runs. You move, enemies move, violence happens, stuff dies. Has a wave system, Fighter/Mage classes, leveling, basic rests.

## Code Structure (Current)

Standard Go layout. Files mostly do what they say:
- `main.go`: Entry point.
- `game.go`: Main loop, state, entity/wave management, **Intent handling**, action building, button init.
- `types.go`: Core structs (Entity, Player, Enemy, Condition, Actions, **Intents**, **UIButtons**...).
- `combat.go`: Attack logic, damage calculation, condition application, action handling.
- `ai.go`: Enemy pathfinding/attack logic (A*, step-based), stun check. Player pathfinding (A*).
- `input.go`: Mouse/keyboard input processing (generates Intents), technique choices. **(Keyboard movement/action keys removed)**.
- `ui.go`: Ebiten drawing (map, sprites, HP, log, overlays, status, **UI Bar/Buttons**, tooltips). Handles **font loading**.
- `data.go`: Enemy/Class/Action definitions.
- `state.go`: Turn management, game state logic (player/enemy/gameover), condition ticks.
- `constants.go`: Sizes, ranges, magic numbers, condition names, **Intent types**, **UI constants**, **Icon font mapping**.
- `utils.go`: Helpers (distance, modifiers, saves, condition management).
- `main_test.go`: Pathfinding, class feature, and condition tests.

Uses Ebiten. Assets embedded (including icon font).

## What's Working / Implemented Features

- **Core Loop:** Turn-based player vs enemy waves.
- **Input System:** Refactored to handle player input via an **Intent queue** (`IntentMove`, `IntentAction`, `IntentEndTurn`, `IntentCancelAction`). Primarily mouse-driven now.
- **Movement:** Player uses **mouse click-to-move**. Uses **A* pathfinding** to generate path. Actor follows path step-by-step each turn update. Enemies use basic A* step pathfinding. **AoO logic triggered correctly during path movement**.
- **Combat:** Melee/ranged attacks, AC vs Attack Roll, Crits/Fumbles. HP bars, damage numbers. Bleeding condition applied on critical hits. Saving throw mechanic implemented.
- **Classes:**
    - Fighter: Attacks, Second Wind, Action Surge, L3 Style & L4 Technique choices (Stunning Strike, Defensive Stance, Quick Strike). Base +2 AC bonus.
    - Mage: Firebolt, Magic Missile, Shield + unlockable spells/cantrips (Ray of Frost w/ Slow, Shocking Grasp w/ NoReactions), Spell Slots, Arcane Recovery.
- **Actions:** Standard/Bonus/Reaction/Free action types tracked. Dash, Disengage. Stunning Strike technique for Fighters. Actions are now **primed/executed via UI buttons**.
- **Conditions System:** Refactored status effects into a core system handling duration and data.
    - **Implemented Conditions:** Slowed (Movement halved), Shielded (+5 AC, 1 turn), MagicArmor (+2 AC, 3 turns), FeatherFall (50% dodge chance), ExpeditiousRetreat (+3 Move), NoReactions (Prevents AoO), Bleeding (1d4 DoT), Stunned (Skip Turn).
- **Enemy AI:** Minimal. Move-to-attack (melee) or maintain-range (ranged). Basic retreat logic. Skips turn if Stunned. No special abilities used yet. Does not use AoO if target applied NoReactions.
- **Waves:** Defined enemy groups spawn sequentially. Boss waves give level ups.
- **UI:**
    - Basic combat log, player status display (shows active conditions), C for character sheet. Effective AC displayed.
    - **Bottom UI action bar implemented.**
    - **Clickable buttons** for available player actions and ending turn.
    - Buttons use **icons rendered from a dingbat font** (`FantasyRPGDings`).
    - **Tooltips** display on button hover.
    - Dedicated **Quit button** added.
- **State:** Game Over on death, Victory on clearing waves (theoretically), Short Rest prompts, Long Rest after level ups. Conditions tick down each turn.

## Known Issues / Bugs / Rough Edges

- **AI is Primitive:** Enemies just follow simple rules. Easily exploitable. No real tactics or use of conditions/saves. Pathfinding surely breaks on weird maps.
- **Thin Content:** Only 2 classes playable. Enemy variety is mostly stat-based. Needs more meaningful differences and abilities.
- **Unbalanced:** Combat difficulty is completely un-tuned. Probably swings wildly. Bleed/Stun values likely need adjustment.
- **Barebones UI:** Action buttons are functional with font icons, but icon clarity/choice/rendering could be improved. Enemy conditions not displayed on map. Tooltips functional but basic.
- **Error Handling:** Likely crashes on unexpected input or state. no refunds.
- **Hardcoded Data:** Waves, spawn points are just lists in code. Not flexible atm. Condition damage dice (e.g., Bleed 1d4) hardcoded in `TickConditions`. Icon mapping (`actionIconMap`) hardcoded in `constants.go`.

## Potential Next Steps / Ideas

- **Enemy Abilities & AI:** Top priority. Implement OnHit/OnDeath behaviors. Give enemies unique actions (that potentially use conditions/saves) instead of just different stats. Make AI smarter about conditions.
- **Balance Pass:** Play it. Tweak numbers (HP, AC, damage, condition durations/saves/damage) until it feels right. Earned difficulty, not just stat bloat.
- **More Content:** Add classes, enemies with distinct behaviors, more spells/items/conditions. Expand the toybox.
- **Visuals / UI Refinement:**
    - Improve action button icon clarity (better mapping, maybe custom bitmap icons later).
    - Add targeting overlay/cursor changes for primed actions.
    * Add condition icons on map for player/enemies.
    * General UI polish.
- **Map Generation/Variety:** Static 10x10 grid gets old fast. Needs procedural maps or at least more layouts.
- **Refactor `TickConditions`?:** Current DoT logic is basic. Might need refactoring if more complex damage types or interactions are added. Passing `*Game` to `utils.go` isn't ideal long-term.

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
