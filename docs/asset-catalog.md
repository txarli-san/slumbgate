# KayKit Asset Catalog & Acquisition Plan

**Author:** Kay Lousberg — [kaylousberg.com](https://kaylousberg.com) / [itch.io](https://kaylousberg.itch.io)
**License:** CC0 (Creative Commons Zero) — free personal & commercial use, no attribution required
**Format:** GLTF/GLB + FBX + OBJ, single gradient atlas texture (1024×1024)
**Last updated:** 2026-03-14

---

## Current Inventory: What We Own & Use

### Models Loaded in Code (12 of 211 files)

| Model | File | Used For |
|-------|------|----------|
| Floor tile | `dungeon/floors/tileBrickB_large.gltf.glb` | Ground tiles |
| Floor cracked A | `dungeon/floors/tileBrickB_largeCrackedA.gltf.glb` | Ground variant |
| Floor cracked B | `dungeon/floors/tileBrickB_largeCrackedB.gltf.glb` | Ground variant |
| Wall | `dungeon/walls/wall.gltf.glb` | All walls |
| Pickaxe | `weapons/axe_common.gltf.glb` | Ground pickup |
| Chest | `loot/chest_common.gltf.glb` | Equipment chest |
| Chest top | `loot/chestTop_common.gltf.glb` | Chest lid |
| Knight (animated) | `characters/animated/Knight.glb` | Brynn (Fighter) |
| Mage (animated) | `characters/animated/Mage.glb` | Elara (Mage) |
| Skeleton Minion (animated) | `characters/animated/Skeleton_Minion.glb` | Enemy mob |
| Skeleton Warrior (animated) | `characters/animated/Skeleton_Warrior.glb` | Enemy mob |
| Skeleton Mage (animated) | `characters/animated/Skeleton_Mage.glb` | Enemy mob |

### Downloaded Packs (already on disk)

| Pack | Location | Models | Status |
|------|----------|--------|--------|
| **Dungeon Remastered** | `assets/models/dungeon/` | ~130 files | 5 used, ~125 unused |
| **Adventurers (free)** | `assets/models/characters/` | 15 static + 2 animated | 2 animated used |
| **Skeletons (free)** | `assets/models/characters/animated/` + `enemies/` | 3 animated + 4 static | 3 animated used |
| **Weapons (from Dungeon pack)** | `assets/models/weapons/` | 27 files | 1 used |
| **Loot (from Dungeon pack)** | `assets/models/loot/` | 27 files | 2 used |

---

## SECTION 1: Assets We Already Have But Don't Use

### 1A. Dungeon Decoration (IMMEDIATE — zero cost, zero download)

These are **already on disk** in `assets/models/dungeon/` and can be loaded right now.

#### Walls & Structural Variety
| Asset | File | Use Case | Priority |
|-------|------|----------|----------|
| Wall broken | `walls/wall_broken.gltf.glb` | Damaged walls, breached walls visual | HIGH |
| Wall with door | `walls/wall_door.gltf.glb` | Room entrances | HIGH |
| Wall with window | `walls/wall_window.gltf.glb` | Variety, LOS mechanic? | MEDIUM |
| Wall gate | `walls/wall_gate.gltf.glb` | Locked passages, ring transitions | MEDIUM |
| Wall gate door | `walls/wall_gateDoor.gltf.glb` | Gated entrances | MEDIUM |
| Wall end | `walls/wall_end.gltf.glb` | Corridor ends | LOW |
| Wall end broken | `walls/wall_end_broken.gltf.glb` | Ruined corridor ends | LOW |
| Wall corner | `walls/wallCorner.gltf.glb` | Corner pieces | MEDIUM |
| Wall intersection | `walls/wallIntersection.gltf.glb` | T/cross junctions | MEDIUM |
| Wall decoration A | `walls/wallDecorationA.gltf.glb` | Wall detail | MEDIUM |
| Wall decoration B | `walls/wallDecorationB.gltf.glb` | Wall detail | MEDIUM |
| Wall split | `walls/wallSplit.gltf.glb` | Branching walls | LOW |
| Pillar | `walls/pillar.gltf.glb` | Room pillars, structural | HIGH |
| Pillar broken | `walls/pillar_broken.gltf.glb` | Ruined rooms | MEDIUM |
| Single wall variants (×10) | `walls/wallSingle_*.gltf.glb` | Freestanding walls, ruins | LOW |

#### Room Props & Furniture
| Asset | File | Use Case | Priority |
|-------|------|----------|----------|
| Barrel | `props/barrel.gltf.glb` | Room dressing, breakable? | HIGH |
| Barrel dark | `props/barrelDark.gltf.glb` | Variant | MEDIUM |
| Crate | `props/crate.gltf.glb` | Room dressing, lootable? | HIGH |
| Crate dark | `props/crateDark.gltf.glb` | Variant | MEDIUM |
| Crate platforms (×3) | `props/cratePlatform_*.gltf.glb` | Stacked crates | LOW |
| Table large | `props/tableLarge.gltf.glb` | Room dressing | HIGH |
| Table medium | `props/tableMedium.gltf.glb` | Room dressing | HIGH |
| Table small | `props/tableSmall.gltf.glb` | Room dressing | MEDIUM |
| Chair | `props/chair.gltf.glb` | Room dressing | MEDIUM |
| Stool | `props/stool.gltf.glb` | Room dressing | MEDIUM |
| Bench | `props/bench.gltf.glb` | Room dressing | MEDIUM |
| Bookcase (×6 variants) | `props/bookcase*.gltf.glb` | Library rooms, lore | HIGH |
| Books (×8 variants) | `props/book*.gltf.glb` | Table dressing, lore | MEDIUM |
| Spell book | `props/spellBook.gltf.glb` | Mage event, loot | HIGH |
| Weapon rack | `props/weaponRack.gltf.glb` | Armory rooms | HIGH |
| Banner | `props/banner.gltf.glb` | Wall/room decoration | MEDIUM |
| Bucket | `props/bucket.gltf.glb` | Room dressing | LOW |
| Pots (×7 variants) | `props/pot*.gltf.glb` | Room dressing | LOW |
| Mug | `props/mug.gltf.glb` | Table dressing | LOW |
| Plate / plate full / plate half | `props/plate*.gltf.glb` | Table dressing | LOW |
| Bricks (rubble) | `props/bricks.gltf.glb` | Broken wall aftermath | HIGH |
| Floor decorations (×6) | `props/floorDecoration_*.gltf.glb` | Floor variety | MEDIUM |

#### Doors & Transitions
| Asset | File | Use Case | Priority |
|-------|------|----------|----------|
| Door | `doors/door.gltf.glb` | Room entrances | HIGH |
| Door gate | `doors/door_gate.gltf.glb` | Locked/special rooms | HIGH |
| Stairs | `doors/stairs.gltf.glb` | Ring transitions, depth | HIGH |
| Stairs wide | `doors/stairs_wide.gltf.glb` | Major transitions | MEDIUM |
| Trapdoor | `doors/trapdoor.gltf.glb` | Hidden passages, events | HIGH |

#### Hazards & Light Sources
| Asset | File | Use Case | Priority |
|-------|------|----------|----------|
| Spike trap | `hazards/tileSpikes.gltf.glb` | Dungeon hazard | HIGH |
| Spike trap large | `hazards/tileSpikes_large.gltf.glb` | Room hazard | MEDIUM |
| Spike trap shallow | `hazards/tileSpikes_shallow.gltf.glb` | Disarmed/partial | MEDIUM |
| Torch (floor) | `hazards/torch.gltf.glb` | Light source prop, light system | HIGH |
| Torch (wall) | `hazards/torchWall.gltf.glb` | Wall lighting | HIGH |

#### Scaffolding (33 pieces)
| Asset | File | Use Case | Priority |
|-------|------|----------|----------|
| Scaffold variants (×33) | `scaffolding/scaffold_*.gltf.glb` | Construction areas, mine shafts, wall-breaking sites | LOW |

#### Additional Floor Variants
| Asset | File | Use Case | Priority |
|-------|------|----------|----------|
| Brick A large/medium/small | `floors/tileBrickA_*.gltf.glb` | Different room themes | MEDIUM |
| Brick B medium/small | `floors/tileBrickB_medium/small.gltf.glb` | Corridor vs room distinction | MEDIUM |

### 1B. Unused Enemies

| Asset | File | Use Case | Priority |
|-------|------|----------|----------|
| **Skeleton Rogue** (animated) | `enemies/Skeleton_Rogue.glb` | **4th skeleton type — already defined in world.go but no model loaded!** | **CRITICAL** |
| Skeleton Rogue (static) | `enemies/Skeleton_Rogue.glb` (331KB) | Same file, needs `loadAnimatedModel` | — |

### 1C. Unused Weapons (all 3 tiers: common/uncommon/rare)

| Weapon | Files | Use Case | Priority |
|--------|-------|----------|----------|
| Sword (×3 tiers) | `weapons/sword_*.gltf.glb` | Fighter/Knight main weapon | Already used via Knight meshes |
| Axe (×3 tiers) | `weapons/axe_*.gltf.glb` | Fighter alt, pickaxe display | 1 used |
| Double axe (×3 tiers) | `weapons/axeDouble_*.gltf.glb` | Two-hand Fighter weapon | MEDIUM |
| Hammer (×3 tiers) | `weapons/hammer_*.gltf.glb` | Cleric weapon | HIGH |
| Dagger (×3 tiers) | `weapons/dagger_*.gltf.glb` | Rogue weapon | HIGH |
| Crossbow (×3 tiers) | `weapons/crossbow_*.gltf.glb` | Rogue ranged weapon | HIGH |
| Staff (×3 tiers) | `weapons/staff_*.gltf.glb` | Mage/Druid weapon | Already used via Mage meshes |
| Shield (×3 tiers) | `weapons/shield_*.gltf.glb` | Fighter/Cleric offhand | Already used via Knight meshes |
| Arrow | `weapons/arrow.gltf.glb` | Projectile visual | MEDIUM |
| Quiver (empty/half/full) | `weapons/quiver_*.gltf.glb` | Rogue gear visual | MEDIUM |

### 1D. Unused Loot & Consumables

| Asset | Files | Use Case | Priority |
|-------|-------|----------|----------|
| Chest uncommon | `loot/chest_uncommon.gltf.glb` | Better loot rooms | HIGH |
| Chest rare | `loot/chest_rare.gltf.glb` | Boss/special rooms | HIGH |
| Chest empty | `loot/chest_common_empty.gltf.glb` | Already-looted chest | HIGH |
| **Mimic chests** (×2) | `loot/chest_*_mimic.gltf.glb` | Trap encounter! | HIGH |
| Potions red (×3 sizes) | `loot/potion*_red.gltf.glb` | Health potions | HIGH |
| Potions blue (×3 sizes) | `loot/potion*_blue.gltf.glb` | Mana potions | HIGH |
| Potions green (×3 sizes) | `loot/potion*_green.gltf.glb` | Antidote/stamina | MEDIUM |
| Coins (×4 variants) | `loot/coin*.gltf.glb` | Currency/loot drops | MEDIUM |
| Loot sacks (×2) | `loot/lootSack*.gltf.glb` | Loot drops | MEDIUM |
| Artifact | `loot/artifact.gltf.glb` | Special quest item, core objective? | HIGH |

### 1E. Unused Character Models (static, not animated)

| Asset | File | Notes |
|-------|------|-------|
| character_rogue.gltf | `characters/` | Static only — need animated version |
| character_barbarian.gltf | `characters/` | Static only — need animated version |
| character_knight.gltf | `characters/` | Static reference (we use animated) |
| character_mage.gltf | `characters/` | Static reference (we use animated) |
| Head variants (×12) | `characters/*Head*.gltf` | Knight/Mage/Rogue/Barbarian heads A/B/C |
| Skull | `characters/skull.gltf.glb` | Death marker, necromancy prop |

---

## SECTION 2: Packs to Download (FREE)

### 2A. Forest Nature Pack — FREE tier

**Source:** [kaylousberg.itch.io/kaykit-forest](https://kaylousberg.itch.io/kaykit-forest)
**Models:** 100+ free (1500+ with paid color variants)
**Cost:** Free

| Category | Content | Use Case | Priority |
|----------|---------|----------|----------|
| Trees (multiple types & sizes) | Oak-like, pine-like, dead trees, saplings | **Outside the dungeon ring** — the world exterior, settlement area, wood gathering | **HIGH** |
| Bushes (multiple types) | Various sizes and shapes | Exterior ground cover, foraging spots | HIGH |
| Rocks (multiple types & sizes) | Boulders, rock clusters, pebbles | **Stone gathering**, exterior terrain, dungeon exterior | **HIGH** |
| Grass (2 styles, clusters) | Grass tufts, patches | Exterior ground cover | MEDIUM |

**Why we need this:**
The area outside the dungeon ring is currently bare. Trees + rocks give us the crafting material sources for the "gather wood/stone → craft pickaxe" loop. Dead trees could mark dungeon entrances.

### 2B. Resource Bits — FREE tier

**Source:** [kaylousberg.itch.io/resource-bits](https://kaylousberg.itch.io/resource-bits)
**Models:** 75+ free
**Cost:** Free

| Category | Content | Use Case | Priority |
|----------|---------|----------|----------|
| Wood | Logs, planks, sticks, bundles | **Crafting material — pickaxe handle, torches** | **CRITICAL** |
| Stone | Stone chunks, blocks, rubble | **Crafting material — pickaxe head, walls** | **CRITICAL** |
| Iron | Ore chunks, ingots, bars | Deeper ring crafting, better tools | HIGH |
| Copper | Ore, ingots | Ring-gated material progression | HIGH |
| Silver | Ore, ingots | Ring-gated material progression | MEDIUM |
| Gold | Ore, ingots | Valuable resource, deeper rings | MEDIUM |
| Textiles | Cloth, leather, rope | **Leather scraps for pickaxe binding!** Armor crafting | **CRITICAL** |
| Fuel | Coal, charcoal, oil | Torch fuel, smelting | HIGH |

**Why we need this:**
This is THE pack for the crafting system. Wood + Stone + Textiles (leather scraps) = craftable pickaxe. The metal ores give material progression for deeper rings. This pack is essential.

### 2C. RPG Tools Bits — FREE tier

**Source:** [kaylousberg.itch.io/rpg-tools-bits](https://kaylousberg.itch.io/rpg-tools-bits)
**Models:** 45+ free (65+ with EXTRA)
**Cost:** Free

| Tool | Use Case | Priority |
|------|----------|----------|
| **Pickaxe** | Craftable pickaxe model (replaces axe_common stand-in) | **CRITICAL** |
| **Hammer** | Cleric weapon, crafting station | HIGH |
| **Axe** | Wood chopping tool, Fighter weapon variant | HIGH |
| **Anvil** | Crafting station prop | HIGH |
| **Lantern** | **Light system — carried light source!** | **CRITICAL** |
| Mallet | Crafting tool | MEDIUM |
| Wrenches | Trap disarming? | LOW |
| Scissors | Textile crafting | LOW |
| Blueprints | Crafting recipes, lore | MEDIUM |
| Handtools (various) | General crafting | LOW |

**EXTRA tier ($4.99)** adds:
| Tool | Use Case | Priority |
|------|----------|----------|
| Fishing rod | Food gathering (if water areas) | LOW |
| Lockpicks | **Rogue class skill — chest/door lockpicking!** | HIGH |
| Locks & Keys | Locked doors/chests mechanic | HIGH |
| Maps & Compass | Navigation aid, minimap item? | MEDIUM |
| Rope | Climbing, utility | MEDIUM |

**Why we need this:**
Proper pickaxe model, lantern for light system, anvil for crafting. The EXTRA tier lockpicks are perfect for Rogue class.

### 2D. Character Animations — FREE

**Source:** [kaylousberg.itch.io/kaykit-character-animations](https://kaylousberg.itch.io/kaykit-character-animations)
**Animations:** 161 (same 41-bone rig as our models)
**Cost:** Free

| Category | Animations | Use Case | Priority |
|----------|------------|----------|----------|
| Sneaking | Sneak walk, crouch | **Rogue class movement** | HIGH |
| Dual wielding | Attack combos | Rogue dual daggers | HIGH |
| Ranged/Bow | Aim, shoot, reload | Crossbow attacks | HIGH |
| Spellcasting | Cast, channel, raise | Already have some, more variety | MEDIUM |
| Blocking | Shield block | Fighter defensive stance | HIGH |
| Crawling | Crawl forward | Exhaustion level 5 (immobile → crawl?) | LOW |
| Digging/Pickaxing | Dig, pickaxe swing | **Wall breaking animation!** | **CRITICAL** |
| Lockpicking | Lockpick action | Rogue skill | HIGH |
| Fishing | Cast, reel | If implemented | LOW |
| Emotes | Wave, cheer, sit, lie | Rest animations, NPC idle | MEDIUM |
| Tool use | Hammering, sawing | Crafting animations | MEDIUM |

**Why we need this:**
161 animations on the same rig = drop-in compatible. Pickaxe swing, sneaking, lockpicking, and blocking are huge gameplay additions.

### 2E. Fantasy Weapons Bits — FREE tier

**Source:** [kaylousberg.itch.io/fantasy-weapons-bits](https://kaylousberg.itch.io/fantasy-weapons-bits)
**Models:** 25+ free
**Cost:** Free

| Category | Content | Use Case | Priority |
|----------|---------|----------|----------|
| Swords | Various styles | Fighter weapon variety | LOW (already have) |
| Axes | Various styles | Fighter/crafting | LOW (already have) |
| Hammers | Warhammer variants | **Cleric weapon** | MEDIUM |
| Bows | Shortbow, longbow | Ranged attacks | MEDIUM |
| Staves/Wands | Caster weapons | LOW (already have) |
| Fist weapons | Gauntlets, claws | Unarmed combat? | LOW |
| Shields | More shield styles | Fighter variety | LOW (already have) |
| Spears | Polearm weapons | Possible Druid/Fighter | MEDIUM |

**EXTRA tier ($4.99)** adds elemental weapon variants (15 models).

---

## SECTION 3: Packs to Purchase

### 3A. Adventurers EXTRA — $7.95

**Source:** [kaylousberg.itch.io/kaykit-adventurers](https://kaylousberg.itch.io/kaykit-adventurers)

| Character | Use Case | Priority |
|-----------|----------|----------|
| **Rogue** (animated) | Rogue class PC — already designed in DESIGN.md | **CRITICAL** |
| **Barbarian** (animated) | Repurpose as **Cleric** (medium armor + hammer/mace) | **HIGH** |
| **Druid** (animated) | Druid class PC — planned class | HIGH |
| Engineer + Turret | Could repurpose — trapper NPC? | LOW |
| 3 alt texture sets | Color variety per character | MEDIUM |

**Note:** The free tier may already include Rogue and Barbarian animated models — need to verify by re-downloading. The static `.gltf` files we have are NOT animated. We need the `.glb` animated versions.

**Decision needed:** Do we get the free tier re-download first to check, or go straight to EXTRA for Druid?

### 3B. Skeletons EXTRA — $7.95

**Source:** [kaylousberg.itch.io/kaykit-skeletons](https://kaylousberg.itch.io/kaykit-skeletons)

| Character | Use Case | Priority |
|-----------|----------|----------|
| **Skeleton Golem** | Boss enemy — room guardian, ring boss | **HIGH** |
| **Necromancer** | Mini-boss — summons minions, event enemy | **HIGH** |
| Alt texture set | Color variants (ice skeletons? fire?) | MEDIUM |

**Why:** Currently all enemies are regular skeletons. A Golem boss and Necromancer add encounter variety and boss fights for the deeper rings.

### 3C. RPG Tools EXTRA — $4.99

Adds lockpicks, locks, keys, fishing rod, maps, compass, rope, and more (20 extra models).

| Notable Items | Use Case | Priority |
|---------------|----------|----------|
| **Lockpicks** | Rogue class mechanic | HIGH |
| **Locks & Keys** | Locked door/chest system | HIGH |
| **Maps** | Found maps revealing dungeon areas? | MEDIUM |
| **Rope** | Utility item | LOW |

### 3D. Resource Bits EXTRA — $3.95

Adds 55+ models: money, gems, containers, food items.

| Category | Use Case | Priority |
|----------|----------|----------|
| **Food items** | **Food/hunger system — core pressure #2!** | **CRITICAL** |
| Gems | Valuable loot, deeper rings | MEDIUM |
| Containers | Storage, bags | MEDIUM |
| Money/coins | Currency (if economy added) | LOW |

### 3E. Forest Nature EXTRA — $4.99

Adds 100+ more models: more trees, more rocks, modular terrain, 8 color variants per model.

| Category | Use Case | Priority |
|----------|----------|----------|
| Modular terrain | Exterior landscape building | MEDIUM |
| More tree types | Biome variety for different rings | MEDIUM |
| 8 color variants | Season/biome theming | MEDIUM |

### 3F. Dungeon Remastered EXTRA — $7.95

Adds 75+ more assets + 6 alternate texture themes.

| Category | Use Case | Priority |
|----------|----------|----------|
| Extra props & pieces | More dungeon variety | MEDIUM |
| **6 texture themes** (Golden, Sepia, B&W, Night) | **Biome ring theming!** Different rings = different palette | **HIGH** |

### 3G. Platformer Pack EXTRA — $9.99

| Category | Use Case | Priority |
|----------|----------|----------|
| Spike traps & spike balls | Dungeon hazards | MEDIUM |
| Sawblades | Mechanical traps | LOW |
| Chains | Dungeon dressing, bridge chains | MEDIUM |
| Cannon | Trap / siege weapon? | LOW |
| Conveyor belts | Puzzle rooms? | LOW |

### 3H. Halloween Bits — FREE + EXTRA

**Source:** [kaylousberg.itch.io/halloween-bits](https://kaylousberg.itch.io/halloween-bits)
**Models:** 60+ free

| Category | Use Case | Priority |
|----------|----------|----------|
| Gravestones | Undead biome, cemetery rooms | MEDIUM |
| Coffins | Skeleton spawn points, loot | HIGH |
| Dead trees | Corrupted areas, deep ring flora | MEDIUM |
| Cobwebs | Dungeon dressing (abandoned rooms) | HIGH |
| Candles | Light sources, altars | HIGH |
| Pumpkins | Seasonal / hidden rooms | LOW |
| Fences | Area boundaries, cemetery | LOW |
| Skulls/bones | Dungeon floor dressing | MEDIUM |

### 3I. Furniture Bits — FREE

**Source:** [kaylousberg.itch.io/furniture-bits](https://kaylousberg.itch.io/furniture-bits)
**Models:** 50+ free

| Category | Use Case | Priority |
|----------|----------|----------|
| Beds | Rest areas, settlement | MEDIUM |
| Shelves/cabinets | Room dressing, loot containers | MEDIUM |
| Desks | Study rooms, lore | LOW |
| Lamps | Light sources | MEDIUM |
| Rugs | Floor variety | LOW |

**Note:** We already have tables, chairs, benches, bookcases in the Dungeon pack. Furniture Bits would add settlement-specific pieces (beds for rest areas).

### 3J. Medieval Hexagon Pack — FREE

**Source:** [kaylousberg.itch.io/kaykit-medieval-hexagon](https://kaylousberg.itch.io/kaykit-medieval-hexagon)
**Models:** 200+ free

| Category | Use Case | Priority |
|----------|----------|----------|
| Buildings (blacksmith, tavern, church, etc.) | **Settlement layer buildings** | HIGH (when settlement implemented) |
| Windmill, watermill, mine, well | Settlement resource buildings | HIGH |
| Trees, rocks, hills, mountains | Overworld terrain | MEDIUM |
| Roads, rivers | Overworld paths | LOW |
| Units & horses (EXTRA $9.99) | NPCs, mounts? | LOW |

**Note:** This is the settlement/strategic layer pack. Not needed now but critical when that system is designed.

---

## SECTION 4: Purchase Summary

### Tier 1: Free Downloads (do now)

| Pack | Models | Use Case |
|------|--------|----------|
| Resource Bits | 75+ | Crafting materials (wood, stone, leather, ore) |
| RPG Tools Bits | 45+ | Pickaxe model, lantern, anvil, tools |
| Forest Nature Pack | 100+ | Trees, rocks, grass for exterior |
| Character Animations | 161 anims | Pickaxe swing, sneak, lockpick, block |
| Fantasy Weapons Bits | 25+ | Additional weapon variety |
| Halloween Bits | 60+ | Cobwebs, candles, coffins, gravestones |
| Furniture Bits | 50+ | Beds, shelves, lamps |
| **Total** | **~516+ models, 161 anims** | **$0** |

### Tier 2: Essential Purchases

| Pack | Cost | Why |
|------|------|-----|
| Adventurers EXTRA | $7.95 | Animated Rogue + Barbarian(→Cleric) + Druid |
| Skeletons EXTRA | $7.95 | Skeleton Golem (boss) + Necromancer |
| Resource Bits EXTRA | $3.95 | Food items (hunger system) |
| RPG Tools EXTRA | $4.99 | Lockpicks, keys, locks (Rogue mechanics) |
| **Subtotal** | **$24.84** | |

### Tier 3: Nice to Have

| Pack | Cost | Why |
|------|------|-----|
| Dungeon Remastered EXTRA | $7.95 | 6 texture themes for biome rings |
| Forest Nature EXTRA | $4.99 | More trees, modular terrain, color variants |
| Platformer EXTRA | $9.99 | Spike traps, chains, mechanical hazards |
| Medieval Hexagon EXTRA | $9.99 | Settlement units & horses |
| Fantasy Weapons EXTRA | $4.99 | Elemental weapon variants |
| **Subtotal** | **$37.91** | |

### Nuclear Option

| Pack | Cost | Why |
|------|------|-----|
| **The Complete KayKit** | **$150** | Everything current + all future packs forever |

**Total (Tier 1 + 2):** $24.84
**Total (Tier 1 + 2 + 3):** $62.75
**Total (Complete):** $150.00

---

## SECTION 5: Crafting System Asset Mapping

The user's vision: gather wood + stone + leather scraps outside → craft a pickaxe → enter dungeon.

| Crafting Step | Asset Source | Specific Models |
|---------------|-------------|-----------------|
| **Chop tree** | Forest Nature (free) | Tree models + RPG Tools axe |
| **Wood drops** | Resource Bits (free) | Wood logs, planks, sticks |
| **Mine rock** | Forest Nature (free) | Rock models |
| **Stone drops** | Resource Bits (free) | Stone chunks, blocks |
| **Skin leather** | Animals (NOT AVAILABLE from Kay) | Need separate source or placeholder |
| **Leather scraps** | Resource Bits (free) | Textile category (cloth, leather, rope) |
| **Craft pickaxe** | RPG Tools (free) | Proper pickaxe model + anvil for crafting |
| **Craft torch** | RPG Tools (free) + Resource Bits | Stick + fuel → torch (from dungeon/hazards) |
| **Better pickaxe** | Resource Bits (free) | Iron/copper ore → ingot → iron pickaxe |

### Missing: Animals

Kay Lousberg does **NOT** have an animal/creature pack. For animals outside the dungeon (deer, wolves, rabbits for leather), options are:

1. **No animals** — leather scraps found as dungeon loot or from enemy drops (skeletons wear leather armor scraps)
2. **Repurpose** Halloween Bits for hostile wildlife (spooky creatures near dungeon entrance)
3. **Third-party CC0** — look for compatible low-poly animal packs (Quaternius, Kenney, etc.)
4. **Placeholder** — use existing skull model or simplified shapes until proper models found

**Decision needed:** Where does leather come from in the game design? Enemy drops? Foraging? Animals?

---

## SECTION 6: Class-to-Asset Mapping

| Class | Character Model | Weapon Models | Animations Needed |
|-------|----------------|---------------|-------------------|
| **Fighter** (done) | Knight.glb ✅ | Sword, shield, 2H sword ✅ | Melee, block ✅ + blocking (new) |
| **Mage** (done) | Mage.glb ✅ | Staff, wand, spellbook ✅ | Spellcast ✅ |
| **Rogue** (planned) | Need animated Rogue.glb | Dagger ✅, crossbow ✅, quiver ✅ | **Sneak, dual wield, ranged, lockpick** (from Animations pack) |
| **Cleric** (planned) | Barbarian.glb → repurpose | Hammer ✅, shield ✅ | Melee ✅ + spellcast (healing) |
| **Druid** (planned) | Druid.glb (EXTRA $7.95) | Staff ✅ | Spellcast ✅ |

---

## SECTION 7: Enemy Variety Expansion

### Currently Available (on disk)

| Enemy | Model | Status |
|-------|-------|--------|
| Skeleton Minion | ✅ Loaded | Working |
| Skeleton Warrior | ✅ Loaded | Working |
| Skeleton Mage | ✅ Loaded | Working |
| **Skeleton Rogue** | ❌ On disk, not loaded | **Wire up immediately** |

### Available with Purchase (Skeletons EXTRA $7.95)

| Enemy | Model | Use Case |
|-------|-------|----------|
| **Skeleton Golem** | Big bulky boss | Ring guardian, room boss |
| **Necromancer** | Caster boss | Summons minions, deeper ring events |

### Potential Future Enemies (from 2D sprite reference)

The `monsters.png` and `monsters.txt` reference sheets show 60+ creature types that could inform future biome enemies if 3D models are found:
- Orcs (brute, shaman, warrior, chieftain) — orc ring?
- Goblins (scout, archer, warlock, brute) — goblin caves?
- Slimes (small, large, spitter) — sewer/cave biome?
- Undead beyond skeletons (zombie, ghoul, wraith, death knight)
- Beasts (spider, wolf, bat, rat, centipede, worm)
- Mythical (dragon, naga, medusa, golem, centaur)
- Cultists (chaos acolyte, chaos cultist)
- Mushroom folk (spore mushroom, elder spore)

These are 2D references only — no 3D models exist from Kay Lousberg for these. Would need third-party assets or a different art direction for non-skeleton enemies.

---

## Sources

- [Kay Lousberg - itch.io store](https://kaylousberg.itch.io/)
- [KayKit Adventurers](https://kaylousberg.itch.io/kaykit-adventurers)
- [KayKit Skeletons](https://kaylousberg.itch.io/kaykit-skeletons)
- [KayKit Forest Nature](https://kaylousberg.itch.io/kaykit-forest)
- [KayKit Resource Bits](https://kaylousberg.itch.io/resource-bits)
- [KayKit RPG Tools](https://kaylousberg.itch.io/rpg-tools-bits)
- [KayKit Fantasy Weapons](https://kaylousberg.itch.io/fantasy-weapons-bits)
- [KayKit Character Animations](https://kaylousberg.itch.io/kaykit-character-animations)
- [KayKit Dungeon Remastered](https://kaylousberg.itch.io/kaykit-dungeon-remastered)
- [KayKit Halloween Bits](https://kaylousberg.itch.io/halloween-bits)
- [KayKit Furniture Bits](https://kaylousberg.itch.io/furniture-bits)
- [KayKit Medieval Hexagon](https://kaylousberg.itch.io/kaykit-medieval-hexagon)
- [KayKit Platformer Pack](https://kaylousberg.itch.io/kaykit-platformer)
- [The Complete KayKit](https://kaylousberg.itch.io/kaykit-complete)
