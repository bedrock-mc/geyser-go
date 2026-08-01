package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJavaSoundEffect(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(8) // ambient.cave: registry ordinal 7, holder ID 8.
	_ = w.VarInt(4)
	_ = w.Int32(80)
	_ = w.Int32(576)
	_ = w.Int32(-160)
	_ = w.Float32(0.75)
	_ = w.Float32(1.25)
	_ = w.Int64(99)
	sound, err := DecodeJavaSoundEffect(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !sound.Known || sound.Sound != "ambient.cave" || sound.Category != 4 || sound.Position.X() != 10 || sound.Position.Y() != 72 || sound.Position.Z() != -20 || sound.Volume != 0.75 || sound.Pitch != 1.25 || sound.Seed != 99 {
		t.Fatalf("sound = %+v", sound)
	}

	custom := javaprotocol.NewWriter()
	_ = custom.VarInt(0)
	_ = custom.String("example.custom")
	_ = custom.Bool(true)
	_ = custom.Float32(24)
	_ = custom.VarInt(0)
	_ = custom.Int32(0)
	_ = custom.Int32(0)
	_ = custom.Int32(0)
	_ = custom.Float32(1)
	_ = custom.Float32(1)
	_ = custom.Int64(0)
	customSound, err := DecodeJavaSoundEffect(custom.Bytes())
	if err != nil || !customSound.Known || customSound.Sound != "example.custom" {
		t.Fatalf("custom sound = %+v, err=%v", customSound, err)
	}
}

func TestDecodeJavaEntityAndStopSound(t *testing.T) {
	entityWriter := javaprotocol.NewWriter()
	_ = entityWriter.VarInt(0)
	_ = entityWriter.String("entity.custom")
	_ = entityWriter.Bool(false)
	_ = entityWriter.VarInt(7)
	_ = entityWriter.VarInt(42)
	_ = entityWriter.Float32(1)
	_ = entityWriter.Float32(0.5)
	_ = entityWriter.Int64(123)
	entity, err := DecodeJavaEntitySoundEffect(entityWriter.Bytes())
	if err != nil || entity.EntityID != 42 || entity.Sound != "entity.custom" || entity.Category != 7 {
		t.Fatalf("entity sound = %+v, err=%v", entity, err)
	}

	stopWriter := javaprotocol.NewWriter()
	_ = stopWriter.Byte(0x03)
	_ = stopWriter.VarInt(2)
	_ = stopWriter.String("minecraft:music.game")
	stop, err := DecodeJavaStopSound(stopWriter.Bytes())
	if err != nil || stop.Category == nil || *stop.Category != 2 || stop.Sound == nil || *stop.Sound != "minecraft:music.game" {
		t.Fatalf("stop sound = %+v, err=%v", stop, err)
	}

	allWriter := javaprotocol.NewWriter()
	_ = allWriter.Byte(0)
	all, err := DecodeJavaStopSound(allWriter.Bytes())
	if err != nil || all.Sound != nil || all.Category != nil {
		t.Fatalf("stop-all = %+v, err=%v", all, err)
	}
}
