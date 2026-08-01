package translate

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaExplode is the 1.21.4 ClientboundExplodePacket projection. The packet
// carries one particle union and one sound holder in addition to the center
// and optional player knockback.
type JavaExplode struct {
	Center          mgl32.Vec3
	PlayerKnockback *mgl32.Vec3
	Particle        JavaParticle
	ParticleKnown   bool
	Sound           string
	SoundKnown      bool
}

// DecodeJavaExplode reads the 1.21.4 explosion shape. An unknown particle
// registry ID is retained as an omitted particle so the following sound holder
// remains aligned; the rest of a well-formed explosion is still useful.
func DecodeJavaExplode(dataBytes []byte, nextStackID func() int32) (JavaExplode, error) {
	r := javaprotocol.NewReader(dataBytes)
	x, err := r.Float64()
	if err != nil {
		return JavaExplode{}, fmt.Errorf("translate: explosion X: %w", err)
	}
	y, err := r.Float64()
	if err != nil {
		return JavaExplode{}, fmt.Errorf("translate: explosion Y: %w", err)
	}
	z, err := r.Float64()
	if err != nil {
		return JavaExplode{}, fmt.Errorf("translate: explosion Z: %w", err)
	}
	explode := JavaExplode{Center: mgl32.Vec3{float32(x), float32(y), float32(z)}, ParticleKnown: true}
	hasKnockback, err := r.Bool()
	if err != nil {
		return JavaExplode{}, fmt.Errorf("translate: explosion knockback presence: %w", err)
	}
	if hasKnockback {
		knockback := mgl32.Vec3{}
		for i, field := range []string{"X", "Y", "Z"} {
			value, readErr := r.Float64()
			if readErr != nil {
				return JavaExplode{}, fmt.Errorf("translate: explosion knockback %s: %w", field, readErr)
			}
			knockback[i] = float32(value)
		}
		explode.PlayerKnockback = &knockback
	}
	particle, particleErr := decodeJavaParticle(r, nextStackID)
	if particleErr != nil {
		if !errors.Is(particleErr, ErrUnsupportedJavaParticle) && !errors.Is(particleErr, ErrUnsupportedJavaItemComponent) {
			return JavaExplode{}, particleErr
		}
		explode.ParticleKnown = false
	}
	explode.Particle = particle
	explode.Sound, explode.SoundKnown, err = decodeJavaSoundHolder(r)
	if err != nil {
		return JavaExplode{}, fmt.Errorf("translate: explosion sound: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaExplode{}, fmt.Errorf("translate: explosion has %d trailing bytes", r.Remaining())
	}
	return explode, nil
}

func (b *Basic) translateJavaExplode(bedrock *minecraft.Conn, dataBytes []byte) error {
	explode, err := DecodeJavaExplode(dataBytes, b.nextStackNetworkID)
	if err != nil {
		return err
	}
	if !finiteVec3(explode.Center) {
		b.logSemanticAnomaly("skipping Java explosion with non-finite center")
		return nil
	}
	if explode.ParticleKnown {
		particle := JavaLevelParticles{Position: explode.Center, Particle: explode.Particle}
		if err := b.translateJavaParticle(bedrock, particle); err != nil {
			return err
		}
	} else {
		b.logSemanticAnomaly("skipping unsupported Java explosion particle", "particle", explode.Particle.ID)
	}

	eventData, err := marshalBedrockTagValue(map[string]any{
		"originX": explode.Center[0],
		"originY": explode.Center[1],
		"originZ": explode.Center[2],
	})
	if err != nil {
		return fmt.Errorf("translate: explosion Bedrock event data: %w", err)
	}
	if err := bedrock.WritePacket(&packet.LevelEventGeneric{
		EventID:             packet.LevelEventParticlesBlockExplosion,
		SerialisedEventData: eventData,
	}); err != nil {
		return err
	}

	if explode.SoundKnown && explode.Sound != "" {
		pitch := (1 + (rand.Float32()-rand.Float32())*0.2) * 0.7
		if err := bedrock.WritePacket(&packet.PlaySound{
			SoundName: data.BedrockSoundName(explode.Sound),
			Position:  explode.Center,
			Volume:    4,
			Pitch:     pitch,
		}); err != nil {
			return err
		}
	} else if !explode.SoundKnown {
		b.logSemanticAnomaly("skipping Java explosion sound outside generated registry", "sound", explode.Sound)
	}

	if explode.PlayerKnockback != nil {
		if !finiteVec3(*explode.PlayerKnockback) {
			b.logSemanticAnomaly("skipping Java explosion with non-finite player knockback")
			return nil
		}
		if err := bedrock.WritePacket(&packet.SetActorMotion{
			EntityRuntimeID: b.gameData.EntityRuntimeID,
			Velocity:        *explode.PlayerKnockback,
		}); err != nil {
			return err
		}
	}
	return nil
}
