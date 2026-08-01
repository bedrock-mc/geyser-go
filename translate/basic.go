// Package translate contains versioned Java/Bedrock packet translators.
// Basic is deliberately a narrow first slice: it establishes a real Java
// play session and Bedrock world bootstrap, then translates movement and
// keep-alives. Unsupported gameplay packets are counted and ignored until a
// typed translator is added for their protocol revision.
package translate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
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
)

// Basic owns the minimum session state needed to translate a Java world
// bootstrap into a Bedrock StartGame and keep the two transports alive.
type Basic struct {
	Profile javaprotocol.Profile
	Logger  *slog.Logger

	mu       sync.Mutex
	gameData minecraft.GameData
	position javaPosition
	unknown  map[int32]uint64
}

type javaPosition struct {
	x, y, z    float64
	yaw, pitch float32
}

func NewBasic(profile javaprotocol.Profile, logger *slog.Logger) *Basic {
	if profile.Name == "" {
		profile = javaprotocol.Java1214
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Basic{Profile: profile, Logger: logger, unknown: make(map[int32]uint64)}
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
			b.gameData = join.GameData(bedrock, items)
			b.position = javaPosition{
				x:   float64(b.gameData.PlayerPosition.X()),
				y:   float64(b.gameData.PlayerPosition.Y()),
				z:   float64(b.gameData.PlayerPosition.Z()),
				yaw: b.gameData.Yaw, pitch: b.gameData.Pitch,
			}
			if err := bedrock.SendStartGame(b.gameData); err != nil {
				return fmt.Errorf("translate: send Bedrock StartGame: %w", err)
			}
			if err := bedrock.WritePacket(&packet.NetworkChunkPublisherUpdate{
				Position: b.gameData.WorldSpawn,
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
	errorsCh := make(chan error, 2)
	go func() { errorsCh <- b.pumpJava(runCtx, bedrock, java) }()
	go func() { errorsCh <- b.pumpBedrock(runCtx, bedrock, java) }()
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
			if err := b.translateBedrockPacket(java, pk); err != nil {
				return err
			}
		}
	}
}

func (b *Basic) translateJavaPacket(bedrock *minecraft.Conn, java *javaprotocol.Client, pk javaprotocol.Packet) error {
	switch pk.ID {
	case b.Profile.PlayClientboundKeepAlivePacketID:
		value, err := javaprotocol.DecodeLongPayload(pk.Data)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundKeepAlivePacketID, javaprotocol.EncodeLongPayload(value))
	case b.Profile.PlayClientboundPositionPacketID:
		position, err := DecodePositionUpdate(pk.Data)
		if err != nil {
			return err
		}
		b.applyPosition(position)
		if err := java.Conn.WritePacket(b.Profile.PlayServerboundTeleportConfirmID, encodeTeleportConfirm(position.TeleportID)); err != nil {
			return err
		}
		b.mu.Lock()
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
		return bedrock.WritePacket(&move)
	case b.Profile.PlayClientboundUpdateHealthID:
		health, err := DecodeHealthUpdate(pk.Data)
		if err != nil {
			return err
		}
		return bedrock.WritePacket(&packet.SetHealth{Health: int32(math.Round(float64(health)))})
	case b.Profile.PlayClientboundDisconnectPacketID:
		_ = bedrock.Disconnect("The Java server disconnected")
		return fmt.Errorf("translate: Java server disconnected")
	default:
		b.logUnknown(pk.ID, "Java play")
		return nil
	}
}

func (b *Basic) translateBedrockPacket(java *javaprotocol.Client, pk packet.Packet) error {
	switch pk := pk.(type) {
	case *packet.MovePlayer:
		data, err := encodePositionLook(pk)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundPositionLookID, data)
	case *packet.Text:
		if pk.Message == "" {
			return nil
		}
		data, err := encodeChatMessage(pk.Message)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundChatMessageID, data)
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
	w := javaprotocol.NewWriter()
	if err := w.Float64(float64(pk.Position.X())); err != nil {
		return nil, err
	}
	if err := w.Float64(float64(pk.Position.Y())); err != nil {
		return nil, err
	}
	if err := w.Float64(float64(pk.Position.Z())); err != nil {
		return nil, err
	}
	if err := w.Float32(pk.Yaw); err != nil {
		return nil, err
	}
	if err := w.Float32(pk.Pitch); err != nil {
		return nil, err
	}
	if err := w.Byte(0); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodeChatMessage(message string) ([]byte, error) {
	w := javaprotocol.NewWriter()
	if err := w.String(message); err != nil {
		return nil, err
	}
	if err := w.Int64(time.Now().UnixMilli()); err != nil {
		return nil, err
	}
	if err := w.Int64(0); err != nil {
		return nil, err
	}
	if err := w.Bool(false); err != nil {
		return nil, err
	}
	if err := w.VarInt(0); err != nil {
		return nil, err
	}
	if err := w.BytesValue([]byte{0, 0, 0}); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}
