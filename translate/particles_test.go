package translate

import (
	"encoding/json"
	"errors"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestDecodeJavaLevelParticlesBlockState(t *testing.T) {
	w := javaprotocol.NewWriter()
	writeJavaLevelParticlesPrefix(t, w, 0)
	_ = w.VarInt(1) // block particle
	_ = w.VarInt(42)
	particles, err := DecodeJavaLevelParticles(w.Bytes(), func() int32 { return 9 })
	if err != nil {
		t.Fatal(err)
	}
	if particles.Position[0] != 12.5 || particles.Offset[1] != 0.25 || particles.Amount != 0 {
		t.Fatalf("unexpected particle envelope: %#v", particles)
	}
	if particles.Particle.ID != 1 || particles.Particle.Kind != javaParticleBlockState || particles.Particle.BlockStateID != 42 {
		t.Fatalf("unexpected block particle: %#v", particles.Particle)
	}
}

func TestDecodeJavaLevelParticlesPayloadVariants(t *testing.T) {
	tests := []struct {
		name string
		id   int32
		body func(*javaprotocol.Writer)
		kind javaParticleDataKind
	}{
		{name: "dust", id: 13, body: func(w *javaprotocol.Writer) {
			_ = w.Int32(0x102030)
			_ = w.Float32(0.75)
		}, kind: javaParticleDust},
		{name: "vibration entity", id: 46, body: func(w *javaprotocol.Writer) {
			_ = w.VarInt(1)
			_ = w.VarInt(7)
			_ = w.Float32(1.62)
			_ = w.VarInt(12)
		}, kind: javaParticleVibration},
		{name: "trail", id: 47, body: func(w *javaprotocol.Writer) {
			_ = w.Float64(1)
			_ = w.Float64(2)
			_ = w.Float64(3)
			_ = w.Int32(0x102030)
			_ = w.VarInt(12)
		}, kind: javaParticleTrail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := javaprotocol.NewWriter()
			writeJavaLevelParticlesPrefix(t, w, 2)
			_ = w.VarInt(test.id)
			test.body(w)
			particles, err := DecodeJavaLevelParticles(w.Bytes(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if particles.Particle.Kind != test.kind || particles.Amount != 2 {
				t.Fatalf("particle=%#v envelope=%#v", particles.Particle, particles)
			}
			if test.id == 13 && (particles.Particle.Color != 0x102030 || particles.Particle.Scale != 0.75) {
				t.Fatalf("dust payload = color %#x scale %v", particles.Particle.Color, particles.Particle.Scale)
			}
		})
	}
}

func TestDecodeJavaLevelParticlesRejectsUnknownRegistry(t *testing.T) {
	w := javaprotocol.NewWriter()
	writeJavaLevelParticlesPrefix(t, w, 0)
	_ = w.VarInt(999)
	_, err := DecodeJavaLevelParticles(w.Bytes(), nil)
	if !errors.Is(err, ErrUnsupportedJavaParticle) {
		t.Fatalf("error=%v, want unsupported particle", err)
	}
}

func TestJavaParticleMappingsIncludeCommonEffects(t *testing.T) {
	for _, id := range []int32{3, 5, 22, 31, 43, 64, 96, 99, 101} {
		mapping, ok := javaParticleMappings[id]
		if !ok || (mapping.eventType == 0 && mapping.identifier == "") {
			t.Fatalf("particle %d has no mapping: %#v", id, mapping)
		}
	}
}

func TestJavaParticleMappingsUseCloudburstParticleTypeEvents(t *testing.T) {
	for _, test := range []struct {
		id      int32
		ordinal int32
	}{
		{id: 31, ordinal: 8},   // FLAME
		{id: 45, ordinal: 13},  // ICON_CRACK
		{id: 57, ordinal: 37},  // RAIN_SPLASH
		{id: 106, ordinal: 91}, // VAULT_CONNECTION
	} {
		mapping, ok := javaParticleMappings[test.id]
		if !ok || mapping.eventType != bedrockParticleType(test.ordinal) {
			t.Fatalf("particle %d mapping=%#v, want Cloudburst ordinal %d", test.id, mapping, test.ordinal)
		}
	}
	for _, id := range []int32{2, 23, 34, 104, 105, 107, 108, 109, 110, 111} {
		if _, ok := javaParticleMappings[id]; ok {
			t.Fatalf("particle %d unexpectedly has a Geyser 1.21.4 mapping", id)
		}
	}
}

func TestMarshalJavaVibrationEventIsNamelessNetworkNBT(t *testing.T) {
	data, err := marshalJavaVibrationEvent(mgl32.Vec3{1, 2, 3}, mgl32.Vec3{4, 5, 6}, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || data[0] != 10 {
		t.Fatalf("root tag=%x, want compound", data)
	}
	framed := append([]byte{data[0], 0}, data[1:]...)
	var decoded map[string]any
	if err := nbt.Unmarshal(framed, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["origin"].(map[string]any); !ok {
		t.Fatalf("origin=%#v, want vec3 compound", decoded["origin"])
	}
	if decoded["timeToLive"] != float32(2) {
		t.Fatalf("timeToLive=%#v, want 2", decoded["timeToLive"])
	}
}

func TestMarshalJavaTrailVariables(t *testing.T) {
	data, err := marshalJavaTrailVariables(mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 0, 10}, 0x336699, 20)
	if err != nil {
		t.Fatal(err)
	}
	var variables []map[string]any
	if err := json.Unmarshal(data, &variables); err != nil {
		t.Fatal(err)
	}
	if len(variables) != 4 || variables[0]["name"] != "variable.direction" || variables[3]["name"] != "variable.particle_initial_speed" {
		t.Fatalf("variables=%#v", variables)
	}
}

func writeJavaLevelParticlesPrefix(t *testing.T, w *javaprotocol.Writer, amount int32) {
	t.Helper()
	_ = w.Bool(true)
	_ = w.Bool(false)
	_ = w.Float64(12.5)
	_ = w.Float64(64)
	_ = w.Float64(-3.25)
	_ = w.Float32(0.5)
	_ = w.Float32(0.25)
	_ = w.Float32(0.125)
	_ = w.Float32(0.01)
	_ = w.Int32(amount)
}
