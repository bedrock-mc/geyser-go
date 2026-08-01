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
  packets; cover the wire decoders, unsupported-component skip behavior, and
  a live Paper readback.
- [x] Forward Java entity velocity/equipment, held-slot/player-inventory
  updates, generic entity metadata, and the Java player-info/list lifecycle to
  typed Bedrock actor packets; cover bounded decoders and a live Paper readback.
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
tranche. It does not claim Geyser gameplay parity: 318 Java block states still
fall back to Bedrock air in the generated 1.21.4 mapping, and lighting,
type-specific block-entity transforms, item components, arbitrary container
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
components it cannot yet decode. Generic
block-entity identity/coordinates, flags/name/pose metadata,
and player-list/player-actor packets are present, but entity-specific metadata,
Java skin properties, item actors, equipment fidelity, and animation are not
yet parity-complete. Common Java menu open/close/content packets now have a
virtual-holder path, and the generic mapped-window stack-request path is
live-tested; Bedrock window interaction is still incomplete for unsupported
menus, properties, recipes, and holder restoration. Java difficulty, game-state
mode/credits/weather cues, default spawn position, title/action-bar packets,
and the initial respawn/dimension path now have typed bounded paths, but dynamic
dimension registries, particles, and the remaining world-event mappings are
still open. Boss-bar add/remove/health/title/style packets have a typed,
live-tested path, and positional/entity sound plus stop-sound packets now have
a generated, live-tested path; flags, custom sound packs, and broader
HUD/scoreboard behavior remain open. A first scoreboard objective/sidebar path
is now live-tested, but scoreboard styling, team/name-tag semantics, and the
remaining HUD surfaces are not parity-complete.
Passenger links now have a typed ordered actor-link path and a live Paper/Pig
readback, while vehicle input and entity-specific riding offsets remain open.
Block cracking and common Java world effects now have typed bounded paths and a
live Bedrock readback; Java particle packets with registry-specific payloads,
effect-specific NBT, and the remaining Geyser mappings are not parity-complete.

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
