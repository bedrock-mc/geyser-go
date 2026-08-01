package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaEntityEvent is the wire shape of ClientboundEntityEventPacket. Java
// uses a signed entity ID followed by an ordinal byte from EntityEvent.
type JavaEntityEvent struct {
	EntityID int32
	EventID  byte
}

func DecodeJavaEntityEvent(data []byte) (JavaEntityEvent, error) {
	r := javaprotocol.NewReader(data)
	entityID, err := r.Int32()
	if err != nil {
		return JavaEntityEvent{}, fmt.Errorf("translate: entity event entity: %w", err)
	}
	eventID, err := r.Byte()
	if err != nil {
		return JavaEntityEvent{}, fmt.Errorf("translate: entity event type: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaEntityEvent{}, fmt.Errorf("translate: entity event has %d trailing bytes", r.Remaining())
	}
	return JavaEntityEvent{EntityID: entityID, EventID: eventID}, nil
}

// JavaTakeItem is the wire shape of ClientboundTakeItemEntityPacket.
type JavaTakeItem struct {
	CollectedEntityID int32
	CollectorEntityID int32
	ItemCount         int32
}

func DecodeJavaTakeItem(data []byte) (JavaTakeItem, error) {
	r := javaprotocol.NewReader(data)
	collected, err := r.VarInt()
	if err != nil {
		return JavaTakeItem{}, fmt.Errorf("translate: take-item collected entity: %w", err)
	}
	collector, err := r.VarInt()
	if err != nil {
		return JavaTakeItem{}, fmt.Errorf("translate: take-item collector entity: %w", err)
	}
	count, err := r.VarInt()
	if err != nil {
		return JavaTakeItem{}, fmt.Errorf("translate: take-item count: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaTakeItem{}, fmt.Errorf("translate: take-item has %d trailing bytes", r.Remaining())
	}
	return JavaTakeItem{CollectedEntityID: collected, CollectorEntityID: collector, ItemCount: count}, nil
}

// These ordinals are the stable 1.21.4 MCProtocolLib EntityEvent order. Only
// events with a direct Bedrock actor-event equivalent are included here;
// unsupported entity-specific side effects are logged and skipped.
const (
	javaEntityEventLivingHurt             = 2
	javaEntityEventLivingDeath            = 3
	javaEntityEventAttack                 = 4
	javaEntityEventStopAttack             = 5
	javaEntityEventTamingFailed           = 6
	javaEntityEventTamingSucceeded        = 7
	javaEntityEventWolfShake              = 8
	javaEntityEventFinishUsingItem        = 9
	javaEntityEventSheepGrazeOrTNTExplode = 10
	javaEntityEventGolemHoldPoppy         = 11
	javaEntityEventVillagerMate           = 12
	javaEntityEventVillagerAngry          = 13
	javaEntityEventVillagerHappy          = 14
	javaEntityEventWitchEmitParticles     = 15
	javaEntityEventFireworkExplode        = 17
	javaEntityEventAnimalEmitHearts       = 18
	javaEntityEventGuardianMakeSound      = 21
	javaEntityEventGolemEmptyHand         = 34
	javaEntityEventDolphinHappy           = 38
	javaEntityEventOcelotTamingFailed     = 40
	javaEntityEventOcelotTamingSucceeded  = 41
	javaEntityEventWolfShakeStop          = 56
	javaEntityEventGoatLoweringHead       = 58
	javaEntityEventGoatStopLoweringHead   = 59
	javaEntityEventWardenReceiveSignal    = 61
	javaEntityEventShake                  = 66
)

var javaEntityEventActorTypes = map[byte]byte{
	javaEntityEventLivingHurt:            packet.ActorEventHurt,
	javaEntityEventLivingDeath:           packet.ActorEventDeath,
	javaEntityEventAttack:                packet.ActorEventStartAttacking,
	javaEntityEventStopAttack:            packet.ActorEventStopAttacking,
	javaEntityEventTamingFailed:          packet.ActorEventTamingFailed,
	javaEntityEventTamingSucceeded:       packet.ActorEventTamingSucceeded,
	javaEntityEventWolfShake:             packet.ActorEventShakeWetness,
	javaEntityEventFinishUsingItem:       packet.ActorEventUseItem,
	javaEntityEventGolemHoldPoppy:        packet.ActorEventStartOfferFlower,
	javaEntityEventVillagerMate:          packet.ActorEventLoveHearts,
	javaEntityEventVillagerAngry:         packet.ActorEventVillagerAngry,
	javaEntityEventVillagerHappy:         packet.ActorEventVillagerHappy,
	javaEntityEventWitchEmitParticles:    packet.ActorEventWitchHatMagic,
	javaEntityEventFireworkExplode:       packet.ActorEventFireworksExplode,
	javaEntityEventAnimalEmitHearts:      packet.ActorEventInLoveHearts,
	javaEntityEventGuardianMakeSound:     packet.ActorEventGuardianAttackSound,
	javaEntityEventGolemEmptyHand:        packet.ActorEventStopOfferFlower,
	javaEntityEventDolphinHappy:          packet.ActorEventLoveHearts,
	javaEntityEventOcelotTamingFailed:    packet.ActorEventTamingFailed,
	javaEntityEventOcelotTamingSucceeded: packet.ActorEventTamingSucceeded,
	javaEntityEventWolfShakeStop:         packet.ActorEventWetnessStop,
	javaEntityEventGoatLoweringHead:      packet.ActorEventStartAttacking,
	javaEntityEventGoatStopLoweringHead:  packet.ActorEventStopAttacking,
	javaEntityEventWardenReceiveSignal:   packet.ActorEventVibrationDetected,
	javaEntityEventShake:                 packet.ActorEventShake,
}

func (b *Basic) javaEntityStateLocked(entityID int32) (javaEntityState, bool) {
	if b.gameData.EntityUniqueID != 0 && int64(entityID) == b.gameData.EntityUniqueID {
		return javaEntityState{
			runtimeID:  b.gameData.EntityRuntimeID,
			entityType: "minecraft:player",
			position:   mgl32.Vec3{float32(b.position.x), float32(b.position.y), float32(b.position.z)},
			player:     true,
		}, true
	}
	entity, ok := b.entities[entityID]
	if !ok || entity == nil {
		return javaEntityState{}, false
	}
	return *entity, true
}

func (b *Basic) translateJavaEntityEvent(bedrock *minecraft.Conn, data []byte) error {
	event, err := DecodeJavaEntityEvent(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	entity, known := b.javaEntityStateLocked(event.EntityID)
	b.mu.Unlock()
	if !known {
		b.logSemanticAnomaly("skipping Java entity event for unknown entity", "entity", event.EntityID, "event", event.EventID)
		return nil
	}
	actorEvent, ok := javaEntityEventActorTypes[event.EventID]
	if event.EventID == javaEntityEventSheepGrazeOrTNTExplode {
		if entity.entityType == "minecraft:tnt_minecart" {
			actorEvent = packet.ActorEventCartWithPrimeTNT
		} else {
			actorEvent = packet.ActorEventEatGrass
		}
		ok = true
	}
	if !ok {
		b.logSemanticAnomaly("skipping Java entity event without a Bedrock actor mapping", "entity", event.EntityID, "event", event.EventID)
		return nil
	}
	return bedrock.WritePacket(&packet.ActorEvent{EntityRuntimeID: entity.runtimeID, EventType: actorEvent})
}

func (b *Basic) translateJavaTakeItem(bedrock *minecraft.Conn, data []byte) error {
	take, err := DecodeJavaTakeItem(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	collected, collectedKnown := b.javaEntityStateLocked(take.CollectedEntityID)
	collector, collectorKnown := b.javaEntityStateLocked(take.CollectorEntityID)
	b.mu.Unlock()
	if !collectedKnown || !collectorKnown {
		b.logSemanticAnomaly("skipping Java take-item event for unknown entity", "collected", take.CollectedEntityID, "collector", take.CollectorEntityID)
		return nil
	}
	if collected.entityType == "minecraft:experience_orb" || collected.entityType == "minecraft:xp_orb" {
		return bedrock.WritePacket(&packet.LevelEvent{
			EventType: packet.LevelEventSoundExperienceOrbPickup,
			Position:  collected.position,
		})
	}
	return bedrock.WritePacket(&packet.TakeItemActor{
		ItemEntityRuntimeID:  collected.runtimeID,
		TakerEntityRuntimeID: collector.runtimeID,
	})
}
