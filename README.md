# Geyser-Go

Geyser-Go is a Go implementation of the Geyser Bedrock-to-Java bridge. It is
separate from Cinnabar and is being built as a reusable library first, with a
small standalone command for local interoperability testing.

The project target is version-matched Geyser parity. The current branch is an
incomplete foundation: it has the Java packet framing/session boundary and the
Gophertunnel Bedrock ownership boundary, but it does not yet claim gameplay or
protocol parity.

## Current architecture

- `java/protocol` owns bounded Java TCP framing, VarInt/VarLong and primitive
  codecs, zlib packet compression, AES/CFB8 stream encryption, and the initial
  handshake/login packet shapes.
- `bridge` owns Bedrock listener lifecycle, Java connection ownership, and the
  typed translator/session boundary. The listener uses lunar batch boundaries;
  it deliberately disconnects rather than relaying packets without a
  translator contract.
- `data` and `cmd/registrygen` will own versioned, generated Bedrock/Java
  registries. The current generator reads Cloudburst's complete item and block
  state JSON, preserving every source record (including multiple state hashes
  for one block name). Dragonfly/Lunar data is a source for complete vanilla
  catalogs, not a runtime excuse to omit unsupported blocks or items.
- `translate/chunks.go` decodes Java 1.21.4 paletted chunk sections and emits
  Bedrock subchunk/biome payloads. The generated
  `data/generated_java1214.go` table is built from the matching Geyser mapping
  revision and Cloudburst's Bedrock palette; all Java states currently resolve
  to a versioned Bedrock runtime ID, while type-specific block-entity and
  lighting semantics remain explicitly incomplete.
- The current play translator also forwards versioned single-block changes,
  chunk unloads, time, and the basic non-player entity lifecycle (spawn,
  absolute/relative movement, rotation, and removal). It translates Java
  player-window snapshots and slot updates into Bedrock inventory, armor,
  offhand, and crafting containers, and converts Java system/player/profileless
  chat into Bedrock `Text` packets. Java item and entity registries are
  generated alongside the block table. Java entity velocity, equipment,
  generic metadata, player-info/list records, and player spawns have typed
  translators. Java 1.21.4 block-entity registry records are normalized into
  Bedrock tile-entity NBT in chunk payloads, and standalone Java tile-entity
  updates become `BlockActorData`. Sign and hanging-sign records now translate
  front/back text, dye colors, glow, and wax state. Campfire records now
  translate Java item lists into Bedrock `Item1`-`Item4` compounds through the
  generated complete item registry, including safe custom-data/name/lore
  fields. Beacon effect holders, end-gateway exit arrays, and decorated-pot
  sherd lists also have direct Geyser-compatible projections. Mob spawners
  now project generated Bedrock entity identifiers, bounded timing fields,
  movable state, and the complete versioned Java 1.21.4 display-dimension
  table resolved from Geyser's entity-definition inheritance, with explicit
  omissions for identifiers Geyser does not define on the network.
  Vault block entities project display items and particle range with exact
  Bedrock NBT types. Session-aware standalone and chunk translation resolves
  Java vault UUID arrays against live Bedrock player actor IDs and skips
  unknown or malformed values; the stateless helper deliberately remains
  unable to resolve session actors.
  The generated Java state-name table and bounded recent-state cache now feed
  state-aware banner base/patterns, skull rotation/mouth state, jigsaw joints,
  command conditional mode, and core structure metadata for both chunk and
  standalone block-entity updates; custom skull profiles and the remaining
  type-specific NBT transforms remain open. Java 1.21.4 item slots now have bounded component decoding and project
  common custom NBT, names/lore, durability, enchantments, glint, repair cost,
  dyed colors, and map IDs into Gophertunnel item stacks. Java `textures`
  profile properties now resolve bounded Mojang skin/cape images with slim-arm
  selection and a per-session URL cache; authenticated remote-profile validation
  remains open. Behavior-heavy item components, arbitrary container windows, and richer
  entity-specific behavior are also still open. The bounded entity-metadata
  decoder follows the pinned Java 1.21.4 registry, dropped Java item entities
  use Bedrock's dedicated item-actor packet and typed stack-count updates, and
  paintings use `AddPainting` with negotiated variant order and direction
  offsets. Java item frames and glow item frames use Bedrock frame block states
  plus `BlockActorData`, including facing, item tags, rotation, and cleanup.
  Experience orbs and falling blocks receive their special Bedrock actor
  metadata, including complete Java block-state mapping for falling blocks;
  owned fishing-hook metadata is projected and covered by focused tests.
  Bedrock identifier overrides and special metadata are also projected for
  End Crystals, Area Effect Clouds, primed TNT, and Lightning sounds; wider
  entity metadata coverage remains open.
  Spectral-arrow texture flags, arrow critical/tipped-display metadata, and
  trident critical/enchantment flags are also projected; native projectile
  rendering and collision remain open.
  Throwable Java projectile aliases and Bedrock defaults now cover eggs,
  snowballs, ender pearls, experience bottles, potions, and eyes of ender,
  including half-scale metadata and the short invisible draw window. Fishing
  hook owners and hooked targets resolve through the Java-to-Bedrock actor map;
  PotionContents now project to Bedrock aux/enchanted/lingering metadata,
  bounded firework components are retained, and player-attached fireworks emit
  Bedrock's Elytra boost effect. A bounded 20 Hz simulator now replays the
  Geyser projectile families, including firework acceleration and arrow
  in-ground state. The firework actor display payload (the selected
  Gophertunnel fork does not expose its metadata key), hook casting,
  water/collision/slipperiness integration, and native rendering remain open.
  Java `text_display` and `interaction` entities now use Geyser's
  armor-stand-backed projection with text/name-tag metadata, multiline offset,
  display translation, and interaction size updates. Item/block display
  entities, display transforms, and native rendering/interaction remain open.
  Common Java 1.21.4 living-entity metadata now projects ageable/tameable state,
  sheep colors/shearing, armor-stand low/high flag words, fox/rabbit/bee state,
  tropical-fish packed colors/variants, negotiated cat/wolf variants, and owner
  EIDs for known actors. The pinned entity matrix also covers pose-derived
  states, horse-family fields, aquatic/ambient states, targets, goat horns,
  and stale-flag replacement. Late owner discovery, equipment-driven saddle
  state, custom variant assets, animation timing, and the broader entity matrix
  remain open.
  Java wood/chest boats and minecart variants now normalize to Bedrock's
  vehicle actor families, with bounded boat buoyancy/variant and minecart
  display/damage metadata projection. Java's implicit clientbound vehicle
  movement packet now snaps the Bedrock vehicle carrying the local player,
  preserving the linked rider state; Bedrock vehicle movement prediction,
  paddling, riding offsets, vehicle input, and vehicle-specific container
  behavior remain open.
  Pickup, merge, interaction, and broader entity-specific behavior are still
  open. Common
  Bedrock server-authoritative player-inventory stack requests (take/place/
  swap/drop plus mine-stack validation) now map to Java 1.21.4 hashed
  container-click packets, with cursor/state tracking, typed Bedrock responses,
  and focused simulation tests. The initial mapped menu path now also handles
  bounded take/place/swap/drop requests for supported Java windows. Java 1.21.4
  recipe-book and stonecutter packets now project bounded recipe displays into
  Bedrock `CraftingData` and unlock/remove packets; native recipe-book rendering,
  crafting transactions, custom recipe components, and the full catalog remain
  open.
  Java cursor-item and entity-attribute packets now use typed bounded
  translators, rotation-only player updates reach Bedrock movement state, and
  Java add/remove entity effects reach Bedrock `MobEffect` packets. Common
  Bedrock container-close events are also forwarded to Java; opening and
  synchronizing arbitrary Java windows now has an initial typed path for common
  vanilla menu types, including Java-to-Bedrock IDs, virtual block holders,
  content, and slot updates. Common mapped windows also accept bounded Bedrock
  stack requests and return typed responses; properties, merchant/recipe
  behavior, and exact virtual-holder restoration remain open. Legacy Bedrock
  `InventoryTransaction` block-use, block-break, item-use, and entity
  interact-at/attack packets now map to Java 1.21.4 packets with held-slot
  ordering and sneaking/hit-position preservation; off-hand, spectator, and
  reconciliation behavior remain open.
  Java difficulty/game-state notifications, default spawn position, and title
  text/subtitle/action-bar/timing/clear packets now have bounded typed Bedrock
  paths.
  Java respawn packets now update the initial Bedrock dimension/game-mode path
  with typed `ChangeDimension` and `Respawn` packets.
  Java boss-bar add/remove/health/title/style packets now use a UUID-keyed
  state cache, a Bedrock-compatible invisible backing actor, and typed
  `BossEvent` updates; Java boss-bar flags remain recorded but have no field in
  the current Gophertunnel codec.
  Java positional and entity sound effects now decode the versioned
  `ItemSoundHolder`, apply the generated Geyser playsound mappings, and emit
  typed Bedrock `PlaySound`/`StopSound` packets. Unknown sound registry IDs are
  logged and skipped without tearing down the session.
  Java scoreboard objective/display/score/reset/team packets now have bounded
  decoders and a stateful Bedrock HUD projection: list/sidebar/below-name slots,
  stable fake-player identities, sidebar line limiting, score ordering, and
  team prefix/suffix decoration are covered. Java number formats, scoreboard
  colors, name-tag visibility, collision rules, and exact multi-sidebar-slot
  semantics remain open.
  Java SetPassengers packets now maintain bounded vehicle/passenger state and
  emit typed Bedrock actor links, including rider-versus-additional-passenger
  ordering. Java block-destruction stages now use a position cache to estimate
  Bedrock cracking durations, and the versioned Java world-event table covers
  the common Geyser 1.21.4 sound, block-particle, weather-effect, sculk,
  trial-spawner, and vault mappings; unknown effects remain logged and
  leniently skipped. Full vehicle input, riding offsets, native particle
  rendering, and the remaining world-event mappings are still open.
  Java entity-status events now cover the common hurt/death/taming/attack/
  villager/guardian/firework/wolf/goat mappings, and Java item/experience-orb
  pickup packets now emit typed Bedrock pickup animation or level-event
  packets. Entity-specific side effects and a dedicated live status-event
  fixture remain open.
  Java cooldown packets now have a bounded typed path to Bedrock item
  cooldowns, including Geyser's vanilla shield/goat-horn category aliases;
  a real Paper fixture now delivers the Bedrock start and clear durations;
  active cooldown enforcement remains open.
  Java block events now project chest-like, end-gateway, mob-spawner, and note
  block actions to typed Bedrock block events. Piston animation, bell and
  decorated-pot block-entity effects, and broader block-event coverage remain
  open.
  Java 1.21.4 level-particle packets now decode the bounded particle union and
  project common mapped effects, block-state particles, dust, items,
  vibration, trail, and bounded sample counts to Bedrock level events or
  particle effects. Native visual fidelity and the remaining unmapped
  particles remain open.
  Bedrock Text and CommandRequest chat input now normalizes Java-style
  whitespace, drops empty/oversized input, forwards ordinary messages as the
  unsigned Java chat packet, and forwards slash commands as Geyser's unsigned
  signed-command packet. The command tree, suggestions, secure-chat session,
  and full text-component localization remain open. Simple Java translatable
  components now preserve their key and arguments in Bedrock translation text
  packets; nested components use a bounded plain-text fallback.
  Java 1.21.4 Brigadier `declare_commands` packets now have a bounded decoder
  and a Bedrock `AvailableCommands` projection for top-level literals, common
  argument types, boolean enums, bounded overload traversal, and redirect
  aliases. Nested redirects are followed within the same bounded traversal;
  server-backed suggestions, registry-backed argument enums, and command
  descriptions remain open.
  Java configuration registry_data, feature flags, reset-chat, and tag packets
  are now decoded and retained on the negotiated client. Valid custom
  `dimension_type` entries receive deterministic Bedrock data-driven IDs and
  vertical layouts for joins, respawns, chunks, and unloads; dynamic biome
  registry IDs are projected into Bedrock chunk palettes, while broader
  Bedrock registry projection remains follow-up work.
  Java 1.21.4 resource-pack push/pop packets now have bounded decoders, and
  required packs receive the Java accepted/downloaded/successfully-loaded
  status sequence while optional packs receive declined. Bedrock pack hosting,
  download, cache, stack delivery, and removal remain open.
  Bedrock `PlayerAuthInput` and legacy `MovePlayer` now emit the Java
  position/look packet with Bedrock's eye-height and collision conversion;
  bounded server-authoritative block-break and click-air/block item actions are
  also emitted as Java packets. Bedrock auth-input sprint/sneak/glide edges,
  held-slot changes, arm swings, and basic interact/attack actions now emit
  Java state, held-item, arm-animation, and use-entity packets. Java section
  multi-block updates are emitted as one Bedrock subchunk update. Java
  experience, player-ability, and entity-animation updates now reach the
  Bedrock HUD/player state. Full stack-request validation, target-specific
  entity semantics, vehicle input, and prediction reconciliation remain open.

## Authoritative references

- Geyser: <https://github.com/GeyserMC/Geyser>
- MCProtocolLib: <https://github.com/GeyserMC/MCProtocolLib>
- Gophertunnel fork: <https://github.com/HashimTheArab/gophertunnel/tree/lunar>
- Dragonfly: <https://github.com/df-mc/dragonfly>
- Cloudburst data: <https://github.com/CloudburstMC/Data>
- Lunar all-vanilla tooling: <https://github.com/lunar-bedrock/lunar/tree/main/cmd/bedrockdata-gen>

The initial dependency pin uses `hashimthearab/gophertunnel` `lunar` commit
`60c66ae560608f209f67b7c432cbc5e29e38170a`, which includes the fork's batch
forwarding hooks, declared compression handling, and Snappy implementation.
The generated catalog command was exercised against Cloudburst data commit
`4634cd1725b4fb765c1c4431ae16a0bca77f93d0` and produced 1,933 item records and
16,913 block-state records; the payload checkout remains external to this repo.
The Java 1.21.4 block mapping was generated from Geyser mappings commit
`5d38942`; its committed table records the input hashes and fallback counts.
The sound table was generated from Geyser mappings commit
`47949016f0079578a9e979c93a2b3765354235ea`, with 1,651 Java sound entries and
1,574 playsound mappings.

## Live bootstrap evidence

The initial real-connection gate now passes locally. Paper `1.21.4-232` was run
in offline mode with Temurin Java `21.0.12`; the automated Gophertunnel client
joined through the bridge as Bedrock protocol `1.26.33` and received a complete
1,933-entry item table. A current readback run also observed translated
`InventoryContent`, `InventorySlot`, `Text`, `LevelChunk`, `AddActor`, movement,
block-update, and actor-removal packets before its bounded read window ended.
The installed Bedrock client (`1.26.3301.0`) also joined the same listener
through the native UI, and the Paper log recorded the native player entering
the Java world. A later native add-server retry produced Bedrock's own `U-000`
modal without reaching the listener; it is retained as a UI/environment
limitation rather than a connection or terrain pass. A Snappy-enabled automated
Bedrock probe subsequently sent
auth-input, held-slot, arm-swing, and self-interact packets on listener
`127.0.0.1:19146`; Paper recorded `GeyserHeld` joining and the bridge reported
no translation errors. A second Snappy probe on `127.0.0.1:19148` received
`UpdateAbilities` and `UpdateAttributes` from the Java login path; Paper
recorded `GeyserState` joining cleanly. A fresh Snappy probe on
`127.0.0.1:19151` also observed repeated typed `UpdateAttributes` packets
from Paper with no bridge translation errors. The native capture and probe logs are
intentionally temporary and ignored by Git.

A temporary offline Paper `WindowTest` plugin opened a generic 9x3 chest on
join. The Snappy probe on `127.0.0.1:19153` received the resulting
`ContainerOpen`, `InventoryContent`, and typed `InventorySlot` sequence with no
bridge translation errors. A follow-up probe on `127.0.0.1:19154` requested a
take from the Java chest: Paper accepted the Java `container_click`, and Bedrock
received `ItemStackResponse` status `0` with the chest slot emptied and the
cursor populated. Native UI rendering and broader menu interaction are not yet
acceptance-complete.

The same temporary Paper plugin sent a title, changed the player to creative,
and changed the world difficulty after join. The Snappy probe on
`127.0.0.1:19155` received typed Bedrock title timing/title/subtitle/action-bar
packets, game type `1`, and difficulty `3`; Paper logged a normal disconnect
and the bridge emitted no translation errors.

The same listener also passed a cross-dimension Paper teleport: the probe
received `ChangeDimension` for Bedrock Nether dimension `1`, `Respawn` state
`1`, and the subsequent Java advancement without a bridge error. Custom
dimension registry projection is covered by focused tests; a live custom-world
readback, dimension-specific world settings, and full respawn inventory/entity
reconciliation remain open.

The temporary Paper plugin also created a segmented boss bar and changed its
health, color, and title. The Snappy probe on `127.0.0.1:19155` observed typed
Bedrock boss-bar events `Show`, `HealthPercentage`, `AppearanceProperties`, and
`Title` with a stable synthetic actor ID. The Java flags action is intentionally
leniently recorded because the current Bedrock codec has no darken-sky/music/
fog fields.

The same Paper plugin emitted a positional note sound, an entity level-up
sound, and a stop-sound packet. The Snappy probe observed `note.pling`,
`random.levelup`, and the typed stop event on the same listener with no bridge
translation errors.

The same temporary Paper plugin installed a sidebar objective with two scores
and a team prefix/suffix. The Snappy probe observed Bedrock
`SetDisplayObjective` followed by fake-player score entries for
`[P] First line!` and `Second line`; the Java score ordering update was also
read back without a bridge error. Full scoreboard styling, player-list and
below-name semantics, and native HUD rendering remain open.

The Java cooldown packet has a bounded decoder and typed Bedrock projection,
including the vanilla shield/goat-horn category aliases. The Paper 1.21.4
Bukkit fixture delivered a real Bedrock start duration of 20 ticks followed by
the clear packet; client-side enforcement and visual timing remain open.

Java block-action packets have a bounded decoder and typed Bedrock projections
for chest-like blocks, end gateways, mob spawners, and note blocks. Piston
movement, bell/decorated-pot effects, and a live block-action fixture remain
open.

Java 1.21.4 explosion packets now have a bounded decoder and typed Bedrock
projection for the explosion particle, generic origin event, sound, and player
knockback. The Paper fixture delivered Bedrock generic event `2026`,
`random.explode`, and player motion over Snappy; explosion block-particle lists
and exact motion accumulation remain open.

The Java level-particle path has focused coverage for the 1.21.4 envelope,
block-state payloads, dust/vibration/trail wire variants, unknown-registry
skipping, and common Geyser mapping entries. Dust, item, vibration, trail, and
named-effect variants now have typed Bedrock projections. A real Paper flame
fixture delivered three Bedrock FLAME `LevelEvent` packets (`16392`) through
the Snappy bridge and the bridge remained alive; native visual rendering,
payload fidelity, and the remaining mappings remain open acceptance work.

The same plugin spawned a Pig and mounted the Bedrock player. A fresh Snappy
probe on `127.0.0.1:19155` observed a real `SetActorLink` for the Java vehicle
and Bedrock player, alongside the normal Paper join/leave log. The plugin also
emitted Java block and smoke effects plus two block-cracking stages; the probe
observed typed Bedrock level events `2001`, `2000`, `3600`, and `3602` with no
bridge translation error. This closes only the common actor-link and effect
packet paths; vehicle control, riding offsets, particle visual fidelity, and
full native rendering remain open.

The resource-pack push/pop path is also live-safe: a temporary Paper fixture
scheduled a required pack, and the Snappy Bedrock probe on `127.0.0.1:19158`
joined and survived its bounded read window with no bridge translation error.
The current tranche only acknowledges Java pack requests; it does not yet
deliver the pack to Bedrock or validate native pack rendering.

The command tree projection now also groups Java redirect aliases. The Snappy
probe on `127.0.0.1:19164` received 28 command entries and verified Bedrock
aliases `tell`, `w`, `minecraft:tell`, and `minecraft:w` under `msg`, plus
`tm` under `teammsg`, with no bridge translation error.

Simple Java translatable components now survive the Bedrock boundary with
their key and parameters. The Snappy probe on `127.0.0.1:19163` received
`multiplayer.player.joined` as a Bedrock translation packet with the player
name parameter, and command feedback remained connected with the bounded
plain-text fallback. Locale coverage, styled/nested component fidelity, and
exact Java-to-Bedrock translation-key differences remain open.

This remains an incomplete transport/world tranche: the automated Bedrock
probe receives forwarded Java chunks and the partial play-state updates, while
native terrain rendering, lighting, the remaining type-specific block-entity transforms,
behavior-heavy item components, arbitrary inventory windows, complex transaction state, Java skin fidelity,
entity-specific metadata, item pickup/merge behavior, target-specific interaction semantics, and the rest of
the Geyser gameplay translators are still open acceptance work.

The temporary Paper fixture also sent a Java oak sign and a campfire containing
cod×2 and chain×4. The Snappy probe on `127.0.0.1:19215` observed the translated
sign text/color/glow state, and the probe on `127.0.0.1:19216` observed Bedrock
`Campfire` NBT with `Item1` `minecraft:cod` and `Item4`
`minecraft:iron_chain`; both sessions had zero bridge translation errors.
The campfire item names use the generated Java 1.21.4 registry and complete
Bedrock palette, independent of Dragonfly behavior coverage.
The same fixture emitted beacon, end-gateway, and decorated-pot updates. The
Snappy probe on `127.0.0.1:19218` observed beacon `primary=1` and `secondary=8`,
end-gateway `Age=1234` with `ExitPortal=[12,64,-8]`, and the normalized pot
sherd list, with zero bridge translation errors.
The same fixture emitted a Java zombie spawner. The probe on
`127.0.0.1:19220` observed Bedrock `EntityIdentifier="minecraft:zombie"`
with its delay/count/range fields and no Java `SpawnData` compound; the bridge
reported zero translation errors.

The state-aware fixture placed a red banner with a blue stripe and a player
head. The clean Snappy probe on `127.0.0.1:19223` observed Bedrock `Banner`
NBT with `Base=1` and `Patterns=[{Pattern="bs", Color=4}]`, plus `Skull`
`Rotation=270`; the bridge reported zero translation errors during the
state-entity window. The standalone path consumed the preceding Java block
state update through the bounded cache.

The modern block-entity fixture then placed suspicious sand containing a
diamond and a trial spawner configured for a zombie. The clean Snappy probe on
`127.0.0.1:19224` observed `BrushableBlock` item `minecraft:diamond`,
`brush_count=0`, and `type=minecraft:suspicious_sand`, plus `TrialSpawner`
`spawn_data={TypeId=minecraft:zombie, Weight=1}`; the bridge reported no
translation errors. Brush-progress animation, trial-spawner ticking, and
native rendering remain open.

The temporary Paper entity fixture also dropped a diamond stack and changed its
count. The Snappy probe on `127.0.0.1:19171` received typed `AddItemActor`
packets and clean item removals; this is an automated actor-projection result,
not full item-entity parity or native rendering evidence.

The same fixture spawned Kebab and Pool paintings. The Snappy probe on
`127.0.0.1:19172` received typed `AddPainting` packets for both motives,
including the variant update, with no bridge translation error. Painting
interaction, custom motive assets, and native rendering remain open.

The temporary frame fixture spawned normal and glow item frames. The Snappy
probe on `127.0.0.1:19173` received typed frame block updates and actor NBT,
including a named diamond sword, facing-specific runtime IDs, rotation, and
air cleanup, with no bridge translation error. Native frame rendering and
Bedrock interaction remain open.

The temporary special-entity fixture spawned a Java experience orb with amount
17 and a falling stone. The Snappy probe on `127.0.0.1:19176` received the typed
Bedrock `SpawnExperienceOrb` amount and a `minecraft:falling_block` actor with
display-tile runtime 2706, with no bridge translation error. Fishing-hook
projection is unit-tested; a live fishing-hook fixture plus native rendering
and interaction remain open.

The same fixture spawned an End Crystal, Area Effect Cloud, primed TNT, and
Lightning. The Snappy probe on `127.0.0.1:19178` received the Bedrock
`ender_crystal` identifier with fire-immunity/block-target metadata, cloud
defaults plus a radius update, TNT fuse/ignited metadata updates, and both
Lightning sound packets, with no bridge translation error. Native rendering,
interaction, and the broader entity metadata matrix remain open.

The same fixture spawned normal, spectral, and trident projectiles. The Snappy
probe on `127.0.0.1:19179` received `arrow`, `spectral_arrow`, and Bedrock's
`thrown_trident`, observed the critical-arrow and spectral-texture flags, and
reported no bridge translation error. The tipped-color and trident-enchantment
updates are covered by focused tests; native projectile rendering, collision,
and the remaining projectile metadata remain open.

A follow-up Paper fixture spawned real egg, snowball, ender-pearl,
experience-bottle, potion, and eye-of-ender entities. The Snappy probe on
`127.0.0.1:19197` received `egg`, `snowball`, `ender_pearl`, `xp_bottle`,
`splash_potion`, and `eye_of_ender_signal` actors at scale `0.5`; throwable
actors then received the invisible-bit clear update on their velocity packet,
including zero-motion projectiles, with no bridge translation error. Paper's
API rejects direct `FishHook` fixture spawning, so fishing-hook owner/target
behavior is unit-tested and not represented as a live-closed gate yet.

The same Paper fixture assigned a real `LONG_SLOWNESS` item to its thrown
potion. The clean-source Snappy probe on `127.0.0.1:19202` observed the initial
`splash_potion`, the later `AuxValueData=18` plus enchanted metadata update for
Java potion ordinal 17, and the zero-motion visibility clear, with no bridge
translation error. Potion registry mapping, firework component decoding, and
firework attachment tracking are covered by focused tests; the firework actor
display payload remains blocked on the missing metadata key in the selected
Gophertunnel fork.

The clean-source Snappy projectile-tick probe on `127.0.0.1:19203` connected to
the same Paper fixture and received translated projectile actors plus repeated
typed `MoveActorDelta` updates from the 20 Hz simulator, with no bridge
translation error. This closes only bounded re-simulation; authoritative
collision, water drag, item slipperiness, and native rendering remain open.

The Paper entity fixture also spawned a baby/sheared blue sheep, a flagged armor
stand, a baby/tamed/sitting cat, a sleeping/interested fox, a killer rabbit, an
angry bee, and a tropical fish. The Snappy probe on `127.0.0.1:19184` observed
their Bedrock flag words (including `FlagsTwo`), color/variant metadata, and the
bee anger clear update with no bridge translation error. This validates the
pinned Java 1.21.4 metadata layout; registry-backed cat/wolf mapping and
known-owner EIDs are covered by the follow-up below, while custom variant
assets, animation, interaction, and the broader entity matrix remain open.

A follow-up Paper fixture assigned a Siamese cat and an owned Ashen wolf to the
local Java player. The Snappy probe on `127.0.0.1:19186` observed Bedrock cat
variant `3`, wolf variant `1`, tame/sit flags, collar color, and owner EID `248`,
with no bridge translation error. The bounded wolf registry-holder decoder is
covered by focused tests; direct custom variant assets and late owner discovery
remain open.

The corrected entity-matrix fixture was rerun through the clean binary on
`127.0.0.1:19193`. A real Bedrock protocol probe connected with Snappy to
Paper 1.21.4 and received the matrix actors, including strider, pufferfish,
polar bear, shulker, turtle, and the special-entity fixture, with no bridge
translation error. Native client rendering, equipment-driven saddle state,
and the remaining entity matrix are still open.

The temporary Paper fixture then spawned an oak boat, a stone-display minecart,
and a chest minecart. The Snappy probe on `127.0.0.1:19195` received Bedrock
`minecraft:boat` and `minecraft:minecart` actors, including the boat's variant,
buoyancy, and collidable metadata plus the minecart display runtime/offset and
damage updates, with a clean bridge disconnect. Vehicle movement, paddling,
riding offsets, input, container behavior, and native rendering remain open.

## Local checks

```powershell
go test ./...
go vet ./...
go run ./cmd/geyser-go --help
```

The live Bedrock/Paper/BDS procedure and acceptance gates are recorded in
`PLAN.md`. No Mojang payloads, server binaries, credentials, or captures belong
in this repository.
