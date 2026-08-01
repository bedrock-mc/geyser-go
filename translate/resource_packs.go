package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

// JavaResourcePackPush is the 1.21.4 common ClientboundResourcePackPush
// payload. The URL/hash are retained for future Bedrock pack hosting even
// though this compatibility slice only acknowledges the Java request.
type JavaResourcePackPush struct {
	UUID          [16]byte
	URL           string
	Hash          string
	Required      bool
	PromptMessage any
}

func DecodeJavaResourcePackPush(data []byte) (JavaResourcePackPush, error) {
	r := javaprotocol.NewReader(data)
	var result JavaResourcePackPush
	uuid, err := r.Bytes(16)
	if err != nil {
		return JavaResourcePackPush{}, fmt.Errorf("translate: resource pack UUID: %w", err)
	}
	copy(result.UUID[:], uuid)
	if result.URL, err = r.String(); err != nil {
		return JavaResourcePackPush{}, fmt.Errorf("translate: resource pack URL: %w", err)
	}
	if result.Hash, err = r.String(); err != nil {
		return JavaResourcePackPush{}, fmt.Errorf("translate: resource pack hash: %w", err)
	}
	if result.Required, err = r.Bool(); err != nil {
		return JavaResourcePackPush{}, fmt.Errorf("translate: resource pack required flag: %w", err)
	}
	present, err := r.Bool()
	if err != nil {
		return JavaResourcePackPush{}, fmt.Errorf("translate: resource pack prompt presence: %w", err)
	}
	if present {
		result.PromptMessage, err = decodeJavaNBTValue(r)
		if err != nil {
			return JavaResourcePackPush{}, fmt.Errorf("translate: resource pack prompt: %w", err)
		}
	}
	if r.Remaining() != 0 {
		return JavaResourcePackPush{}, fmt.Errorf("translate: resource pack push has %d trailing bytes", r.Remaining())
	}
	return result, nil
}

func DecodeJavaResourcePackPop(data []byte) (*[16]byte, error) {
	r := javaprotocol.NewReader(data)
	present, err := r.Bool()
	if err != nil {
		return nil, fmt.Errorf("translate: resource pack pop UUID presence: %w", err)
	}
	if !present {
		if r.Remaining() != 0 {
			return nil, fmt.Errorf("translate: resource pack pop has %d trailing bytes", r.Remaining())
		}
		return nil, nil
	}
	uuid, err := r.Bytes(16)
	if err != nil {
		return nil, fmt.Errorf("translate: resource pack pop UUID: %w", err)
	}
	var result [16]byte
	copy(result[:], uuid)
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("translate: resource pack pop has %d trailing bytes", r.Remaining())
	}
	return &result, nil
}

func (b *Basic) translateJavaResourcePackPush(java *javaprotocol.Client, data []byte) error {
	pack, err := DecodeJavaResourcePackPush(data)
	if err != nil {
		return err
	}
	if pack.Required {
		for _, status := range []int32{3, 4, 0} { // ACCEPTED, DOWNLOADED, SUCCESSFULLY_LOADED
			if err := java.Conn.WritePacket(b.Profile.PlayServerboundResourcePackID, encodeJavaResourcePackStatus(pack.UUID, status)); err != nil {
				return err
			}
		}
		return nil
	}
	return java.Conn.WritePacket(b.Profile.PlayServerboundResourcePackID, encodeJavaResourcePackStatus(pack.UUID, 1)) // DECLINED
}

func encodeJavaResourcePackStatus(uuid [16]byte, status int32) []byte {
	w := javaprotocol.NewWriter()
	_ = w.BytesValue(uuid[:])
	_ = w.VarInt(status)
	return append([]byte(nil), w.Bytes()...)
}
