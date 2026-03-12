# Animation System & Asset Details

## Asset Sources
- KayKit Character Pack Adventures (Knight, Mage) — CC0
- KayKit Character Pack Skeletons (Minion, Warrior, Mage) — CC0
- All models: animated GLB, same art style, 41-bone skeleton

## Animated Model Files
- `assets/models/characters/animated/Knight.glb` — 15 meshes, 76 animations
- `assets/models/characters/animated/Mage.glb` — 12 meshes, 76 animations
- `assets/models/characters/animated/Skeleton_Minion.glb` — 9 meshes, 95 animations
- `assets/models/characters/animated/Skeleton_Warrior.glb` — 10 meshes, 95 animations
- `assets/models/characters/animated/Skeleton_Mage.glb` — 9 meshes, 95 animations

## Knight Mesh Map (15 meshes)
| Idx | Node Name | Verts | Category | Bone Weights |
|-----|-----------|-------|----------|-------------|
| 0 | 1H_Sword_Offhand | 359 | Weapon | No |
| 1 | Badge_Shield | 285 | Shield variant | No |
| 2 | Rectangle_Shield | 286 | Shield variant | No |
| 3 | Round_Shield | 322 | Shield variant | No |
| 4 | Spike_Shield | 592 | Shield variant | No |
| 5 | 1H_Sword | 359 | Weapon (main) | No |
| 6 | 2H_Sword | 545 | Weapon (two-hand) | No |
| 7 | Knight_Helmet | 504 | Gear | No |
| 8 | Knight_Cape | 56 | Gear | No |
| 9 | Knight_ArmLeft | 486 | Body | Yes |
| 10 | Knight_ArmRight | 486 | Body | Yes |
| 11 | Knight_Body | 1141 | Body | Yes |
| 12 | Knight_Head | 727 | Body | Yes |
| 13 | Knight_LegLeft | 438 | Body | Yes |
| 14 | Knight_LegRight | 438 | Body | Yes |

NOTE: Mesh indices 0-8 have NO bone weights (static gear, parented to bones).
Meshes 9-14 have bone weights (deformable body). Raylib bug workaround allocates
zeroed boneWeights/boneIds for 0-8 via C.calloc to prevent SIGSEGV.

## Mage Mesh Map (12 meshes)
| Idx | Node Name | Verts | Category | Bone Weights |
|-----|-----------|-------|----------|-------------|
| 0 | Spellbook | 399 | Weapon (closed) | No |
| 1 | Spellbook_open | 418 | Weapon (open) | No |
| 2 | 1H_Wand | 158 | Weapon | No |
| 3 | 2H_Staff | 498 | Weapon | No |
| 4 | Mage_Hat | 402 | Gear | No |
| 5 | Mage_Cape | 56 | Gear | No |
| 6 | Mage_ArmLeft | 330 | Body | Yes |
| 7 | Mage_ArmRight | 330 | Body | Yes |
| 8 | Mage_Body | 1275 | Body | Yes |
| 9 | Mage_Head | 761 | Body | Yes |
| 10 | Mage_LegLeft | 326 | Body | Yes |
| 11 | Mage_LegRight | 326 | Body | Yes |

## Skeleton Minion Mesh Map (9 meshes)
| Idx | Node Name | Verts | Category |
|-----|-----------|-------|----------|
| 0 | Skeleton_Minion_ArmLeft | 406 | Body |
| 1 | Skeleton_Minion_ArmRight | 406 | Body |
| 2 | Skeleton_Minion_Body | 1833 | Body |
| 3 | Skeleton_Minion_Cloak | 344 | Gear |
| 4 | Skeleton_Minion_Eyes | 80 | Body detail |
| 5 | Skeleton_Minion_Head | 595 | Body |
| 6 | Skeleton_Minion_Jaw | 176 | Body detail |
| 7 | Skeleton_Minion_LegLeft | 509 | Body |
| 8 | Skeleton_Minion_LegRight | 509 | Body |

## Skeleton Warrior Mesh Map (10 meshes)
| Idx | Node Name | Verts | Category |
|-----|-----------|-------|----------|
| 0 | Skeleton_Warrior_ArmLeft | 634 | Body |
| 1 | Skeleton_Warrior_Helmet | 1069 | Gear |
| 2 | Skeleton_Warrior_ArmRight | 634 | Body |
| 3 | Skeleton_Warrior_Body | 1273 | Body |
| 4 | Skeleton_Warrior_Cloak | 344 | Gear |
| 5 | Skeleton_Warrior_Eyes | 80 | Body detail |
| 6 | Skeleton_Warrior_Head | 462 | Body |
| 7 | Skeleton_Warrior_Jaw | 209 | Body detail |
| 8 | Skeleton_Warrior_LegLeft | 737 | Body |
| 9 | Skeleton_Warrior_LegRight | 675 | Body |

## Skeleton Mage Mesh Map (9 meshes)
| Idx | Node Name | Verts | Category |
|-----|-----------|-------|----------|
| 0 | Skeleton_Mage_ArmLeft | 544 | Body |
| 1 | Skeleton_Mage_Hat | 340 | Gear |
| 2 | Skeleton_Mage_ArmRight | 544 | Body |
| 3 | Skeleton_Mage_Body | 1093 | Body |
| 4 | Skeleton_Mage_Eyes | 80 | Body detail |
| 5 | Skeleton_Mage_Jaw | 195 | Body detail |
| 6 | Skeleton_Mage_LegLeft | 405 | Body |
| 7 | Skeleton_Mage_LegRight | 405 | Body |
| 8 | Skeleton_Mage_Skull | 450 | Body |

## Gear Bone Bindings

Gear meshes have NO bone weights in the GLTF — they're parented to bone nodes in the scene
hierarchy, not skinned. Raylib's `UpdateModelAnimation` skips them (zeroed calloc'd weights →
`boneWeight == 0.0` → continue, `updated` stays false → GPU vertex buffer never overwritten).
The GPU retains the original vertex data loaded from the GLTF.

**Critical finding:** Raylib bakes node world transforms into mesh vertices at GLTF load time
(rmodels.c line 5437: `cgltf_node_transform_world` → `Vector3Transform` on every vertex).
So gear vertices are already in model space (bind pose), NOT in node-local space.

This means gear only needs the **skinning delta** at draw time:
`boneMatrices[parentBone]` = `animPose * inverse(bindPose)` — moves vertices from bind-pose
model space to current-animation model space.

### Joint Indices (shared skeleton, 41 bones)
| Joint | Bone Name | Used For |
|-------|-----------|----------|
| 3 | chest | Cape |
| 8 | handslot.l | Shields, offhand weapons, spellbooks |
| 13 | handslot.r | Main-hand weapons (sword, wand, staff) |
| 14 | head | Helmet, hat |

### Per-Mesh Parent Bones
**Knight:** 0-4→handslot.l(8), 5-6→handslot.r(13), 7→head(14), 8→chest(3), 9-14→body(-1)
**Mage:** 0-1→handslot.l(8), 2-3→handslot.r(13), 4→head(14), 5→chest(3), 6-11→body(-1)

### DrawFiltered Per-Mesh Transform
```
Body meshes (bone == -1): standard model transform (skinning already applied by UpdateModelAnimation)
Gear meshes (bone >= 0):  modelTransform * boneMatrices[parentBone]  (skinning delta only)
```

## Visual Progression Plan (Knight)
Gear meshes to toggle per level:
- **L1**: body only (meshes 9-14)
- **L2**: + 1H_Sword (5)
- **L3**: + Cape (8)
- **L4**: + Helmet (7) + Round_Shield (3)

## Visual Progression Plan (Mage)
- **L1**: body only (meshes 6-11)
- **L2**: + 1H_Wand (2)
- **L3**: + Cape (5)
- **L4**: + Hat (4) + Spellbook (0)

## Key Animation Clips (shared by all models)
- `Idle` — standing, subtle breathing
- `Idle_Combat` — combat ready stance (used for threats)
- `Walking_A` / `Walking_B` / `Walking_C` — walk cycles
- `Running_A` / `Running_B` — run cycles
- `1H_Melee_Attack_Chop` / `_Slice` / `_Stab` — melee attacks
- `2H_Melee_Attack_Chop` / `_Slice` / `_Spin` — two-hand attacks
- `Spellcast_Shoot` / `Spellcast_Raise` / `Spellcast_Long` — casting
- `Hit_A` / `Hit_B` — taking damage
- `Death_A` / `Death_A_Pose` — death + held pose
- `Dodge_Right` / `Dodge_Left` / `Dodge_Backward` — dodge
- `Interact` / `Pickup` / `Cheer` — utility
- `Jump_Full_Short` / `Jump_Full_Long` — jumps

## Skeleton-Only Animations (95 total, 19 extra)
- `Skeletons_Awaken_Floor` / `Skeletons_Awaken_Standing` — spawn from ground
- `Death_C_Skeletons` — skeleton-specific death (collapse)
- `Spawn_Ground_Skeletons` / `Spawn_Air_Skeletons` — spawn variants
- `Walking_D_Skeletons` — shambling skeleton walk

## Technical Notes
- Custom GLSL shader must NOT be applied to animated models (SIGSEGV on UpdateModelAnimation)
- AnimatedModel.Model is *rl.Model (heap pointer) to prevent C pointer invalidation
- loadAnimatedModel allocates zeroed boneWeights/boneIds via C.calloc for meshes missing them
- Animations play at authored speed (~60fps in GLB) via time accumulator, not frame-per-render
- stepInterval = 0.25s controls both game tick and visual lerp duration
- Threats have PrevX/PrevZ + StepProgress + Moving for visual interpolation (same as entities)

### Raylib Animation Internals (rmodels.c)
- `UpdateModelAnimationBones`: computes `boneMatrices[boneId] = animPose * inverse(bindPose)` per mesh per bone. All meshes get boneMatrices allocated (even gear meshes without real bone weights).
- `UpdateModelAnimation`: zeros `animVertices`, then for each vertex iterates 4 bone influences. If `boneWeight == 0.0f` → skip. For gear meshes (all-zero calloc'd weights), all influences skip, `updated` stays false, `rlUpdateVertexBuffer` is NOT called → GPU retains original vertices.
- `MatrixMultiply(left, right)` computes `right * left` in standard math (column-major storage).
- GLTF loader (`cgltf_node_transform_world`) bakes full node hierarchy transform into vertex positions at load time. Gear mesh vertices arrive in model space, not node-local space.
