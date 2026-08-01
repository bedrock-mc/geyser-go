package translate

import (
	"bytes"
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaPlayerInfoAddPlayer byte = 1 << iota
	javaPlayerInfoInitializeChat
	javaPlayerInfoUpdateGameMode
	javaPlayerInfoUpdateListed
	javaPlayerInfoUpdateLatency
	javaPlayerInfoUpdateDisplayName
	javaPlayerInfoUpdateHat
	javaPlayerInfoUpdateListOrder
)

type JavaPlayerInfoUpdate struct {
	Actions byte
	Entries []JavaPlayerInfoEntry
}

type JavaPlayerInfoEntry struct {
	UUID           [16]byte
	Name           string
	Properties     []javaprotocol.Property
	HasGameMode    bool
	GameMode       int32
	HasListed      bool
	Listed         bool
	HasLatency     bool
	Latency        int32
	HasDisplayName bool
	DisplayName    any
	HasListOrder   bool
	ListOrder      int32
	HasShowHat     bool
	ShowHat        bool
}

func DecodePlayerInfoUpdate(payload []byte) (JavaPlayerInfoUpdate, error) {
	r := javaprotocol.NewReader(payload)
	actions, err := r.Uint8()
	if err != nil {
		return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info action flags: %w", err)
	}
	count, err := boundedJavaCount(r, "player info entry count")
	if err != nil {
		return JavaPlayerInfoUpdate{}, err
	}
	entries := make([]JavaPlayerInfoEntry, count)
	for i := range entries {
		uuid, err := r.Bytes(16)
		if err != nil {
			return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d UUID: %w", i, err)
		}
		copy(entries[i].UUID[:], uuid)
		if actions&javaPlayerInfoAddPlayer != 0 {
			if entries[i].Name, err = r.String(); err != nil {
				return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d name: %w", i, err)
			}
			propertyCount, countErr := boundedJavaCount(r, "player info property count")
			if countErr != nil {
				return JavaPlayerInfoUpdate{}, countErr
			}
			entries[i].Properties = make([]javaprotocol.Property, propertyCount)
			for property := range entries[i].Properties {
				name, readErr := r.String()
				if readErr != nil {
					return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d property %d name: %w", i, property, readErr)
				}
				value, readErr := r.String()
				if readErr != nil {
					return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d property %d value: %w", i, property, readErr)
				}
				propertyValue := javaprotocol.Property{Name: name, Value: value}
				signed, readErr := r.Bool()
				if readErr != nil {
					return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d property %d signature flag: %w", i, property, readErr)
				}
				if signed {
					if propertyValue.Signature, readErr = r.String(); readErr != nil {
						return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d property %d signature: %w", i, property, readErr)
					}
				}
				entries[i].Properties[property] = propertyValue
			}
		}
		if actions&javaPlayerInfoInitializeChat != 0 {
			if err := skipOptionalChatSession(r, i); err != nil {
				return JavaPlayerInfoUpdate{}, err
			}
		}
		if actions&javaPlayerInfoUpdateGameMode != 0 {
			entries[i].HasGameMode = true
			if entries[i].GameMode, err = r.VarInt(); err != nil {
				return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d game mode: %w", i, err)
			}
		}
		if actions&javaPlayerInfoUpdateListed != 0 {
			entries[i].HasListed = true
			if entries[i].Listed, err = r.Bool(); err != nil {
				return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d listed: %w", i, err)
			}
		}
		if actions&javaPlayerInfoUpdateLatency != 0 {
			entries[i].HasLatency = true
			if entries[i].Latency, err = r.VarInt(); err != nil {
				return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d latency: %w", i, err)
			}
		}
		if actions&javaPlayerInfoUpdateDisplayName != 0 {
			entries[i].HasDisplayName = true
			present, readErr := r.Bool()
			if readErr != nil {
				return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d display-name presence: %w", i, readErr)
			}
			if present {
				if entries[i].DisplayName, err = decodeJavaNBTValue(r); err != nil {
					return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d display name: %w", i, err)
				}
			}
		}
		if actions&javaPlayerInfoUpdateListOrder != 0 {
			entries[i].HasListOrder = true
			if entries[i].ListOrder, err = r.VarInt(); err != nil {
				return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d list order: %w", i, err)
			}
		}
		if actions&javaPlayerInfoUpdateHat != 0 {
			entries[i].HasShowHat = true
			if entries[i].ShowHat, err = r.Bool(); err != nil {
				return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info entry %d hat flag: %w", i, err)
			}
		}
	}
	if r.Remaining() != 0 {
		return JavaPlayerInfoUpdate{}, fmt.Errorf("translate: player info has %d trailing bytes", r.Remaining())
	}
	return JavaPlayerInfoUpdate{Actions: actions, Entries: entries}, nil
}

func skipOptionalChatSession(r *javaprotocol.Reader, entry int) error {
	present, err := r.Bool()
	if err != nil {
		return fmt.Errorf("translate: player info entry %d chat-session presence: %w", entry, err)
	}
	if !present {
		return nil
	}
	if _, err := r.Bytes(16); err != nil {
		return fmt.Errorf("translate: player info entry %d chat-session UUID: %w", entry, err)
	}
	if _, err := r.Int64(); err != nil {
		return fmt.Errorf("translate: player info entry %d chat-session expiry: %w", entry, err)
	}
	for _, field := range []string{"chat-session key", "chat-session signature"} {
		length, err := r.VarInt()
		if err != nil {
			return fmt.Errorf("translate: player info entry %d %s length: %w", entry, field, err)
		}
		if length < 0 || length > 64*1024 {
			return fmt.Errorf("translate: player info entry %d %s length %d exceeds limit", entry, field, length)
		}
		if _, err := r.Bytes(int(length)); err != nil {
			return fmt.Errorf("translate: player info entry %d %s: %w", entry, field, err)
		}
	}
	return nil
}

type JavaPlayerInfoRemove struct {
	UUIDs [][16]byte
}

func DecodePlayerInfoRemove(payload []byte) (JavaPlayerInfoRemove, error) {
	r := javaprotocol.NewReader(payload)
	count, err := boundedJavaCount(r, "player remove count")
	if err != nil {
		return JavaPlayerInfoRemove{}, err
	}
	uuids := make([][16]byte, count)
	for i := range uuids {
		value, readErr := r.Bytes(16)
		if readErr != nil {
			return JavaPlayerInfoRemove{}, fmt.Errorf("translate: player remove UUID %d: %w", i, readErr)
		}
		copy(uuids[i][:], value)
	}
	if r.Remaining() != 0 {
		return JavaPlayerInfoRemove{}, fmt.Errorf("translate: player remove has %d trailing bytes", r.Remaining())
	}
	return JavaPlayerInfoRemove{UUIDs: uuids}, nil
}

type javaPlayerState struct {
	UUID       [16]byte
	Name       string
	Properties []javaprotocol.Property
	GameMode   int32
	Listed     bool
	Latency    int32
}

func (b *Basic) translatePlayerInfo(bedrock *minecraft.Conn, update JavaPlayerInfoUpdate) error {
	for _, entry := range update.Entries {
		b.mu.Lock()
		state := b.players[entry.UUID]
		if state == nil {
			state = &javaPlayerState{UUID: entry.UUID, Listed: true}
			b.players[entry.UUID] = state
		}
		if update.Actions&javaPlayerInfoAddPlayer != 0 {
			state.Name = entry.Name
			state.Properties = append(state.Properties[:0], entry.Properties...)
		}
		if entry.HasGameMode {
			state.GameMode = entry.GameMode
		}
		if entry.HasListed {
			state.Listed = entry.Listed
		}
		if entry.HasLatency {
			state.Latency = entry.Latency
		}
		snapshot := *state
		b.mu.Unlock()

		if snapshot.Name == "" {
			b.logSemanticAnomaly("Java player info has no profile name", "uuid", fmt.Sprintf("%x", snapshot.UUID))
			continue
		}
		// Bedrock has no in-place player-list update. Re-sending the add form
		// refreshes the skin/name/latency-bearing entry for this session.
		if err := bedrock.WritePacket(&packet.PlayerList{
			ActionType: packet.PlayerListActionAdd,
			Entries:    []gtprotocol.PlayerListEntry{b.playerListEntry(snapshot, 0)},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *Basic) translatePlayerRemove(bedrock *minecraft.Conn, removed JavaPlayerInfoRemove) error {
	for _, playerUUID := range removed.UUIDs {
		if err := bedrock.WritePacket(&packet.PlayerList{
			ActionType: packet.PlayerListActionRemove,
			Entries:    []gtprotocol.PlayerListEntry{{UUID: uuid.UUID(playerUUID)}},
		}); err != nil {
			return err
		}
		b.mu.Lock()
		delete(b.players, playerUUID)
		var actorIDs []int64
		for entityID, entity := range b.entities {
			if entity.player && entity.playerUUID == playerUUID {
				delete(b.entities, entityID)
				actorIDs = append(actorIDs, int64(entityID))
			}
		}
		b.mu.Unlock()
		for _, actorID := range actorIDs {
			if err := bedrock.WritePacket(&packet.RemoveActor{EntityUniqueID: actorID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Basic) translatePlayerSpawn(bedrock *minecraft.Conn, spawn JavaSpawnEntity) error {
	if !finiteVec3(spawn.Position) {
		b.logSemanticAnomaly("skipping player spawn with non-finite position", "entity", spawn.EntityID)
		return nil
	}
	b.mu.Lock()
	state := b.players[spawn.UUID]
	if state == nil {
		b.mu.Unlock()
		b.logSemanticAnomaly("skipping player spawn without player-info state", "entity", spawn.EntityID)
		return nil
	}
	runtimeID := uint64(uint32(spawn.EntityID))
	b.entities[spawn.EntityID] = &javaEntityState{
		runtimeID:  runtimeID,
		position:   spawn.Position,
		rotation:   mgl32.Vec3{spawn.Pitch, spawn.Yaw, spawn.HeadYaw},
		metadata:   gtprotocol.NewEntityMetadata(),
		player:     true,
		playerUUID: spawn.UUID,
	}
	snapshot := *state
	b.mu.Unlock()

	if err := bedrock.WritePacket(&packet.PlayerList{
		ActionType: packet.PlayerListActionAdd,
		Entries:    []gtprotocol.PlayerListEntry{b.playerListEntry(snapshot, int64(spawn.EntityID))},
	}); err != nil {
		return err
	}
	return bedrock.WritePacket(&packet.AddPlayer{
		UUID:            uuid.UUID(spawn.UUID),
		Username:        snapshot.Name,
		EntityRuntimeID: runtimeID,
		Position:        spawn.Position,
		Velocity:        spawn.Velocity,
		Pitch:           spawn.Pitch,
		Yaw:             spawn.Yaw,
		HeadYaw:         spawn.HeadYaw,
		GameType:        normalizeBedrockGameMode(snapshot.GameMode),
		EntityMetadata:  gtprotocol.NewEntityMetadata(),
		AbilityData: gtprotocol.AbilityData{
			EntityUniqueID: int64(spawn.EntityID),
			Layers: []gtprotocol.AbilityLayer{{
				Type:             gtprotocol.AbilityLayerTypeBase,
				FlySpeed:         gtprotocol.AbilityBaseFlySpeed,
				VerticalFlySpeed: gtprotocol.AbilityBaseVerticalFlySpeed,
				WalkSpeed:        gtprotocol.AbilityBaseWalkSpeed,
			}},
		},
		DeviceID:      "geyser-go",
		BuildPlatform: 0,
	})
}

func (b *Basic) playerListEntry(state javaPlayerState, entityID int64) gtprotocol.PlayerListEntry {
	return gtprotocol.PlayerListEntry{
		UUID:           uuid.UUID(state.UUID),
		EntityUniqueID: entityID,
		Username:       state.Name,
		Skin:           defaultPlayerSkin(state.Name),
	}
}

func defaultPlayerSkin(name string) gtprotocol.Skin {
	return gtprotocol.Skin{
		SkinID:                    "geyser-go-default-" + name,
		SkinResourcePatch:         minecraft.DefaultSkinResourcePatch(),
		SkinImageWidth:            64,
		SkinImageHeight:           32,
		SkinData:                  bytes.Repeat([]byte{0, 0, 0, 255}, 64*32),
		SkinGeometry:              minecraft.DefaultSkinGeometry(),
		GeometryDataEngineVersion: []byte("0.0.0"),
		ArmSize:                   "wide",
		SkinColour:                "#b37b62",
		Trusted:                   true,
	}
}

func normalizeBedrockGameMode(mode int32) int32 {
	if mode < 0 || mode > 3 {
		return packet.GameTypeSurvival
	}
	return mode
}
