package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeMultiBlockChange(t *testing.T) {
	sectionPosition := (int64(3) << 42) | (int64(5) << 20) | int64(2)
	first := int32((17 << 12) | (1 << 8) | (2 << 4) | 3)
	second := int32((23 << 12) | (15 << 8) | (0 << 4) | 15)
	w := javaprotocol.NewWriter()
	if err := w.Int64(sectionPosition); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(2); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(first); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(second); err != nil {
		t.Fatal(err)
	}
	update, err := DecodeMultiBlockChange(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if update.SectionX != 3 || update.SectionY != 2 || update.SectionZ != 5 || len(update.Changes) != 2 {
		t.Fatalf("section update = %+v", update)
	}
	if update.Changes[0].StateID != 17 || update.Changes[0].Position != [3]int32{49, 35, 82} {
		t.Fatalf("first change = %+v", update.Changes[0])
	}
	if update.Changes[1].StateID != 23 || update.Changes[1].Position != [3]int32{63, 47, 80} {
		t.Fatalf("second change = %+v", update.Changes[1])
	}
}

func TestDecodeMultiBlockChangeNegativeSection(t *testing.T) {
	packed := (uint64(0x3ffffe) << 42) | (uint64(0x3ffffc) << 20) | uint64(0xffffd)
	sectionPosition := int64(packed)
	w := javaprotocol.NewWriter()
	if err := w.Int64(sectionPosition); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(0); err != nil {
		t.Fatal(err)
	}
	update, err := DecodeMultiBlockChange(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if update.SectionX != -2 || update.SectionY != -3 || update.SectionZ != -4 {
		t.Fatalf("negative section = %+v", update)
	}
}
