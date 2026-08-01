// Package translate contains versioned Java/Bedrock packet translators.
// Basic owns the versioned packet pumps for the current implementation slice.
// It is intentionally explicit: packets without a translator are counted and
// logged rather than copied across incompatible wire formats.
package translate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"strings"
	"sync"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaPositionRelativeX     = 1 << 0
	javaPositionRelativeY     = 1 << 1
	javaPositionRelativeZ     = 1 << 2
	javaPositionRelativeYaw   = 1 << 3
	javaPositionRelativePitch = 1 << 4
	defaultChunkRadius        = 8
	maxJavaCollectionSize     = 4096
	maxJavaItemDamage         = int32(1<<31 - 1)
)

// Basic owns the minimum session state needed to translate a Java world
// bootstrap into a Bedrock StartGame and keep the two transports alive.
type Basic struct {
	Profile javaprotocol.Profile
	Logger  *slog.Logger

	mu                    sync.Mutex
	gameData              minecraft.GameData
	playerUUID            [16]byte
	dimensionIDs          map[string]int32
	dimensionLayouts      map[int32]javaDimensionLayout
	biomeRuntimeIDs       []uint32
	entityVariants        javaEntityVariantMappings
	paintings             []javaPaintingDefinition
	position              javaPosition
	playerItems           [46]gtprotocol.ItemInstance
	cursorItem            gtprotocol.ItemInstance
	inventoryStateID      int32
	selectedSlot          byte
	entities              map[int32]*javaEntityState
	players               map[[16]byte]*javaPlayerState
	skinCache             map[string]gtprotocol.Skin
	bossBars              map[[16]byte]*javaBossBarState
	scoreboardObjectives  map[string]*javaScoreboardObjectiveState
	scoreboardTeams       map[string]*javaScoreboardTeamState
	passengers            map[int32][]int32
	vehicleID             int32
	hasVehicle            bool
	blockBreaks           map[gtprotocol.BlockPos]javaBlockBreakState
	blockEntityStates     map[gtprotocol.BlockPos]int32
	blockEntityChunks     map[gtprotocol.ChunkPos]map[gtprotocol.BlockPos]struct{}
	blockStates           javaBlockStateCache
	windows               map[int32]*javaWindowState
	bedrockWindows        map[byte]*javaWindowState
	activeWindowID        byte
	nextWindowID          byte
	nextBossBarID         int64
	nextScoreboardID      int64
	nextScoreboardEntryID int64
	recipeNetworkID       uint32
	recipeIDs             map[int32][]string
	nextStackID           int32
	nextSequence          int32
	clientTick            uint64
	sprinting             bool
	sneaking              bool
	gliding               bool
	fireworkAttachments   map[int32]struct{}
	unknown               map[int32]uint64
}

type javaPosition struct {
	x, y, z    float64
	yaw, pitch float32
}

type javaEntityState struct {
	runtimeID                uint64
	entityType               string
	entityUUID               [16]byte
	projectileHidden         bool
	ownerUUID                [16]byte
	hasOwnerUUID             bool
	position                 mgl32.Vec3
	displayTranslation       mgl32.Vec3
	displayLineOffset        float32
	rotation                 mgl32.Vec3 // pitch, yaw, head yaw
	velocity                 mgl32.Vec3
	item                     gtprotocol.ItemInstance
	hasItem                  bool
	framePosition            gtprotocol.BlockPos
	frameDirection           int32
	frameItem                gtprotocol.ItemInstance
	frameRotation            int32
	frameSpawned             bool
	painting                 JavaPaintingVariant
	hasPainting              bool
	paintingDirection        int32
	paintingSpawned          bool
	goatLeftHorn             bool
	goatRightHorn            bool
	goatLeftHornKnown        bool
	goatRightHornKnown       bool
	fireworkAttachedToPlayer bool
	fireworkAttachedToEntity bool
	fireworkShotAtAngle      bool
	projectileInGround       bool
	projectileNoGravity      bool
	equipment                [7]gtprotocol.ItemInstance
	metadata                 gtprotocol.EntityMetadata
	player                   bool
	playerUUID               [16]byte
}

func NewBasic(profile javaprotocol.Profile, logger *slog.Logger) *Basic {
	if profile.Name == "" {
		profile = javaprotocol.Java1214
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Basic{
		Profile:               profile,
		Logger:                logger,
		dimensionIDs:          make(map[string]int32),
		dimensionLayouts:      make(map[int32]javaDimensionLayout),
		entities:              make(map[int32]*javaEntityState),
		players:               make(map[[16]byte]*javaPlayerState),
		skinCache:             make(map[string]gtprotocol.Skin),
		bossBars:              make(map[[16]byte]*javaBossBarState),
		scoreboardObjectives:  make(map[string]*javaScoreboardObjectiveState),
		scoreboardTeams:       make(map[string]*javaScoreboardTeamState),
		passengers:            make(map[int32][]int32),
		blockBreaks:           make(map[gtprotocol.BlockPos]javaBlockBreakState),
		blockEntityStates:     make(map[gtprotocol.BlockPos]int32),
		blockEntityChunks:     make(map[gtprotocol.ChunkPos]map[gtprotocol.BlockPos]struct{}),
		windows:               make(map[int32]*javaWindowState),
		bedrockWindows:        make(map[byte]*javaWindowState),
		nextWindowID:          1,
		nextBossBarID:         1 << 32,
		nextScoreboardID:      1,
		nextScoreboardEntryID: 1,
		recipeNetworkID:       1,
		recipeIDs:             make(map[int32][]string),
		nextStackID:           1,
		nextSequence:          1,
		fireworkAttachments:   make(map[int32]struct{}),
		unknown:               make(map[int32]uint64),
	}
}

// Bootstrap reads Java play packets until Login is received, builds a full
// vanilla item table from Lunar's Dragonfly fork, and starts the Bedrock game.
func (b *Basic) Bootstrap(ctx context.Context, bedrock *minecraft.Conn, java *javaprotocol.Client) error {
	if bedrock == nil || java == nil {
		return fmt.Errorf("translate: bootstrap requires both connections")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pk, err := java.Conn.ReadPacket()
		if err != nil {
			return fmt.Errorf("translate: read Java bootstrap packet: %w", err)
		}
		switch pk.ID {
		case b.Profile.PlayClientboundLoginPacketID:
			join, err := DecodeJoinGame(pk.Data)
			if err != nil {
				return err
			}
			items, err := data.DragonflyItemEntries()
			if err != nil {
				return err
			}
			catalog := newJavaDimensionCatalog(java.Configuration)
			gameData := join.GameData(bedrock, items, java.Configuration)
			b.mu.Lock()
			b.dimensionIDs = catalog.IDs
			b.dimensionLayouts = catalog.Layouts
			b.biomeRuntimeIDs = catalog.BiomeRuntimeIDs
			b.playerUUID = java.Login.UUID
			b.entityVariants = newJavaEntityVariantMappings(java.Configuration)
			b.paintings = javaPaintingCatalog(java.Configuration)
			b.gameData = gameData
			b.position = javaPosition{
				x:   float64(b.gameData.PlayerPosition.X()),
				y:   float64(b.gameData.PlayerPosition.Y()),
				z:   float64(b.gameData.PlayerPosition.Z()),
				yaw: b.gameData.Yaw, pitch: b.gameData.Pitch,
			}
			b.mu.Unlock()
			if err := bedrock.SendStartGame(gameData); err != nil {
				return fmt.Errorf("translate: send Bedrock StartGame: %w", err)
			}
			if err := bedrock.WritePacket(&packet.NetworkChunkPublisherUpdate{
				Position: gameData.WorldSpawn,
				Radius:   defaultChunkRadius * 16,
			}); err != nil {
				return fmt.Errorf("translate: send chunk publisher: %w", err)
			}
			return nil
		case b.Profile.PlayClientboundKeepAlivePacketID:
			value, err := javaprotocol.DecodeLongPayload(pk.Data)
			if err != nil {
				return err
			}
			if err := java.Conn.WritePacket(b.Profile.PlayServerboundKeepAlivePacketID, javaprotocol.EncodeLongPayload(value)); err != nil {
				return err
			}
		default:
			b.logUnknown(pk.ID, "Java bootstrap")
		}
	}
}

// Run pumps both directions until either connection closes. The pump uses
// Gophertunnel's preserved batch boundary so a future translator can preserve
// packet order and batching decisions without changing session ownership.
func (b *Basic) Run(ctx context.Context, bedrock *minecraft.Conn, java *javaprotocol.Client) error {
	if bedrock == nil || java == nil {
		return fmt.Errorf("translate: run requires both connections")
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errorsCh := make(chan error, 3)
	go func() { errorsCh <- b.pumpJava(runCtx, bedrock, java) }()
	go func() { errorsCh <- b.pumpBedrock(runCtx, bedrock, java) }()
	go func() { errorsCh <- b.pumpProjectiles(runCtx, bedrock) }()
	go func() {
		<-runCtx.Done()
		_ = java.Close()
		_ = bedrock.Close()
	}()
	err := <-errorsCh
	cancel()
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func (b *Basic) pumpJava(ctx context.Context, bedrock *minecraft.Conn, java *javaprotocol.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pk, err := java.Conn.ReadPacket()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("translate: Java read: %w", err)
		}
		if err := b.translateJavaPacket(bedrock, java, pk); err != nil {
			return err
		}
	}
}

func (b *Basic) pumpBedrock(ctx context.Context, bedrock *minecraft.Conn, java *javaprotocol.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		batch, err := bedrock.ReadBatch()
		if err != nil {
			if ctx.Err() != nil || bedrock.Context().Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("translate: Bedrock read batch: %w", err)
		}
		for _, pk := range batch {
			if err := b.translateBedrockPacket(bedrock, java, pk); err != nil {
				return err
			}
		}
	}
}

func (b *Basic) translateJavaPacket(bedrock *minecraft.Conn, java *javaprotocol.Client, pk javaprotocol.Packet) error {
	switch pk.ID {
	case b.Profile.PlayClientboundBlockChangeID:
		change, err := DecodeBlockChange(pk.Data)
		if err != nil {
			return err
		}
		b.rememberBlockStateChange(change.Position, change.StateID)
		runtimeID, known := JavaBlockRuntimeID(change.StateID)
		if !known {
			b.logSemanticAnomaly("Java block state outside generated registry", "state", change.StateID)
		}
		return bedrock.WritePacket(&packet.UpdateBlock{
			Position:          gtprotocol.BlockPos{change.Position[0], change.Position[1], change.Position[2]},
			NewBlockRuntimeID: runtimeID,
			Flags:             packet.BlockUpdateNeighbours | packet.BlockUpdateNetwork,
			Layer:             0,
		})
	case b.Profile.PlayClientboundMultiBlockChangeID:
		update, err := DecodeMultiBlockChange(pk.Data)
		if err != nil {
			return err
		}
		entries := make([]gtprotocol.BlockChangeEntry, 0, len(update.Changes))
		for _, change := range update.Changes {
			b.rememberBlockStateChange(change.Position, change.StateID)
			runtimeID, known := JavaBlockRuntimeID(change.StateID)
			if !known {
				b.logSemanticAnomaly("Java multi-block state outside generated registry", "state", change.StateID)
			}
			entries = append(entries, gtprotocol.BlockChangeEntry{
				BlockPos:       change.Position,
				BlockRuntimeID: runtimeID,
				Flags:          packet.BlockUpdateNeighbours | packet.BlockUpdateNetwork,
			})
		}
		if len(entries) == 0 {
			return nil
		}
		return bedrock.WritePacket(&packet.UpdateSubChunkBlocks{
			Position: gtprotocol.BlockPos{update.SectionX * 16, update.SectionY * 16, update.SectionZ * 16},
			Blocks:   entries,
		})
	case b.Profile.PlayClientboundAnimationID:
		return b.translateAnimation(bedrock, pk.Data)
	case b.Profile.PlayClientboundBlockEntityDataID:
		update, err := DecodeBlockEntityUpdate(pk.Data)
		if err != nil {
			return err
		}
		b.rememberBlockEntityPosition(update.Position)
		stateID, hasState := b.cachedBlockEntityState(update.Position)
		var actorRuntimeIDs map[[16]byte]int64
		if javaName, ok := JavaBlockEntityTypeName(update.Type); ok && javaName == "vault" {
			actorRuntimeIDs = b.snapshotJavaVaultActorRuntimeIDs()
		}
		var tag map[string]any
		var known bool
		if hasState {
			tag, known = bedrockBlockEntityTagWithStateAndResolver(update.Type, update.Position[0], update.Position[1], update.Position[2], update.Data, stateID, actorRuntimeIDs)
		} else {
			tag, known = bedrockBlockEntityTagWithResolver(update.Type, update.Position[0], update.Position[1], update.Position[2], update.Data, actorRuntimeIDs)
		}
		if !known {
			b.logSemanticAnomaly("Java block entity type outside generated registry", "type", update.Type)
			return nil
		}
		if javaName, ok := JavaBlockEntityTypeName(update.Type); ok && javaName == "mob_spawner" && javaMobSpawnerPayloadNeedsReset(update.Data) {
			if err := b.resetJavaMobSpawnerBlock(bedrock, update.Position); err != nil {
				return err
			}
		}
		return bedrock.WritePacket(&packet.BlockActorData{
			Position: update.Position,
			NBTData:  tag,
		})
	case b.Profile.PlayClientboundMapChunkPacketID:
		chunk, err := DecodeMapChunk(pk.Data)
		if err != nil {
			return err
		}
		b.mu.Lock()
		dimension := b.gameData.Dimension
		layout, hasLayout := b.dimensionLayouts[dimension]
		biomeRuntimeIDs := b.biomeRuntimeIDs
		actorRuntimeIDs := b.snapshotJavaVaultActorRuntimeIDsLocked()
		b.mu.Unlock()
		minSection := 0
		if hasLayout {
			minSection = layout.MinSection
		} else {
			_, minSection = bedrockDimensionSections(dimension)
		}
		b.rememberChunkBlockEntityStates(chunk, minSection)
		var raw []byte
		var sections uint32
		if hasLayout {
			raw, sections, err = encodeBedrockChunkWithResolver(chunk, dimension, layout.SectionCount, layout.MinSection, biomeRuntimeIDs, actorRuntimeIDs)
		} else {
			sectionCount, minSection := bedrockDimensionSections(dimension)
			raw, sections, err = encodeBedrockChunkWithResolver(chunk, dimension, sectionCount, minSection, biomeRuntimeIDs, actorRuntimeIDs)
		}
		if err != nil {
			return err
		}
		for _, entity := range chunk.BlockEntities {
			if _, _, known := BedrockBlockEntityForChunk(chunk.X, chunk.Z, entity); !known {
				b.logSemanticAnomaly("Java chunk block entity outside generated registry", "type", entity.Type)
			}
		}
		for _, position := range javaMobSpawnerChunkResetPositions(chunk) {
			if err := b.resetJavaMobSpawnerBlock(bedrock, position); err != nil {
				return err
			}
		}
		return bedrock.WritePacket(&packet.LevelChunk{
			Position:      gtprotocol.ChunkPos{chunk.X, chunk.Z},
			Dimension:     dimension,
			SubChunkCount: sections,
			RawPayload:    raw,
		})
	case b.Profile.PlayClientboundKeepAlivePacketID:
		value, err := javaprotocol.DecodeLongPayload(pk.Data)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundKeepAlivePacketID, javaprotocol.EncodeLongPayload(value))
	case b.Profile.PlayClientboundDifficultyID:
		return b.translateJavaDifficulty(bedrock, pk.Data)
	case b.Profile.PlayClientboundBlockDestructionID:
		return b.translateJavaBlockDestruction(bedrock, pk.Data)
	case b.Profile.PlayClientboundBlockEventID:
		return b.translateJavaBlockEvent(bedrock, pk.Data)
	case b.Profile.PlayClientboundBossBarID:
		return b.translateJavaBossBar(bedrock, pk.Data)
	case b.Profile.PlayClientboundResetScoreID:
		return b.translateJavaResetScore(bedrock, pk.Data)
	case b.Profile.PlayClientboundScoreboardDisplayObjectiveID:
		return b.translateJavaScoreboardDisplay(bedrock, pk.Data)
	case b.Profile.PlayClientboundScoreboardObjectiveID:
		return b.translateJavaScoreboardObjective(bedrock, pk.Data)
	case b.Profile.PlayClientboundTeamsID:
		return b.translateJavaScoreboardTeam(bedrock, pk.Data)
	case b.Profile.PlayClientboundScoreboardScoreID:
		return b.translateJavaScoreboardScore(bedrock, pk.Data)
	case b.Profile.PlayClientboundSetPassengersID:
		return b.translateJavaSetPassengers(bedrock, pk.Data)
	case b.Profile.PlayClientboundSoundEffectID:
		return b.translateJavaSoundEffect(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntitySoundEffectID:
		return b.translateJavaEntitySoundEffect(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntityEventID:
		return b.translateJavaEntityEvent(bedrock, pk.Data)
	case b.Profile.PlayClientboundCollectID:
		return b.translateJavaTakeItem(bedrock, pk.Data)
	case b.Profile.PlayClientboundCooldownID:
		return b.translateJavaCooldown(bedrock, pk.Data)
	case b.Profile.PlayClientboundStopSoundID:
		return b.translateJavaStopSound(bedrock, pk.Data)
	case b.Profile.PlayClientboundClearTitlesID:
		return b.translateJavaClearTitles(bedrock, pk.Data)
	case b.Profile.PlayClientboundGameStateChangeID:
		return b.translateJavaGameStateChange(bedrock, pk.Data)
	case b.Profile.PlayClientboundWorldEventID:
		return b.translateJavaWorldEvent(bedrock, pk.Data)
	case b.Profile.PlayClientboundExplodeID:
		return b.translateJavaExplode(bedrock, pk.Data)
	case b.Profile.PlayClientboundLevelParticlesID:
		return b.translateJavaLevelParticles(bedrock, pk.Data)
	case b.Profile.PlayClientboundPositionPacketID:
		position, err := DecodePositionUpdate(pk.Data)
		if err != nil {
			return err
		}
		if !finitePositionUpdate(position) {
			b.logSemanticAnomaly("skipping Java position update with non-finite movement", "teleport_id", position.TeleportID)
			return nil
		}
		b.applyPosition(position)
		if err := java.Conn.WritePacket(b.Profile.PlayServerboundTeleportConfirmID, encodeTeleportConfirm(position.TeleportID)); err != nil {
			return err
		}
		b.mu.Lock()
		publisherPosition := floorBlockPosition(b.position)
		move := packet.MovePlayer{
			EntityRuntimeID: b.gameData.EntityRuntimeID,
			Position:        mgl32.Vec3{float32(b.position.x), float32(b.position.y), float32(b.position.z)},
			Yaw:             b.position.yaw,
			HeadYaw:         b.position.yaw,
			Pitch:           b.position.pitch,
			Mode:            packet.MoveModeTeleport,
			TeleportCause:   0,
		}
		b.mu.Unlock()
		if err := bedrock.WritePacket(&packet.NetworkChunkPublisherUpdate{
			Position: publisherPosition,
			Radius:   defaultChunkRadius * 16,
		}); err != nil {
			return fmt.Errorf("translate: send chunk publisher update: %w", err)
		}
		return bedrock.WritePacket(&move)
	case b.Profile.PlayClientboundPlayerRotationID:
		rotation, err := DecodePlayerRotation(pk.Data)
		if err != nil {
			return err
		}
		if !finiteFloat32(rotation.Yaw) || !finiteFloat32(rotation.Pitch) {
			b.logSemanticAnomaly("skipping Java player rotation with non-finite values", "yaw", rotation.Yaw, "pitch", rotation.Pitch)
			return nil
		}
		b.mu.Lock()
		b.position.yaw = rotation.Yaw
		b.position.pitch = rotation.Pitch
		move := packet.MovePlayer{
			EntityRuntimeID: b.gameData.EntityRuntimeID,
			Position:        mgl32.Vec3{float32(b.position.x), float32(b.position.y), float32(b.position.z)},
			Yaw:             b.position.yaw,
			HeadYaw:         b.position.yaw,
			Pitch:           b.position.pitch,
			Mode:            packet.MoveModeNormal,
		}
		b.mu.Unlock()
		return bedrock.WritePacket(&move)
	case b.Profile.PlayClientboundUpdateHealthID:
		health, err := DecodeHealthUpdate(pk.Data)
		if err != nil {
			return err
		}
		return bedrock.WritePacket(&packet.SetHealth{Health: int32(math.Round(float64(health)))})
	case b.Profile.PlayClientboundExperienceID:
		return b.translateExperience(bedrock, pk.Data)
	case b.Profile.PlayClientboundPlayerAbilitiesID:
		return b.translatePlayerAbilities(bedrock, pk.Data)
	case b.Profile.PlayClientboundUpdateTimeID:
		update, err := DecodeTimeUpdate(pk.Data)
		if err != nil {
			return err
		}
		value := int32(-1)
		if update.TickDayTime {
			value = clampJavaTime(update.Time)
		}
		return bedrock.WritePacket(&packet.SetTime{Time: value})
	case b.Profile.PlayClientboundUnloadChunkID:
		chunk, err := DecodeUnloadChunk(pk.Data)
		if err != nil {
			return err
		}
		b.forgetBlockEntityChunk(chunk.X, chunk.Z)
		b.mu.Lock()
		dimension := b.gameData.Dimension
		layout, hasLayout := b.dimensionLayouts[dimension]
		b.mu.Unlock()
		emptyPayload := EmptyBedrockChunkPayload(dimension)
		if hasLayout {
			emptyPayload = EmptyBedrockChunkPayloadWithLayout(layout)
		}
		return bedrock.WritePacket(&packet.LevelChunk{
			Position:      gtprotocol.ChunkPos{chunk.X, chunk.Z},
			Dimension:     dimension,
			SubChunkCount: 0,
			RawPayload:    emptyPayload,
		})
	case b.Profile.PlayClientboundOpenWindowID:
		window, err := DecodeJavaOpenWindow(pk.Data)
		if err != nil {
			return err
		}
		return b.translateOpenWindow(bedrock, java, window)
	case b.Profile.PlayClientboundCloseWindowID:
		window, err := DecodeJavaContainerClose(pk.Data)
		if err != nil {
			return err
		}
		return b.translateJavaWindowClose(bedrock, window.WindowID)
	case b.Profile.PlayClientboundWindowItemsID:
		items, err := DecodeJavaWindowItems(pk.Data, b.nextStackNetworkID)
		if err != nil {
			if errors.Is(err, ErrUnsupportedJavaItemComponent) {
				b.logSemanticAnomaly("skipping inventory update with unsupported Java item component", "error", err)
				return nil
			}
			return err
		}
		if items.UnknownItems != 0 {
			b.logSemanticAnomaly("Java inventory contains unknown item IDs", "count", items.UnknownItems)
		}
		return b.translateWindowItems(bedrock, items)
	case b.Profile.PlayClientboundSetSlotID:
		update, err := DecodeJavaSetSlot(pk.Data, b.nextStackNetworkID)
		if err != nil {
			if errors.Is(err, ErrUnsupportedJavaItemComponent) {
				b.logSemanticAnomaly("skipping slot update with unsupported Java item component", "error", err)
				return nil
			}
			return err
		}
		if !update.Known {
			b.logSemanticAnomaly("Java slot update contains an unknown item ID", "slot", update.Slot)
		}
		return b.translateSetSlot(bedrock, update)
	case b.Profile.PlayClientboundSetCursorItemID:
		item, err := DecodeJavaCursorItem(pk.Data, b.nextStackNetworkID)
		if err != nil {
			if errors.Is(err, ErrUnsupportedJavaItemComponent) {
				b.logSemanticAnomaly("skipping cursor update with unsupported Java item component", "error", err)
				return nil
			}
			return err
		}
		if !item.Known {
			b.logSemanticAnomaly("Java cursor update contains an unknown item ID", "item", item.ItemID)
		}
		b.mu.Lock()
		b.cursorItem = item.Item
		b.mu.Unlock()
		return bedrock.WritePacket(&packet.InventorySlot{
			WindowID:  0,
			Slot:      0,
			Container: gtprotocol.Option(gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerCursor}),
			NewItem:   item.Item,
		})
	case b.Profile.PlayClientboundDeclareRecipesID:
		return b.translateJavaDeclareRecipes(bedrock, pk.Data)
	case b.Profile.PlayClientboundRecipeBookAddID:
		return b.translateJavaRecipeBookAdd(bedrock, pk.Data)
	case b.Profile.PlayClientboundRecipeBookRemoveID:
		return b.translateJavaRecipeBookRemove(bedrock, pk.Data)
	case b.Profile.PlayClientboundDeclareCommandsID:
		return b.translateJavaCommands(bedrock, pk.Data)
	case b.Profile.PlayClientboundAddResourcePackID:
		return b.translateJavaResourcePackPush(java, pk.Data)
	case b.Profile.PlayClientboundRemoveResourcePackID:
		if _, err := DecodeJavaResourcePackPop(pk.Data); err != nil {
			return err
		}
		return nil
	case b.Profile.PlayClientboundSetPlayerInventoryID:
		update, err := DecodeSetPlayerInventory(pk.Data, b.nextStackNetworkID)
		if err != nil {
			if errors.Is(err, ErrUnsupportedJavaItemComponent) {
				b.logSemanticAnomaly("skipping player inventory update with unsupported Java item component", "error", err)
				return nil
			}
			return err
		}
		if !update.Item.Known {
			b.logSemanticAnomaly("Java player inventory update contains an unknown item ID", "slot", update.Slot)
		}
		return b.translateSetSlot(bedrock, JavaSetSlot{
			WindowID: 0, Slot: int16(update.Slot), Item: update.Item.Item, Known: update.Item.Known,
		})
	case b.Profile.PlayClientboundRespawnID:
		return b.translateJavaRespawn(bedrock, pk.Data)
	case b.Profile.PlayClientboundSpawnPositionID:
		return b.translateJavaSpawnPosition(bedrock, pk.Data)
	case b.Profile.PlayClientboundSetTitleSubtitleID:
		return b.translateJavaTitleText(bedrock, pk.Data, packet.TitleActionSetSubtitle)
	case b.Profile.PlayClientboundSetTitleTextID:
		return b.translateJavaTitleText(bedrock, pk.Data, packet.TitleActionSetTitle)
	case b.Profile.PlayClientboundSetTitleTimeID:
		return b.translateJavaTitleTimes(bedrock, pk.Data)
	case b.Profile.PlayClientboundActionBarID:
		return b.translateJavaTitleText(bedrock, pk.Data, packet.TitleActionSetActionBar)
	case b.Profile.PlayClientboundPlayerInfoID:
		info, err := DecodePlayerInfoUpdate(pk.Data)
		if err != nil {
			return err
		}
		return b.translatePlayerInfo(bedrock, info)
	case b.Profile.PlayClientboundPlayerRemoveID:
		removed, err := DecodePlayerInfoRemove(pk.Data)
		if err != nil {
			return err
		}
		return b.translatePlayerRemove(bedrock, removed)
	case b.Profile.PlayClientboundHeldItemSlotID:
		update, err := DecodeHeldItemSlot(pk.Data)
		if err != nil {
			return err
		}
		return b.translateHeldItemSlot(bedrock, update)
	case b.Profile.PlayClientboundEntityVelocityID:
		velocity, err := DecodeEntityVelocity(pk.Data)
		if err != nil {
			return err
		}
		b.mu.Lock()
		entity := b.entities[velocity.EntityID]
		var visibility gtprotocol.EntityMetadata
		if entity != nil {
			entity.velocity = velocity.Velocity
			visibility = revealJavaProjectileLocked(entity)
		}
		b.mu.Unlock()
		if entity == nil {
			return nil
		}
		if err := bedrock.WritePacket(&packet.SetActorMotion{
			EntityRuntimeID: entity.runtimeID,
			Velocity:        velocity.Velocity,
		}); err != nil {
			return err
		}
		if visibility != nil {
			return bedrock.WritePacket(&packet.SetActorData{EntityRuntimeID: entity.runtimeID, EntityMetadata: visibility})
		}
		return nil
	case b.Profile.PlayClientboundEntityEquipmentID:
		equipment, err := DecodeEntityEquipment(pk.Data, b.nextStackNetworkID)
		if err != nil {
			if errors.Is(err, ErrUnsupportedJavaItemComponent) {
				b.logSemanticAnomaly("skipping entity equipment with unsupported Java item component", "error", err)
				return nil
			}
			return err
		}
		return b.translateEntityEquipment(bedrock, equipment)
	case b.Profile.PlayClientboundEntityAttributesID:
		return b.translateEntityAttributes(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntityEffectID:
		return b.translateEntityEffect(bedrock, pk.Data)
	case b.Profile.PlayClientboundRemoveEntityEffectID:
		return b.translateRemoveEntityEffect(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntityMetadataID:
		metadata, err := DecodeEntityMetadata(pk.Data, b.nextStackNetworkID)
		if err != nil {
			if errors.Is(err, ErrUnsupportedJavaEntityMetadata) || errors.Is(err, ErrUnsupportedJavaItemComponent) {
				b.logSemanticAnomaly("skipping entity metadata with unsupported field", "error", err)
				return nil
			}
			return err
		}
		return b.translateEntityMetadata(bedrock, metadata)
	case b.Profile.PlayClientboundSystemChatID:
		chat, err := DecodeSystemChat(pk.Data)
		if err != nil {
			return err
		}
		projection := ProjectJavaTextComponent(chat.Content)
		if projection.TranslationKey != "" && !chat.ActionBar {
			return bedrock.WritePacket(&packet.Text{
				TextType:         packet.TextTypeTranslation,
				NeedsTranslation: true,
				Message:          projection.TranslationKey,
				Parameters:       projection.Parameters,
			})
		}
		if projection.PlainText == "" {
			b.logSemanticAnomaly("skipping empty Java system chat component")
			return nil
		}
		textType := byte(packet.TextTypeSystem)
		if chat.ActionBar {
			textType = packet.TextTypeTip
		}
		return bedrock.WritePacket(&packet.Text{TextType: textType, Message: projection.PlainText})
	case b.Profile.PlayClientboundPlayerChatID:
		chat, err := DecodePlayerChat(pk.Data)
		if err != nil {
			return err
		}
		if chat.Message == "" {
			return nil
		}
		return bedrock.WritePacket(&packet.Text{
			TextType:   packet.TextTypeChat,
			SourceName: JavaTextComponentText(chat.NetworkName),
			Message:    chat.Message,
		})
	case b.Profile.PlayClientboundProfilelessChatID:
		chat, err := DecodeProfilelessChat(pk.Data)
		if err != nil {
			return err
		}
		message := JavaTextComponentText(chat.Message)
		if message == "" {
			return nil
		}
		return bedrock.WritePacket(&packet.Text{
			TextType:   packet.TextTypeChat,
			SourceName: JavaTextComponentText(chat.Name),
			Message:    message,
		})
	case b.Profile.PlayClientboundSpawnEntityID:
		return b.translateSpawnEntity(bedrock, pk.Data)
	case b.Profile.PlayClientboundSpawnExperienceOrbID:
		return b.translateSpawnExperienceOrb(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntityTeleportID:
		return b.translateEntityTeleport(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntityRelMoveID:
		return b.translateEntityRelativeMove(bedrock, pk.Data, false)
	case b.Profile.PlayClientboundEntityMoveLookID:
		return b.translateEntityRelativeMove(bedrock, pk.Data, true)
	case b.Profile.PlayClientboundEntityLookID:
		return b.translateEntityLook(bedrock, pk.Data)
	case b.Profile.PlayClientboundVehicleMoveID:
		return b.translateJavaVehicleMove(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntityHeadRotationID:
		return b.translateEntityHeadRotation(bedrock, pk.Data)
	case b.Profile.PlayClientboundEntityDestroyID:
		return b.translateEntityDestroy(bedrock, pk.Data)
	case b.Profile.PlayClientboundDisconnectPacketID:
		_ = bedrock.Disconnect("The Java server disconnected")
		return fmt.Errorf("translate: Java server disconnected")
	default:
		b.logUnknown(pk.ID, "Java play")
		return nil
	}
}

func (b *Basic) nextStackNetworkID() int32 {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextStackID
	if id <= 0 {
		id = 1
	}
	b.nextStackID = id + 1
	if b.nextStackID <= 0 {
		b.nextStackID = 1
	}
	return id
}

func (b *Basic) nextRecipeNetworkID() uint32 {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.recipeNetworkID
	if id == 0 {
		id = 1
	}
	b.recipeNetworkID = id + 1
	if b.recipeNetworkID == 0 {
		b.recipeNetworkID = 1
	}
	return id
}

func (b *Basic) translateWindowItems(bedrock *minecraft.Conn, update JavaWindowItems) error {
	if update.WindowID != 0 {
		return b.translateWindowContent(bedrock, update)
	}
	b.mu.Lock()
	for slot := range b.playerItems {
		b.playerItems[slot] = gtprotocol.ItemInstance{}
	}
	copy(b.playerItems[:], update.Items)
	b.cursorItem = update.CarriedItem
	b.inventoryStateID = update.StateID
	b.mu.Unlock()
	contents := make([]gtprotocol.ItemInstance, 36)
	for slot := 9; slot <= 35 && slot < len(update.Items); slot++ {
		contents[slot] = update.Items[slot]
	}
	for slot := 36; slot <= 44 && slot < len(update.Items); slot++ {
		contents[slot-36] = update.Items[slot]
	}
	if err := bedrock.WritePacket(&packet.InventoryContent{
		WindowID: 0,
		Content:  contents,
		Container: gtprotocol.FullContainerName{
			ContainerID: gtprotocol.ContainerInventory,
		},
	}); err != nil {
		return err
	}
	armor := make([]gtprotocol.ItemInstance, 4)
	for slot := 5; slot <= 8 && slot < len(update.Items); slot++ {
		armor[slot-5] = update.Items[slot]
	}
	if err := bedrock.WritePacket(&packet.InventoryContent{
		WindowID: 0, Content: armor,
		Container: gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerArmor},
	}); err != nil {
		return err
	}
	var offhand gtprotocol.ItemInstance
	if len(update.Items) > 45 {
		offhand = update.Items[45]
	}
	if err := bedrock.WritePacket(&packet.InventoryContent{
		WindowID: 0, Content: []gtprotocol.ItemInstance{offhand},
		Container: gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerOffhand},
	}); err != nil {
		return err
	}
	for slot := 1; slot <= 4 && slot < len(update.Items); slot++ {
		if err := bedrock.WritePacket(&packet.InventorySlot{
			WindowID: 0,
			Slot:     uint32(slot + 27),
			Container: gtprotocol.Option(gtprotocol.FullContainerName{
				ContainerID: gtprotocol.ContainerCraftingInput,
			}),
			NewItem: update.Items[slot],
		}); err != nil {
			return err
		}
	}
	if err := bedrock.WritePacket(&packet.InventorySlot{
		WindowID:  0,
		Slot:      0,
		Container: gtprotocol.Option(gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerCursor}),
		NewItem:   update.CarriedItem,
	}); err != nil {
		return err
	}
	return nil
}

func (b *Basic) translateSetSlot(bedrock *minecraft.Conn, update JavaSetSlot) error {
	if handled, err := b.translateWindowSlot(bedrock, update); handled {
		return err
	}
	if update.WindowID == -1 && update.Slot == -1 {
		b.mu.Lock()
		b.cursorItem = update.Item
		b.inventoryStateID = update.StateID
		b.mu.Unlock()
		return bedrock.WritePacket(&packet.InventorySlot{
			WindowID:  0,
			Slot:      0,
			Container: gtprotocol.Option(gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerCursor}),
			NewItem:   update.Item,
		})
	}
	if update.WindowID != 0 && update.WindowID != -2 {
		b.logSemanticAnomaly("skipping slot update for unsupported Java window", "window", update.WindowID)
		return nil
	}
	if update.Slot >= 0 && update.Slot < int16(len(b.playerItems)) {
		b.mu.Lock()
		b.playerItems[update.Slot] = update.Item
		b.inventoryStateID = update.StateID
		b.mu.Unlock()
	}
	containerID, slot, ok := javaPlayerSlot(update.Slot)
	if !ok {
		b.logSemanticAnomaly("skipping Java player slot outside Bedrock containers", "slot", update.Slot)
		return nil
	}
	return bedrock.WritePacket(&packet.InventorySlot{
		WindowID:  0,
		Slot:      slot,
		Container: gtprotocol.Option(gtprotocol.FullContainerName{ContainerID: containerID}),
		NewItem:   update.Item,
	})
}

func (b *Basic) translateHeldItemSlot(bedrock *minecraft.Conn, update JavaHeldItemSlot) error {
	b.mu.Lock()
	b.selectedSlot = byte(update.Slot)
	item := b.playerItems[36+update.Slot]
	runtimeID := b.gameData.EntityRuntimeID
	b.mu.Unlock()
	return bedrock.WritePacket(&packet.MobEquipment{
		EntityRuntimeID: runtimeID,
		NewItem:         item,
		InventorySlot:   byte(update.Slot),
		HotBarSlot:      byte(update.Slot),
		WindowID:        0,
	})
}

func (b *Basic) translateEntityEquipment(bedrock *minecraft.Conn, update JavaEntityEquipment) error {
	b.mu.Lock()
	entity := b.entities[update.EntityID]
	if entity != nil {
		for slot, item := range update.Items {
			if int(slot) < len(entity.equipment) {
				entity.equipment[slot] = item
			}
		}
	}
	b.mu.Unlock()
	if entity == nil {
		return nil
	}
	armorChanged := false
	if _, ok := update.Items[javaEquipmentBoots]; ok {
		armorChanged = true
	}
	if _, ok := update.Items[javaEquipmentLeggings]; ok {
		armorChanged = true
	}
	if _, ok := update.Items[javaEquipmentChestplate]; ok {
		armorChanged = true
	}
	if _, ok := update.Items[javaEquipmentHelmet]; ok {
		armorChanged = true
	}
	if _, ok := update.Items[javaEquipmentBody]; ok {
		armorChanged = true
	}
	if armorChanged {
		b.mu.Lock()
		armor := packet.MobArmourEquipment{
			EntityRuntimeID: entity.runtimeID,
			Helmet:          entity.equipment[javaEquipmentHelmet],
			Chestplate:      entity.equipment[javaEquipmentChestplate],
			Leggings:        entity.equipment[javaEquipmentLeggings],
			Boots:           entity.equipment[javaEquipmentBoots],
			Body:            entity.equipment[javaEquipmentBody],
		}
		b.mu.Unlock()
		if err := bedrock.WritePacket(&armor); err != nil {
			return err
		}
	}
	if item, ok := update.Items[javaEquipmentMainHand]; ok {
		if err := bedrock.WritePacket(&packet.MobEquipment{
			EntityRuntimeID: entity.runtimeID,
			NewItem:         item,
			InventorySlot:   0,
			HotBarSlot:      0,
			WindowID:        0,
		}); err != nil {
			return err
		}
	}
	if item, ok := update.Items[javaEquipmentOffHand]; ok {
		if err := bedrock.WritePacket(&packet.MobEquipment{
			EntityRuntimeID: entity.runtimeID,
			NewItem:         item,
			InventorySlot:   0,
			HotBarSlot:      0,
			WindowID:        gtprotocol.ContainerOffhand,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *Basic) translateEntityMetadata(bedrock *minecraft.Conn, update JavaEntityMetadata) error {
	metadata := translateGenericEntityMetadata(update.Entries)
	var healthValue float32
	hasHealthUpdate := false
	for _, entry := range update.Entries {
		if entry.Index == 9 {
			if value, ok := entry.Value.(float32); ok {
				healthValue = value
				hasHealthUpdate = true
			}
		}
	}
	var item gtprotocol.ItemInstance
	hasItemUpdate := false
	hasItemMetadataUpdate := false
	var entityItemMetadata JavaEntityItemMetadata
	var frameItem gtprotocol.ItemInstance
	hasFrameItemUpdate := false
	var frameRotationUpdate int32
	hasFrameRotationUpdate := false
	for _, entry := range update.Entries {
		if entry.Index == 8 {
			hasItemMetadataUpdate = true
			switch value := entry.Value.(type) {
			case gtprotocol.ItemInstance:
				item = value
				hasItemUpdate = true
				frameItem = value
				hasFrameItemUpdate = true
			case JavaEntityItemMetadata:
				item = value.Item
				entityItemMetadata = value
				hasItemUpdate = true
				frameItem = value.Item
				hasFrameItemUpdate = true
			}
		}
		if entry.Index == 9 {
			if value, ok := entry.Value.(int32); ok {
				frameRotationUpdate = value
				hasFrameRotationUpdate = true
			}
		}
	}
	if hasItemUpdate && (item.Stack.Count == 0 || item.Stack.ItemType.NetworkID == 0) {
		b.logSemanticAnomaly("skipping Java item entity metadata with an empty or unknown item", "entity", update.EntityID)
		hasItemUpdate = false
	}
	var paintingUpdate JavaPaintingVariant
	hasPaintingUpdate := false
	var paintingDirectionUpdate int32 = 3 // Java's default hanging direction is SOUTH.
	hasDirectionUpdate := false
	for _, entry := range update.Entries {
		switch entry.Type {
		case 12:
			if value, ok := entry.Value.(int32); ok {
				paintingDirectionUpdate = value
				hasDirectionUpdate = true
			}
		case 26:
			if value, ok := entry.Value.(JavaPaintingVariant); ok {
				paintingUpdate = value
				hasPaintingUpdate = true
			}
		}
	}
	b.mu.Lock()
	entity := b.entities[update.EntityID]
	if entity == nil {
		b.mu.Unlock()
		return nil
	}
	entityType := entity.entityType
	updateJavaProjectileMetadataLocked(entity, update.Entries)
	displayTranslationChanged := false
	displayLineOffsetChanged := false
	if entityType == "minecraft:text_display" {
		for _, entry := range update.Entries {
			switch entry.Index {
			case 11:
				translation, ok := entry.Value.(mgl32.Vec3)
				if !ok || !finiteVec3(translation) {
					continue
				}
				if entity.displayTranslation != translation {
					entity.displayTranslation = translation
					displayTranslationChanged = true
				}
			case 23:
				lineOffset := javaTextDisplayLineOffset(JavaTextComponentText(entry.Value))
				if entity.displayLineOffset != lineOffset {
					entity.displayLineOffset = lineOffset
					displayLineOffsetChanged = true
				}
			}
		}
	}
	currentFlags, _ := entity.metadata[gtprotocol.EntityDataKeyFlags].(int64)
	currentFlagsTwo, _ := entity.metadata[gtprotocol.EntityDataKeyFlagsTwo].(int64)
	if genericFlags, ok := metadata[gtprotocol.EntityDataKeyFlags].(int64); ok {
		genericMask, genericMaskTwo := javaGenericEntityFlagMasks(update.Entries)
		// Geyser owns the initial invisible state for throwable item
		// projectiles. Java's shared flag byte does not control that state;
		// the first movement update below reveals the actor after its draw
		// window. Preserve the bridge-owned bit when Java sends defaults.
		if javaThrowableItemEntity(entityType) && entity.projectileHidden {
			genericMask &^= int64(1) << gtprotocol.EntityDataFlagInvisible
		}
		currentFlags = (currentFlags &^ genericMask) | (genericFlags & genericMask)
		metadata[gtprotocol.EntityDataKeyFlags] = currentFlags
		if genericFlagsTwo, hasGenericFlagsTwo := metadata[gtprotocol.EntityDataKeyFlagsTwo].(int64); hasGenericFlagsTwo || genericMaskTwo != 0 {
			currentFlagsTwo = (currentFlagsTwo &^ genericMaskTwo) | (genericFlagsTwo & genericMaskTwo)
			metadata[gtprotocol.EntityDataKeyFlagsTwo] = currentFlagsTwo
		}
	}
	specialMetadata := translateSpecialEntityMetadataWithVariants(entityType, update.Entries, b.entityVariants)
	b.translateEntityTargetMetadataLocked(entityType, update.Entries, specialMetadata)
	b.translateGoatHornMetadataLocked(entityType, update.Entries, entity, specialMetadata)
	translateJavaPotionEntityMetadata(entityType, entityItemMetadata, hasItemMetadataUpdate, specialMetadata)
	fireworkDuration, fireworkEffectChanged := b.updateJavaFireworkAttachmentLocked(update.EntityID, entity, update.Entries)
	if ownerUUID, hasOwnerEntry, hasOwnerUUID := javaTameableOwnerMetadata(entityType, update.Entries); hasOwnerEntry {
		ownerID := int64(0)
		if hasOwnerUUID {
			ownerID = b.javaOwnerEntityIDLocked(ownerUUID)
			entity.ownerUUID = ownerUUID
			entity.hasOwnerUUID = true
		} else {
			entity.ownerUUID = [16]byte{}
			entity.hasOwnerUUID = false
		}
		specialMetadata[gtprotocol.EntityDataKeyOwner] = ownerID
	}
	specialFlags, hasSpecialFlags := specialMetadata[gtprotocol.EntityDataKeyFlags].(int64)
	specialFlagsTwo, hasSpecialFlagsTwo := specialMetadata[gtprotocol.EntityDataKeyFlagsTwo].(int64)
	if hasSpecialFlags || hasSpecialFlagsTwo {
		flagMask, flagMaskTwo := specialEntityFlagMasks(entityType, update.Entries)
		if flagMask != 0 {
			specialFlags = (currentFlags &^ flagMask) | (specialFlags & flagMask)
		} else {
			specialFlags |= currentFlags
		}
		if flagMaskTwo != 0 {
			specialFlagsTwo = (currentFlagsTwo &^ flagMaskTwo) | (specialFlagsTwo & flagMaskTwo)
		} else if hasSpecialFlagsTwo {
			specialFlagsTwo |= currentFlagsTwo
		}
		specialMetadata[gtprotocol.EntityDataKeyFlags] = specialFlags
		if hasSpecialFlagsTwo || flagMaskTwo != 0 {
			specialMetadata[gtprotocol.EntityDataKeyFlagsTwo] = specialFlagsTwo
		}
	}
	for key, value := range specialMetadata {
		metadata[key] = value
	}
	if entity.metadata == nil {
		entity.metadata = gtprotocol.NewEntityMetadata()
	}
	for key, value := range metadata {
		entity.metadata[key] = value
	}
	merged := make(gtprotocol.EntityMetadata, len(entity.metadata))
	for key, value := range entity.metadata {
		merged[key] = value
	}
	runtimeID := entity.runtimeID
	position := entity.position
	rotation := entity.rotation
	velocity := entity.velocity
	if entityType == "minecraft:text_display" {
		position = position.Add(entity.displayTranslation)
		position[1] += entity.displayLineOffset
	}
	previousItem := entity.item
	hadItem := entity.hasItem
	itemChanged := false
	itemCountChanged := false
	if entityType == "minecraft:item" && hasItemUpdate {
		itemChanged = !hadItem || !sameProjectedItem(previousItem, item)
		itemCountChanged = hadItem && itemChanged && sameProjectedItemExceptCount(previousItem, item)
		entity.item = item
		entity.hasItem = true
	}
	paintingChanged := false
	directionChanged := false
	currentPainting := entity.painting
	currentPaintingDirection := entity.paintingDirection
	paintingSpawned := entity.paintingSpawned
	if entityType == "minecraft:painting" {
		if hasPaintingUpdate {
			paintingChanged = !entity.hasPainting || entity.painting != paintingUpdate
			entity.painting = paintingUpdate
			entity.hasPainting = true
		}
		if hasDirectionUpdate {
			directionChanged = entity.paintingDirection != paintingDirectionUpdate
			entity.paintingDirection = paintingDirectionUpdate
		}
		currentPainting = entity.painting
		currentPaintingDirection = entity.paintingDirection
		paintingSpawned = entity.paintingSpawned
	}
	frameChanged := false
	frameRotationChanged := false
	currentFrameItem := entity.frameItem
	currentFrameRotation := entity.frameRotation
	framePosition := entity.framePosition
	frameDirection := entity.frameDirection
	frameSpawned := entity.frameSpawned
	if entityType == "minecraft:item_frame" || entityType == "minecraft:glow_item_frame" {
		if hasFrameItemUpdate {
			frameChanged = !sameProjectedItem(currentFrameItem, frameItem)
			entity.frameItem = frameItem
		}
		if hasFrameRotationUpdate {
			frameRotationChanged = entity.frameRotation != frameRotationUpdate
			entity.frameRotation = frameRotationUpdate
		}
		if hasDirectionUpdate && paintingDirectionUpdate >= 0 && paintingDirectionUpdate <= 5 {
			directionChanged = entity.frameDirection != paintingDirectionUpdate
			entity.frameDirection = paintingDirectionUpdate
		}
		currentFrameItem = entity.frameItem
		currentFrameRotation = entity.frameRotation
		framePosition = entity.framePosition
		frameDirection = entity.frameDirection
		frameSpawned = entity.frameSpawned
	}
	b.mu.Unlock()
	if fireworkEffectChanged {
		if err := b.writeFireworkMovementEffect(bedrock, fireworkDuration); err != nil {
			return err
		}
	}
	if entityType == "minecraft:item" {
		if hasItemUpdate && (!hadItem || itemChanged) {
			if !hadItem {
				return bedrock.WritePacket(&packet.AddItemActor{
					EntityUniqueID:  int64(update.EntityID),
					EntityRuntimeID: runtimeID,
					Item:            item,
					Position:        position,
					Velocity:        velocity,
					EntityMetadata:  merged,
				})
			}
			if itemCountChanged {
				return bedrock.WritePacket(&packet.ActorEvent{
					EntityRuntimeID: runtimeID,
					EventType:       packet.ActorEventUpdateStackSize,
					EventData:       int32(item.Stack.Count),
				})
			}
			if err := bedrock.WritePacket(&packet.RemoveActor{EntityUniqueID: int64(update.EntityID)}); err != nil {
				return err
			}
			return bedrock.WritePacket(&packet.AddItemActor{
				EntityUniqueID:  int64(update.EntityID),
				EntityRuntimeID: runtimeID,
				Item:            item,
				Position:        position,
				Velocity:        velocity,
				EntityMetadata:  merged,
			})
		}
		if !hadItem {
			return nil
		}
	}
	if entityType == "minecraft:item_frame" || entityType == "minecraft:glow_item_frame" {
		if !hasFrameItemUpdate && !hasFrameRotationUpdate && !hasDirectionUpdate {
			return nil
		}
		if !frameChanged && !frameRotationChanged && !directionChanged && frameSpawned {
			return nil
		}
		return b.writeJavaItemFrame(bedrock, entityType, framePosition, frameDirection, currentFrameItem, currentFrameRotation)
	}
	if entityType == "minecraft:painting" {
		if !hasPaintingUpdate && !hasDirectionUpdate {
			return nil
		}
		if !paintingChanged && !directionChanged && paintingSpawned {
			return nil
		}
		definition, ok := b.paintingDefinition(currentPainting)
		if !ok {
			b.logSemanticAnomaly("skipping Java painting with no valid Bedrock motive", "entity", update.EntityID)
			return nil
		}
		if paintingSpawned {
			if err := bedrock.WritePacket(&packet.RemoveActor{EntityUniqueID: int64(update.EntityID)}); err != nil {
				return err
			}
		}
		if err := bedrock.WritePacket(&packet.AddPainting{
			EntityUniqueID:  int64(update.EntityID),
			EntityRuntimeID: runtimeID,
			Position:        javaPaintingPosition(position, currentPaintingDirection, definition.Width, definition.Height),
			Direction:       javaPaintingDirection(currentPaintingDirection),
			Title:           definition.Motive,
		}); err != nil {
			return err
		}
		b.mu.Lock()
		if entity := b.entities[update.EntityID]; entity != nil {
			entity.paintingSpawned = true
		}
		b.mu.Unlock()
		return nil
	}
	if entityType == "minecraft:text_display" && (displayTranslationChanged || displayLineOffsetChanged) {
		if err := bedrock.WritePacket(&packet.MoveActorAbsolute{
			EntityRuntimeID: runtimeID,
			Position:        position,
			Rotation:        rotation,
		}); err != nil {
			return err
		}
	}
	if len(merged) == 0 {
		if !hasHealthUpdate {
			return nil
		}
	} else if err := bedrock.WritePacket(&packet.SetActorData{
		EntityRuntimeID: runtimeID,
		EntityMetadata:  merged,
	}); err != nil {
		return err
	}
	if hasHealthUpdate {
		if health, ok := projectJavaHealth(healthValue); ok {
			return bedrock.WritePacket(&packet.UpdateAttributes{
				EntityRuntimeID: runtimeID,
				Attributes:      []gtprotocol.Attribute{health},
			})
		}
		b.logSemanticAnomaly("skipping Java entity health metadata with invalid value", "entity", update.EntityID, "value", healthValue)
	}
	return nil
}

func sameProjectedItem(a, b gtprotocol.ItemInstance) bool {
	a.StackNetworkID = 0
	b.StackNetworkID = 0
	return reflect.DeepEqual(a, b)
}

func sameProjectedItemExceptCount(a, b gtprotocol.ItemInstance) bool {
	a.StackNetworkID = 0
	b.StackNetworkID = 0
	a.Stack.Count = 0
	b.Stack.Count = 0
	return reflect.DeepEqual(a, b)
}

func javaPlayerSlot(slot int16) (containerID byte, bedrockSlot uint32, ok bool) {
	switch {
	case slot >= 9 && slot <= 35:
		return gtprotocol.ContainerInventory, uint32(slot), true
	case slot >= 36 && slot <= 44:
		return gtprotocol.ContainerHotBar, uint32(slot - 36), true
	case slot >= 5 && slot <= 8:
		return gtprotocol.ContainerArmor, uint32(slot - 5), true
	case slot == 45:
		return gtprotocol.ContainerOffhand, 0, true
	case slot >= 1 && slot <= 4:
		return gtprotocol.ContainerCraftingInput, uint32(slot + 27), true
	case slot == 0:
		return gtprotocol.ContainerCreatedOutput, 0, true
	default:
		return 0, 0, false
	}
}

func (b *Basic) translateSpawnEntity(bedrock *minecraft.Conn, payload []byte) error {
	spawn, err := DecodeSpawnEntity(payload)
	if err != nil {
		return err
	}
	if !finiteVec3(spawn.Position) {
		b.logSemanticAnomaly("skipping entity spawn with non-finite position", "entity", spawn.EntityID)
		return nil
	}
	entityType, known := data.JavaEntityTypeName(spawn.Type)
	if !known {
		b.logSemanticAnomaly("Java entity type outside generated registry", "type", spawn.Type)
		return nil
	}
	if entityType == "minecraft:player" {
		return b.translatePlayerSpawn(bedrock, spawn)
	}
	position := javaEntitySpawnPosition(entityType, spawn.Position)
	runtimeID := uint64(uint32(spawn.EntityID))
	metadata, projectable := javaSpawnEntityProjection(entityType, spawn.ObjectData)
	if !projectable {
		if entityType == "minecraft:falling_block" {
			b.logSemanticAnomaly("projecting Java falling block with air fallback", "entity", spawn.EntityID, "state", spawn.ObjectData)
		} else {
			b.logSemanticAnomaly("skipping Java entity without a safe Bedrock spawn projection", "entity", spawn.EntityID, "type", entityType, "object_data", spawn.ObjectData)
			return nil
		}
	}
	if javaFishingHookEntity(entityType) {
		b.mu.Lock()
		ownerRuntimeID := b.javaEntityRuntimeIDLocked(spawn.ObjectData)
		b.mu.Unlock()
		if ownerRuntimeID == 0 {
			b.logSemanticAnomaly("skipping Java fishing hook without a visible owner", "entity", spawn.EntityID, "owner", spawn.ObjectData)
			return nil
		}
		metadata[gtprotocol.EntityDataKeyOwner] = ownerRuntimeID
	}
	b.mu.Lock()
	entity := &javaEntityState{
		runtimeID:         runtimeID,
		entityUUID:        spawn.UUID,
		entityType:        entityType,
		position:          position,
		rotation:          mgl32.Vec3{spawn.Pitch, spawn.Yaw, spawn.HeadYaw},
		velocity:          spawn.Velocity,
		paintingDirection: 3, // Java Direction.SOUTH, the hanging-entity default.
		framePosition:     javaItemFramePosition(spawn.Position),
		frameDirection:    spawn.ObjectData,
		metadata:          metadata,
	}
	if javaThrowableItemEntity(entityType) {
		entity.projectileHidden = true
	}
	if entity.frameDirection < 0 || entity.frameDirection > 5 {
		entity.frameDirection = javaItemFrameDefaultDirection
	}
	b.entities[spawn.EntityID] = entity
	b.mu.Unlock()
	if entityType == "minecraft:item_frame" || entityType == "minecraft:glow_item_frame" {
		if err := b.writeJavaItemFrame(bedrock, entityType, entity.framePosition, entity.frameDirection, entity.frameItem, entity.frameRotation); err != nil {
			return err
		}
		b.mu.Lock()
		if current := b.entities[spawn.EntityID]; current != nil {
			current.frameSpawned = true
		}
		b.mu.Unlock()
		return nil
	}
	if entityType == "minecraft:item" || entityType == "minecraft:painting" {
		// Java sends the item/painting-specific state in following metadata
		// packets. Bedrock has dedicated actor packets for both, so wait until
		// that state is available instead of emitting a generic actor.
		return nil
	}
	if err := bedrock.WritePacket(&packet.AddActor{
		EntityUniqueID:  int64(spawn.EntityID),
		EntityRuntimeID: runtimeID,
		EntityType:      bedrockEntityType(entityType),
		Position:        position,
		Velocity:        spawn.Velocity,
		Pitch:           spawn.Pitch,
		Yaw:             spawn.Yaw,
		HeadYaw:         spawn.HeadYaw,
		BodyYaw:         spawn.Yaw,
		EntityMetadata:  metadata,
	}); err != nil {
		return err
	}
	if entityType == "minecraft:lightning_bolt" {
		for _, sound := range javaLightningSounds(position) {
			if err := bedrock.WritePacket(&sound); err != nil {
				return err
			}
		}
	}
	return nil
}

// revealJavaProjectileLocked applies the short-lived ThrowableItemEntity
// camera-occlusion guard. Java's metadata flag byte does not own this Bedrock
// presentation detail; the first server movement is the earliest reliable
// point at which the actor has entered the draw stream in this translator.
func revealJavaProjectileLocked(entity *javaEntityState) gtprotocol.EntityMetadata {
	if entity == nil || !entity.projectileHidden || !javaThrowableItemEntity(entity.entityType) {
		return nil
	}
	if entity.metadata == nil {
		entity.metadata = gtprotocol.NewEntityMetadata()
	}
	setProjectedFlag(entity.metadata, gtprotocol.EntityDataFlagInvisible, false)
	entity.projectileHidden = false
	return gtprotocol.EntityMetadata{
		gtprotocol.EntityDataKeyFlags:    entity.metadata[gtprotocol.EntityDataKeyFlags],
		gtprotocol.EntityDataKeyFlagsTwo: entity.metadata[gtprotocol.EntityDataKeyFlagsTwo],
	}
}

func (b *Basic) translateEntityTeleport(bedrock *minecraft.Conn, payload []byte) error {
	teleport, err := DecodeEntityTeleport(payload)
	if err != nil {
		return err
	}
	if !finiteVec3(teleport.Position) {
		b.logSemanticAnomaly("skipping entity teleport with non-finite position", "entity", teleport.EntityID)
		return nil
	}
	b.mu.Lock()
	entity := b.entities[teleport.EntityID]
	if entity == nil {
		b.mu.Unlock()
		return nil
	}
	entity.position = teleport.Position
	entity.rotation[0] = teleport.Pitch
	entity.rotation[1] = teleport.Yaw
	runtimeID := entity.runtimeID
	position := entity.position
	rotation := entity.rotation
	visibility := revealJavaProjectileLocked(entity)
	if entity.entityType == "minecraft:text_display" {
		position = position.Add(entity.displayTranslation)
		position[1] += entity.displayLineOffset
	}
	b.mu.Unlock()
	flags := byte(0)
	if teleport.OnGround {
		flags |= packet.MoveFlagOnGround
	}
	if err := bedrock.WritePacket(&packet.MoveActorAbsolute{
		EntityRuntimeID: runtimeID,
		Flags:           flags,
		Position:        position,
		Rotation:        rotation,
	}); err != nil {
		return err
	}
	if visibility != nil {
		return bedrock.WritePacket(&packet.SetActorData{EntityRuntimeID: runtimeID, EntityMetadata: visibility})
	}
	return nil
}

func (b *Basic) translateEntityRelativeMove(bedrock *minecraft.Conn, payload []byte, hasRotation bool) error {
	move, err := DecodeEntityRelativeMove(payload, hasRotation)
	if err != nil {
		return err
	}
	b.mu.Lock()
	entity := b.entities[move.EntityID]
	if entity == nil {
		b.mu.Unlock()
		return nil
	}
	entity.position = entity.position.Add(move.Delta)
	if move.HasRotation {
		entity.rotation[0], entity.rotation[1] = move.Pitch, move.Yaw
	}
	runtimeID := entity.runtimeID
	position := entity.position
	rotation := entity.rotation
	visibility := revealJavaProjectileLocked(entity)
	if entity.entityType == "minecraft:text_display" {
		position = position.Add(entity.displayTranslation)
		position[1] += entity.displayLineOffset
	}
	b.mu.Unlock()
	flags := uint16(packet.MoveActorDeltaFlagHasX | packet.MoveActorDeltaFlagHasY | packet.MoveActorDeltaFlagHasZ)
	if move.HasRotation {
		flags |= packet.MoveActorDeltaFlagHasRotX | packet.MoveActorDeltaFlagHasRotY
	}
	if move.OnGround {
		flags |= packet.MoveActorDeltaFlagOnGround
	}
	if err := bedrock.WritePacket(&packet.MoveActorDelta{
		EntityRuntimeID: runtimeID,
		Flags:           flags,
		Position:        position,
		Rotation:        rotation,
	}); err != nil {
		return err
	}
	if visibility != nil {
		return bedrock.WritePacket(&packet.SetActorData{EntityRuntimeID: runtimeID, EntityMetadata: visibility})
	}
	return nil
}

func (b *Basic) translateEntityLook(bedrock *minecraft.Conn, payload []byte) error {
	look, err := DecodeEntityLook(payload)
	if err != nil {
		return err
	}
	b.mu.Lock()
	entity := b.entities[look.EntityID]
	var runtimeID uint64
	var rotation mgl32.Vec3
	if entity != nil {
		entity.rotation[0], entity.rotation[1] = look.Pitch, look.Yaw
		runtimeID = entity.runtimeID
		rotation = entity.rotation
	}
	b.mu.Unlock()
	if entity == nil {
		return nil
	}
	flags := uint16(packet.MoveActorDeltaFlagHasRotX | packet.MoveActorDeltaFlagHasRotY)
	if look.OnGround {
		flags |= packet.MoveActorDeltaFlagOnGround
	}
	return bedrock.WritePacket(&packet.MoveActorDelta{
		EntityRuntimeID: runtimeID,
		Flags:           flags,
		Rotation:        rotation,
	})
}

func (b *Basic) translateEntityHeadRotation(bedrock *minecraft.Conn, payload []byte) error {
	rotation, err := DecodeEntityHeadRotation(payload)
	if err != nil {
		return err
	}
	b.mu.Lock()
	entity := b.entities[rotation.EntityID]
	var runtimeID uint64
	var entityRotation mgl32.Vec3
	if entity != nil {
		entity.rotation[2] = rotation.HeadYaw
		runtimeID = entity.runtimeID
		entityRotation = entity.rotation
	}
	b.mu.Unlock()
	if entity == nil {
		return nil
	}
	return bedrock.WritePacket(&packet.MoveActorDelta{
		EntityRuntimeID: runtimeID,
		Flags:           packet.MoveActorDeltaFlagHasRotZ,
		Rotation:        entityRotation,
	})
}

func (b *Basic) translateEntityDestroy(bedrock *minecraft.Conn, payload []byte) error {
	ids, err := DecodeEntityDestroy(payload)
	if err != nil {
		return err
	}
	for _, id := range ids {
		b.mu.Lock()
		entity := b.entities[id]
		delete(b.entities, id)
		delete(b.passengers, id)
		if b.hasVehicle && b.vehicleID == id {
			b.vehicleID = 0
			b.hasVehicle = false
		}
		stopFireworkBoost := false
		if entity != nil && entity.fireworkAttachedToPlayer {
			delete(b.fireworkAttachments, id)
			entity.fireworkAttachedToPlayer = false
			stopFireworkBoost = len(b.fireworkAttachments) == 0
		}
		b.mu.Unlock()
		if entity == nil {
			continue
		}
		if stopFireworkBoost {
			if err := b.writeFireworkMovementEffect(bedrock, 0); err != nil {
				return err
			}
		}
		if entity.entityType == "minecraft:item_frame" || entity.entityType == "minecraft:glow_item_frame" {
			if entity.frameSpawned {
				if err := b.clearJavaItemFrame(bedrock, entity.framePosition); err != nil {
					return err
				}
			}
			continue
		}
		if entity.entityType == "minecraft:item" && !entity.hasItem {
			continue
		}
		if err := bedrock.WritePacket(&packet.RemoveActor{EntityUniqueID: int64(id)}); err != nil {
			return err
		}
	}
	return nil
}

func clampJavaTime(value int64) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}

func (b *Basic) logSemanticAnomaly(message string, args ...any) {
	b.Logger.Debug(message, args...)
}

func (b *Basic) translateBedrockPacket(bedrock *minecraft.Conn, java *javaprotocol.Client, pk packet.Packet) error {
	switch pk := pk.(type) {
	case *packet.PlayerAuthInput:
		if err := b.translatePlayerAuthInputActions(bedrock, java, pk); err != nil {
			return err
		}
		data, err := encodePlayerAuthInput(pk)
		if err != nil {
			b.logSemanticAnomaly("skipping Bedrock player auth input with invalid movement", "error", err)
			return nil
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundPositionLookID, data)
	case *packet.MovePlayer:
		data, err := encodePositionLook(pk)
		if err != nil {
			b.logSemanticAnomaly("skipping Bedrock movement with invalid position or rotation", "error", err)
			return nil
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundPositionLookID, data)
	case *packet.Text:
		message := normalizeIncomingChat(pk.Message)
		if message == "" {
			return nil
		}
		if strings.HasPrefix(message, "/") {
			command := message[1:]
			if command == "" || tooLongChat(command) {
				b.logSemanticAnomaly("skipping Bedrock command outside Java limit", "length", len([]rune(command)))
				return nil
			}
			data, err := encodeChatCommandSigned(command)
			if err != nil {
				return err
			}
			return java.Conn.WritePacket(b.Profile.PlayServerboundChatCommandSignedID, data)
		}
		if tooLongChat(message) {
			b.logSemanticAnomaly("skipping Bedrock chat message over Java limit", "length", len([]rune(message)))
			return nil
		}
		data, err := encodeChatMessage(message)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundChatMessageID, data)
	case *packet.CommandRequest:
		message := normalizeIncomingChat(pk.CommandLine)
		if message == "" || len(message) == 1 || !strings.HasPrefix(message, "/") {
			return nil
		}
		command := message[1:]
		if tooLongChat(command) {
			b.logSemanticAnomaly("skipping Bedrock command outside Java limit", "length", len([]rune(command)))
			return nil
		}
		data, err := encodeChatCommandSigned(command)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundChatCommandSignedID, data)
	case *packet.MobEquipment:
		return b.translateBedrockEquipment(java, pk)
	case *packet.Animate:
		return b.translateBedrockAnimate(java, pk)
	case *packet.Interact:
		return b.translateBedrockInteract(java, pk)
	case *packet.PlayerAction:
		return b.translateBedrockPlayerAction(java, pk)
	case *packet.ItemStackRequest:
		return b.translateItemStackRequests(bedrock, java, pk.Requests)
	case *packet.ContainerClose:
		return b.translateBedrockContainerClose(bedrock, java, pk)
	case *packet.Unknown:
		b.Logger.Debug("unknown Bedrock packet", "id", pk.ID())
		return nil
	default:
		// Login/resource-pack and movement acknowledgement packets are handled
		// by Gophertunnel. Gameplay packets get a typed translator as coverage
		// is added; silently copying their bytes would corrupt Java state.
		return nil
	}
}

func (b *Basic) applyPosition(update PositionUpdate) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if update.Flags&javaPositionRelativeX != 0 {
		b.position.x += update.X
	} else {
		b.position.x = update.X
	}
	if update.Flags&javaPositionRelativeY != 0 {
		b.position.y += update.Y
	} else {
		b.position.y = update.Y
	}
	if update.Flags&javaPositionRelativeZ != 0 {
		b.position.z += update.Z
	} else {
		b.position.z = update.Z
	}
	if update.Flags&javaPositionRelativeYaw != 0 {
		b.position.yaw += update.Yaw
	} else {
		b.position.yaw = update.Yaw
	}
	if update.Flags&javaPositionRelativePitch != 0 {
		b.position.pitch += update.Pitch
	} else {
		b.position.pitch = update.Pitch
	}
}

func floorBlockPosition(position javaPosition) gtprotocol.BlockPos {
	return gtprotocol.BlockPos{
		int32(math.Floor(position.x)),
		int32(math.Floor(position.y)),
		int32(math.Floor(position.z)),
	}
}

func finitePositionUpdate(update PositionUpdate) bool {
	values := []float64{update.X, update.Y, update.Z, update.DeltaX, update.DeltaY, update.DeltaZ,
		float64(update.Yaw), float64(update.Pitch)}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

func (b *Basic) logUnknown(id int32, state string) {
	b.mu.Lock()
	b.unknown[id]++
	count := b.unknown[id]
	b.mu.Unlock()
	if count == 1 || count%1000 == 0 {
		b.Logger.Debug("well-formed packet has no translator", "state", state, "id", id, "count", count)
	}
}

func encodeTeleportConfirm(id int32) []byte {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(id)
	return append([]byte(nil), w.Bytes()...)
}

func encodePositionLook(pk *packet.MovePlayer) ([]byte, error) {
	if pk == nil {
		return nil, fmt.Errorf("translate: nil MovePlayer")
	}
	return encodeBedrockPositionLook(pk.Position, pk.Yaw, pk.Pitch, pk.OnGround, false)
}
