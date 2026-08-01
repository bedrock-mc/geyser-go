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
windows, cursor/transaction state, player/entity metadata, interactions, and
most Java play protocol remain open. Bedrock auth-input movement and a bounded
block/item/selection/entity-action path is present, but stack requests,
target-specific entity semantics, vehicle input, and client prediction
reconciliation remain open. Java section block updates and auth-input state
edges are present. The inventory slice is
limited to the Java player window and safely skips updates containing components
it cannot yet decode. Generic
block-entity identity/coordinates, flags/name/pose metadata,
and player-list/player-actor packets are present, but entity-specific metadata,
Java skin properties, item actors, equipment fidelity, and animation are not
yet parity-complete.

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
pass. The automated probe now receives translated Java chunks, inventory/chat,
generic metadata, equipment, velocity, and player-list updates; native terrain
rendering, BDS, and gameplay behavior remain acceptance gates. A separate
Snappy-enabled probe on `127.0.0.1:19146` sent auth-input, held-slot,
arm-swing, and self-interact packets; Paper recorded `GeyserHeld` joining and
the bridge emitted no translation errors.
