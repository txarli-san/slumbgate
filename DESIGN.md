# Slumbgate — Game Design Document

## The Premise

D&D has two systems that every game treats as separate: tactical turn-based combat, and survival (rests, exhaustion, rations, time). Every D&D video game either ignores the survival side (Baldur's Gate 3 — long rest is free) or drops the combat depth for a survival sim.

Nobody has made the d20 rest/time system actually matter.

Slumbgate lives in that gap.

## Core Identity

**A d20 tactical survival RPG where time is the real enemy.**

You manage a party of adventurers pushing inward through a massive, hostile, procedurally generated concentric dungeon. The goal: reach the core. The obstacle: everything between you and it, and the fact that your people need to eat, sleep, and not die.

The dungeon is not a series of levels. It's a **physical space** — a concentric world generated from the core outward, divided into biomes. You start at the outermost ring. You find paths, break through walls, and push deeper.

## The Macro Structure

### The Dungeon

Generated concentrically — the core at the center, biome rings expanding outward. You enter at the outer edge.

- **Non-linear progression.** There is no fixed path. You explore rooms, find passages, break through walls to create routes. The dungeon is a physical space, not a graph.
- **Biomes.** Each ring has a distinct biome with unique enemies, resources, and environmental hazards. The fungal caves, the flooded depths, the forge halls, the bone warrens.
- **Destructible walls.** You can break through to create shortcuts, flanking routes, or connections between areas. But breaking a wall takes time, makes noise, and you might not like what's on the other side.

### The Party

You start with one hero and recruit allies found in the dungeon. Classic RPG party — each member has their own class, level, stats, equipment, exhaustion, and spell slots.

- **Party management.** Who leads the push? Who's too exhausted to fight? Who gets the healing potion? Who carries the torch?
- **Character value.** Each adventurer levels up, acquires gear, develops. Losing a level 4 Fighter with good equipment hurts — that's hours of investment gone. But no single death is game over.
- **Recruitment.** Allies are found through events deeper in the dungeon — prisoners to rescue, NPCs in trouble. The party grows organically through exploration, not menus.
- **Game over condition.** Total party kill. Everyone's dead.

### Push Forward

The game is designed so that **pushing deeper is always more rewarding than lingering.**

- Better loot and gear are deeper. You need them to survive what comes next.
- Staying in the outer ring means weak enemies but also weak rewards — you'll plateau.
- The dungeon pushes back — rest ambushes, and later: reclamation, patrols, escalation.
- Risk is rewarded. The inner biomes have what you need, but the threats scale to match.

## The Unified Clock

Everything runs on one time track. The clock applies to the **entire party simultaneously.**

| Activity | Time Cost |
|---|---|
| Combat round | 6 seconds |
| Exploring a room | ~5 minutes |
| Breaking through a wall | ~1-4 hours |
| Short rest | 1 hour |
| Long rest | 8 hours |

This makes d20 durations real:
- Mage Armor lasts 8 hours — strategic decision about when to cast it
- A torch lasts 1 hour — ~12 combat encounters or 1 short rest
- Exhaustion recovers on long rest — 8 hours of real vulnerability
- Spell durations measured in minutes actually span multiple encounters

**While your Fighter rests, the Mage has no front-liner.** Scheduling who rests when is a real problem.

## The Three Pressures

### 1. Exhaustion (The Body)

d20 exhaustion rules, played straight:

| Level | Effect |
|---|---|
| 1 | Disadvantage on ability checks |
| 2 | Speed halved |
| 3 | Disadvantage on attacks and saves |
| 4 | Max HP halved |
| 5 | Speed reduced to 0 |
| 6 | Death |

Exhaustion increases from:
- Going too long without sleep (~16 hours of game clock)
- Going too long without food/water
- Certain enemy effects (cold, poison, drains)
- Environmental hazards in deeper biomes

Exhaustion decreases from:
- Long rest (remove 1 level) — but that's 8 hours of vulnerability
- Food and water (prevents gain, doesn't remove)
- Rare consumables found deeper in

With a small party, exhaustion is a **rotation problem.** Push too hard and everyone's at disadvantage. Rest too much and you burn through food. There's a rhythm to find.

### 2. Supplies (The Pack)

Finite, depletable, meaningful — carried by the party:

- **Food/Water** — prevents exhaustion gain. Found in chests, dropped by enemies, scavenged. Running out means the exhaustion clock starts ticking.
- **Light sources** — torches, lanterns, magic light. Darkness is mechanically dangerous (disadvantage, can't target at range, enemies get advantage). Light has a time cost.
- **Spell slots** — only recover on long rest. Your mage is on a budget. Every cast is a survival decision.
- **Hit dice** — spend on short rest to heal. Recover half on long rest. The bridge between combat damage and rest economy.
- **Hit points** — don't regenerate passively. Healing requires resources (hit dice, spells, potions).
- **Consumables** — potions, scrolls, bandages. Found or looted. Never enough.
- **Equipment** — weapons, armor, gear. Found in chests or dropped. Who gets the magic sword changes who leads the next fight.

### 3. The Dungeon (The Clock)

Time passing has consequences:

- **Rest ambushes.** Short resting inside the dungeon has a chance to spawn enemies. The deeper you are, the worse they get. Resting is never free.
- **Enemies regroup.** Clear a room, leave, come back later — it might not be empty. The dungeon repopulates undefended areas over time.
- **Environmental hazards.** Biome-specific dangers that create pressure to keep moving or find safe ground.
- **Alert escalation.** Combat makes noise. Breaking walls makes noise. The deeper you push, the more the local ecosystem responds.

The party can never turtle. Standing still costs food. Resting costs time and risks ambush. Pushing forward costs HP and spell slots. Every decision is a tradeoff.

## The Two Layers

### Strategic Layer (Party Management)

This is where most of the game happens:

- **Party status.** View each member's class, level, HP, exhaustion, equipment, spell slots, conditions. Decide who's fit to push forward.
- **Task assignment.** Select a party member, give orders — scout ahead, move to a location, rest, follow the leader. Point-and-click in the 3D world.
- **Equipment.** Open chests to find gear. Distribute items among party members. Who gets the shield? The helmet? The two-handed sword?
- **Exploration.** Reveal the dungeon room by room. Break walls to create new paths. Find the way deeper.
- **Time management.** Every order costs time. Scout commands, rest periods, wall-breaking — all tick the clock. The clock drives exhaustion, hunger, torch burn, and ambush risk.

### Tactical Layer (d20 Combat)

When the party encounters threats, the game shifts to full d20 tactical combat:

- **Attack rolls**: d20 + ability mod + proficiency + gear bonuses vs AC
- **Damage**: dice + modifiers, weapon-dependent, gear bonuses
- **Action economy**: Movement + Standard Action + Bonus Action per turn
- **Conditions**: Exhausted party members fight at disadvantage — the strategic layer bleeds into combat
- **Spell slots**: Recovered only on long rest. Every cast is a survival decision.
- **Advantage/Disadvantage**: From conditions, exhaustion, darkness, positioning
- **Critical hits**: Nat 20 doubles damage dice

Combat is lethal but fair. A fight against 3 skeletons is trivial when your party is rested and equipped. The same fight is deadly when your Fighter is at exhaustion 3, your Mage has no spell slots, and the last healing potion was two rooms ago.

**The same encounter is a completely different experience depending on your survival state.** That's the core design insight. The strategic layer creates the context. The tactical layer plays out the consequences.

## Classes

### Fighter
- High HP, heavy armor, martial weapons
- Recovers key abilities on short rest (1 hour) — the **efficient survivor**
- Less dependent on long rest than casters. The backbone of any party.
- Combat styles at level 3 (Gladiator, Ranger, Juggernaut), techniques at level 4 (Power Attack, Defensive Stance, Quick Strike)
- **Party role**: Reliable front-liner. Can fight longer before needing a long rest.

### Mage
- Low HP, no armor, devastating spells
- Spell slots recover ONLY on long rest (8 hours)
- Cantrips are free but the power gap between a cantrip and a leveled spell is enormous
- Learns new spells at levels 2 and 4, cantrips at level 3
- **Party role**: Force multiplier. Turns impossible fights into easy ones — when they have slots. Needs careful resource management.

### Cleric (planned)
- Medium HP, medium armor, healing and support spells
- Healing spells reduce consumable drain — one Cleric saves potions for everyone
- **Party role**: Sustainer. Keeps the party going longer between rests. Losing your Cleric is a crisis.

### Rogue (planned)
- Medium HP, light armor, stealth and precision
- No spell slots, no rest dependency
- Stealth to scout without triggering encounters. Trap disarming.
- **Party role**: Scout and infiltrator. Avoids fights entirely. Saves the party time and HP.

## The Core Loop

```
EXPLORE → FIGHT → LOOT → REST → PUSH DEEPER
```

1. **Explore**: Scout the dungeon. Reveal rooms, find paths, break walls. Every step costs time.
2. **Fight**: Encounter threats. D&D tactical combat with the party's current resources and condition.
3. **Loot**: Open chests, collect drops. Distribute gear and supplies among the party.
4. **Rest**: Heal up — but resting costs time, burns food, and risks ambush. Choose when and where carefully.
5. **Push deeper**: The outer ring is running dry. Better loot, harder enemies, new biomes await. Go deeper or stagnate.

## Two Game Modes

**Continuous mode** (default) — 3D isometric view. Point-and-click party management. Click to select members, click ground to move, Tab to cycle, hotkeys for orders (Scout, Stop, Rest, Follow). Camera follows selected member. Time advances per action.

**Combat mode** (triggered by threat encounter) — D&D initiative turns. Action bar with class abilities. Select action → click target → execute. Space to end turn. Initiative order on screen. Movement points per turn. Full d20 resolution. Enemy AI pathfinds, attacks, and leashes back when losing interest. Combat ends on victory or disengage.

Player never controls a "main character" outside combat. You're the party leader, not a hero.

## Action-Driven Clock

**Time only advances when someone acts. No real-time ticking.**

- Each entity action (step, break wall, etc.) costs time
- Global clock driven by entity actions. If everyone is idle, time freezes
- Entities with active tasks (scout, move-to) drive the clock automatically
- Idle entities still experience elapsed time (exhaustion, hunger, torch burn)

This creates tension: every order has weight. Scouting burns time. Resting burns time. Even walking to that chest burns time. The clock is always the enemy.

## Dungeon Generation

**Concentric ring structure, chunked world, sector-based rooms.**

- World centered at origin, dungeon defined by concentric radii with noise-warped irregular boundaries
- 16×16 tile chunks, deterministic from seed
- Outer ring filled with TileSolid (breakable), carved into rooms and corridors
- 16 angular sectors per ring, each with room clusters connected by corridors
- Inner area (TileCore) is impenetrable until the right material is found — natural progression gating
- Fog of war: dungeon interior hidden until explored. Line-of-sight visibility (Bresenham)

**Material progression:** Each ring's walls require a specific tool found in the current ring. Stone pickaxe breaks into the outer ring. Better tools found inside break into the next ring.

**Wall breaking is a core mechanic, not a shortcut.** Players punch through wherever they want. The dungeon provides content, the player provides topology.

## Recruitment

**Start solo, build the party from allies found in the dungeon.**

- Brynn (Fighter) starts alone. First room cleared rescues Elara (Mage).
- Future allies found through events deeper in — prisoners, NPCs, survivors.
- Party grows organically through exploration.

## Open Questions
- **Biome count and depth.** How many rings? 5 (short campaign)? 10+ (long haul)?
- **How visible is the clock?** Explicit time display? Or felt through consequences (torch dimming, yawning, hunger icon)?
- **What's at the core?** What are they pushing toward? The motivation needs to sustain the whole game.
- **Narrative.** Found journals, NPC encounters, environmental storytelling? Or pure systems?
- **Party size cap.** How many members before it becomes unwieldy? 4-6 feels right for the tactical layer.
