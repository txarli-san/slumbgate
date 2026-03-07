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
- Skeleton enemy packs spawning in rooms (Minion/Warrior/Rogue/Mage)
- Combat trigger on threat encounter with initiative rolls
- Basic combat UI: initiative panel, turn cycling, Space to end turn
- 3D rendering with custom GLSL lighting shader
- Camera orbit (right-drag) and zoom (scroll)

### WIP
- Click-to-move in combat (movement points)
- Click-to-attack (d20 resolution)
- Enemy AI turns
- Combat end → back to continuous mode

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
