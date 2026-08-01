package translate

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ErrUnsupportedJavaParticle means that a well-formed Java particle uses a
// registry/data variant this tranche cannot project. The enclosing packet is
// already bounded by Java framing, so callers skip it without disconnecting.
var ErrUnsupportedJavaParticle = errors.New("translate: unsupported Java particle")

type javaParticleDataKind byte

const (
	javaParticleNoData javaParticleDataKind = iota
	javaParticleBlockState
	javaParticleDust
	javaParticleDustTransition
	javaParticleEntityEffect
	javaParticleItem
	javaParticleSculkCharge
	javaParticleShriek
	javaParticleVibration
	javaParticleTrail
)

// JavaParticle is the bounded subset of the protocol's Particle union. The
// type ID is the 1.21.4 particle registry ID; data is retained only for the
// variants that need it for a Bedrock event or for safe diagnostics.
type JavaParticle struct {
	ID                    int32
	Kind                  javaParticleDataKind
	BlockStateID          int32
	Item                  JavaItemSlot
	Color                 int32
	TransitionColor       int32
	Scale                 float32
	ShriekTicks           int32
	VibrationPositionType int32
	VibrationBlock        gtprotocol.BlockPos
	VibrationEntityID     int32
	VibrationYOffset      float32
	VibrationArrivalTicks int32
	TrailTarget           mgl32.Vec3
	TrailColor            int32
	TrailDuration         int32
}

type JavaLevelParticles struct {
	LongDistance bool
	AlwaysShow   bool
	Position     mgl32.Vec3
	Offset       mgl32.Vec3
	Velocity     float32
	Amount       int32
	Particle     JavaParticle
}

// DecodeJavaLevelParticles consumes ClientboundLevelParticlesPacket for the
// 1.21.4 profile. The Particle union is parsed in full for every known wire
// variant, which prevents a semantically unsupported particle from shifting
// the rest of a packet or session.
func DecodeJavaLevelParticles(dataBytes []byte, nextStackID func() int32) (JavaLevelParticles, error) {
	r := javaprotocol.NewReader(dataBytes)
	longDistance, err := r.Bool()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles long-distance flag: %w", err)
	}
	alwaysShow, err := r.Bool()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles always-show flag: %w", err)
	}
	x, err := r.Float64()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles X: %w", err)
	}
	y, err := r.Float64()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles Y: %w", err)
	}
	z, err := r.Float64()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles Z: %w", err)
	}
	offsetX, err := r.Float32()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles offset X: %w", err)
	}
	offsetY, err := r.Float32()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles offset Y: %w", err)
	}
	offsetZ, err := r.Float32()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles offset Z: %w", err)
	}
	velocity, err := r.Float32()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles velocity offset: %w", err)
	}
	amount, err := r.Int32()
	if err != nil {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles amount: %w", err)
	}
	particle, err := decodeJavaParticle(r, nextStackID)
	if err != nil {
		return JavaLevelParticles{}, err
	}
	if r.Remaining() != 0 {
		return JavaLevelParticles{}, fmt.Errorf("translate: particles has %d trailing bytes", r.Remaining())
	}
	return JavaLevelParticles{
		LongDistance: longDistance,
		AlwaysShow:   alwaysShow,
		Position:     mgl32.Vec3{float32(x), float32(y), float32(z)},
		Offset:       mgl32.Vec3{offsetX, offsetY, offsetZ},
		Velocity:     velocity,
		Amount:       amount,
		Particle:     particle,
	}, nil
}

func decodeJavaParticle(r *javaprotocol.Reader, nextStackID func() int32) (JavaParticle, error) {
	id, err := r.VarInt()
	if err != nil {
		return JavaParticle{}, fmt.Errorf("translate: particle type: %w", err)
	}
	particle := JavaParticle{ID: id}
	switch id {
	case 1, 2, 28, 107, 111: // block, block marker, falling dust, dust pillar, block crumble
		particle.Kind = javaParticleBlockState
		particle.BlockStateID, err = r.VarInt()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: particle %d block state: %w", id, err)
		}
	case 13: // dust
		particle.Kind = javaParticleDust
		particle.Color, err = r.Int32()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: dust particle color: %w", err)
		}
		particle.Scale, err = r.Float32()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: dust particle scale: %w", err)
		}
	case 14: // dust color transition
		particle.Kind = javaParticleDustTransition
		particle.Color, err = r.Int32()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: dust transition particle color: %w", err)
		}
		particle.TransitionColor, err = r.Int32()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: dust transition particle target color: %w", err)
		}
		particle.Scale, err = r.Float32()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: dust transition particle scale: %w", err)
		}
	case 20: // entity effect color
		particle.Kind = javaParticleEntityEffect
		particle.Color, err = r.Int32()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: entity-effect particle color: %w", err)
		}
	case 36: // sculk charge
		particle.Kind = javaParticleSculkCharge
		particle.Scale, err = r.Float32()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: sculk-charge particle scale: %w", err)
		}
	case 45: // item
		particle.Kind = javaParticleItem
		particle.Item, err = decodeJavaItemSlot(r, nextStackID)
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: item particle item: %w", err)
		}
	case 46: // vibration
		particle.Kind = javaParticleVibration
		vibration, decodeErr := decodeJavaVibration(r)
		if decodeErr != nil {
			return JavaParticle{}, decodeErr
		}
		particle.VibrationPositionType = vibration.PositionType
		particle.VibrationBlock = vibration.Block
		particle.VibrationEntityID = vibration.EntityID
		particle.VibrationYOffset = vibration.EntityYOffset
		particle.VibrationArrivalTicks = vibration.ArrivalTicks
	case 47: // trail
		particle.Kind = javaParticleTrail
		trail, decodeErr := decodeJavaTrail(r)
		if decodeErr != nil {
			return JavaParticle{}, decodeErr
		}
		particle.TrailTarget = trail.Target
		particle.TrailColor = trail.Color
		particle.TrailDuration = trail.Duration
	case 101: // shriek
		particle.Kind = javaParticleShriek
		particle.ShriekTicks, err = r.VarInt()
		if err != nil {
			return JavaParticle{}, fmt.Errorf("translate: shriek particle delay: %w", err)
		}
	case 0, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 15, 16, 17, 18, 19, 21, 22, 23, 24, 25, 26, 27, 29, 30, 31, 32, 33, 34, 35, 37, 38, 39, 40, 41, 42, 43, 44, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 102, 103, 104, 105, 106, 108, 109, 110:
		// No additional payload.
	default:
		return particle, fmt.Errorf("%w: registry ID %d", ErrUnsupportedJavaParticle, id)
	}
	return particle, nil
}

type javaVibrationData struct {
	PositionType  int32
	Block         gtprotocol.BlockPos
	EntityID      int32
	EntityYOffset float32
	ArrivalTicks  int32
}

func decodeJavaVibration(r *javaprotocol.Reader) (javaVibrationData, error) {
	positionType, err := r.VarInt()
	if err != nil {
		return javaVibrationData{}, fmt.Errorf("translate: vibration position type: %w", err)
	}
	vibration := javaVibrationData{PositionType: positionType}
	switch positionType {
	case 0:
		packed, readErr := r.Int64()
		if readErr != nil {
			return javaVibrationData{}, fmt.Errorf("translate: vibration block position: %w", readErr)
		}
		vibration.Block = decodeJavaPosition(packed)
	case 1:
		vibration.EntityID, err = r.VarInt()
		if err != nil {
			return javaVibrationData{}, fmt.Errorf("translate: vibration entity ID: %w", err)
		}
		vibration.EntityYOffset, err = r.Float32()
		if err != nil {
			return javaVibrationData{}, fmt.Errorf("translate: vibration entity eye height: %w", err)
		}
	default:
		return javaVibrationData{}, fmt.Errorf("%w: vibration position type %d", ErrUnsupportedJavaParticle, positionType)
	}
	vibration.ArrivalTicks, err = r.VarInt()
	if err != nil {
		return javaVibrationData{}, fmt.Errorf("translate: vibration arrival ticks: %w", err)
	}
	if vibration.ArrivalTicks < 0 {
		return javaVibrationData{}, fmt.Errorf("%w: negative vibration arrival ticks %d", ErrUnsupportedJavaParticle, vibration.ArrivalTicks)
	}
	return vibration, nil
}

type javaTrailData struct {
	Target   mgl32.Vec3
	Color    int32
	Duration int32
}

func decodeJavaTrail(r *javaprotocol.Reader) (javaTrailData, error) {
	var target [3]float64
	for i := 0; i < 3; i++ {
		value, err := r.Float64()
		if err != nil {
			return javaTrailData{}, fmt.Errorf("translate: trail target coordinate: %w", err)
		}
		target[i] = value
	}
	color, err := r.Int32()
	if err != nil {
		return javaTrailData{}, fmt.Errorf("translate: trail color: %w", err)
	}
	duration, err := r.VarInt()
	if err != nil {
		return javaTrailData{}, fmt.Errorf("translate: trail duration: %w", err)
	}
	if duration < 0 {
		return javaTrailData{}, fmt.Errorf("%w: negative trail duration %d", ErrUnsupportedJavaParticle, duration)
	}
	return javaTrailData{Target: mgl32.Vec3{float32(target[0]), float32(target[1]), float32(target[2])}, Color: color, Duration: duration}, nil
}

type javaParticleMapping struct {
	eventType  int32
	identifier string
}

func bedrockParticleType(ordinal int32) int32 {
	return packet.LevelEventParticleLegacyEvent + ordinal
}

// javaParticleMappings is the versioned common projection from Geyser's
// 1.21.4 particles.json. Entries without a Bedrock equivalent are omitted and
// are logged as semantic skips. The specialized payload variants above are
// handled before this table.
var javaParticleMappings = map[int32]javaParticleMapping{
	0:   {eventType: bedrockParticleType(38), identifier: "minecraft:villager_angry"},
	3:   {eventType: bedrockParticleType(1), identifier: "minecraft:basic_bubble_particle_manual"},
	4:   {eventType: bedrockParticleType(7), identifier: "minecraft:water_evaporation_bucket_emitter"},
	5:   {eventType: bedrockParticleType(3), identifier: "minecraft:critical_hit_emitter"},
	6:   {identifier: "geyseropt:damage_indicator"},
	7:   {eventType: bedrockParticleType(47), identifier: "minecraft:dragon_breath_lingering"},
	8:   {eventType: bedrockParticleType(27), identifier: "minecraft:lava_drip_particle"},
	9:   {eventType: bedrockParticleType(27), identifier: "minecraft:lava_drip_particle"},
	10:  {identifier: "geyseropt:landing_lava"},
	11:  {eventType: bedrockParticleType(26), identifier: "minecraft:water_drip_particle"},
	12:  {identifier: "minecraft:water_splash_particle"},
	15:  {identifier: "minecraft:splash_spell_emitter"},
	16:  {eventType: packet.LevelEventParticleSoundGuardianGhost},
	17:  {identifier: "geyseropt:enchanted_hit_single"},
	18:  {eventType: bedrockParticleType(40), identifier: "minecraft:enchanting_table_particle"},
	19:  {eventType: bedrockParticleType(46), identifier: "minecraft:endrod"},
	20:  {eventType: bedrockParticleType(32), identifier: "minecraft:evoker_spell"},
	21:  {identifier: "minecraft:huge_explosion_emitter"},
	22:  {eventType: bedrockParticleType(16), identifier: "minecraft:large_explosion"},
	26:  {eventType: packet.LevelEventParticlesWindExplosion},
	27:  {identifier: "minecraft:sonic_explosion"},
	28:  {eventType: bedrockParticleType(30)},
	29:  {eventType: bedrockParticleType(50), identifier: "minecraft:sparkler_emitter"},
	30:  {eventType: bedrockParticleType(25), identifier: "minecraft:water_wake_particle"},
	31:  {eventType: bedrockParticleType(8), identifier: "minecraft:basic_flame_particle"},
	33:  {eventType: bedrockParticleType(86), identifier: "minecraft:cherry_leaves_particle"},
	35:  {eventType: bedrockParticleType(83), identifier: "minecraft:sculk_soul_particle"},
	36:  {eventType: packet.LevelEventSculkCharge},
	37:  {eventType: packet.LevelEventSculkChargePop},
	38:  {eventType: bedrockParticleType(70), identifier: "minecraft:blue_flame_particle"},
	39:  {eventType: bedrockParticleType(71), identifier: "minecraft:soul_particle"},
	40:  {identifier: "geyseropt:flash"},
	41:  {eventType: bedrockParticleType(39), identifier: "minecraft:villager_happy"},
	42:  {eventType: bedrockParticleType(39), identifier: "minecraft:villager_happy"},
	43:  {eventType: bedrockParticleType(18), identifier: "minecraft:heart_particle"},
	44:  {eventType: bedrockParticleType(34), identifier: "minecraft:mobspell_emitter"},
	45:  {eventType: bedrockParticleType(13), identifier: "minecraft:breaking_item_icon"},
	48:  {eventType: bedrockParticleType(36)},
	50:  {eventType: bedrockParticleType(14)},
	51:  {eventType: bedrockParticleType(10), identifier: "minecraft:water_evaporation_actor_emitter"},
	52:  {eventType: bedrockParticleType(9), identifier: "minecraft:lava_particle"},
	53:  {identifier: "minecraft:mycelium_dust_particle"},
	54:  {eventType: bedrockParticleType(42), identifier: "minecraft:note_particle"},
	55:  {eventType: bedrockParticleType(6), identifier: "minecraft:explosion_manual"},
	56:  {eventType: bedrockParticleType(21), identifier: "minecraft:mob_portal"},
	57:  {eventType: bedrockParticleType(37)},
	58:  {eventType: bedrockParticleType(5)},
	59:  {eventType: bedrockParticleType(88)},
	60:  {eventType: bedrockParticleType(60)},
	61:  {eventType: bedrockParticleType(48), identifier: "minecraft:llama_spit_smoke"},
	62:  {eventType: bedrockParticleType(35)},
	63:  {identifier: "geyseropt:sweep_attack"},
	64:  {eventType: bedrockParticleType(49), identifier: "minecraft:totem_particle"},
	65:  {identifier: "geyseropt:underwater"},
	66:  {eventType: bedrockParticleType(23), identifier: "minecraft:water_splash_particle_manual"},
	67:  {eventType: bedrockParticleType(43)},
	68:  {eventType: bedrockParticleType(1), identifier: "minecraft:bubble_column_bubble"},
	69:  {eventType: bedrockParticleType(59), identifier: "minecraft:bubble_column_down_particle"},
	70:  {eventType: bedrockParticleType(58), identifier: "minecraft:bubble_column_up_particle"},
	71:  {identifier: "geyseropt:nautilus"},
	72:  {identifier: "minecraft:dolphin_move_particle"},
	73:  {eventType: bedrockParticleType(66), identifier: "minecraft:campfire_smoke_particle"},
	74:  {eventType: bedrockParticleType(67), identifier: "minecraft:campfire_tall_smoke_particle"},
	75:  {eventType: bedrockParticleType(28)},
	76:  {identifier: "minecraft:honey_drip_particle"},
	77:  {identifier: "geyseropt:landing_honey"},
	78:  {identifier: "minecraft:nectar_drip_particle"},
	79:  {eventType: bedrockParticleType(77), identifier: "minecraft:spore_blossom_shower_particle"},
	80:  {identifier: "geyseropt:ash"},
	81:  {identifier: "geyseropt:crimson_spore"},
	82:  {identifier: "geyseropt:warped_spore"},
	83:  {identifier: "minecraft:spore_blossom_ambient_particle"},
	84:  {eventType: bedrockParticleType(72)},
	85:  {eventType: bedrockParticleType(72)},
	86:  {identifier: "geyseropt:landing_obsidian_tear"},
	87:  {eventType: bedrockParticleType(73), identifier: "minecraft:portal_reverse_particle"},
	88:  {identifier: "geyseropt:white_ash"},
	89:  {eventType: bedrockParticleType(81), identifier: "minecraft:candle_flame_particle"},
	90:  {eventType: bedrockParticleType(74)},
	91:  {eventType: bedrockParticleType(30), identifier: "minecraft:stalactite_lava_drip_particle"},
	92:  {eventType: bedrockParticleType(30), identifier: "minecraft:stalactite_lava_drip_particle"},
	93:  {eventType: bedrockParticleType(29), identifier: "minecraft:stalactite_water_drip_particle"},
	94:  {eventType: bedrockParticleType(29), identifier: "minecraft:stalactite_water_drip_particle"},
	95:  {eventType: bedrockParticleType(35)},
	96:  {identifier: "minecraft:glow_particle"},
	97:  {eventType: bedrockParticleType(79)},
	98:  {eventType: bedrockParticleType(79)},
	99:  {eventType: bedrockParticleType(80)},
	100: {eventType: packet.LevelEventScrape},
	101: {eventType: bedrockParticleType(82), identifier: "minecraft:shriek_particle"},
	102: {identifier: "minecraft:villager_happy"},
	103: {eventType: bedrockParticleType(87)},
	106: {eventType: bedrockParticleType(91)},
}

func (b *Basic) translateJavaLevelParticles(bedrock *minecraft.Conn, dataBytes []byte) error {
	particles, err := DecodeJavaLevelParticles(dataBytes, b.nextStackNetworkID)
	if err != nil {
		if errors.Is(err, ErrUnsupportedJavaParticle) || errors.Is(err, ErrUnsupportedJavaItemComponent) {
			b.logSemanticAnomaly("skipping Java particle with unsupported registry/data", "error", err)
			return nil
		}
		return err
	}
	if !finiteVec3(particles.Position) || !finiteVec3(particles.Offset) || !finiteFloat32(particles.Velocity) || particles.Amount < 0 {
		b.logSemanticAnomaly("skipping Java particle with invalid position, offset, velocity, or amount", "amount", particles.Amount)
		return nil
	}
	if particles.Amount > 100000 {
		b.logSemanticAnomaly("clamping Java particle amount", "amount", particles.Amount)
		particles.Amount = 100000
	}
	if particles.Particle.Kind == javaParticleBlockState {
		runtimeID, known := JavaBlockRuntimeID(particles.Particle.BlockStateID)
		if !known {
			b.logSemanticAnomaly("skipping Java block particle outside generated registry", "state", particles.Particle.BlockStateID)
			return nil
		}
		eventType := int32(0)
		switch particles.Particle.ID {
		case 1: // BLOCK
			eventType = packet.LevelEventParticlesCrackBlock
		case 28: // FALLING_DUST
			eventType = bedrockParticleType(30)
		default:
			b.logSemanticAnomaly("skipping Java block particle without a lossless Bedrock mapping", "particle", particles.Particle.ID)
			return nil
		}
		return b.writeJavaParticleInstances(bedrock, particles, func(position mgl32.Vec3) packet.Packet {
			return &packet.LevelEvent{EventType: eventType, Position: position, EventData: int32(runtimeID)}
		})
	}
	if particles.Particle.Kind == javaParticleDust || particles.Particle.Kind == javaParticleDustTransition {
		if !finiteFloat32(particles.Particle.Scale) {
			b.logSemanticAnomaly("skipping Java dust particle with non-finite scale", "particle", particles.Particle.ID)
			return nil
		}
		return b.writeJavaParticleInstances(bedrock, particles, func(position mgl32.Vec3) packet.Packet {
			return &packet.LevelEvent{EventType: bedrockParticleType(30), Position: position, EventData: particles.Particle.Color}
		})
	}
	if particles.Particle.Kind == javaParticleItem {
		if !particles.Particle.Item.Present || !particles.Particle.Item.Known {
			b.logSemanticAnomaly("skipping Java item particle with unknown item", "item", particles.Particle.Item.ItemID)
			return nil
		}
		runtimeID := particles.Particle.Item.Item.Stack.ItemType.NetworkID
		if runtimeID < 0 {
			b.logSemanticAnomaly("skipping Java item particle with invalid Bedrock item runtime", "runtime_id", runtimeID)
			return nil
		}
		eventData := int32(uint32(runtimeID)<<16 | (particles.Particle.Item.Item.Stack.MetadataValue & 0xffff))
		return b.writeJavaParticleInstances(bedrock, particles, func(position mgl32.Vec3) packet.Packet {
			return &packet.LevelEvent{EventType: bedrockParticleType(13), Position: position, EventData: eventData}
		})
	}
	if particles.Particle.Kind == javaParticleVibration {
		target, known := b.javaVibrationTarget(particles.Particle)
		if !known {
			return nil
		}
		if !finiteVec3(target) {
			b.logSemanticAnomaly("skipping Java vibration particle with non-finite target")
			return nil
		}
		return b.writeJavaParticleInstances(bedrock, particles, func(position mgl32.Vec3) packet.Packet {
			data, err := marshalJavaVibrationEvent(position, target, particles.Particle.VibrationArrivalTicks)
			if err != nil {
				b.logSemanticAnomaly("skipping Java vibration particle with unencodable Bedrock payload", "error", err)
				return nil
			}
			return &packet.LevelEventGeneric{EventID: packet.LevelEventParticlesVibrationSignal, SerialisedEventData: data}
		})
	}
	if particles.Particle.Kind == javaParticleTrail {
		if !finiteVec3(particles.Particle.TrailTarget) {
			b.logSemanticAnomaly("skipping Java trail particle with non-finite target")
			return nil
		}
		dimension := byte(0)
		if b.gameData.Dimension >= 0 && b.gameData.Dimension <= math.MaxUint8 {
			dimension = byte(b.gameData.Dimension)
		}
		return b.writeJavaParticleInstances(bedrock, particles, func(position mgl32.Vec3) packet.Packet {
			variables, err := marshalJavaTrailVariables(position, particles.Particle.TrailTarget, particles.Particle.TrailColor, particles.Particle.TrailDuration)
			if err != nil {
				b.logSemanticAnomaly("skipping Java trail particle with unencodable Molang payload", "error", err)
				return nil
			}
			return &packet.SpawnParticleEffect{
				Dimension:       dimension,
				EntityUniqueID:  -1,
				Position:        position,
				ParticleName:    "minecraft:creaking_heart_trail",
				MoLangVariables: gtprotocol.Option[[]byte](variables),
			}
		})
	}
	if particles.Particle.ID == 29 || particles.Particle.ID == 96 || particles.Particle.ID == 97 || particles.Particle.ID == 98 || particles.Particle.ID == 100 {
		dimension := byte(0)
		if b.gameData.Dimension >= 0 && b.gameData.Dimension <= math.MaxUint8 {
			dimension = byte(b.gameData.Dimension)
		}
		return b.writeJavaParticleInstances(bedrock, particles, func(position mgl32.Vec3) packet.Packet {
			var variables []byte
			var err error
			name := "minecraft:wax_particle"
			switch particles.Particle.ID {
			case 29: // FIREWORK
				name = "minecraft:sparkler_emitter"
				variables, err = marshalJavaColorVariables(1, 1, 1)
			case 96: // GLOW
				name = "minecraft:glow_particle"
				if rand.Intn(2) == 0 {
					variables, err = marshalJavaColorVariables(0.6, 1, 0.8)
				} else {
					variables, err = marshalJavaColorVariables(0.08, 0.4, 0.4)
				}
			case 97: // WAX_ON
				variables, err = marshalJavaColorVariables(0.91, 0.55, 0.08)
			case 98: // WAX_OFF
				variables, err = marshalJavaColorVariables(1, 0.9, 1)
			case 100: // SCRAPE
				if rand.Intn(2) == 0 {
					variables, err = marshalJavaColorVariables(0.29, 0.58, 0.51)
				} else {
					variables, err = marshalJavaColorVariables(0.43, 0.77, 0.62)
				}
			}
			if err != nil {
				b.logSemanticAnomaly("skipping Java named particle with unencodable Molang payload", "particle", particles.Particle.ID, "error", err)
				return nil
			}
			return &packet.SpawnParticleEffect{
				Dimension:       dimension,
				EntityUniqueID:  -1,
				Position:        position,
				ParticleName:    name,
				MoLangVariables: gtprotocol.Option[[]byte](variables),
			}
		})
	}
	mapping, ok := javaParticleMappings[particles.Particle.ID]
	if !ok {
		b.logSemanticAnomaly("skipping Java particle without a Bedrock mapping", "particle", particles.Particle.ID)
		return nil
	}
	if mapping.eventType == 0 && mapping.identifier == "" {
		b.logSemanticAnomaly("skipping Java particle mapping without a Bedrock packet", "particle", particles.Particle.ID)
		return nil
	}
	dimension := byte(0)
	if b.gameData.Dimension >= 0 && b.gameData.Dimension <= math.MaxUint8 {
		dimension = byte(b.gameData.Dimension)
	}
	return b.writeJavaParticleInstances(bedrock, particles, func(position mgl32.Vec3) packet.Packet {
		if mapping.eventType != 0 {
			return &packet.LevelEvent{EventType: mapping.eventType, Position: position, EventData: javaParticleEventData(particles.Particle)}
		}
		if mapping.identifier != "" {
			return &packet.SpawnParticleEffect{
				Dimension:      dimension,
				EntityUniqueID: -1,
				Position:       position,
				ParticleName:   mapping.identifier,
			}
		}
		return nil
	})
}

func (b *Basic) writeJavaParticleInstances(bedrock *minecraft.Conn, particles JavaLevelParticles, create func(mgl32.Vec3) packet.Packet) error {
	amount := particles.Amount
	if amount <= 0 {
		created := create(particles.Position)
		if created == nil {
			return nil
		}
		return bedrock.WritePacketImmediate(created)
	}
	if amount > 100 {
		amount = 100
	}
	packets := make([]packet.Packet, 0, amount)
	for i := int32(0); i < amount; i++ {
		position := particles.Position
		position[0] += float32(rand.NormFloat64()) * particles.Offset[0]
		position[1] += float32(rand.NormFloat64()) * particles.Offset[1]
		position[2] += float32(rand.NormFloat64()) * particles.Offset[2]
		if created := create(position); created != nil {
			packets = append(packets, created)
		}
	}
	if len(packets) == 0 {
		return nil
	}
	return bedrock.WritePacketImmediate(packets...)
}

func (b *Basic) javaVibrationTarget(particle JavaParticle) (mgl32.Vec3, bool) {
	switch particle.VibrationPositionType {
	case 0:
		return mgl32.Vec3{
			float32(particle.VibrationBlock[0]) + 0.5,
			float32(particle.VibrationBlock[1]) + 0.5,
			float32(particle.VibrationBlock[2]) + 0.5,
		}, true
	case 1:
		b.mu.Lock()
		entity, known := b.javaEntityStateLocked(particle.VibrationEntityID)
		b.mu.Unlock()
		if !known {
			b.logSemanticAnomaly("skipping Java vibration particle for unknown entity", "entity", particle.VibrationEntityID)
			return mgl32.Vec3{}, false
		}
		return entity.position.Add(mgl32.Vec3{0, particle.VibrationYOffset, 0}), true
	default:
		b.logSemanticAnomaly("skipping Java vibration particle with unknown position source", "source", particle.VibrationPositionType)
		return mgl32.Vec3{}, false
	}
}

func marshalJavaVibrationEvent(origin, target mgl32.Vec3, arrivalTicks int32) ([]byte, error) {
	return marshalBedrockTagValue(map[string]any{
		"origin":     bedrockVec3Tag(origin),
		"target":     bedrockVec3Tag(target),
		"speed":      float32(20),
		"timeToLive": float32(arrivalTicks) / 20,
	})
}

func marshalBedrockTagValue(value map[string]any) ([]byte, error) {
	data, err := nbt.Marshal(value)
	if err != nil {
		return nil, err
	}
	// nbt.Marshal emits an empty root name for NetworkLittleEndian. Bedrock's
	// generic level-event value is a nameless root tag, matching Cloudburst's
	// writeTagValue/readTagValue pair.
	if len(data) < 2 || data[0] != 10 || data[1] != 0 {
		return nil, fmt.Errorf("translate: unexpected Bedrock NBT root encoding")
	}
	return append([]byte{data[0]}, data[2:]...), nil
}

func bedrockVec3Tag(position mgl32.Vec3) map[string]any {
	return map[string]any{
		"type": "vec3",
		"x":    position[0],
		"y":    position[1],
		"z":    position[2],
	}
}

func marshalJavaColorVariables(red, green, blue float32) ([]byte, error) {
	return json.Marshal([]javaParticleMolangVariable{{
		Name: "variable.color",
		Value: javaParticleMolangValue{Type: "member_array", Value: []javaParticleMolangMember{
			{Name: ".r", Value: javaParticleMolangValue{Type: "float", Value: red}},
			{Name: ".g", Value: javaParticleMolangValue{Type: "float", Value: green}},
			{Name: ".b", Value: javaParticleMolangValue{Type: "float", Value: blue}},
		}},
	}})
}

type javaParticleMolangValue struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type javaParticleMolangVariable struct {
	Name  string                  `json:"name"`
	Value javaParticleMolangValue `json:"value"`
}

type javaParticleMolangMember struct {
	Name  string                  `json:"name"`
	Value javaParticleMolangValue `json:"value"`
}

func marshalJavaTrailVariables(origin, target mgl32.Vec3, color, duration int32) ([]byte, error) {
	direction := target.Sub(origin)
	distance := direction.Len()
	if distance > 0 {
		direction = direction.Mul(1 / distance)
	}
	lifetime := float32(maxInt32(duration, 1)) / 20
	red := float32(uint8(uint32(color)>>16)) / 255
	green := float32(uint8(uint32(color)>>8)) / 255
	blue := float32(uint8(color)) / 255
	variables := []javaParticleMolangVariable{
		{Name: "variable.direction", Value: javaParticleMolangValue{Type: "member_array", Value: []javaParticleMolangMember{
			{Name: ".x", Value: javaParticleMolangValue{Type: "float", Value: direction[0]}},
			{Name: ".y", Value: javaParticleMolangValue{Type: "float", Value: direction[1]}},
			{Name: ".z", Value: javaParticleMolangValue{Type: "float", Value: direction[2]}},
		}}},
		{Name: "variable.color", Value: javaParticleMolangValue{Type: "member_array", Value: []javaParticleMolangMember{
			{Name: ".r", Value: javaParticleMolangValue{Type: "float", Value: red}},
			{Name: ".g", Value: javaParticleMolangValue{Type: "float", Value: green}},
			{Name: ".b", Value: javaParticleMolangValue{Type: "float", Value: blue}},
		}}},
		{Name: "variable.max_lifetime", Value: javaParticleMolangValue{Type: "float", Value: lifetime}},
		{Name: "variable.particle_initial_speed", Value: javaParticleMolangValue{Type: "float", Value: distance / lifetime}},
	}
	return json.Marshal(variables)
}

func maxInt32(value, minimum int32) int32 {
	if value < minimum {
		return minimum
	}
	return value
}

func javaParticleEventData(particle JavaParticle) int32 {
	if particle.Kind == javaParticleShriek {
		return particle.ShriekTicks
	}
	if particle.Kind == javaParticleEntityEffect {
		return particle.Color
	}
	return 0
}
