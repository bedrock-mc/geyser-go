package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaSetPassengers is the wire shape of Java's Set Passengers packet. The
// order is meaningful: Java treats the first passenger as the controlling
// rider and the remaining passengers as ordinary riders.
type JavaSetPassengers struct {
	EntityID     int32
	PassengerIDs []int32
}

func DecodeJavaSetPassengers(data []byte) (JavaSetPassengers, error) {
	r := javaprotocol.NewReader(data)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaSetPassengers{}, fmt.Errorf("translate: set passengers vehicle: %w", err)
	}
	count, err := boundedJavaCount(r, "set passengers count")
	if err != nil {
		return JavaSetPassengers{}, err
	}
	passengerIDs := make([]int32, count)
	for i := range passengerIDs {
		if passengerIDs[i], err = r.VarInt(); err != nil {
			return JavaSetPassengers{}, fmt.Errorf("translate: set passengers passenger %d: %w", i, err)
		}
	}
	if r.Remaining() != 0 {
		return JavaSetPassengers{}, fmt.Errorf("translate: set passengers has %d trailing bytes", r.Remaining())
	}
	return JavaSetPassengers{EntityID: entityID, PassengerIDs: passengerIDs}, nil
}

func (b *Basic) javaEntityUniqueIDLocked(entityID int32) (int64, bool) {
	if b.gameData.EntityUniqueID != 0 && b.gameData.EntityUniqueID == int64(entityID) {
		return b.gameData.EntityUniqueID, true
	}
	if _, ok := b.entities[entityID]; !ok {
		return 0, false
	}
	return int64(entityID), true
}

func passengerLinkType(index int) byte {
	if index == 0 {
		return gtprotocol.EntityLinkRider
	}
	return gtprotocol.EntityLinkPassenger
}

func (b *Basic) translateJavaSetPassengers(bedrock *minecraft.Conn, data []byte) error {
	update, err := DecodeJavaSetPassengers(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	vehicleID, vehicleKnown := b.javaEntityUniqueIDLocked(update.EntityID)
	oldPassengers := append([]int32(nil), b.passengers[update.EntityID]...)
	newPassengers := append([]int32(nil), update.PassengerIDs...)
	b.passengers[update.EntityID] = newPassengers
	playerID := int32(uint32(b.gameData.EntityUniqueID))
	playerPassenger := b.gameData.EntityUniqueID != 0 && containsInt32(newPassengers, playerID)
	if playerPassenger {
		b.vehicleID = update.EntityID
		b.hasVehicle = true
	} else if b.hasVehicle && b.vehicleID == update.EntityID {
		b.vehicleID = 0
		b.hasVehicle = false
	}
	b.mu.Unlock()
	if !vehicleKnown {
		b.logSemanticAnomaly("skipping Java passenger links for an unknown vehicle", "entity", update.EntityID)
		return nil
	}

	contains := func(ids []int32, wanted int32) bool {
		for _, id := range ids {
			if id == wanted {
				return true
			}
		}
		return false
	}
	for _, passengerID := range oldPassengers {
		if contains(newPassengers, passengerID) {
			continue
		}
		if err := bedrock.WritePacket(&packet.SetActorLink{EntityLink: gtprotocol.EntityLink{
			RiddenEntityUniqueID: vehicleID,
			RiderEntityUniqueID:  int64(passengerID),
			Type:                 gtprotocol.EntityLinkRemove,
		}}); err != nil {
			return err
		}
	}
	for index, passengerID := range newPassengers {
		b.mu.Lock()
		passengerKnown := false
		if passengerID == int32(b.gameData.EntityUniqueID) && b.gameData.EntityUniqueID != 0 {
			passengerKnown = true
		} else {
			_, passengerKnown = b.entities[passengerID]
		}
		b.mu.Unlock()
		if !passengerKnown {
			// Java may send the mount before the passenger enters the client's
			// tracking range. The later entity spawn/set-passengers sequence will
			// complete the link; a missing actor is not a wire failure.
			b.logSemanticAnomaly("skipping Java passenger link for an unknown passenger", "entity", passengerID)
			continue
		}
		if err := bedrock.WritePacket(&packet.SetActorLink{EntityLink: gtprotocol.EntityLink{
			RiddenEntityUniqueID: vehicleID,
			RiderEntityUniqueID:  int64(passengerID),
			Type:                 passengerLinkType(index),
		}}); err != nil {
			return err
		}
	}
	return nil
}

func containsInt32(values []int32, wanted int32) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// javaCurrentVehicleLocked returns the vehicle carrying the local Java
// player. Clientbound MoveVehicle has no entity ID; Java applies it to this
// vehicle implicitly. The tracked ID is preferred, while the bounded scan
// repairs state if a server sends SetPassengers before the local entity is
// fully known.
func (b *Basic) javaCurrentVehicleLocked() (int32, *javaEntityState, bool) {
	if b.gameData.EntityUniqueID == 0 {
		return 0, nil, false
	}
	playerID := int32(uint32(b.gameData.EntityUniqueID))
	if b.hasVehicle && containsInt32(b.passengers[b.vehicleID], playerID) {
		return b.vehicleID, b.entities[b.vehicleID], true
	}
	for vehicleID, passengerIDs := range b.passengers {
		if !containsInt32(passengerIDs, playerID) {
			continue
		}
		b.vehicleID = vehicleID
		b.hasVehicle = true
		return vehicleID, b.entities[vehicleID], true
	}
	b.vehicleID = 0
	b.hasVehicle = false
	return 0, nil, false
}
