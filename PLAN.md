# Geyser-Go parity plan

This plan is the durable status record for the port. A feature is not complete
when its types compile: implementation, focused tests, Java/BDS interoperability,
native Bedrock validation, and performance evidence are separate gates.

## Status

- [x] Create a separate Go repository and pin the requested Gophertunnel fork.
- [x] Establish bounded Java packet framing, cryptographic transport seams, and
  a tested Java 1.21.4 handshake/login/configuration slice.
- [x] Close the first local Bedrock bootstrap gate with both an automated
  Gophertunnel client and the installed native Bedrock client against Paper
  1.21.4; record the native result as bootstrap-only evidence.
- [x] Decode Java 1.21.4 paletted chunk sections and forward typed Bedrock
  `LevelChunk` packets through the lunar Gophertunnel fork; cover the codec and
  Bedrock subchunk writer with focused tests.
- [x] Normalize the Java 1.21.4 block-entity registry and forward generic
  tile-entity NBT in chunk payloads and standalone `BlockActorData` updates;
  type-specific Geyser transforms remain explicitly incomplete.
- [x] Translate Bedrock `PlayerAuthInput` and legacy `MovePlayer` positions and
  rotations to Java position/look packets, including eye-height and collision
  flag conversion, and emit bounded Java block-dig/block-place/use-item
  packets. Forward Bedrock held-slot, arm-swing, and basic interact/attack
  packets to their Java equivalents; stack requests, target-specific entity
  semantics, vehicles, and reconciliation remain open.
- [x] Forward Java 1.21.4 single-block and section multi-block changes, chunk
  unloads, world time, and the basic non-player entity lifecycle with generated
  Java registry lookups.
- [x] Translate the Java player-window inventory snapshot/slot slice and the
  common Java system/player/profileless chat packets into typed Bedrock
  packets; cover the 1.21.4 slot component wire boundary, common item NBT
  projections, unsupported-component skip behavior, and a live Paper readback.
- [x] Forward Java entity velocity/equipment, held-slot/player-inventory
  updates, generic entity metadata, and the Java player-info/list lifecycle to
  typed Bedrock actor packets; cover bounded decoders and a live Paper readback.
- [x] Project Java dropped-item entities through Bedrock's dedicated item-actor
  packet, including delayed stack metadata, count-only actor events, and
  changed-stack remove/re-add behavior; cover the item metadata codec and a
  live Snappy Paper readback.
- [x] Align the bounded entity-metadata decoder with the pinned Java 1.21.4
  metadata registry and project Java painting variants through Bedrock's
  dedicated `AddPainting` packet, including negotiated registry order,
  direction, offsets, and variant changes; cover a live Kebab/Pool fixture.
- [x] Project Java item-frame and glow-item-frame entities as Bedrock frame
  block states plus `BlockActorData`, including object-data facing, item NBT,
  rotation updates, empty-frame updates, and air cleanup; cover complete
  Dragonfly state lookup tests and a live Paper/Snappy frame fixture.
- [x] Project Java experience orbs, falling blocks, and owned fishing hooks
  with their Bedrock actor metadata, including Java block-state to complete
  Bedrock runtime mapping and safe unknown-state handling; cover unit tests and
  a live Paper/Snappy experience-orb/falling-block fixture.
- [x] Apply Geyser's Bedrock entity-identifier overrides and special metadata
  defaults/updates for End Crystals, Area Effect Clouds, and primed TNT, plus
  the leash-knot offset and Lightning thunder/impact sounds; cover focused
  projection tests and a live Paper/Snappy fixture.
- [x] Project spectral-arrow texture flags, arrow critical/tipped-display
  metadata, and trident critical/enchantment flags into Bedrock actor metadata;
  cover the version-pinned color table with focused tests and a live
  Paper/Snappy arrow-family fixture.
- [x] Project Java `text_display` and `interaction` entities through Geyser's
  armor-stand backing contract, including text/name-tag metadata, multiline
  vertical offset, display translation, and interaction width/height updates;
  cover a live Paper/Snappy fixture on `127.0.0.1:19180`. Item-display,
  block-display, transformation, billboard, brightness, and native interaction
  fidelity remain open.
- [x] Project common Java 1.21.4 living-entity metadata into Bedrock flags and
  variants, including ageable/tameable state, sheep colors/shearing, armor-stand
  flag words, fox/rabbit/bee state, tropical-fish packed colors, negotiated
  cat/wolf variants, and optional owner EIDs for known actors; cover the exact
  pinned metadata indices with focused tests and live Paper/Snappy fixtures on
  `127.0.0.1:19184` and `127.0.0.1:19186`. Late owner discovery, custom
  variant assets, and the broader entity metadata matrix remain open.
- [x] Translate Java experience, player abilities, and basic entity animation
  packets into Gophertunnel player-state/animation packets; cover bounded
  decoders and a live Snappy Paper readback.
- [x] Translate common server-authoritative player-inventory stack requests
  (take, place, swap, drop, and mine-stack validation) into Java 1.21.4
  hashed container-click packets, synchronize the Java cursor/state ID, and
  return typed Bedrock stack responses with focused codec and simulation tests.
- [x] Forward Java cursor-item packets and the versioned entity-attribute
  update packet into typed Bedrock cursor/attribute updates, with bounded
  modifier decoding and a live Snappy Paper readback. Java rotation-only
  player updates are also forwarded as Bedrock movement rotations.
- [x] Forward Java add/remove entity-effect packets into typed Bedrock
  `MobEffect` updates, and translate Bedrock common-container close events into
  the Java close-window packet; cover both bounded effect decoders and the
  close-window codec.
- [x] Add the first Java window lifecycle slice: decode/open/close common
  vanilla menus, maintain Java-to-Bedrock window IDs, forward bounded content
  and slot updates, use a versioned virtual block holder, and translate bounded
  take/place/swap/drop stack requests for supported mapped windows. Per-menu
  properties, merchant/recipe behavior, and exact holder restoration remain
  open.
- [x] Translate Java difficulty, game-state mode/credits/weather cues, default
  spawn position, and title clear/text/subtitle/action-bar/timing packets into
  typed Bedrock packets with bounded decoders and a live Snappy Paper readback.
- [x] Decode Java respawn `SpawnInfo` plus metadata flags and forward the
  initial cross-dimension path through Bedrock `ChangeDimension`/`Respawn`,
  with a live Paper Nether teleport readback.
- [x] Translate Java boss-bar add/remove/health/title/style actions through a
  UUID-keyed state cache, including the invisible Bedrock backing actor;
  validate the typed Bedrock events over a Snappy Paper probe. Java boss-bar
  flags remain leniently recorded because the current codec has no matching
  Bedrock fields.
- [x] Generate the Java 1.21.4 sound registry and Geyser playsound mappings;
  translate positional/entity sound effects and stop-sound packets to typed
  Bedrock packets, with bounded custom-holder decoding and a live Paper probe.
- [x] Decode Java scoreboard objective/display/score/reset/team packets and
  project the initial objective/sidebar/fake-player/team-decoration path to
  Bedrock; validate it with a live Snappy Paper probe. Number formats, colors,
  exact team/name-tag behavior, and full multi-slot semantics remain open.
- [x] Translate Java `SetPassengers` state into ordered Bedrock actor links,
  including removal updates and lenient unknown-entity handling; validate a
  real Pig/player mount over the Snappy Paper probe.
- [x] Translate Java block-destruction stages into estimated Bedrock block
  cracking events and map the common 1.21.4 Geyser world-effect table into
  typed Bedrock level/sound events; validate block, smoke, and cracking events
  over the Snappy Paper probe. Typed Java particle payloads and the remaining
  effect-specific NBT/sound semantics remain open.
- [x] Add bounded Java entity-status and item/experience-orb pickup
  translators with focused wire and mapping tests; a dedicated live status/
  pickup fixture remains an open acceptance gate.
- [x] Add a bounded Java cooldown decoder and typed Bedrock item-cooldown
  projection, including the vanilla shield/goat-horn category aliases; active
  cooldown enforcement remains open; a real Paper fixture now validates the
  Bedrock start and clear durations.
- [x] Add a bounded Java block-event decoder and typed Bedrock projections for
  chest-like blocks, end gateways, mob spawners, and note blocks; piston
  animation, bell/decorated-pot effects, and a live block-action fixture remain
  open.
- [x] Add a bounded Java 1.21.4 explosion decoder and project the explosion
  particle, generic origin event, sound, and player knockback paths; block
  particle lists and exact motion accumulation remain open; the Paper fixture
  now delivers Bedrock explosion event `2026`, sound, and player motion.
- [x] Add a bounded Java 1.21.4 level-particle decoder covering the particle
  union's block, dust, item, vibration, trail, and common no-data variants;
  project mapped common effects to typed Bedrock packets while keeping
  unsupported payloads lenient. Dust/item visual fidelity, native trail/
  vibration rendering, and the remaining unmapped particles remain open; a
  real Paper flame fixture now survives the Snappy bridge and delivers three
  repeated typed particle events.
- [x] Forward Bedrock `Text` and `CommandRequest` input through Java 1.21.4
  chat and unsigned signed-command packets, including Java-style whitespace
  normalization and the 256-character limit. Command-tree negotiation,
  suggestions, secure-chat sessions, and full text-component localization
  remain open.
- [x] Decode Java 1.21.4 Brigadier `declare_commands` packets and project the
  bounded top-level literal/common-argument tree to Bedrock `AvailableCommands`;
  redirect aliases and bounded nested redirects are followed; server-backed
  suggestions, registry-backed enum values, and exact command descriptions
  remain open.
- [x] Preserve simple Java translatable text components and their arguments in
  Bedrock translation packets, with bounded plain-text fallback for nested
  components; locale coverage, styling, and exact key mappings remain open.
- [x] Retain Java 1.21.4 configuration registry data, feature flags, reset-chat,
  and tag registries with bounded decoders; project valid custom
  `dimension_type` entries to deterministic Bedrock data-driven definitions and
  matching vertical chunk layouts, and project the negotiated `worldgen/biome`
  registry into Bedrock chunk biome storage. Broader Bedrock registry projection
  remains open.
- [x] Decode Java 1.21.4 resource-pack push/pop packets and acknowledge the
  required accepted/downloaded/successfully-loaded sequence plus optional
  declined status; Bedrock pack hosting, download, cache, stack delivery, and
  removal remain open.
- [ ] Implement Java handshake/login/configuration/play negotiation for the
  supported protocol matrix.
- [ ] Implement session ownership and typed Java <-> Bedrock translator registries.
- [ ] Generate complete versioned vanilla blocks/items from Cloudburst plus
  Lunar/Dragonfly sources, including entries without Dragonfly behavior.
- [ ] Port world bootstrap, dimensions, chunks, block entities, entities,
  movement, interactions, inventory, recipes, commands, text, sound, packs,
  skins, auth, transfers, and extension APIs.
- [ ] Run real Bedrock client connections against local Paper and BDS, then the
  designated remote compatibility targets.
- [ ] Close independent review, native, and performance gates before calling a
  tranche complete.

The current implementation is an explicitly incomplete bootstrap/world
tranche. It does not claim Geyser gameplay parity: 200 Java block states still
fall back to Bedrock air in the generated 1.21.4 mapping. The generator now
resolves 118 palette-checked Java-to-Bedrock name/state transforms (including
chains, standing pale-oak signs, and skeleton/wither skull variants); the
remaining head states require Geyser's custom-skull/entity path rather than an
arbitrary static block alias. Lighting,
type-specific block-entity transforms, behavior-heavy item components, arbitrary container
windows, complex transaction state, player/entity metadata, interactions, and
most Java play protocol remain open. Bedrock auth-input movement and a bounded
block/item/selection/entity-action path is present, including auth-input
sprint/sneak/glide edges, Java section block updates, experience, abilities,
and basic animation. Common player-inventory and initial mapped-window stack
requests now have a Java hashed-click bridge, and Java attribute/rotation
updates now have typed paths,
entity effects now have a typed add/remove path, and common non-player-window
close events are forwarded to Java, but complex transactions, recipes,
target-specific entity semantics, vehicle input, and client prediction
reconciliation remain open. The inventory slice covers the Java player window
and the initial mapped common-menu path, and safely skips updates containing
components whose behavior it cannot yet project. Generic
block-entity identity/coordinates, flags/name/pose metadata,
and player-list/player-actor packets are present. The pinned 1.21.4 entity
metadata registry is decoded, dropped-item entities have a dedicated item-actor
path with stack-count updates, paintings have a typed `AddPainting` path, and
item frames have a typed Bedrock block/actor projection. Experience orbs use a
dedicated Bedrock spawn packet, falling blocks carry complete display-tile
runtime metadata, and owned fishing-hook metadata is projected; broader
entity-specific metadata, Java skin properties, equipment fidelity, and
animation are not yet parity-complete. Common Java menu open/close/content
packets now have a
virtual-holder path, and the generic mapped-window stack-request path is
live-tested; Bedrock window interaction is still incomplete for unsupported
menus, properties, recipes, and holder restoration. Java difficulty, game-state
mode/credits/weather cues, default spawn position, title/action-bar packets,
and the initial respawn/dimension path now have typed bounded paths, including
custom dimension definitions and vertical layouts; Java biome registry IDs now
populate Bedrock chunk biome palettes, while native particle rendering and the
remaining world-event
mappings are still open. Boss-bar add/remove/health/title/style packets have a typed,
live-tested path, and positional/entity sound plus stop-sound packets now have
a generated, live-tested path; flags, custom sound packs, and broader
HUD/scoreboard behavior remain open. A first scoreboard objective/sidebar path
is now live-tested, but scoreboard styling, team/name-tag semantics, and the
remaining HUD surfaces are not parity-complete.
Passenger links now have a typed ordered actor-link path and a live Paper/Pig
readback, while vehicle input and entity-specific riding offsets remain open.
Block cracking and common Java world effects now have typed bounded paths and a
live Bedrock readback; Java particle packets now have bounded registry-specific
decoders and common typed projections, but native visual fidelity, effect-
specific NBT, and the remaining Geyser mappings are not parity-complete.
Java cooldown packets and common direct block-event projections now have typed
bounded paths and focused tests; the real Paper fixture observes cooldown start
and clear packets, while active cooldown state and piston/bell/pot semantics
remain open.
The Java level-particle envelope and common mapping path are also typed and
covered by focused wire/mapping tests; native rendering and broad particle
coverage remain open.
Bedrock chat text and command requests now have a bounded downstream path:
the Paper fixture accepted `hello from bedrock`, `/time set day` through a
Bedrock `Text` packet, and `/time set night` through `CommandRequest` over the
Snappy listener. Paper logged both commands as issued by the bridge player;
the current Bedrock text projection still flattens command response
translation arguments, so exact localized command feedback remains open.
The same Paper login emitted its Brigadier tree twice during the bounded
session; the probe received a typed Bedrock `AvailableCommands` packet with
34 top-level commands and no bridge translation error on listener
`127.0.0.1:19157`. Server suggestions, registry-backed argument enums, and
descriptions remain incomplete. A follow-up Snappy probe on
`127.0.0.1:19164` received 28 command entries and grouped Java redirect
aliases `tell`, `w`, `minecraft:tell`, and `minecraft:w` under `msg`, plus
`tm` under `teammsg`, with no bridge translation error.
The temporary Paper fixture also scheduled a required resource pack. The
Snappy Bedrock probe on `127.0.0.1:19158` joined and survived its bounded read
window with no bridge translation error. The Java status acknowledgment path
is covered by focused tests; actual Bedrock pack hosting, delivery, caching,
stack updates, removal, and native rendering remain incomplete.
The same Snappy probe on `127.0.0.1:19163` received a typed Bedrock
translation packet for `multiplayer.player.joined` with its player-name
parameter, and command feedback remained connected through the bounded
plain-text fallback. Locale coverage, styled/nested component fidelity, and
exact Java-to-Bedrock translation-key mappings remain incomplete.

The temporary Paper special-entity fixture spawned a Java experience orb with
amount 17 and a falling stone. The Snappy probe on `127.0.0.1:19176` observed
the typed Bedrock `SpawnExperienceOrb` amount and a `minecraft:falling_block`
actor with display-tile runtime 2706, with no bridge translation error. Fishing
hook projection is covered by focused tests; a live fishing-hook fixture and
native rendering/interaction remain open.

The same fixture also spawned an End Crystal, Area Effect Cloud, primed TNT,
and Lightning. The Snappy probe on `127.0.0.1:19178` received the Bedrock
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

A Paper display fixture then spawned a two-line Java text display and an
`interaction` entity. The Snappy probe on `127.0.0.1:19180` received both as
Bedrock armor-stand-backed actors, observed the text name field and zero-scale
hitbox carrier, the multiline `MoveActorAbsolute` offset, and interaction
width/height updates, with no bridge translation error. Display transformation,
item/block displays, and native Bedrock rendering/interaction remain open.

The same Paper fixture spawned a baby/sheared blue sheep, a flagged armor stand,
a baby/tamed/sitting cat, a sleeping/interested fox, a killer rabbit, an angry
bee, and a tropical fish. The Snappy probe on `127.0.0.1:19184` observed the
corresponding Bedrock flag words (including the high `FlagsTwo` word), color and
variant metadata, and the bee anger clear update, with no bridge translation
error. This validates the pinned Java 1.21.4 metadata layout; registry-backed
cat/wolf mapping and known-owner EIDs are covered by the follow-up below, while
custom variant assets, animation, interaction, and the broader entity matrix
remain open.

A follow-up Paper fixture assigned a Siamese cat and an owned Ashen wolf to the
local Java player. The Snappy probe on `127.0.0.1:19186` observed Bedrock cat
variant `3`, wolf variant `1`, tame/sit flags, collar color, and owner EID `248`,
with no bridge translation error. The registry-holder decoder is now bounded
for Java's wolf variant type; direct custom variant assets and late owner
discovery remain open.

## Non-negotiable contracts

1. Wire truncation, invalid lengths, decompression overflow, malformed UTF-8,
   and cryptographic failures are fatal to the affected connection.
2. A well-formed packet with an unknown or semantically odd field is handled
   leniently when the client does not use that field; it is counted/logged and
   does not become an arbitrary disconnect.
3. Every mapping is versioned and generated from an identified authoritative
   source. A Dragonfly omission is not evidence that a vanilla item or block is
   unsupported.
4. Batch forwarding stays bounded and preserves packet order. Snappy is opt-in
   until native Bedrock testing proves the target client/device matrix.
5. No Java `.jar` extension compatibility is implied by a Go API. The eventual
   extension boundary must be explicit (Go, RPC, or WASM).

## First live acceptance slice

The first live gate is a local offline-mode Paper server with a matching Java
protocol profile and a real Bedrock client joining through `geyser-go`. Record
the Java server version, Bedrock client version, protocol profile, OS, exact
binary, connection duration, visible result, and any disconnect reason. A unit
test or Java-only socket test cannot close this gate.

Recorded result: Paper `1.21.4-232` on Temurin `21.0.12`, Bedrock client
`1.26.3301.0` / protocol `1.26.33`, Java profile `java-1.21.4` / protocol 769,
Windows 11, bridge listener `127.0.0.1:19132`, and the lunar fork at
`60c66ae560608f209f67b7c432cbc5e29e38170a`. Both the automated client and the
native client reached the bridge; the native client reached the in-world HUD.
The visible empty world is an expected incomplete-state finding, not a parity
pass. A later native add-server retry hit the Bedrock UI's `U-000` modal before
contacting the listener, so it is recorded as a client-environment limitation,
not a terrain result. The automated probe now receives translated Java chunks, inventory/chat,
generic metadata, equipment, velocity, and player-list updates; native terrain
rendering, BDS, and gameplay behavior remain acceptance gates. A separate
Snappy-enabled probe on `127.0.0.1:19146` sent auth-input, held-slot,
arm-swing, and self-interact packets; Paper recorded `GeyserHeld` joining and
the bridge emitted no translation errors. A temporary Paper `WindowTest`
plugin opened a generic 9x3 chest menu on join; a Snappy probe on
`127.0.0.1:19153` observed the translated `ContainerOpen`, `InventoryContent`,
and typed `InventorySlot` sequence with no bridge translation errors. This
closes only the automated window-packet gate; native UI rendering and menu
interaction are still open. On `127.0.0.1:19154`, the probe then requested a
take from chest slot 0; Paper accepted the translated Java `container_click`,
and Bedrock returned request `-41` with status `0`, an empty chest slot, and a
diamond on the cursor. This is a generic-window action slice, not full
inventory parity.

The same temporary Paper plugin sent a title, changed the player to creative,
and changed the world difficulty after join. The Snappy probe on
`127.0.0.1:19155` observed typed title timing/title/subtitle packets, game type
`1`, and difficulty `3`; Paper logged a normal disconnect and the bridge
reported no translation errors.

The same plugin created a segmented boss bar and changed its health, color, and
title. The Snappy probe on `127.0.0.1:19155` observed Bedrock boss-bar events
`Show`, `HealthPercentage`, `AppearanceProperties`, and `Title` with a stable
synthetic actor ID. Boss-bar flags are retained as a leniently logged semantic
gap because the current Bedrock codec has no darken-sky/music/fog fields.

The same Paper plugin emitted a positional note sound, an entity level-up
sound, and a stop-sound packet. The Snappy probe observed `note.pling`,
`random.levelup`, and the typed stop event with no bridge translation errors.

The same plugin installed a sidebar objective with two scores and a team
prefix/suffix. The Snappy probe observed the typed Bedrock display objective and
fake-player score entries for `[P] First line!` and `Second line`, including the
score update path, with no bridge translation error. Native HUD rendering and
full scoreboard styling/name-tag semantics remain open.

The temporary Paper entity fixture dropped a stack of diamonds and changed its
count after spawn. A fresh Snappy probe on `127.0.0.1:19171` observed typed
`AddItemActor` packets and item removals with no bridge translation error. This
closes only the dropped-item actor projection; item pickup, merge physics,
per-entity metadata, and native Bedrock rendering remain open.

The same Paper fixture spawned Kebab and Pool paintings, then the Snappy probe
on `127.0.0.1:19172` observed typed `AddPainting` packets for both motives,
including the variant update, with no bridge translation error. Painting
interaction, custom motive assets, and native rendering remain open.

The temporary Paper frame fixture spawned a normal frame containing a named
diamond sword and a glow frame with distinct facing. The Snappy probe on
`127.0.0.1:19173` observed frame block runtime IDs `6480` and `1051`, the
`ItemFrame`/`GlowItemFrame` actor tags, the sword's `minecraft:diamond_sword`
item tag, a 45-degree rotation update, and air cleanup, with no bridge
translation error. Native frame rendering and Bedrock interaction remain open.
