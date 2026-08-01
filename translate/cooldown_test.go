package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJavaCooldown(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.String("minecraft:shield"); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(20); err != nil {
		t.Fatal(err)
	}
	cooldown, err := DecodeJavaCooldown(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if cooldown.Group != "minecraft:shield" || cooldown.Ticks != 20 {
		t.Fatalf("unexpected cooldown: %#v", cooldown)
	}
}

func TestDecodeJavaCooldownRejectsTrailingBytes(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.String("minecraft:shield")
	_ = w.VarInt(20)
	_ = w.Byte(1)
	if _, err := DecodeJavaCooldown(w.Bytes()); err == nil {
		t.Fatal("expected trailing-byte error")
	}
}

func TestBedrockCooldownCategory(t *testing.T) {
	if got := bedrockCooldownCategory("minecraft:goat_horn"); got != "goat_horn" {
		t.Fatalf("goat horn category=%q", got)
	}
	if got := bedrockCooldownCategory("minecraft:shield"); got != "shield" {
		t.Fatalf("shield category=%q", got)
	}
	if got := bedrockCooldownCategory("example:custom"); got != "example:custom" {
		t.Fatalf("custom category=%q", got)
	}
}
