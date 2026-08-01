package protocol

import "fmt"

// Profile describes packet IDs and wire choices for one Java protocol
// revision. Packet IDs are deliberately data rather than scattered constants:
// Java protocol revisions change state layouts independently.
type Profile struct {
	Name            string
	ProtocolVersion int32

	// LoginStartUUIDOptional selects the older boolean-plus-optional-UUID
	// encoding. Java 1.21.4 uses a direct 16-byte UUID field.
	LoginStartUUIDOptional bool

	LoginStartPacketID          int32
	LoginDisconnectPacketID     int32
	LoginEncryptionPacketID     int32
	LoginSuccessPacketID        int32
	LoginSetCompressionPacketID int32
	LoginPluginRequestPacketID  int32
	LoginPluginResponsePacketID int32
	LoginCookieRequestPacketID  int32
	LoginAcknowledgedPacketID   int32
	LoginCookieResponsePacketID int32

	ConfigClientInformationPacketID int32
	ConfigCookieRequestPacketID     int32
	ConfigCustomPayloadPacketID     int32
	ConfigDisconnectPacketID        int32
	ConfigFinishPacketID            int32
	ConfigKeepAlivePacketID         int32
	ConfigPingPacketID              int32
	ConfigSelectKnownPacksPacketID  int32

	ConfigServerboundCookieResponsePacketID int32
	ConfigServerboundCustomPayloadPacketID  int32
	ConfigServerboundFinishPacketID         int32
	ConfigServerboundKeepAlivePacketID      int32
	ConfigServerboundPongPacketID           int32
	ConfigServerboundSelectKnownPacksID     int32

	PlayClientboundDisconnectPacketID   int32
	PlayClientboundKeepAlivePacketID    int32
	PlayClientboundLoginPacketID        int32
	PlayClientboundMapChunkPacketID     int32
	PlayClientboundPositionPacketID     int32
	PlayClientboundUpdateHealthID       int32
	PlayClientboundBlockEntityDataID    int32
	PlayClientboundBlockChangeID        int32
	PlayClientboundMultiBlockChangeID   int32
	PlayClientboundSpawnEntityID        int32
	PlayClientboundEntityTeleportID     int32
	PlayClientboundEntityDestroyID      int32
	PlayClientboundEntityRelMoveID      int32
	PlayClientboundEntityMoveLookID     int32
	PlayClientboundEntityLookID         int32
	PlayClientboundEntityHeadRotationID int32
	PlayClientboundUpdateTimeID         int32
	PlayClientboundUnloadChunkID        int32
	PlayClientboundWindowItemsID        int32
	PlayClientboundSetSlotID            int32
	PlayClientboundProfilelessChatID    int32
	PlayClientboundPlayerChatID         int32
	PlayClientboundSystemChatID         int32
	PlayClientboundPlayerRemoveID       int32
	PlayClientboundPlayerInfoID         int32
	PlayClientboundEntityMetadataID     int32
	PlayClientboundEntityVelocityID     int32
	PlayClientboundEntityEquipmentID    int32
	PlayClientboundExperienceID         int32
	PlayClientboundHeldItemSlotID       int32
	PlayClientboundSetPlayerInventoryID int32
	PlayServerboundKeepAlivePacketID    int32
	PlayServerboundTeleportConfirmID    int32
	PlayServerboundPositionLookID       int32
	PlayServerboundBlockDigID           int32
	PlayServerboundBlockPlaceID         int32
	PlayServerboundUseItemID            int32
	PlayServerboundUseEntityID          int32
	PlayServerboundHeldItemSlotID       int32
	PlayServerboundEntityActionID       int32
	PlayServerboundArmAnimationID       int32
	PlayServerboundChatMessageID        int32
}

// Java1214 is the 1.21.4 protocol profile (protocol 769), sourced from the
// versioned minecraft-data protocol definition. It is the first profile used
// by the bridge; later profiles must be added explicitly and tested separately.
var Java1214 = Profile{
	Name:            "java-1.21.4",
	ProtocolVersion: 769,

	LoginStartUUIDOptional: false,

	LoginStartPacketID:          0,
	LoginDisconnectPacketID:     0,
	LoginEncryptionPacketID:     1,
	LoginSuccessPacketID:        2,
	LoginSetCompressionPacketID: 3,
	LoginPluginRequestPacketID:  4,
	LoginPluginResponsePacketID: 2,
	LoginCookieRequestPacketID:  5,
	LoginAcknowledgedPacketID:   3,
	LoginCookieResponsePacketID: 4,

	ConfigClientInformationPacketID: 0,
	ConfigCookieRequestPacketID:     0,
	ConfigCustomPayloadPacketID:     1,
	ConfigDisconnectPacketID:        2,
	ConfigFinishPacketID:            3,
	ConfigKeepAlivePacketID:         4,
	ConfigPingPacketID:              5,
	ConfigSelectKnownPacksPacketID:  14,

	ConfigServerboundCookieResponsePacketID: 1,
	ConfigServerboundCustomPayloadPacketID:  2,
	ConfigServerboundFinishPacketID:         3,
	ConfigServerboundKeepAlivePacketID:      4,
	ConfigServerboundPongPacketID:           5,
	ConfigServerboundSelectKnownPacksID:     7,

	PlayClientboundDisconnectPacketID:   0x1d,
	PlayClientboundKeepAlivePacketID:    0x27,
	PlayClientboundLoginPacketID:        0x2c,
	PlayClientboundMapChunkPacketID:     0x28,
	PlayClientboundPositionPacketID:     0x42,
	PlayClientboundUpdateHealthID:       0x62,
	PlayClientboundBlockEntityDataID:    0x07,
	PlayClientboundBlockChangeID:        0x09,
	PlayClientboundMultiBlockChangeID:   0x4e,
	PlayClientboundSpawnEntityID:        0x01,
	PlayClientboundEntityTeleportID:     0x77,
	PlayClientboundEntityDestroyID:      0x47,
	PlayClientboundEntityRelMoveID:      0x2f,
	PlayClientboundEntityMoveLookID:     0x30,
	PlayClientboundEntityLookID:         0x32,
	PlayClientboundEntityHeadRotationID: 0x4d,
	PlayClientboundUpdateTimeID:         0x6b,
	PlayClientboundUnloadChunkID:        0x22,
	PlayClientboundWindowItemsID:        0x13,
	PlayClientboundSetSlotID:            0x15,
	PlayClientboundProfilelessChatID:    0x1e,
	PlayClientboundPlayerChatID:         0x3b,
	PlayClientboundSystemChatID:         0x73,
	PlayClientboundPlayerRemoveID:       0x3f,
	PlayClientboundPlayerInfoID:         0x40,
	PlayClientboundEntityMetadataID:     0x5d,
	PlayClientboundEntityVelocityID:     0x5f,
	PlayClientboundEntityEquipmentID:    0x60,
	PlayClientboundExperienceID:         0x61,
	PlayClientboundHeldItemSlotID:       0x63,
	PlayClientboundSetPlayerInventoryID: 0x66,
	PlayServerboundKeepAlivePacketID:    0x1a,
	PlayServerboundTeleportConfirmID:    0x00,
	PlayServerboundPositionLookID:       0x1d,
	PlayServerboundBlockDigID:           0x27,
	PlayServerboundBlockPlaceID:         0x3c,
	PlayServerboundUseItemID:            0x3d,
	PlayServerboundUseEntityID:          0x18,
	PlayServerboundHeldItemSlotID:       0x33,
	PlayServerboundEntityActionID:       0x28,
	PlayServerboundArmAnimationID:       0x3a,
	PlayServerboundChatMessageID:        0x07,
}

func (p Profile) Validate() error {
	if p.Name == "" || p.ProtocolVersion <= 0 {
		return fmt.Errorf("%w: missing name or protocol version", ErrUnsupportedProtocol)
	}
	if p.LoginStartPacketID < 0 || p.LoginSuccessPacketID < 0 || p.ConfigFinishPacketID < 0 {
		return fmt.Errorf("%w: negative required packet ID", ErrUnsupportedProtocol)
	}
	return nil
}
