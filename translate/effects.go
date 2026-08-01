package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type JavaEntityEffect struct {
	EntityID  int32
	EffectID  int32
	Amplifier int32
	Duration  int32
	Flags     byte
}

func DecodeEntityEffect(payload []byte) (JavaEntityEffect, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityEffect{}, fmt.Errorf("translate: effect entity ID: %w", err)
	}
	effectID, err := r.VarInt()
	if err != nil {
		return JavaEntityEffect{}, fmt.Errorf("translate: effect ID: %w", err)
	}
	amplifier, err := r.VarInt()
	if err != nil {
		return JavaEntityEffect{}, fmt.Errorf("translate: effect amplifier: %w", err)
	}
	duration, err := r.VarInt()
	if err != nil {
		return JavaEntityEffect{}, fmt.Errorf("translate: effect duration: %w", err)
	}
	flags, err := r.Byte()
	if err != nil {
		return JavaEntityEffect{}, fmt.Errorf("translate: effect flags: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaEntityEffect{}, fmt.Errorf("translate: entity effect has %d trailing bytes", r.Remaining())
	}
	return JavaEntityEffect{EntityID: entityID, EffectID: effectID, Amplifier: amplifier, Duration: duration, Flags: flags}, nil
}

type JavaRemoveEntityEffect struct {
	EntityID int32
	EffectID int32
}

func DecodeRemoveEntityEffect(payload []byte) (JavaRemoveEntityEffect, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaRemoveEntityEffect{}, fmt.Errorf("translate: remove effect entity ID: %w", err)
	}
	effectID, err := r.VarInt()
	if err != nil {
		return JavaRemoveEntityEffect{}, fmt.Errorf("translate: remove effect ID: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaRemoveEntityEffect{}, fmt.Errorf("translate: remove entity effect has %d trailing bytes", r.Remaining())
	}
	return JavaRemoveEntityEffect{EntityID: entityID, EffectID: effectID}, nil
}

func (b *Basic) effectRuntimeID(entityID int32) (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if entityID == int32(b.gameData.EntityUniqueID) {
		return b.gameData.EntityRuntimeID, true
	}
	entity := b.entities[entityID]
	if entity == nil {
		return 0, false
	}
	return entity.runtimeID, true
}

func (b *Basic) translateEntityEffect(bedrock *minecraft.Conn, payload []byte) error {
	effect, err := DecodeEntityEffect(payload)
	if err != nil {
		return err
	}
	if effect.EffectID < 0 || effect.EffectID > 4095 {
		b.logSemanticAnomaly("skipping Java effect outside bounded ID range", "effect", effect.EffectID)
		return nil
	}
	runtimeID, ok := b.effectRuntimeID(effect.EntityID)
	if !ok {
		return nil
	}
	return bedrock.WritePacket(&packet.MobEffect{
		EntityRuntimeID: runtimeID,
		Operation:       packet.MobEffectAdd,
		EffectType:      effect.EffectID + 1,
		Amplifier:       effect.Amplifier,
		Particles:       effect.Flags&0x02 != 0,
		Duration:        effect.Duration,
		Ambient:         effect.Flags&0x01 != 0,
	})
}

func (b *Basic) translateRemoveEntityEffect(bedrock *minecraft.Conn, payload []byte) error {
	effect, err := DecodeRemoveEntityEffect(payload)
	if err != nil {
		return err
	}
	if effect.EffectID < 0 || effect.EffectID > 4095 {
		b.logSemanticAnomaly("skipping Java remove-effect ID outside bounded range", "effect", effect.EffectID)
		return nil
	}
	runtimeID, ok := b.effectRuntimeID(effect.EntityID)
	if !ok {
		return nil
	}
	return bedrock.WritePacket(&packet.MobEffect{
		EntityRuntimeID: runtimeID,
		Operation:       packet.MobEffectRemove,
		EffectType:      effect.EffectID + 1,
	})
}
