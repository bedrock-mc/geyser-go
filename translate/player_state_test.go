package translate

import (
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeExperience(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.Float32(0.75); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(12); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(1234); err != nil {
		t.Fatal(err)
	}
	experience, err := DecodeExperience(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if experience.Progress != 0.75 || experience.Level != 12 || experience.Total != 1234 {
		t.Fatalf("experience = %+v", experience)
	}
}

func TestDecodePlayerAbilities(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.Byte(0x0f); err != nil {
		t.Fatal(err)
	}
	if err := w.Float32(0.05); err != nil {
		t.Fatal(err)
	}
	if err := w.Float32(0.1); err != nil {
		t.Fatal(err)
	}
	abilities, err := DecodePlayerAbilities(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if abilities.Flags != 0x0f || abilities.FlyingSpeed != 0.05 || abilities.WalkingSpeed != 0.1 {
		t.Fatalf("abilities = %+v", abilities)
	}
}

func TestDecodeAnimation(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(-4); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(4); err != nil {
		t.Fatal(err)
	}
	animation, err := DecodeAnimation(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if animation.EntityID != -4 || animation.Animation != 4 {
		t.Fatalf("animation = %+v", animation)
	}
	if !finiteFloat32(1) || finiteFloat32(float32(math.NaN())) {
		t.Fatal("finiteFloat32 validation failed")
	}
}
