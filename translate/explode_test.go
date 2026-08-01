package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJavaExplode(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Float64(12.5)
	_ = w.Float64(64)
	_ = w.Float64(-3.25)
	_ = w.Bool(true)
	_ = w.Float64(0.1)
	_ = w.Float64(0.2)
	_ = w.Float64(0.3)
	_ = w.VarInt(31) // FLAME
	_ = w.VarInt(1)  // first built-in sound holder

	explode, err := DecodeJavaExplode(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if explode.Center[0] != 12.5 || explode.Center[1] != 64 || explode.Center[2] != -3.25 {
		t.Fatalf("center=%v", explode.Center)
	}
	if explode.PlayerKnockback == nil || explode.PlayerKnockback[2] != 0.3 {
		t.Fatalf("knockback=%v", explode.PlayerKnockback)
	}
	if !explode.ParticleKnown || explode.Particle.ID != 31 || explode.Particle.Kind != javaParticleNoData {
		t.Fatalf("particle=%#v known=%v", explode.Particle, explode.ParticleKnown)
	}
	if !explode.SoundKnown || explode.Sound == "" {
		t.Fatalf("sound=%q known=%v", explode.Sound, explode.SoundKnown)
	}
}

func TestDecodeJavaExplodeCustomSound(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Float64(0)
	_ = w.Float64(1)
	_ = w.Float64(2)
	_ = w.Bool(false)
	_ = w.VarInt(55) // POOF
	_ = w.VarInt(0)
	_ = w.String("minecraft:test.explosion")
	_ = w.Bool(true)
	_ = w.Float32(32)

	explode, err := DecodeJavaExplode(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if explode.Sound != "minecraft:test.explosion" || !explode.SoundKnown {
		t.Fatalf("sound=%q known=%v", explode.Sound, explode.SoundKnown)
	}
}
