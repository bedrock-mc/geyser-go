package translate

import (
	"fmt"
	"math"
	"time"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type javaBlockBreakState struct {
	stage int32
	at    time.Time
}

// JavaBlockDestruction is ClientboundBlockDestructionPacket. The breaker ID
// is retained for diagnostics; Bedrock's block-cracking level event uses a
// duration estimate rather than the Java breaker identity.
type JavaBlockDestruction struct {
	BreakerEntityID int32
	Position        gtprotocol.BlockPos
	Stage           int32
}

func DecodeJavaBlockDestruction(data []byte) (JavaBlockDestruction, error) {
	r := javaprotocol.NewReader(data)
	breaker, err := r.VarInt()
	if err != nil {
		return JavaBlockDestruction{}, fmt.Errorf("translate: block destruction breaker: %w", err)
	}
	packed, err := r.Int64()
	if err != nil {
		return JavaBlockDestruction{}, fmt.Errorf("translate: block destruction position: %w", err)
	}
	stage, err := r.VarInt()
	if err != nil {
		return JavaBlockDestruction{}, fmt.Errorf("translate: block destruction stage: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaBlockDestruction{}, fmt.Errorf("translate: block destruction has %d trailing bytes", r.Remaining())
	}
	return JavaBlockDestruction{BreakerEntityID: breaker, Position: decodeJavaPosition(packed), Stage: stage}, nil
}

func (b *Basic) translateJavaBlockDestruction(bedrock *minecraft.Conn, data []byte) error {
	destruction, err := DecodeJavaBlockDestruction(data)
	if err != nil {
		return err
	}
	if destruction.Stage < 0 || destruction.Stage > 10 {
		b.logSemanticAnomaly("skipping Java block destruction with unknown stage", "stage", destruction.Stage)
		return nil
	}

	now := time.Now()
	b.mu.Lock()
	previous, hadPrevious := b.blockBreaks[destruction.Position]
	if destruction.Stage == 10 {
		delete(b.blockBreaks, destruction.Position)
	} else {
		b.blockBreaks[destruction.Position] = javaBlockBreakState{stage: destruction.Stage, at: now}
	}
	b.mu.Unlock()

	position := mgl32.Vec3{
		float32(destruction.Position[0]),
		float32(destruction.Position[1]),
		float32(destruction.Position[2]),
	}
	if destruction.Stage == 10 {
		return bedrock.WritePacket(&packet.LevelEvent{
			EventType: packet.LevelEventStopBlockCracking,
			Position:  position,
		})
	}

	if !hadPrevious || destruction.Stage == 0 {
		// This is the same conservative initial estimate Geyser uses. A later
		// Java stage lets us refine it from the observed stage cadence.
		return bedrock.WritePacket(&packet.LevelEvent{
			EventType: packet.LevelEventStartBlockCracking,
			Position:  position,
			EventData: 65535 / 6000,
		})
	}

	ticksSince := int(time.Since(previous.at) / (50 * time.Millisecond))
	if ticksSince < 1 {
		ticksSince = 1
	}
	stagesSince := int(destruction.Stage - previous.stage)
	if stagesSince < 1 {
		stagesSince = 1
	}
	ticksPerStage := ticksSince / stagesSince
	if ticksPerStage < 1 {
		ticksPerStage = 1
	}
	remainingStages := int64(10 - destruction.Stage)
	if remainingStages < 1 {
		remainingStages = 1
	}
	dataValue := int64(65535) / remainingStages * int64(ticksPerStage)
	if dataValue > math.MaxInt32 {
		dataValue = math.MaxInt32
	}
	return bedrock.WritePacket(&packet.LevelEvent{
		EventType: packet.LevelEventUpdateBlockCracking,
		Position:  position,
		EventData: int32(dataValue),
	})
}

type javaWorldEventKind byte

const (
	javaWorldEventLevel javaWorldEventKind = iota + 1
	javaWorldEventSound
)

type javaWorldEventMapping struct {
	kind      javaWorldEventKind
	eventType int32
	sound     string
	volume    float32
	pitch     float32
	data      func(int32) int32
}

func constantWorldEventData(value int32) func(int32) int32 {
	return func(int32) int32 { return value }
}

// javaWorldEvents is derived from the 1.21.4 Geyser effects mapping and the
// Bedrock LevelEvent constants in the pinned lunar Gophertunnel fork. It is
// intentionally explicit so a newer Java effect cannot silently acquire a
// wrong meaning in an older profile.
var javaWorldEvents = map[int32]javaWorldEventMapping{
	1000: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundClick, data: constantWorldEventData(1000)},
	1001: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundClickFail, data: constantWorldEventData(1200)},
	1002: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundClick, data: constantWorldEventData(1000)},
	1004: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundLaunch},
	1009: {kind: javaWorldEventSound, sound: "random.fizz", volume: 1, pitch: 1},
	1015: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundGhastWarning},
	1016: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundGhastFireball},
	1017: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundGhastFireball},
	1018: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundBlazeFireball},
	1019: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundZombieWoodenDoor},
	1021: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundZombieDoorCrash},
	1022: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundZombieDoorCrash},
	1023: {kind: javaWorldEventSound, sound: "mob.wither.spawn", volume: 1, pitch: 1},
	1024: {kind: javaWorldEventSound, sound: "mob.wither.shoot", volume: 1, pitch: 1},
	1025: {kind: javaWorldEventSound, sound: "mob.bat.takeoff", volume: 1, pitch: 1},
	1027: {kind: javaWorldEventSound, sound: "mob.zombie.unfect", volume: 1, pitch: 1},
	1028: {kind: javaWorldEventSound, sound: "mob.enderdragon.death", volume: 1, pitch: 1},
	1029: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundAnvilBroken},
	1030: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundAnvilUsed},
	1031: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundAnvilLand},
	1032: {kind: javaWorldEventSound, sound: "portal.travel", volume: 0.25, pitch: 1},
	1033: {kind: javaWorldEventSound, sound: "block.chorus_flower.grow", volume: 1, pitch: 1},
	1034: {kind: javaWorldEventSound, sound: "block.chorus_flower.death", volume: 1, pitch: 1},
	1035: {kind: javaWorldEventSound, sound: "block.brewing_stand.brew", volume: 1, pitch: 1},
	1038: {kind: javaWorldEventSound, sound: "block.end_portal.spawn", volume: 1, pitch: 1},
	1042: {kind: javaWorldEventSound, sound: "block.grindstone.use", volume: 1, pitch: 1},
	1043: {kind: javaWorldEventSound, sound: "item.book.page_turn", volume: 1, pitch: 1},
	1044: {kind: javaWorldEventSound, sound: "block.smithing_table.use", volume: 1, pitch: 1},
	1045: {kind: javaWorldEventLevel, eventType: packet.LevelEventSoundPointedDripstoneLand},
	1046: {kind: javaWorldEventSound, sound: "block.pointed_dripstone.drip_lava", volume: 1, pitch: 1},
	1047: {kind: javaWorldEventSound, sound: "block.pointed_dripstone.drip_water", volume: 1, pitch: 1},
	1048: {kind: javaWorldEventSound, sound: "entity.skeleton.converted_to_stray", volume: 1, pitch: 1},
	1049: {kind: javaWorldEventSound, sound: "block.crafter.craft", volume: 1, pitch: 1},
	1050: {kind: javaWorldEventSound, sound: "block.crafter.fail", volume: 1, pitch: 1},
	1051: {kind: javaWorldEventSound, sound: "entity.breeze.wind_charge", volume: 1, pitch: 1},
	1500: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticleCropGrowth},
	1501: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesEvaporate},
	1502: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesEvaporate},
	1503: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesEvaporate},
	1504: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesDripstoneDrip},
	1505: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticleCropGrowth, data: func(value int32) int32 { return value }},
	2000: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesShoot, data: javaSmokeEventData},
	2001: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesDestroyBlock, data: javaBlockParticleData},
	2002: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesPotionSplash, data: func(value int32) int32 { return value }},
	2003: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesEyeOfEnderDeath},
	2004: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesMobBlockSpawn},
	2006: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesEyeOfEnderDeath},
	2007: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesPotionSplash, data: func(value int32) int32 { return value }},
	2009: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesEvaporateWater},
	2010: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesShootWhiteSmoke, data: javaSmokeEventData},
	2011: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticleTurtleEgg},
	2012: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesBreakingEgg},
	2013: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticleSmashAttackGroundDust},
	3000: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesExplosion},
	3002: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesElectricSpark, data: func(value int32) int32 { return value }},
	3003: {kind: javaWorldEventLevel, eventType: packet.LevelEventWaxOn},
	3004: {kind: javaWorldEventLevel, eventType: packet.LevelEventWaxOff},
	3005: {kind: javaWorldEventLevel, eventType: packet.LevelEventScrape},
	3006: {kind: javaWorldEventLevel, eventType: packet.LevelEventSculkCharge, data: func(value int32) int32 { return value }},
	3007: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticleSculkShriek},
	3009: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesBreakingEgg},
	3011: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesTrialSpawnerSpawning, data: func(value int32) int32 { return value }},
	3012: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesTrialSpawnerSpawning, data: func(value int32) int32 { return value }},
	3013: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesTrialSpawnerDetection, data: func(value int32) int32 { return value }},
	3014: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesTrialSpawnerEjecting},
	3015: {kind: javaWorldEventLevel, eventType: packet.LevelEventAnimationVaultActivate},
	3016: {kind: javaWorldEventLevel, eventType: packet.LevelEventAnimationVaultDeactivate},
	3017: {kind: javaWorldEventLevel, eventType: packet.LevelEventAnimationVaultEjectItem},
	3018: {kind: javaWorldEventLevel, eventType: packet.LevelEventAnimationSpawnCobweb},
	3019: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesTrialSpawnerDetectionCharged},
	3020: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesTrialSpawnerBecomeCharged},
	3021: {kind: javaWorldEventLevel, eventType: packet.LevelEventParticlesTrialSpawnerEjecting},
}

func javaSmokeEventData(value int32) int32 {
	direction := value % 6
	if direction < 0 {
		direction = -direction
	}
	switch direction {
	case 0, 1:
		return 4
	case 2:
		return 1
	case 3:
		return 7
	case 4:
		return 3
	default:
		return 5
	}
}

func javaBlockParticleData(value int32) int32 {
	runtimeID, known := JavaBlockRuntimeID(value)
	if !known {
		return 0
	}
	if runtimeID > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(runtimeID)
}

func (b *Basic) translateJavaWorldEvent(bedrock *minecraft.Conn, data []byte) error {
	event, err := DecodeJavaWorldEvent(data)
	if err != nil {
		return err
	}
	mapping, ok := javaWorldEvents[event.EffectID]
	if !ok {
		b.logSemanticAnomaly("skipping Java world event without a versioned effect mapping", "effect", event.EffectID)
		return nil
	}
	position := mgl32.Vec3{
		float32(event.Position[0]) + 0.5,
		float32(event.Position[1]) + 0.5,
		float32(event.Position[2]) + 0.5,
	}
	if mapping.kind == javaWorldEventSound {
		volume, pitch := mapping.volume, mapping.pitch
		if volume == 0 {
			volume = 1
		}
		if pitch == 0 {
			pitch = 1
		}
		return bedrock.WritePacket(&packet.PlaySound{SoundName: mapping.sound, Position: position, Volume: volume, Pitch: pitch})
	}
	eventData := event.Data
	if mapping.data != nil {
		eventData = mapping.data(event.Data)
	}
	return bedrock.WritePacket(&packet.LevelEvent{EventType: mapping.eventType, Position: position, EventData: eventData})
}
