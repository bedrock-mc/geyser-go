package translate

import (
	"fmt"
	"math"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaBossBarAdd = iota
	javaBossBarRemove
	javaBossBarHealth
	javaBossBarTitle
	javaBossBarStyle
	javaBossBarFlags
)

// JavaBossBar is the 1.21.4 clientbound boss-bar packet. Its action-specific
// fields stay optional so a semantically unknown action can be logged and
// ignored without desynchronising the Java session.
type JavaBossBar struct {
	UUID   [16]byte
	Action int32
	Title  any
	Health float32
	Color  int32
	Style  int32
	Flags  byte
}

type javaBossBarState struct {
	entityID int64
	title    string
	health   float32
	color    uint8
	overlay  uint8
	flags    byte
}

func DecodeJavaBossBar(data []byte) (JavaBossBar, error) {
	r := javaprotocol.NewReader(data)
	rawUUID, err := r.Bytes(16)
	if err != nil {
		return JavaBossBar{}, fmt.Errorf("translate: boss bar UUID: %w", err)
	}
	var uuid [16]byte
	copy(uuid[:], rawUUID)
	action, err := r.VarInt()
	if err != nil {
		return JavaBossBar{}, fmt.Errorf("translate: boss bar action: %w", err)
	}
	bar := JavaBossBar{UUID: uuid, Action: action}
	switch action {
	case javaBossBarAdd:
		bar.Title, err = decodeJavaNBTValue(r)
		if err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar title: %w", err)
		}
		if bar.Health, err = r.Float32(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar health: %w", err)
		}
		if bar.Color, err = r.VarInt(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar color: %w", err)
		}
		if bar.Style, err = r.VarInt(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar style: %w", err)
		}
		if bar.Flags, err = r.Byte(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar flags: %w", err)
		}
	case javaBossBarHealth:
		if bar.Health, err = r.Float32(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar health: %w", err)
		}
	case javaBossBarTitle:
		bar.Title, err = decodeJavaNBTValue(r)
		if err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar title: %w", err)
		}
	case javaBossBarStyle:
		if bar.Color, err = r.VarInt(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar color: %w", err)
		}
		if bar.Style, err = r.VarInt(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar style: %w", err)
		}
	case javaBossBarFlags:
		if bar.Flags, err = r.Byte(); err != nil {
			return JavaBossBar{}, fmt.Errorf("translate: boss bar flags: %w", err)
		}
	}
	if r.Remaining() != 0 {
		return JavaBossBar{}, fmt.Errorf("translate: boss bar action %d has %d trailing bytes", action, r.Remaining())
	}
	return bar, nil
}

func (b *Basic) translateJavaBossBar(bedrock *minecraft.Conn, data []byte) error {
	bar, err := DecodeJavaBossBar(data)
	if err != nil {
		return err
	}
	switch bar.Action {
	case javaBossBarAdd:
		state := &javaBossBarState{
			title:   bossBarText(bar.Title),
			health:  clampBossBarHealth(bar.Health),
			color:   bossBarColor(bar.Color),
			overlay: bossBarOverlay(bar.Style),
			flags:   bar.Flags,
		}
		if state.title == "" {
			state.title = " "
		}
		b.mu.Lock()
		state.entityID = b.nextBossBarID
		b.nextBossBarID++
		b.bossBars[bar.UUID] = state
		position := mgl32.Vec3{float32(b.position.x), float32(b.position.y - 10), float32(b.position.z)}
		playerID := b.gameData.EntityUniqueID
		b.mu.Unlock()
		if err := bedrock.WritePacket(&packet.AddActor{
			EntityUniqueID:  state.entityID,
			EntityRuntimeID: uint64(state.entityID),
			EntityType:      "minecraft:creeper",
			Position:        position,
			Velocity:        mgl32.Vec3{},
			EntityMetadata:  bossBarEntityMetadata(),
		}); err != nil {
			return err
		}
		return bedrock.WritePacket(bossBarPacket(state, playerID, packet.BossEventShow))
	case javaBossBarRemove:
		b.mu.Lock()
		state := b.bossBars[bar.UUID]
		playerID := b.gameData.EntityUniqueID
		if state != nil {
			delete(b.bossBars, bar.UUID)
		}
		b.mu.Unlock()
		if state == nil {
			b.logSemanticAnomaly("skipping Java boss-bar removal for unknown UUID")
			return nil
		}
		if err := bedrock.WritePacket(bossBarPacket(state, playerID, packet.BossEventHide)); err != nil {
			return err
		}
		return bedrock.WritePacket(&packet.RemoveActor{EntityUniqueID: state.entityID})
	case javaBossBarHealth, javaBossBarTitle, javaBossBarStyle, javaBossBarFlags:
		b.mu.Lock()
		state := b.bossBars[bar.UUID]
		playerID := b.gameData.EntityUniqueID
		if state != nil {
			switch bar.Action {
			case javaBossBarHealth:
				state.health = clampBossBarHealth(bar.Health)
			case javaBossBarTitle:
				state.title = bossBarText(bar.Title)
				if state.title == "" {
					state.title = " "
				}
			case javaBossBarStyle:
				state.color = bossBarColor(bar.Color)
				state.overlay = bossBarOverlay(bar.Style)
			case javaBossBarFlags:
				state.flags = bar.Flags
			}
		}
		b.mu.Unlock()
		if state == nil {
			b.logSemanticAnomaly("skipping Java boss-bar update for unknown UUID", "action", bar.Action)
			return nil
		}
		eventType := uint8(packet.BossEventHealthPercentage)
		switch bar.Action {
		case javaBossBarTitle:
			eventType = packet.BossEventTitle
		case javaBossBarStyle:
			eventType = packet.BossEventAppearanceProperties
		case javaBossBarFlags:
			// Gophertunnel's Bedrock BossEvent model has no darken-sky, music,
			// or fog fields. Keep the state for future codec support and do not
			// disconnect over this well-formed Java update.
			b.logSemanticAnomaly("Java boss-bar flags have no Bedrock field in the current codec", "flags", bar.Flags)
			return nil
		}
		return bedrock.WritePacket(bossBarPacket(state, playerID, eventType))
	default:
		b.logSemanticAnomaly("skipping unknown Java boss-bar action", "action", bar.Action)
		return nil
	}
}

func bossBarEntityMetadata() gtprotocol.EntityMetadata {
	metadata := gtprotocol.NewEntityMetadata()
	metadata[gtprotocol.EntityDataKeyScale] = float32(0)
	metadata[gtprotocol.EntityDataKeyWidth] = float32(0)
	metadata[gtprotocol.EntityDataKeyHeight] = float32(0)
	return metadata
}

func bossBarPacket(state *javaBossBarState, playerID int64, eventType uint8) *packet.BossEvent {
	return &packet.BossEvent{
		BossEntityUniqueID:   state.entityID,
		PlayerUniqueID:       playerID,
		EventType:            eventType,
		BossBarTitle:         state.title,
		FilteredBossBarTitle: state.title,
		HealthPercentage:     state.health,
		Colour:               state.color,
		Overlay:              state.overlay,
	}
}

func bossBarText(value any) string {
	text := JavaTextComponentText(value)
	runes := []rune(text)
	if len(runes) > 256 {
		return string(runes[:253]) + "..."
	}
	return text
}

func clampBossBarHealth(value float32) float32 {
	if math.IsNaN(float64(value)) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func bossBarColor(value int32) uint8 {
	if value < packet.BossEventColourPink || value > 6 {
		return packet.BossEventColourWhite
	}
	// Java's seventh color is WHITE. The current Bedrock codec inserts
	// RebeccaPurple before WHITE, so Java WHITE maps to Bedrock value 7.
	if value == 6 {
		return packet.BossEventColourWhite
	}
	return uint8(value)
}

func bossBarOverlay(value int32) uint8 {
	if value < packet.BossEventOverlayProgress || value > packet.BossEventOverlayNotched20 {
		return packet.BossEventOverlayProgress
	}
	return uint8(value)
}
