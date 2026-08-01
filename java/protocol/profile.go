package protocol

import "fmt"

// Profile describes packet IDs and wire choices for one Java protocol
// revision. Packet IDs are deliberately data rather than scattered constants:
// Java protocol revisions change state layouts independently.
type Profile struct {
	Name            string
	ProtocolVersion int32

	LoginStartHasUUID bool

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
}

// Java1214 is the 1.21.4 protocol profile (protocol 769), sourced from the
// versioned minecraft-data protocol definition. It is the first profile used
// by the bridge; later profiles must be added explicitly and tested separately.
var Java1214 = Profile{
	Name:            "java-1.21.4",
	ProtocolVersion: 769,

	LoginStartHasUUID: true,

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
