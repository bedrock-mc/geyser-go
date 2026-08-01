package protocol

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// LoginSuccess is the server's authenticated/offline login identity. Property
// values are retained because skin and profile properties are part of Geyser's
// eventual Bedrock identity translation.
type LoginSuccess struct {
	UUID       [16]byte
	Username   string
	Properties []Property
}

type Property struct {
	Name      string
	Value     string
	Signature string
}

// DisconnectError preserves a Java disconnect reason when it is represented
// as a string by the active protocol state.
type DisconnectError struct {
	State  State
	Reason string
}

func (e *DisconnectError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("java protocol: disconnected during state %d", e.State)
	}
	return fmt.Sprintf("java protocol: disconnected during state %d: %s", e.State, e.Reason)
}

// Client owns a negotiated Java connection. Once Login returns, the
// connection is in Play and the caller owns packet translation and closure.
type Client struct {
	Conn    *Conn
	Profile Profile
	Login   LoginSuccess
}

func (c *Client) Close() error { return c.Conn.Close() }

// DialAndLogin performs the transport and login/configuration handshake. It
// intentionally stops at the Play boundary; gameplay packets belong to a
// versioned translator, not an untyped pass-through.
func DialAndLogin(ctx context.Context, address string, profile Profile, username string, playerUUID *[16]byte) (*Client, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	if username == "" {
		return nil, fmt.Errorf("java protocol: empty player username")
	}
	conn, err := Dial(ctx, address, ConnConfig{})
	if err != nil {
		return nil, err
	}
	client := &Client{Conn: conn, Profile: profile}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = conn.Close()
		}
	}()

	clearDeadline := applyContextDeadline(ctx, conn.NetConn())
	defer clearDeadline()

	host, port := handshakeAddress(address)
	handshake, err := (Handshake{
		ProtocolVersion: profile.ProtocolVersion,
		ServerAddress:   host,
		ServerPort:      port,
		NextState:       2, // login
	}).Encode()
	if err != nil {
		return nil, err
	}
	if err := conn.WritePacket(0, handshake); err != nil {
		return nil, fmt.Errorf("java protocol: write handshake: %w", err)
	}
	conn.SetState(StateLogin)

	loginStart, err := (LoginStart{Username: username, UUID: playerUUID}).Encode(profile.LoginStartUUIDOptional)
	if err != nil {
		return nil, err
	}
	if err := conn.WritePacket(profile.LoginStartPacketID, loginStart); err != nil {
		return nil, fmt.Errorf("java protocol: write login start: %w", err)
	}

	if err := client.negotiate(ctx); err != nil {
		return nil, err
	}
	closeOnError = false
	return client, nil
}

func (c *Client) negotiate(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pk, err := c.Conn.ReadPacket()
		if err != nil {
			return fmt.Errorf("java protocol: read state %d packet: %w", c.Conn.State(), err)
		}
		switch c.Conn.State() {
		case StateLogin:
			if done, err := c.loginPacket(pk); done || err != nil {
				if err != nil {
					return err
				}
				c.Conn.SetState(StateConfiguration)
				if err := c.sendClientInformation(); err != nil {
					return err
				}
			}
		case StateConfiguration:
			if done, err := c.configurationPacket(pk); done || err != nil {
				if err != nil {
					return err
				}
				c.Conn.SetState(StatePlay)
				return nil
			}
		default:
			return fmt.Errorf("%w: state %d packet %#x", ErrUnexpectedPacket, c.Conn.State(), pk.ID)
		}
	}
}

func (c *Client) loginPacket(pk Packet) (bool, error) {
	p := c.Profile
	switch pk.ID {
	case p.LoginDisconnectPacketID:
		reason, err := decodeStringOnly(pk.Data)
		if err != nil {
			return false, err
		}
		return false, &DisconnectError{State: StateLogin, Reason: reason}
	case p.LoginSetCompressionPacketID:
		threshold, err := DecodeSetCompression(pk.Data)
		if err != nil {
			return false, err
		}
		return false, c.Conn.SetCompression(int(threshold))
	case p.LoginEncryptionPacketID:
		return false, ErrOnlineModeRequired
	case p.LoginSuccessPacketID:
		login, err := DecodeLoginSuccess(pk.Data)
		if err != nil {
			return false, err
		}
		c.Login = login
		if p.LoginAcknowledgedPacketID >= 0 {
			if err := c.Conn.WritePacket(p.LoginAcknowledgedPacketID, nil); err != nil {
				return false, fmt.Errorf("java protocol: write login acknowledged: %w", err)
			}
		}
		return true, nil
	case p.LoginPluginRequestPacketID:
		messageID, err := decodePluginRequestID(pk.Data)
		if err != nil {
			return false, err
		}
		response, err := EncodeLoginPluginResponse(messageID, false, nil)
		if err != nil {
			return false, err
		}
		return false, c.Conn.WritePacket(p.LoginPluginResponsePacketID, response)
	case p.LoginCookieRequestPacketID:
		key, err := decodeCookieRequest(pk.Data)
		if err != nil {
			return false, err
		}
		response, err := EncodeCookieResponse(key, nil)
		if err != nil {
			return false, err
		}
		return false, c.Conn.WritePacket(p.LoginCookieResponsePacketID, response)
	default:
		return false, nil
	}
}

func (c *Client) sendClientInformation() error {
	data, err := EncodeClientInformation(ClientInformation{
		Locale:              "en_us",
		ViewDistance:        10,
		ChatFlags:           0,
		ChatColors:          true,
		SkinParts:           0x7f,
		MainHand:            1,
		EnableTextFiltering: false,
		EnableServerListing: true,
		ParticleStatus:      0,
	})
	if err != nil {
		return err
	}
	return c.Conn.WritePacket(c.Profile.ConfigClientInformationPacketID, data)
}

func (c *Client) configurationPacket(pk Packet) (bool, error) {
	p := c.Profile
	switch pk.ID {
	case p.ConfigDisconnectPacketID:
		return false, &DisconnectError{State: StateConfiguration, Reason: "server sent a configuration disconnect"}
	case p.ConfigKeepAlivePacketID:
		value, err := DecodeLongPayload(pk.Data)
		if err != nil {
			return false, err
		}
		return false, c.Conn.WritePacket(p.ConfigServerboundKeepAlivePacketID, EncodeLongPayload(value))
	case p.ConfigPingPacketID:
		value, err := DecodeInt32Payload(pk.Data)
		if err != nil {
			return false, err
		}
		return false, c.Conn.WritePacket(p.ConfigServerboundPongPacketID, EncodeInt32Payload(value))
	case p.ConfigSelectKnownPacksPacketID:
		data, err := EncodeSelectKnownPacks(nil)
		if err != nil {
			return false, err
		}
		return false, c.Conn.WritePacket(p.ConfigServerboundSelectKnownPacksID, data)
	case p.ConfigCookieRequestPacketID:
		key, err := decodeCookieRequest(pk.Data)
		if err != nil {
			return false, err
		}
		data, err := EncodeCookieResponse(key, nil)
		if err != nil {
			return false, err
		}
		return false, c.Conn.WritePacket(p.ConfigServerboundCookieResponsePacketID, data)
	case p.ConfigFinishPacketID:
		if err := c.Conn.WritePacket(p.ConfigServerboundFinishPacketID, nil); err != nil {
			return false, err
		}
		return true, nil
	default:
		// Configuration packets such as registry data, tags, and feature flags
		// are intentionally retained for the future translator and are safe to
		// ignore while negotiating a connection.
		return false, nil
	}
}

func DecodeLoginSuccess(data []byte) (LoginSuccess, error) {
	r := NewReader(data)
	uuidBytes, err := r.Bytes(16)
	if err != nil {
		return LoginSuccess{}, err
	}
	var result LoginSuccess
	copy(result.UUID[:], uuidBytes)
	if result.Username, err = r.String(); err != nil {
		return LoginSuccess{}, err
	}
	count, err := r.VarInt()
	if err != nil {
		return LoginSuccess{}, err
	}
	if count < 0 || count > 4096 {
		return LoginSuccess{}, fmt.Errorf("java protocol: invalid property count %d", count)
	}
	result.Properties = make([]Property, 0, count)
	for i := int32(0); i < count; i++ {
		name, err := r.String()
		if err != nil {
			return LoginSuccess{}, err
		}
		value, err := r.String()
		if err != nil {
			return LoginSuccess{}, err
		}
		hasSignature, err := r.Bool()
		if err != nil {
			return LoginSuccess{}, err
		}
		signature := ""
		if hasSignature {
			if signature, err = r.String(); err != nil {
				return LoginSuccess{}, err
			}
		}
		result.Properties = append(result.Properties, Property{Name: name, Value: value, Signature: signature})
	}
	if r.Remaining() != 0 {
		return LoginSuccess{}, fmt.Errorf("java protocol: login success has %d trailing bytes", r.Remaining())
	}
	return result, nil
}

func decodeStringOnly(data []byte) (string, error) {
	r := NewReader(data)
	value, err := r.String()
	if err != nil {
		return "", err
	}
	if r.Remaining() != 0 {
		return "", fmt.Errorf("java protocol: string packet has %d trailing bytes", r.Remaining())
	}
	return value, nil
}

func decodePluginRequestID(data []byte) (int32, error) {
	r := NewReader(data)
	return r.VarInt()
}

func decodeCookieRequest(data []byte) (string, error) {
	return decodeStringOnly(data)
}

func handshakeAddress(address string) (string, uint16) {
	host, portText, err := net.SplitHostPort(address)
	if err == nil {
		port, parseErr := strconv.ParseUint(portText, 10, 16)
		if parseErr == nil {
			return host, uint16(port)
		}
	}
	return strings.TrimSpace(address), 25565
}

func applyContextDeadline(ctx context.Context, conn net.Conn) func() {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
		return func() { _ = conn.SetDeadline(time.Time{}) }
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.SetDeadline(time.Now())
		case <-done:
		}
	}()
	return func() { close(done) }
}

func EncodeUUID(uuidBytes [16]byte) []byte {
	result := make([]byte, 16)
	copy(result, uuidBytes[:])
	return result
}

func UUIDFromBytes(data []byte) (uuid [16]byte, err error) {
	if len(data) != len(uuid) {
		return uuid, errors.New("java protocol: UUID must contain 16 bytes")
	}
	copy(uuid[:], data)
	return uuid, nil
}

func DecodeInt32Payload(data []byte) (int32, error) {
	if len(data) != 4 {
		return 0, fmt.Errorf("java protocol: expected 4-byte integer, got %d", len(data))
	}
	return int32(binary.BigEndian.Uint32(data)), nil
}
