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

### Fighter (implemented)
- High HP (d10 hit die), heavy armor (AC 13 base), martial weapons
- Recovers class charges on short rest (1 hour) — the **efficient survivor**
- Less dependent on long rest than casters. The backbone of any party.
- **Party role**: Reliable front-liner. Can fight longer before needing a long rest.

**Abilities:**
- L1: Melee Attack, Second Wind (heal 1d10+CON, costs class charge), Dash, Shove
- L2: Action Surge (free action, costs class charge), +1 class charge
- L3: Combat Style choice — Gladiator (+2 melee damage), Ranger (+2 ranged hit), Juggernaut (+2 AC)
- L4: Combat Technique choice — Power Attack (-2 hit, +50% damage), Defensive Stance (+2 AC, -1 move), Quick Strike (bonus action attack, half damage)

### Mage (implemented)
- Low HP (d6 hit die), no armor (AC 12 base), devastating spells
- Spell slots (ClassCharges) recover ONLY on long rest (8 hours)
- Cantrips are free but the power gap between a cantrip and a leveled spell is enormous
- **Party role**: Force multiplier. Turns impossible fights into easy ones — when they have slots. Needs careful resource management.

**Starting kit:** Fire Bolt (cantrip, 1d10+INT, range 7), Magic Missile (spell, 3×1d4+1 auto-hit, range 7)

**Level-up choices:**
- L2: Learn one L1 spell — Arcane Blink, Burning Hands, Frost Nova, or Mind Spike
- L3: Learn one cantrip — Ray of Frost (1d8+INT, range 7) or Shocking Grasp (1d8+INT, melee)
- L4: Learn one L1 spell — Magic Armor, Elemental Strike, Feather Fall, or Expeditious Retreat

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

- World centered at origin, dungeon defined by concentric radii (outer=80, inner=58) with noise-warped irregular boundaries
- 16×16 tile chunks, deterministic from seed
- Outer ring filled with TileSolid (breakable), carved into rooms and corridors
- 16 angular sectors per ring, each with ~6 room clusters connected by MST corridors + 30% extra loops
- Fill rooms placed in empty solid areas to reduce dead space
- Inner area (TileCore) is impenetrable until the right material is found — natural progression gating
- Fog of war: dungeon interior hidden until explored. Line-of-sight visibility (Bresenham)

**Room types** (implemented): Storage, Barracks, Library, Dining, Crypt, Trap, Empty. Each type has a weighted random assignment and a distinct prop palette (wall-adjacent blocking props + center decorative props). 29 prop models total (barrels, bookcases, weapon racks, torches, pillars, tables, etc.).

**Material progression:** Each ring's walls require a specific tool found in the current ring. Stone pickaxe breaks into the outer ring. Better tools found inside break into the next ring. Tool tiers defined (Stone/Iron/Steel Pickaxe) but only Stone Pickaxe is active.

**Wall breaking is a core mechanic, not a shortcut.** Players punch through wherever they want. The dungeon provides content, the player provides topology. A* pathfinding through breakable walls with a +5 cost penalty so entities prefer existing corridors.

## Recruitment

**Start solo, build the party from allies found in the dungeon.**

- Brynn (Fighter, STR 16, DEX 12, CON 14, HP 12, AC 13) starts alone with 5 Rations, 2 Waterskins, 3 Torches.
- First room entered triggers mage rescue event: 2 skeleton minions spawn. Room cleared rescues Elara (Mage, STR 8, DEX 14, INT 16, HP 7, AC 12) with 3 Rations, 1 Waterskin, 1 Torch.
- Future allies found through events deeper in — prisoners, NPCs, survivors.
- Party grows organically through exploration.

## XP and Leveling (implemented)

**5e SRD XP thresholds.** XP awarded per kill, split evenly among allies in combat.

| Enemy Type | CR | XP |
|---|---|---|
| Skeleton Minion | 1/4 | 50 |
| Skeleton Warrior | 1/2 | 100 |
| Skeleton Rogue | 1/2 | 100 |
| Skeleton Mage | 1 | 200 |

Level-up grants: hit die roll + CON mod to MaxHP, +1 hit die, proficiency bonus update, class-specific choices (see Classes).

## Items and Inventory (implemented)

**Per-entity inventory** with 12-slot capacity and item stacking.

Four item categories:
- **Supplies** — Rations (stack 10), Waterskins (stack 5), Torches (stack 5). Have BurnRate field for future passive consumption.
- **Materials** — Stone, Wood, Iron Ore, Bone, Leather Scrap. Reserved for future crafting.
- **Consumables** — Healing Potion (2d4+2 HP), Greater Healing Potion (4d4+4), Scroll of Fire Bolt, Antidote. Items defined but not yet obtainable in-game.
- **Tools** — Stone/Iron/Steel Pickaxe. Tool tier gates wall-breaking per ring.

Transfer function moves items between entity inventories.

## Equipment (implemented)

**4 equipment slots:** MainHand, OffHand, Head, Back. Two-handed weapons block the off-hand slot.

Equipment grants stat bonuses (AC, hit, damage, weapon die override) and controls character model mesh visibility. 15 gear items defined across Fighter (Longsword, Greatsword, Short Sword, Buckler, Kite Shield, Round Shield, Spike Shield, Helmet, Cloak) and Mage (Wand, Staff, Spellbook, Tome of Power, Wizard Hat, Arcane Cloak).

Equipment chest spawns near the starting area. Walking an entity to the chest opens the equipment UI.

## Rest Economy (implemented)

**Short rest** (1 hour = 60 ticks): Spend 1 hit die to heal (roll + CON mod). Fighter recovers class charges. 30% ambush chance inside the dungeon.

**Long rest** (8 hours = 480 ticks): Requires 1 Ration + 1 Waterskin from party inventory. Full HP restore, recover half hit dice (min 1), restore all class charges, reduce exhaustion by 1. Applies to entire party. 30% ambush chance inside dungeon. Advances Day counter.

## Exhaustion (implemented)

Full 5e SRD exhaustion. Gained from time awake without long rest:
- **Threshold:** 960 ticks (16 hours) without a long rest triggers first exhaustion level
- **Interval:** Every 240 ticks (4 hours) after threshold, another level gained
- **Level 2:** Speed halved (skip every other tick)
- **Level 3:** Disadvantage on attacks (d20 take lower)
- **Level 4:** MaxHP halved
- **Level 5:** Speed reduced to 0 (can't move)
- **Level 6:** Death (entity removed via KillEntity)

Long rest removes 1 exhaustion level.

## Enemies (implemented)

**Skeleton packs** in ~50% of rooms. Pack size = room area / 8 (minimum 1).

| Type | Weight | HP | AC | STR | DEX | Speed | Attack | Range |
|---|---|---|---|---|---|---|---|---|
| Minion | 70% | 8 | 11 | 10 | 12 | 4 | 1d4 | 1 (melee) |
| Warrior | 25% | 13 | 13 | 10 | 14 | 4 | 1d6 | 1 (melee) |
| Rogue | 2.5% | 11 | 14 | 8 | 16 | 5 | 1d6 | 1 (melee) |
| Mage | 2.5% | 9 | 12 | 8 | 14 | 4 | 1d8 | 5 (ranged) |

**Enemy AI:** Pathfind toward nearest ally, attack if in range. Pursuit timer of 3 turns — resets on successful attack, decays by 2 if no LOS. When expired, enemy leashes back to spawn position outside initiative.

## Open Questions
- **Biome count and depth.** How many rings? 5 (short campaign)? 10+ (long haul)?
- **How visible is the clock?** Explicit time display? Or felt through consequences (torch dimming, yawning, hunger icon)?
- **What's at the core?** What are they pushing toward? The motivation needs to sustain the whole game.
- **Narrative.** Found journals, NPC encounters, environmental storytelling? Or pure systems?
- **Party size cap.** How many members before it becomes unwieldy? 4-6 feels right for the tactical layer.
