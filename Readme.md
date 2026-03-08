# Slumbgate

A d20 tactical survival management game where time is the real enemy.

You manage a company of adventurers pushing inward through a massive, procedurally generated concentric dungeon. Explore, break walls, fight skeletons, and try not to get everyone killed.

## Current Status

Go + Raylib-go 3D game. Two modes: continuous (point-and-click company management) and combat (D&D initiative turns). The dungeon generates from a seed, entities scout autonomously, and combat triggers when threats are encountered.

### What's working
- Concentric ring dungeon with noise-warped boundaries, rooms, corridors
- Chunked world (16x16 tiles), fog of war with line-of-sight
- Company of entities with tier system (Recruit → Lieutenant)
- Point-and-click: select entities, click to move, Tab to cycle
- Orders: 1=Scout (autonomous perimeter patrol), 2=Stop
- Action-driven clock — time only advances when entities act
- Pickaxe discovery and wall-breaking (A* pathfinding through walls)
- Manual wall breaking — click a wall to path there and break through
- Skeleton enemy packs spawning in rooms (Minion/Warrior/Rogue/Mage)
- Combat: initiative, action bar, movement points, d20 attack resolution
- Fighter action system: Melee Attack, Second Wind, Dash, Shove, Action Surge
- Prime→target flow: select action → click target → execute
- Enemy AI: pathfinding toward allies, melee attacks, pursuit/leash system
- Pursuit leash: enemies give up chase after 3 turns, walk back to spawn
- Event system: trigger→effect events (room entered, room cleared)
- Mage rescue: first room cleared spawns Elara the Mage as ally
- Combat aggro requires line-of-sight + proximity (4 tiles)
- 3D rendering with custom GLSL lighting shader
- Camera orbit (right-drag) and zoom (scroll)

### WIP
- Mage class actions
- Click action bar buttons (currently hotkey-only)
- Entity death handling
- Map generation improvements (room connectivity)

## Code Structure

5 files, one domain each:
- `main.go` — entry point, window, models, input, main loop
- `render.go` — shaders, 3D drawing, HUD/UI
- `world.go` — dungeon generation, chunks, threats, fog of war
- `game.go` — game state, entities, tasks, pathfinding, tick logic
- `combat.go` — combat state, initiative, turns, dice

## Build / Run

Requires Go and Raylib dependencies.

```bash
go run .
```

## Assets

- Environment: [KayKit Dungeon Pack](https://kaylousberg.itch.io/kaykit-dungeon)
- Enemies: [KayKit Skeletons](https://kaylousberg.itch.io/kaykit-skeletons)

## Legal — Third-Party Content

### Dungeons & Dragons SRD 5.1

This work includes material taken from the System Reference Document 5.1
("SRD 5.1") by Wizards of the Coast LLC and available at
https://dnd.wizards.com/resources/systems-reference-document. The SRD 5.1 is
licensed under the Creative Commons Attribution 4.0 International License
available at https://creativecommons.org/licenses/by/4.0/legalcode.

A copy of the SRD 5.1 is included in [`docs/SRD_CC_v5.1.pdf`](docs/SRD_CC_v5.1.pdf)
for reference.

Slumbgate uses game mechanics from the SRD including but not limited to:
ability scores, combat rules, hit dice, spell slots, rest mechanics, and
creature stat blocks. All such content is used under the CC-BY-4.0 license.

Dungeons & Dragons, D&D, and the dragon ampersand are registered trademarks
of Wizards of the Coast LLC. This project is not affiliated with, endorsed,
sponsored, or specifically approved by Wizards of the Coast LLC.
