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
  revision and Cloudburst's Bedrock palette; unresolved state aliases remain
  explicitly incomplete rather than silently being called parity.
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
  updates become `BlockActorData`; type-specific NBT transforms are still
  open. Component-bearing items, arbitrary container windows, Java skin
  properties, and richer entity-specific behavior are also still open. Common
  Bedrock server-authoritative player-inventory stack requests (take/place/
  swap/drop plus mine-stack validation) now map to Java 1.21.4 hashed
  container-click packets, with cursor/state tracking, typed Bedrock responses,
  and focused simulation tests. The initial mapped menu path now also handles
  bounded take/place/swap/drop requests for supported Java windows; complex
  transactions, recipes, and component fidelity are open.
  Java cursor-item and entity-attribute packets now use typed bounded
  translators, rotation-only player updates reach Bedrock movement state, and
  Java add/remove entity effects reach Bedrock `MobEffect` packets. Common
  Bedrock container-close events are also forwarded to Java; opening and
  synchronizing arbitrary Java windows now has an initial typed path for common
  vanilla menu types, including Java-to-Bedrock IDs, virtual block holders,
  content, and slot updates. Common mapped windows also accept bounded Bedrock
  stack requests and return typed responses; properties, merchant/recipe
  behavior, and exact virtual-holder restoration remain open.
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
  and full text-component localization remain open.
  Java 1.21.4 Brigadier `declare_commands` packets now have a bounded decoder
  and a Bedrock `AvailableCommands` projection for top-level literals, common
  argument types, boolean enums, and bounded overload traversal. Exact
  redirects, server-backed suggestions, registry-backed argument enums, and
  command descriptions remain open.
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
`619483eb88140f46b8933506c6263861c0d8fa43` and produced 1,933 item records and
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
`1`, and the subsequent Java advancement without a bridge error. Dynamic
dimension registries, dimension-specific world settings, and full respawn
inventory/entity reconciliation remain open.

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

This remains an incomplete transport/world tranche: the automated Bedrock
probe receives forwarded Java chunks and the partial play-state updates, while
native terrain rendering, lighting, type-specific block-entity transforms,
item components, arbitrary inventory windows, complex transaction state, Java skin fidelity,
entity-specific metadata, target-specific interaction semantics, and the rest of
the Geyser gameplay translators are still open acceptance work.

## Local checks

```powershell
go test ./...
go vet ./...
go run ./cmd/geyser-go --help
```

The live Bedrock/Paper/BDS procedure and acceptance gates are recorded in
`PLAN.md`. No Mojang payloads, server binaries, credentials, or captures belong
in this repository.
