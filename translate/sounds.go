package translate

import (
	"fmt"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const javaSoundCategoryCount = 10

type JavaSoundEffect struct {
	Sound    string
	Known    bool
	Category int32
	Position mgl32.Vec3
	Volume   float32
	Pitch    float32
	Seed     int64
}

type JavaEntitySoundEffect struct {
	Sound    string
	Known    bool
	Category int32
	EntityID int32
	Volume   float32
	Pitch    float32
	Seed     int64
}

type JavaStopSound struct {
	Category *int32
	Sound    *string
	Flags    byte
}

func decodeJavaSoundHolder(r *javaprotocol.Reader) (string, bool, error) {
	id, err := r.VarInt()
	if err != nil {
		return "", false, fmt.Errorf("translate: sound holder ID: %w", err)
	}
	if id == 0 {
		name, err := r.String()
		if err != nil {
			return "", false, fmt.Errorf("translate: custom sound name: %w", err)
		}
		newSystem, err := r.Bool()
		if err != nil {
			return "", false, fmt.Errorf("translate: custom sound range flag: %w", err)
		}
		if newSystem {
			if _, err := r.Float32(); err != nil {
				return "", false, fmt.Errorf("translate: custom sound range: %w", err)
			}
		}
		return name, name != "", nil
	}
	name, known := data.Java1214SoundName(id)
	return name, known, nil
}

func DecodeJavaSoundEffect(dataBytes []byte) (JavaSoundEffect, error) {
	r := javaprotocol.NewReader(dataBytes)
	sound, known, err := decodeJavaSoundHolder(r)
	if err != nil {
		return JavaSoundEffect{}, err
	}
	category, err := r.VarInt()
	if err != nil {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound category: %w", err)
	}
	x, err := r.Int32()
	if err != nil {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound X: %w", err)
	}
	y, err := r.Int32()
	if err != nil {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound Y: %w", err)
	}
	z, err := r.Int32()
	if err != nil {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound Z: %w", err)
	}
	volume, err := r.Float32()
	if err != nil {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound volume: %w", err)
	}
	pitch, err := r.Float32()
	if err != nil {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound pitch: %w", err)
	}
	seed, err := r.Int64()
	if err != nil {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound seed: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaSoundEffect{}, fmt.Errorf("translate: sound effect has %d trailing bytes", r.Remaining())
	}
	return JavaSoundEffect{
		Sound: sound, Known: known, Category: category,
		Position: mgl32.Vec3{float32(x) / 8, float32(y) / 8, float32(z) / 8},
		Volume:   volume, Pitch: pitch, Seed: seed,
	}, nil
}

func DecodeJavaEntitySoundEffect(dataBytes []byte) (JavaEntitySoundEffect, error) {
	r := javaprotocol.NewReader(dataBytes)
	sound, known, err := decodeJavaSoundHolder(r)
	if err != nil {
		return JavaEntitySoundEffect{}, err
	}
	category, err := r.VarInt()
	if err != nil {
		return JavaEntitySoundEffect{}, fmt.Errorf("translate: entity sound category: %w", err)
	}
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntitySoundEffect{}, fmt.Errorf("translate: entity sound entity ID: %w", err)
	}
	volume, err := r.Float32()
	if err != nil {
		return JavaEntitySoundEffect{}, fmt.Errorf("translate: entity sound volume: %w", err)
	}
	pitch, err := r.Float32()
	if err != nil {
		return JavaEntitySoundEffect{}, fmt.Errorf("translate: entity sound pitch: %w", err)
	}
	seed, err := r.Int64()
	if err != nil {
		return JavaEntitySoundEffect{}, fmt.Errorf("translate: entity sound seed: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaEntitySoundEffect{}, fmt.Errorf("translate: entity sound effect has %d trailing bytes", r.Remaining())
	}
	return JavaEntitySoundEffect{
		Sound: sound, Known: known, Category: category, EntityID: entityID,
		Volume: volume, Pitch: pitch, Seed: seed,
	}, nil
}

func DecodeJavaStopSound(dataBytes []byte) (JavaStopSound, error) {
	r := javaprotocol.NewReader(dataBytes)
	flags, err := r.Byte()
	if err != nil {
		return JavaStopSound{}, fmt.Errorf("translate: stop sound flags: %w", err)
	}
	stop := JavaStopSound{Flags: flags}
	if flags&0x01 != 0 {
		category, err := r.VarInt()
		if err != nil {
			return JavaStopSound{}, fmt.Errorf("translate: stop sound category: %w", err)
		}
		stop.Category = &category
	}
	if flags&0x02 != 0 {
		sound, err := r.String()
		if err != nil {
			return JavaStopSound{}, fmt.Errorf("translate: stop sound name: %w", err)
		}
		stop.Sound = &sound
	}
	if r.Remaining() != 0 {
		return JavaStopSound{}, fmt.Errorf("translate: stop sound has %d trailing bytes", r.Remaining())
	}
	return stop, nil
}

func validateJavaSound(category int32, volume, pitch float32) bool {
	return category >= 0 && category < javaSoundCategoryCount && finiteFloat32(volume) && finiteFloat32(pitch)
}

func (b *Basic) translateJavaSoundEffect(bedrock *minecraft.Conn, dataBytes []byte) error {
	sound, err := DecodeJavaSoundEffect(dataBytes)
	if err != nil {
		return err
	}
	if !sound.Known {
		b.logSemanticAnomaly("skipping Java sound outside the generated registry", "sound", sound.Sound)
		return nil
	}
	if !validateJavaSound(sound.Category, sound.Volume, sound.Pitch) {
		b.logSemanticAnomaly("skipping Java sound with invalid category or non-finite volume/pitch", "category", sound.Category)
		return nil
	}
	return bedrock.WritePacket(&packet.PlaySound{
		SoundName: data.BedrockSoundName(sound.Sound),
		Position:  sound.Position,
		Volume:    sound.Volume,
		Pitch:     sound.Pitch,
	})
}

func (b *Basic) translateJavaEntitySoundEffect(bedrock *minecraft.Conn, dataBytes []byte) error {
	sound, err := DecodeJavaEntitySoundEffect(dataBytes)
	if err != nil {
		return err
	}
	if !sound.Known {
		b.logSemanticAnomaly("skipping Java entity sound outside the generated registry", "sound", sound.Sound)
		return nil
	}
	if !validateJavaSound(sound.Category, sound.Volume, sound.Pitch) {
		b.logSemanticAnomaly("skipping Java entity sound with invalid category or non-finite volume/pitch", "category", sound.Category)
		return nil
	}
	b.mu.Lock()
	var position mgl32.Vec3
	var known bool
	if sound.EntityID == int32(b.gameData.EntityUniqueID) {
		position = mgl32.Vec3{float32(b.position.x), float32(b.position.y), float32(b.position.z)}
		known = true
	} else if entity := b.entities[sound.EntityID]; entity != nil {
		position = entity.position
		known = true
	}
	b.mu.Unlock()
	if !known {
		b.logSemanticAnomaly("skipping Java entity sound for unknown entity", "entity", sound.EntityID)
		return nil
	}
	return bedrock.WritePacket(&packet.PlaySound{
		SoundName: data.BedrockSoundName(sound.Sound),
		Position:  position,
		Volume:    sound.Volume,
		Pitch:     sound.Pitch,
	})
}

func (b *Basic) translateJavaStopSound(bedrock *minecraft.Conn, dataBytes []byte) error {
	stop, err := DecodeJavaStopSound(dataBytes)
	if err != nil {
		return err
	}
	if stop.Flags&^byte(0x03) != 0 {
		b.logSemanticAnomaly("Java stop-sound packet has unknown flags", "flags", stop.Flags)
	}
	if stop.Category != nil && (*stop.Category < 0 || *stop.Category >= javaSoundCategoryCount) {
		b.logSemanticAnomaly("Java stop-sound packet has unknown category", "category", *stop.Category)
	}
	if stop.Sound == nil {
		return bedrock.WritePacket(&packet.StopSound{StopAll: true})
	}
	return bedrock.WritePacket(&packet.StopSound{SoundName: data.BedrockSoundName(*stop.Sound)})
}
