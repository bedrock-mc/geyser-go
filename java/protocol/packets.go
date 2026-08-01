package protocol

import (
	"encoding/binary"
	"fmt"
)

type Handshake struct {
	ProtocolVersion int32
	ServerAddress   string
	ServerPort      uint16
	NextState       int32
}

func (p Handshake) Encode() ([]byte, error) {
	w := NewWriter()
	if err := w.VarInt(p.ProtocolVersion); err != nil {
		return nil, err
	}
	if err := w.String(p.ServerAddress); err != nil {
		return nil, err
	}
	if err := w.Uint16(p.ServerPort); err != nil {
		return nil, err
	}
	if err := w.VarInt(p.NextState); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func DecodeHandshake(data []byte) (Handshake, error) {
	r := NewReader(data)
	version, err := r.VarInt()
	if err != nil {
		return Handshake{}, err
	}
	address, err := r.String()
	if err != nil {
		return Handshake{}, err
	}
	port, err := r.Uint16()
	if err != nil {
		return Handshake{}, err
	}
	state, err := r.VarInt()
	if err != nil {
		return Handshake{}, err
	}
	if r.Remaining() != 0 {
		return Handshake{}, fmt.Errorf("java protocol: handshake has %d trailing bytes", r.Remaining())
	}
	return Handshake{ProtocolVersion: version, ServerAddress: address, ServerPort: port, NextState: state}, nil
}

type LoginStart struct {
	Username string
	UUID     *[16]byte
}

func (p LoginStart) Encode(includeUUID bool) ([]byte, error) {
	w := NewWriter()
	if err := w.String(p.Username); err != nil {
		return nil, err
	}
	if includeUUID {
		if p.UUID == nil {
			if err := w.Bool(false); err != nil {
				return nil, err
			}
		} else {
			if err := w.Bool(true); err != nil {
				return nil, err
			}
			if err := w.BytesValue(p.UUID[:]); err != nil {
				return nil, err
			}
		}
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func EncodeStringPayload(value string) ([]byte, error) {
	w := NewWriter()
	if err := w.String(value); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func EncodeSetCompression(threshold int32) ([]byte, error) {
	if threshold < 0 {
		return nil, fmt.Errorf("java protocol: negative compression threshold %d", threshold)
	}
	w := NewWriter()
	if err := w.VarInt(threshold); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func EncodeLongPayload(value int64) []byte {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], uint64(value))
	return data[:]
}

func DecodeSetCompression(data []byte) (int32, error) {
	r := NewReader(data)
	threshold, err := r.VarInt()
	if err != nil {
		return 0, err
	}
	if r.Remaining() != 0 {
		return 0, fmt.Errorf("java protocol: set-compression has %d trailing bytes", r.Remaining())
	}
	return threshold, nil
}

// ClientInformation contains the configuration packet sent immediately after
// login acknowledgement by Java 1.20.2 and newer servers.
type ClientInformation struct {
	Locale              string
	ViewDistance        int8
	ChatFlags           int32
	ChatColors          bool
	SkinParts           byte
	MainHand            int32
	EnableTextFiltering bool
	EnableServerListing bool
	ParticleStatus      int32
}

func EncodeClientInformation(info ClientInformation) ([]byte, error) {
	w := NewWriter()
	if err := w.String(info.Locale); err != nil {
		return nil, err
	}
	if err := w.Byte(byte(info.ViewDistance)); err != nil {
		return nil, err
	}
	if err := w.VarInt(info.ChatFlags); err != nil {
		return nil, err
	}
	if err := w.Bool(info.ChatColors); err != nil {
		return nil, err
	}
	if err := w.Byte(info.SkinParts); err != nil {
		return nil, err
	}
	if err := w.VarInt(info.MainHand); err != nil {
		return nil, err
	}
	if err := w.Bool(info.EnableTextFiltering); err != nil {
		return nil, err
	}
	if err := w.Bool(info.EnableServerListing); err != nil {
		return nil, err
	}
	if err := w.VarInt(info.ParticleStatus); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

type KnownPack struct {
	Namespace string
	ID        string
	Version   string
}

func EncodeSelectKnownPacks(packs []KnownPack) ([]byte, error) {
	w := NewWriter()
	if err := w.VarInt(int32(len(packs))); err != nil {
		return nil, err
	}
	for _, pack := range packs {
		for _, value := range []string{pack.Namespace, pack.ID, pack.Version} {
			if err := w.String(value); err != nil {
				return nil, err
			}
		}
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func EncodeLoginPluginResponse(messageID int32, successful bool, data []byte) ([]byte, error) {
	w := NewWriter()
	if err := w.VarInt(messageID); err != nil {
		return nil, err
	}
	if err := w.Bool(successful); err != nil {
		return nil, err
	}
	if successful {
		if err := w.BytesValue(data); err != nil {
			return nil, err
		}
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func EncodeCookieResponse(key string, data []byte) ([]byte, error) {
	w := NewWriter()
	if err := w.String(key); err != nil {
		return nil, err
	}
	if err := w.Bool(data != nil); err != nil {
		return nil, err
	}
	if data != nil {
		if err := w.BytesValue(data); err != nil {
			return nil, err
		}
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func DecodeLongPayload(data []byte) (int64, error) {
	if len(data) != 8 {
		return 0, fmt.Errorf("java protocol: expected 8-byte integer, got %d", len(data))
	}
	return int64(binary.BigEndian.Uint64(data)), nil
}

func EncodeInt32Payload(value int32) []byte {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], uint32(value))
	return data[:]
}
