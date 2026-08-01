package translate

import (
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeBlockChangePositionAndState(t *testing.T) {
	w := javaprotocol.NewWriter()
	position := gtprotocol.BlockPos{-12345, -17, 54321}
	if err := w.Int64(encodeJavaPosition(position)); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(42); err != nil {
		t.Fatal(err)
	}
	change, err := DecodeBlockChange(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if change.Position != position || change.StateID != 42 {
		t.Fatalf("decoded block change = %+v, want position %v state 42", change, position)
	}
}

func TestDecodeSpawnEntity(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(7); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		if err := w.Byte(byte(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.VarInt(47); err != nil { // experience_orb
		t.Fatal(err)
	}
	for _, value := range []float64{1.5, 64, -2.25} {
		if err := w.Float64(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []int8{-64, 64, 32} {
		if err := w.Byte(byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.VarInt(3); err != nil {
		t.Fatal(err)
	}
	for _, value := range []int16{8000, -4000, 0} {
		if err := w.Int16(value); err != nil {
			t.Fatal(err)
		}
	}
	spawn, err := DecodeSpawnEntity(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if spawn.EntityID != 7 || spawn.Type != 47 || spawn.ObjectData != 3 {
		t.Fatalf("decoded spawn header = %+v", spawn)
	}
	if got := spawn.Position; got[0] != 1.5 || got[1] != 64 || got[2] != -2.25 {
		t.Fatalf("decoded position = %v", got)
	}
	if math.Abs(float64(spawn.Velocity[0]-1)) > 0.0001 || math.Abs(float64(spawn.Velocity[1]+0.5)) > 0.0001 {
		t.Fatalf("decoded velocity = %v", spawn.Velocity)
	}
	if math.Abs(float64(spawn.Pitch+90)) > 0.0001 || math.Abs(float64(spawn.Yaw-90)) > 0.0001 {
		t.Fatalf("decoded rotation = pitch %f yaw %f", spawn.Pitch, spawn.Yaw)
	}
}

func TestDecodeEntityRelativeMove(t *testing.T) {
	w := javaprotocol.NewWriter()
	for _, value := range []any{int32(11), int16(4096), int16(-2048), int16(0), int8(64), int8(-64), true} {
		switch value := value.(type) {
		case int32:
			if err := w.VarInt(value); err != nil {
				t.Fatal(err)
			}
		case int16:
			if err := w.Int16(value); err != nil {
				t.Fatal(err)
			}
		case int8:
			if err := w.Byte(byte(value)); err != nil {
				t.Fatal(err)
			}
		case bool:
			if err := w.Bool(value); err != nil {
				t.Fatal(err)
			}
		}
	}
	move, err := DecodeEntityRelativeMove(w.Bytes(), true)
	if err != nil {
		t.Fatal(err)
	}
	if move.EntityID != 11 || move.Delta != (mgl32.Vec3{1, -0.5, 0}) || !move.OnGround {
		t.Fatalf("decoded relative move = %+v", move)
	}
}

func TestEmptyBedrockChunkPayload(t *testing.T) {
	payload := EmptyBedrockChunkPayload(0)
	if len(payload) != 26 || payload[0] != 1 || payload[1] != 0 || payload[len(payload)-1] != 0 {
		t.Fatalf("unexpected overworld empty payload length/content: len=%d bytes=%v", len(payload), payload)
	}
	for _, value := range payload[2 : len(payload)-1] {
		if value != 0xff {
			t.Fatalf("unexpected biome carry marker 0x%x", value)
		}
	}
}

func TestDecodeWorldPresentationPackets(t *testing.T) {
	difficultyWriter := javaprotocol.NewWriter()
	if err := difficultyWriter.Byte(2); err != nil {
		t.Fatal(err)
	}
	if err := difficultyWriter.Bool(true); err != nil {
		t.Fatal(err)
	}
	difficulty, err := DecodeJavaDifficulty(difficultyWriter.Bytes())
	if err != nil || difficulty.Difficulty != 2 || !difficulty.Locked {
		t.Fatalf("difficulty = %+v, err=%v", difficulty, err)
	}

	stateWriter := javaprotocol.NewWriter()
	if err := stateWriter.Byte(javaGameEventChangeGameMode); err != nil {
		t.Fatal(err)
	}
	if err := stateWriter.Float32(3); err != nil {
		t.Fatal(err)
	}
	state, err := DecodeJavaGameStateChange(stateWriter.Bytes())
	if err != nil || state.Reason != javaGameEventChangeGameMode || state.Value != 3 {
		t.Fatalf("game state = %+v, err=%v", state, err)
	}

	titleWriter := javaprotocol.NewWriter()
	if err := titleWriter.Byte(8); err != nil { // anonymous NBT TAG_String
		t.Fatal(err)
	}
	if err := titleWriter.Int16(5); err != nil {
		t.Fatal(err)
	}
	if err := titleWriter.BytesValue([]byte("Hello")); err != nil {
		t.Fatal(err)
	}
	title, err := DecodeJavaTitleText(titleWriter.Bytes())
	if err != nil || JavaTextComponentText(title) != "Hello" {
		t.Fatalf("title = %#v, text=%q, err=%v", title, JavaTextComponentText(title), err)
	}

	timesWriter := javaprotocol.NewWriter()
	for _, value := range []int32{10, 70, 20} {
		if err := timesWriter.Int32(value); err != nil {
			t.Fatal(err)
		}
	}
	times, err := DecodeJavaTitleTimes(timesWriter.Bytes())
	if err != nil || times != (JavaTitleTimes{FadeIn: 10, Stay: 70, FadeOut: 20}) {
		t.Fatalf("title times = %+v, err=%v", times, err)
	}

	eventWriter := javaprotocol.NewWriter()
	if err := eventWriter.Int32(2001); err != nil {
		t.Fatal(err)
	}
	eventPosition := gtprotocol.BlockPos{-12, 63, 34}
	if err := eventWriter.Int64(encodeJavaPosition(eventPosition)); err != nil {
		t.Fatal(err)
	}
	if err := eventWriter.Int32(7); err != nil {
		t.Fatal(err)
	}
	if err := eventWriter.Bool(false); err != nil {
		t.Fatal(err)
	}
	event, err := DecodeJavaWorldEvent(eventWriter.Bytes())
	if err != nil || event.EffectID != 2001 || event.Position != eventPosition || event.Data != 7 || event.Global {
		t.Fatalf("world event = %+v, err=%v", event, err)
	}

	spawnWriter := javaprotocol.NewWriter()
	spawnPosition := gtprotocol.BlockPos{8, 72, -9}
	if err := spawnWriter.Int64(encodeJavaPosition(spawnPosition)); err != nil {
		t.Fatal(err)
	}
	if err := spawnWriter.Float32(135); err != nil {
		t.Fatal(err)
	}
	spawn, err := DecodeJavaSpawnPosition(spawnWriter.Bytes())
	if err != nil || spawn.Position != spawnPosition || spawn.Angle != 135 {
		t.Fatalf("spawn position = %+v, err=%v", spawn, err)
	}
}

func TestDecodeJavaRespawn(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(1); err != nil {
		t.Fatal(err)
	}
	if err := w.String("minecraft:the_nether"); err != nil {
		t.Fatal(err)
	}
	if err := w.Int64(99); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(2); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(1); err != nil {
		t.Fatal(err)
	}
	if err := w.Bool(false); err != nil {
		t.Fatal(err)
	}
	if err := w.Bool(true); err != nil {
		t.Fatal(err)
	}
	if err := w.Bool(true); err != nil {
		t.Fatal(err)
	}
	if err := w.String("minecraft:the_nether"); err != nil {
		t.Fatal(err)
	}
	if err := w.Int64(encodeJavaPosition(gtprotocol.BlockPos{1, 65, 2})); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(80); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(32); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(3); err != nil {
		t.Fatal(err)
	}
	respawn, err := DecodeJavaRespawn(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if respawn.World.Name != "minecraft:the_nether" || respawn.World.Dimension != 1 || respawn.World.HashedSeed != 99 || respawn.World.GameMode != 2 || !respawn.World.Flat || respawn.World.PortalCooldown != 80 || respawn.World.SeaLevel != 32 || respawn.CopyMetadata != 3 {
		t.Fatalf("decoded respawn = %+v", respawn)
	}
}
