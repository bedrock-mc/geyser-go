package translate

import (
	"bytes"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
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
	// 4096 one-bit indices occupy 128 little-endian uint32 words. The
	// following palette count is a signed ZigZag VarInt: 2 entries -> 4.
	if payload[516] != 4 {
		t.Fatalf("expected signed palette count 4, got byte %d", payload[516])
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

func TestEncodeBedrockChunkUsesZeroBitSingletonPalette(t *testing.T) {
	blocks := make([]int32, javaChunkSectionSize)
	for i := range blocks {
		blocks[i] = 1
	}
	chunk := JavaChunk{Sections: []JavaChunkSection{{Blocks: blocks}}}
	payload, _, err := EncodeBedrockChunk(chunk, 0)
	if err != nil {
		t.Fatal(err)
	}
	// A uniform non-air section still carries its singleton runtime ID, but
	// has no index words and therefore uses the zero-bit palette form.
	if len(payload) < 5 || payload[3] != 1 {
		t.Fatalf("expected zero-bit singleton storage header, got %v", payload[:minTest(len(payload), 5)])
	}
	runtimeID, known := JavaBlockRuntimeID(1)
	if !known {
		t.Fatal("Java block state 1 is not present in the generated mapping")
	}
	var expected bytes.Buffer
	writeSignedVarInt(&expected, int32(runtimeID))
	if len(payload) < 4+expected.Len() || !bytes.Equal(payload[4:4+expected.Len()], expected.Bytes()) {
		t.Fatalf("expected singleton palette runtime ID %d encoded at payload prefix, got %v", runtimeID, payload[4:minTest(len(payload), 4+expected.Len())])
	}
}

func TestDecodeOptionalNBTSupportsEndAndCompoundRoots(t *testing.T) {
	end := javaprotocol.NewWriter()
	_ = end.Byte(0)
	value, err := decodeOptionalNBT(javaprotocol.NewReader(end.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if value != nil {
		t.Fatalf("TAG_End optional NBT = %#v, want nil", value)
	}

	compound := javaprotocol.NewWriter()
	_ = compound.Byte(10) // TAG_Compound, anonymous root.
	_ = compound.Byte(8)  // TAG_String.
	_ = compound.Int16(4)
	_ = compound.BytesValue([]byte("text"))
	_ = compound.Int16(5)
	_ = compound.BytesValue([]byte("hello"))
	_ = compound.Byte(0) // TAG_End.
	value, err = decodeOptionalNBT(javaprotocol.NewReader(compound.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if value["text"] != "hello" {
		t.Fatalf("compound optional NBT text = %#v, want hello", value["text"])
	}
}

func TestEncodeBedrockChunkAppendsBlockEntityNBT(t *testing.T) {
	base, _, err := EncodeBedrockChunk(JavaChunk{X: 2, Z: -3}, 0)
	if err != nil {
		t.Fatal(err)
	}
	withEntity, _, err := EncodeBedrockChunk(JavaChunk{
		X: 2, Z: -3,
		BlockEntities: []JavaBlockEntity{{
			X: 5, Y: 70, Z: 14, Type: 1,
			Data: map[string]any{"Custom": int32(9)},
		}},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(withEntity) <= len(base) {
		t.Fatalf("block entity did not extend chunk payload: base=%d entity=%d", len(base), len(withEntity))
	}
	var tag map[string]any
	if err := nbt.UnmarshalEncoding(withEntity[len(base):], &tag, nbt.NetworkLittleEndian); err != nil {
		t.Fatal(err)
	}
	if tag["id"] != "Chest" || tag["x"] != int32(37) || tag["y"] != int32(70) || tag["z"] != int32(-34) || tag["Custom"] != int32(9) {
		t.Fatalf("encoded block entity tag = %#v", tag)
	}
}

func minTest(a, b int) int {
	if a < b {
		return a
	}
	return b
}
