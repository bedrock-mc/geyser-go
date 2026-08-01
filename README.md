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
  and focused simulation tests. This remains limited to the Java player
  window; complex transactions, recipes, and component fidelity are open.
  Java cursor-item and entity-attribute packets now use typed bounded
  translators, and rotation-only player updates reach Bedrock movement state.
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
