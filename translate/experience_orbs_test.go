package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeSpawnExperienceOrb(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(27)
	_ = w.Float64(12.5)
	_ = w.Float64(79)
	_ = w.Float64(-262.5)
	_ = w.Int16(17)

	orb, err := DecodeSpawnExperienceOrb(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if orb.EntityID != 27 || orb.Position[0] != 12.5 || orb.Position[1] != 79 || orb.Position[2] != -262.5 || orb.Experience != 17 {
		t.Fatalf("decoded experience orb = %+v", orb)
	}
}
