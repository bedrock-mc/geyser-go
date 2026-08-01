package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJavaBossBar(t *testing.T) {
	w := javaprotocol.NewWriter()
	uuid := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := w.BytesValue(uuid[:]); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(javaBossBarAdd); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(8); err != nil { // anonymous NBT TAG_String
		t.Fatal(err)
	}
	if err := w.Int16(4); err != nil {
		t.Fatal(err)
	}
	if err := w.BytesValue([]byte("Test")); err != nil {
		t.Fatal(err)
	}
	if err := w.Float32(0.5); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(1); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(2); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(5); err != nil {
		t.Fatal(err)
	}
	bar, err := DecodeJavaBossBar(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if bar.UUID != uuid || bar.Action != javaBossBarAdd || JavaTextComponentText(bar.Title) != "Test" || bar.Health != 0.5 || bar.Color != 1 || bar.Style != 2 || bar.Flags != 5 {
		t.Fatalf("boss bar = %+v, title=%q", bar, JavaTextComponentText(bar.Title))
	}

	update := javaprotocol.NewWriter()
	_ = update.BytesValue(uuid[:])
	_ = update.VarInt(javaBossBarHealth)
	_ = update.Float32(0.25)
	decoded, err := DecodeJavaBossBar(update.Bytes())
	if err != nil || decoded.Action != javaBossBarHealth || decoded.Health != 0.25 {
		t.Fatalf("health update = %+v, err=%v", decoded, err)
	}
}

func TestBossBarNormalization(t *testing.T) {
	if got := clampBossBarHealth(2); got != 1 {
		t.Fatalf("high health = %f", got)
	}
	if got := clampBossBarHealth(-1); got != 0 {
		t.Fatalf("low health = %f", got)
	}
	if got := bossBarColor(99); got != 7 {
		t.Fatalf("unknown color = %d", got)
	}
	if got := bossBarColor(6); got != 7 {
		t.Fatalf("Java white color = %d", got)
	}
	if got := bossBarOverlay(99); got != 0 {
		t.Fatalf("unknown overlay = %d", got)
	}
}
