package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

type JavaSystemChat struct {
	Content   any
	ActionBar bool
}

func DecodeSystemChat(payload []byte) (JavaSystemChat, error) {
	r := javaprotocol.NewReader(payload)
	content, err := decodeJavaNBTValue(r)
	if err != nil {
		return JavaSystemChat{}, fmt.Errorf("translate: system chat content: %w", err)
	}
	actionBar, err := r.Bool()
	if err != nil {
		return JavaSystemChat{}, fmt.Errorf("translate: system chat action bar: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaSystemChat{}, fmt.Errorf("translate: system chat has %d trailing bytes", r.Remaining())
	}
	return JavaSystemChat{Content: content, ActionBar: actionBar}, nil
}

type JavaPlayerChat struct {
	SenderUUID  [16]byte
	Message     string
	NetworkName any
}

func DecodePlayerChat(payload []byte) (JavaPlayerChat, error) {
	r := javaprotocol.NewReader(payload)
	var result JavaPlayerChat
	uuid, err := r.Bytes(16)
	if err != nil {
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat sender UUID: %w", err)
	}
	copy(result.SenderUUID[:], uuid)
	if _, err := r.VarInt(); err != nil { // message index
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat index: %w", err)
	}
	if err := readOptionalBytes(r, 256, "player chat signature"); err != nil {
		return JavaPlayerChat{}, err
	}
	if result.Message, err = r.String(); err != nil {
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat message: %w", err)
	}
	if _, err := r.Int64(); err != nil { // timestamp
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat timestamp: %w", err)
	}
	if _, err := r.Int64(); err != nil { // salt
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat salt: %w", err)
	}
	previous, err := boundedJavaCount(r, "player chat previous message count")
	if err != nil {
		return JavaPlayerChat{}, err
	}
	for i := 0; i < previous; i++ {
		id, readErr := r.VarInt()
		if readErr != nil {
			return JavaPlayerChat{}, fmt.Errorf("translate: player chat previous message %d ID: %w", i, readErr)
		}
		if id == 0 {
			if _, readErr := r.Bytes(256); readErr != nil {
				return JavaPlayerChat{}, fmt.Errorf("translate: player chat previous message %d signature: %w", i, readErr)
			}
		}
	}
	if err := readOptionalNBT(r, "player chat unsigned content"); err != nil {
		return JavaPlayerChat{}, err
	}
	filterType, err := r.VarInt()
	if err != nil {
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat filter type: %w", err)
	}
	if filterType == 2 {
		count, countErr := boundedJavaCount(r, "player chat filter mask count")
		if countErr != nil {
			return JavaPlayerChat{}, countErr
		}
		for i := 0; i < count; i++ {
			if _, readErr := r.Int64(); readErr != nil {
				return JavaPlayerChat{}, fmt.Errorf("translate: player chat filter mask %d: %w", i, readErr)
			}
		}
	}
	if err := readChatTypeHolder(r); err != nil {
		return JavaPlayerChat{}, err
	}
	if result.NetworkName, err = decodeJavaNBTValue(r); err != nil {
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat network name: %w", err)
	}
	if err := readOptionalNBT(r, "player chat network target name"); err != nil {
		return JavaPlayerChat{}, err
	}
	if r.Remaining() != 0 {
		return JavaPlayerChat{}, fmt.Errorf("translate: player chat has %d trailing bytes", r.Remaining())
	}
	return result, nil
}

type JavaProfilelessChat struct {
	Message any
	Name    any
}

func DecodeProfilelessChat(payload []byte) (JavaProfilelessChat, error) {
	r := javaprotocol.NewReader(payload)
	message, err := decodeJavaNBTValue(r)
	if err != nil {
		return JavaProfilelessChat{}, fmt.Errorf("translate: profileless chat message: %w", err)
	}
	if err := readChatTypeHolder(r); err != nil {
		return JavaProfilelessChat{}, err
	}
	name, err := decodeJavaNBTValue(r)
	if err != nil {
		return JavaProfilelessChat{}, fmt.Errorf("translate: profileless chat name: %w", err)
	}
	if err := readOptionalNBT(r, "profileless chat target"); err != nil {
		return JavaProfilelessChat{}, err
	}
	if r.Remaining() != 0 {
		return JavaProfilelessChat{}, fmt.Errorf("translate: profileless chat has %d trailing bytes", r.Remaining())
	}
	return JavaProfilelessChat{Message: message, Name: name}, nil
}

func readOptionalBytes(r *javaprotocol.Reader, length int, field string) error {
	present, err := r.Bool()
	if err != nil {
		return fmt.Errorf("translate: %s presence: %w", field, err)
	}
	if !present {
		return nil
	}
	if _, err := r.Bytes(length); err != nil {
		return fmt.Errorf("translate: %s: %w", field, err)
	}
	return nil
}

func readOptionalNBT(r *javaprotocol.Reader, field string) error {
	present, err := r.Bool()
	if err != nil {
		return fmt.Errorf("translate: %s presence: %w", field, err)
	}
	if !present {
		return nil
	}
	if _, err := decodeJavaNBTValue(r); err != nil {
		return fmt.Errorf("translate: %s: %w", field, err)
	}
	return nil
}

// ChatTypesHolder is an ID-or-inline registry holder. Chat types received from
// the server normally use the registry ID; zero carries the inline definition.
func readChatTypeHolder(r *javaprotocol.Reader) error {
	holder, err := r.VarInt()
	if err != nil {
		return fmt.Errorf("translate: chat type holder: %w", err)
	}
	if holder != 0 {
		return nil
	}
	if _, err := r.String(); err != nil { // translation key
		return fmt.Errorf("translate: inline chat type translation key: %w", err)
	}
	count, err := boundedJavaCount(r, "inline chat type parameter count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := r.VarInt(); err != nil {
			return fmt.Errorf("translate: inline chat type parameter %d: %w", i, err)
		}
	}
	if _, err := decodeJavaNBTValue(r); err != nil {
		return fmt.Errorf("translate: inline chat type style: %w", err)
	}
	return nil
}

// JavaTextComponentText extracts the useful plain text from a Java network
// text component. It intentionally keeps translation keys readable until the
// registry/locale layer is implemented.
func JavaTextComponentText(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case []any:
		var out string
		for _, part := range value {
			out += JavaTextComponentText(part)
		}
		return out
	case map[string]any:
		if text, ok := value["text"]; ok {
			return JavaTextComponentText(text) + JavaTextComponentText(value["extra"])
		}
		if translate, ok := value["translate"]; ok {
			return JavaTextComponentText(translate) + JavaTextComponentText(value["with"])
		}
		if key, ok := value["keybind"]; ok {
			return JavaTextComponentText(key)
		}
		return JavaTextComponentText(value["extra"])
	default:
		return ""
	}
}
