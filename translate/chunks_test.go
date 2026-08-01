package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJavaPalettedContainerSingleton(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Byte(0)
	_ = w.VarInt(17)
	_ = w.VarInt(0)
	r := javaprotocol.NewReader(w.Bytes())
	values, err := decodeJavaPalettedContainer(r, 64)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 64 || values[0] != 17 || values[63] != 17 {
		t.Fatalf("unexpected singleton values: len=%d first=%d last=%d", len(values), values[0], values[63])
	}
	if r.Remaining() != 0 {
		t.Fatalf("reader has %d bytes remaining", r.Remaining())
	}
}

func TestDecodeJavaPalettedContainerPackedPalette(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Byte(4)
	_ = w.VarInt(2)
	_ = w.VarInt(101)
	_ = w.VarInt(202)
	_ = w.VarInt(4) // four 64-bit words for a 64-entry test container.
	for word := 0; word < 4; word++ {
		var packed int64
		for i := 0; i < 16; i++ {
			index := uint64(0)
			if word*16+i == 63 {
				index = 1
			}
			packed |= int64(index << uint(i*4))
		}
		_ = w.Int64(packed)
	}
	r := javaprotocol.NewReader(w.Bytes())
	values, err := decodeJavaPalettedContainer(r, 64)
	if err != nil {
		t.Fatal(err)
	}
	if values[0] != 101 || values[62] != 101 || values[63] != 202 {
		t.Fatalf("unexpected packed values at edges: first=%d before-last=%d last=%d", values[0], values[62], values[63])
	}
}

func TestEncodeBedrockChunkCarriesMappedTerrain(t *testing.T) {
	blocks := make([]int32, javaChunkSectionSize)
	blocks[0] = 1 // Java stone at local x=0,y=0,z=0.
	chunk := JavaChunk{X: 4, Z: -2, Sections: []JavaChunkSection{{Blocks: blocks}}}
	payload, sections, err := EncodeBedrockChunk(chunk, 0)
	if err != nil {
		t.Fatal(err)
	}
	if sections != 24 {
		t.Fatalf("overworld section count=%d", sections)
	}
	if len(payload) < 8 || payload[0] != 9 || payload[1] != 1 || payload[2] != 0xfc {
		t.Fatalf("unexpected first Bedrock section header: %v", payload[:minTest(len(payload), 8)])
	}
	if payload[3] != 3 {
		t.Fatalf("expected a two-entry 1-bit block palette, header=%d", payload[3])
	}

	airBlocks := make([]int32, javaChunkSectionSize)
	empty, _, err := EncodeBedrockChunk(JavaChunk{Sections: []JavaChunkSection{{Blocks: airBlocks}}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) < 3 || empty[0] != 9 || empty[1] != 0 || empty[2] != 0xfc {
		t.Fatalf("unexpected empty section header: %v", empty[:minTest(len(empty), 3)])
	}
}

func minTest(a, b int) int {
	if a < b {
		return a
	}
	return b
}
