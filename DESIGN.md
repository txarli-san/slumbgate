# Coso — Game Design Document

## The Premise

D&D has two systems that every game treats as separate: tactical turn-based combat, and survival (rests, exhaustion, rations, time). Every D&D video game either ignores the survival side (Baldur's Gate 3 — long rest is free) or drops the combat depth for a survival sim.

Nobody has made the d20 rest/time system actually matter.

Slumbgate lives in that gap.

## Core Identity

**A d20 tactical survival management game where time is the real enemy.**

You manage a company of adventurers pushing inward through a massive, hostile, procedurally generated dungeon. The goal: reach the core. The obstacle: everything between you and it, and the fact that your people need to eat, sleep, and not die.

The dungeon is not a series of levels. It's a **physical space** — a concentric world generated from the core outward, divided into biomes. You start at the outermost ring. You find paths, break through walls, settle conquered areas, extract resources, and push the front line deeper.

This is not an adventure. **This is a war of attrition against a living dungeon.**

## The Macro Structure

### The Dungeon

Generated concentrically — the core at the center, biome rings expanding outward. You enter at the outer edge.

- **Non-linear progression.** There is no fixed path. You explore rooms, find passages, break through walls to create routes. The dungeon is a physical space, not a graph.
- **Biomes.** Each ring has a distinct biome with unique enemies, resources, environmental hazards, and materials. The fungal caves, the flooded depths, the forge halls, the bone warrens.
- **Destructible walls.** You can break through to create shortcuts, flanking routes, or connections between settled areas. But breaking a wall takes time, makes noise, and you might not like what's on the other side.

### The Company

You don't control one hero. You manage a **roster of adventurers** — each with their own class, level, stats, equipment, exhaustion, and spell slots.

- **Roster management.** Who goes on the next expedition? Who stays to defend camp? Who's too exhausted to fight? Who gets the last healing potion?
- **Character value.** Each adventurer levels up, acquires gear, develops. Losing a level 4 Fighter with good equipment hurts — that's days of investment gone. But no single death is game over.
- **Game over condition.** The company is wiped out, or can no longer sustain itself (no food, no able bodies, no way forward).

### Settlement

Conquered areas become your foothold. Settlement is a core mechanic, not a side feature.

- **Establishing camps.** Secure a cleared area to create a rest point. Camps need defending — the dungeon doesn't forget you're there.
- **Resource extraction.** Each biome provides unique materials. Fungal biome yields alchemical ingredients. Forge biome yields metals. Crystal caverns yield magical components.
- **Upgrades.** Resources fuel technology, equipment crafting, skill training, fortifications. You need specific biome resources to push into specific deeper areas — natural non-linear gating without artificial locks.
- **Supply lines.** Your camp at the front needs supplies from your settlements behind it. Longer supply lines are harder to defend. Overextension is real.

### Push Forward

The game is designed so that **pushing deeper is always more rewarding than consolidating.**

- Better resources are deeper. You need them to sustain your growing operation.
- Staying at the outer ring means slow starvation — resources are thin, enemies are weak but so is the loot.
- The dungeon slowly reclaims undefended territory. Hold too much with too few, and your perimeter collapses.
- Risk is rewarded. The inner biomes have what you need, but the threats scale to match.

## The Unified Clock

Everything runs on one time track. The clock applies to the **entire company simultaneously.**

| Activity | Time Cost |
|---|---|
| Combat round | 6 seconds |
| Exploring a room | ~5 minutes |
| Searching/looting | ~10 minutes |
| Breaking through a wall | ~1-4 hours |
| Short rest | 1 hour |
| Long rest | 8 hours |
| Traveling between areas | variable |
| Establishing a camp | ~4 hours |
| Fortifying a position | ~8 hours |

This makes d20 durations real:
- Mage Armor lasts 8 hours — strategic decision about when to cast it
- A torch lasts 1 hour — ~12 combat encounters or 1 short rest
- Exhaustion recovers on long rest — 8 hours of real vulnerability
- Spell durations measured in minutes actually span multiple encounters

**While squad A pushes into the next room, squad B is resting in camp — and the camp needs defending.** The clock runs for everyone. A mage who rested 4 hours ago has full slots. A fighter who's been on watch for 16 hours is hitting exhaustion level 1. Scheduling rest rotations is a real management problem.

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
- Going too long without sleep (~16 hours)
- Going too long without food/water
- Forced marches
- Certain enemy effects (cold, poison, drains)
- Taking damage while already exhausted

Exhaustion decreases from:
- Long rest (remove 1 level) — but that's 8 hours
- Food and water (prevents gain, doesn't remove)
- Rare consumables

With a full company, exhaustion is a **roster management problem.** You need fresh bodies to rotate in. Send everyone on one big push and the whole company is exhausted with nobody to stand guard. Keep too many in reserve and you don't have the firepower to clear the next area.

### 2. Resources (The Pack)

Finite, depletable, meaningful — now multiplied across a whole company:

- **Food/Water** — prevents exhaustion gain. Must be found, foraged, or produced at settlements. Feeding 8 adventurers burns through supplies fast.
- **Light sources** — torches, lanterns, magic light. Darkness is mechanically dangerous (disadvantage, can't target at range, enemies get advantage). Light has a time cost.
- **Spell slots** — only recover on long rest. Your mages are on a rotation schedule. You can't send them on every expedition.
- **Hit dice** — spend on short rest to heal. Recover on long rest. The bridge between combat damage and rest economy.
- **Hit points** — don't regenerate passively. Healing requires resources (hit dice, spells, potions, cleric attention).
- **Consumables** — potions, scrolls, bandages. Found or crafted. Never enough for everyone.
- **Biome materials** — specific to each area. Used for crafting, upgrades, fortifications. The reason you push deeper.
- **Equipment** — weapons, armor, tools. Finite, distributable. Who gets the magic sword changes the calculus of who leads the next expedition.

### 3. The World (The Clock)

Time passing has consequences at the company scale:

- **Enemies regroup.** Clear a room, leave, come back later — it might not be empty. The dungeon repopulates undefended areas.
- **Patrols and raids.** Enemies don't just wait in rooms. They probe your defenses, raid supply lines, test your perimeter. Especially when they've been alerted by noise.
- **The dungeon reclaims.** Undefended settled areas slowly revert. Fortifications degrade. Supply caches get raided. Hold too much territory with too few bodies and your front collapses.
- **Environmental cycles.** Temperature shifts, flooding, magical surges. Biome-specific hazards that change over time.
- **Alert escalation.** Combat makes noise. Breaking walls makes noise. The deeper you push, the more the local ecosystem responds. Early areas might be scavenged quietly. Deep areas know you're coming.

The company can never turtle. Standing still costs food. Holding territory costs guards. Pushing forward costs lives. Every decision is a tradeoff.

## The Two Layers

### Strategic Layer (Company Management)

This is where most of the game happens:

- **Roster.** View your company — each adventurer's class, level, HP, exhaustion, equipment, spell slots, conditions. Decide who's fit for duty.
- **Expedition planning.** Assemble a squad from available (non-exhausted, non-injured) members. Assign equipment and consumables. Choose an objective (explore new area, clear a room, establish a camp, break through a wall, defend a position).
- **Settlement management.** Assign idle adventurers to tasks — fortifying, crafting, foraging, resting, standing watch. Manage resource stockpiles.
- **Map overview.** See explored areas, settled areas, threat levels, supply line status, known enemy positions. Plan your next push.
- **Time controls.** Advance time (to let resting characters recover, to wait for a patrol to pass), or respond to events (attack on camp, resource discovery, adventurer reaching a new exhaustion level).

### Tactical Layer (d20 Combat)

When a squad enters combat, the game zooms into the full d20 tactical system:

- **Attack rolls**: d20 + ability mod + proficiency vs AC
- **Damage**: dice + modifiers, weapon-dependent
- **Ability scores**: STR, DEX, CON, INT, WIS, CHA with standard modifiers
- **Action economy**: Action, Bonus Action, Reaction, Movement per turn
- **Conditions**: Stunned, Slowed, Bleeding, Exhausted, etc. with duration tracking
- **Spell slots**: Recovered only on long rest. Every cast is a survival decision.
- **Advantage/Disadvantage**: Core mechanic. Flows from conditions, positioning, exhaustion, darkness, flanking.
- **Critical hits**: Nat 20 doubles damage dice
- **Saving throws**: Ability-based, class proficiencies
- **Reactions**: Opportunity attacks, Shield spell, Riposte

Combat is lethal but fair. A fight against 3 goblins is trivial when your squad is rested and equipped. The same fight is deadly when your Fighter is at exhaustion 3, your Mage has no spell slots, and your Cleric used the last healing potion two rooms ago.

**The same encounter is a completely different experience depending on your survival state.** That's the core design insight. The strategic layer creates the context. The tactical layer plays out the consequences.

## Classes

Designed around the survival tension and company role:

### Fighter
- High HP, heavy armor, martial weapons
- Recovers key abilities on short rest (1 hour) — the **efficient survivor**
- Less dependent on long rest than caster classes. The backbone of any expedition.
- Combat styles create build variety (defensive, ranged, aggressive)
- **Company role**: Reliable. Can go on more expeditions before needing a long rest. Guards, front-liners, the ones you send when you can't afford to spend spell slots.

### Mage
- Low HP, no armor, devastating spells
- Spell slots recover ONLY on long rest (8 hours)
- Cantrips are free but the power gap between a cantrip and a leveled spell is enormous
- Utility spells have survival applications (light, detect traps, feather fall, knock)
- **Company role**: Force multiplier. Turns impossible fights into easy ones — when they have slots. Needs careful scheduling. You don't waste a fully-rested Mage on a patrol.

### Cleric
- Medium HP, medium armor, healing and support spells
- Healing spells reduce the company's consumable drain — one Cleric saves potions for everyone
- Spell slots on long rest, but domain abilities on short rest
- **Company role**: Medic and force sustainer. Keeps the company going longer between full rests. The most strategically valuable class — losing your only Cleric is a crisis.

### Rogue
- Medium HP, light armor, stealth and precision
- No spell slots, no rest dependency. Skills and cunning.
- Stealth to scout without triggering encounters. Trap disarming. Lock picking.
- **Company role**: Scout and infiltrator. Sends ahead to map unknown areas without committing the full squad. Finds the paths so Fighters don't have to break through walls. Saves the company time and resources by avoiding fights entirely.

### Druid
- Medium HP, natural magic, foraging and survival skills
- Spell slots on long rest, but Wild Shape and nature abilities more flexible
- Can forage for food/water, reducing supply drain. Identify safe rest spots. Navigate environmental hazards.
- **Company role**: Expedition sustainer. A squad with a Druid can stay out longer, eat less, and navigate biome hazards better. Particularly valuable in hostile biomes.

## What Makes This Different

| Other Games | Slumbgate |
|---|---|
| Control one hero or fixed party | Manage a company roster |
| Dungeon is a sequence of floors | Dungeon is a physical space you occupy |
| Rest is free/trivial | Rest is a survival event with real cost |
| Combat exists in isolation | Combat is shaped by survival state |
| Resources reset between runs | Resources persist across real time |
| Time is an abstraction | Time is the central mechanic |
| Exhaustion is a footnote | Exhaustion is the death spiral |
| Clear a room, it stays cleared | The dungeon reclaims undefended territory |
| Push to the exit | Settle, extract, advance, hold |

## The Core Loop

```
PLAN → DEPLOY → EXECUTE → RECOVER → MANAGE → PLAN
```

1. **Plan**: Assess company state. Who's rested? What resources do you have? Where are the threats? What's the objective?
2. **Deploy**: Assemble a squad. Equip them. Assign the objective. Send them out.
3. **Execute**: The squad explores, fights (d20 tactical combat), discovers. Time passes for everyone.
4. **Recover**: Squad returns (or doesn't). Wounded need healing. Exhausted need rest. Loot is distributed.
5. **Manage**: Handle camp events. Assign idle members to tasks. Process resources. Deal with threats to your perimeter.
6. **Plan**: The situation has changed. New information from the expedition. New threats. New opportunities. What next?

## Decided: Camera System

Two camera modes, matching the two gameplay layers:

**Strategic view** — 2D map overlay. Shows explored areas, squad positions, settlements, supply lines, threat zones. This is where company management happens. No 3D rendering needed — stylized abstract map.

**Tactical view** — 3D isometric (current spike). Only renders the local area around the active squad. Combat, exploration, moment-to-moment gameplay. You never see the whole dungeon in 3D.

Player flow: strategic map → pick squad/destination → travel (time passes) → tactical 3D for local area → back to strategic.

This solves the scale problem: the dungeon can be massive without the camera needing to show it all.

## Decided: Multiplayer Time Model

**Real-time shared clock. No sync overhead.**

- All players move freely on the map simultaneously
- Global clock ticks forward as anyone acts — standing still still costs time (food, torch, exhaustion)
- No phases, no turn order, no waiting on other players during exploration
- Combat is the only sync point: enters turn-based mode with initiative when a squad engages

This means a player who wanders inefficiently burns real resources. Time pressure is felt per-step, not per-turn. The spike proved this — watching the tick counter go up with every step creates genuine pressure even without survival mechanics wired up yet.

## Decided: Dungeon Generation (Raylib spike)

**Concentric ring structure, chunked world, sector-based rooms.**

- World centered at origin, dungeon defined by concentric radii with noise-warped irregular boundaries
- 16x16 tile chunks, loaded/unloaded by proximity to player. Deterministic from seed — same chunk always generates the same content
- Outer ring is a band between outer and inner radius. Filled with `TileSolid` (breakable wall mass), carved into rooms and corridors by sector-based generation
- 16 angular sectors per ring, each with room clusters connected by corridors. No guaranteed connections between sectors — wall-breaking is the primary way to create paths
- Inner area (`TileCore`) is impenetrable until the player finds the right material to break through — natural progression gating
- Fog of war: dungeon interior hidden until player explores nearby. Ground (outside) always visible

**Material progression:** Each ring's walls require a specific tool/material found in the current ring. Stone pickaxe (found outside) breaks into the outer ring. Better material found inside the outer ring breaks into the next ring. This repeats inward.

**Wall breaking is a core mechanic, not a shortcut.** Players punch through wherever they want. The dungeon provides content, the player provides topology. Breaking walls may lead to rooms, corridors, solid rock, or the wrong side of a threat.

## Decided: Movement & Controls

- WASD screen-relative movement (transformed by camera orbit angle)
- Diagonal movement with corner-cutting prevention and wall sliding
- E to interact/break walls in facing direction
- Right-drag to orbit camera, scroll to zoom (radius + height scale together)

## Open Questions

- **Company size.** How many adventurers? Start with 4, recruit up to 12? Or start solo, build the company from NPCs found in the dungeon?
- **Recruitment.** Where do new adventurers come from? Found as prisoners in the dungeon? Arrive from the surface periodically? Hired with resources?
- **Biome count and depth.** How many rings to the core? 5 biomes (short campaign)? 10+ (long haul)?
- **How visible is the clock?** Explicit time display? Or felt through consequences (torch dimming, character yawning, hunger icon)?
- **Automation level.** Can you auto-resolve easy fights to keep the management pace? Or is every combat hand-played? (Probably: player choice — auto-resolve with risk, or manual for control.)
- **What's at the core?** What are they pushing toward? A MacGuffin? An entity? An answer? The motivation needs to sustain dozens of hours of play.
- **Narrative.** Is there story in the dungeon? Found journals, NPC encounters, environmental storytelling? Or is it pure systems?
