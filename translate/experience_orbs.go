package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaSpawnExperienceOrb is the 1.21.4 ClientboundAddExperienceOrbPacket.
// Unlike generic entity spawns, the orb amount is a signed Java short.
type JavaSpawnExperienceOrb struct {
	EntityID   int32
	Position   mgl32.Vec3
	Experience int16
}

func DecodeSpawnExperienceOrb(payload []byte) (JavaSpawnExperienceOrb, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaSpawnExperienceOrb{}, fmt.Errorf("translate: experience orb entity ID: %w", err)
	}
	x, err := r.Float64()
	if err != nil {
		return JavaSpawnExperienceOrb{}, fmt.Errorf("translate: experience orb x: %w", err)
	}
	y, err := r.Float64()
	if err != nil {
		return JavaSpawnExperienceOrb{}, fmt.Errorf("translate: experience orb y: %w", err)
	}
	z, err := r.Float64()
	if err != nil {
		return JavaSpawnExperienceOrb{}, fmt.Errorf("translate: experience orb z: %w", err)
	}
	experience, err := r.Int16()
	if err != nil {
		return JavaSpawnExperienceOrb{}, fmt.Errorf("translate: experience orb amount: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaSpawnExperienceOrb{}, fmt.Errorf("translate: experience orb has %d trailing bytes", r.Remaining())
	}
	return JavaSpawnExperienceOrb{
		EntityID:   entityID,
		Position:   mgl32.Vec3{float32(x), float32(y), float32(z)},
		Experience: experience,
	}, nil
}

func (b *Basic) translateSpawnExperienceOrb(bedrock *minecraft.Conn, payload []byte) error {
	orb, err := DecodeSpawnExperienceOrb(payload)
	if err != nil {
		return err
	}
	if !finiteVec3(orb.Position) {
		b.logSemanticAnomaly("skipping experience orb with non-finite position", "entity", orb.EntityID)
		return nil
	}
	amount := int32(orb.Experience)
	if amount < 0 {
		b.logSemanticAnomaly("clamping experience orb with negative amount", "entity", orb.EntityID, "amount", amount)
		amount = 0
	}
	runtimeID := uint64(uint32(orb.EntityID))
	metadata, _ := javaSpawnEntityProjection("minecraft:xp_orb", amount)
	b.mu.Lock()
	b.entities[orb.EntityID] = &javaEntityState{
		runtimeID:  runtimeID,
		entityType: "minecraft:xp_orb",
		position:   orb.Position,
		metadata:   metadata,
	}
	b.mu.Unlock()
	return bedrock.WritePacket(&packet.SpawnExperienceOrb{
		Position:         orb.Position,
		ExperienceAmount: amount,
	})
}
