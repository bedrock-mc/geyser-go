package translate

import (
	"fmt"
	"strings"

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

// JavaTextProjection is the Bedrock-facing projection of a Java network text
// component. A top-level translation can be passed through to Bedrock so the
// client applies its own locale; components that cannot be represented by one
// Bedrock translation packet retain a bounded plain-text fallback.
type JavaTextProjection struct {
	PlainText      string
	TranslationKey string
	Parameters     []string
}

// ProjectJavaTextComponent preserves a simple top-level Java translation for
// Bedrock's TextTypeTranslation packet. Nested/extra components use PlainText
// because the Bedrock text packet has no field for a suffix component.
func ProjectJavaTextComponent(value any) JavaTextProjection {
	projection := JavaTextProjection{PlainText: JavaTextComponentText(value)}
	component, ok := value.(map[string]any)
	if !ok {
		return projection
	}
	key, ok := component["translate"].(string)
	if !ok || key == "" {
		return projection
	}
	parameters, ok := javaTextComponentParameters(component["with"])
	if !ok || component["extra"] != nil {
		return projection
	}
	projection.TranslationKey = key
	projection.Parameters = parameters
	return projection
}

func javaTextComponentParameters(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	parts, ok := value.([]any)
	if !ok || len(parts) > maxJavaCollectionSize {
		return nil, false
	}
	parameters := make([]string, 0, len(parts))
	for _, part := range parts {
		parameters = append(parameters, JavaTextComponentText(part))
	}
	return parameters, true
}

// JavaTextComponentText extracts the useful plain text from a Java network
// text component. It uses a small protocol-level fallback table for the
// common keys that appear in command feedback and chat decoration; unknown
// keys remain visible instead of being silently discarded.
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
			key, _ := translate.(string)
			parameters, _ := javaTextComponentParameters(value["with"])
			return formatJavaTextTranslation(key, parameters) + JavaTextComponentText(value["extra"])
		}
		if key, ok := value["keybind"]; ok {
			return JavaTextComponentText(key)
		}
		return JavaTextComponentText(value["extra"])
	default:
		return ""
	}
}

var javaTextTranslationFallbacks = map[string]string{
	"chat.type.achievement":             "%s has just earned the achievement %s",
	"chat.type.advancement":             "%s has made the advancement %s",
	"chat.type.announcement":            "[%s] %s",
	"chat.type.emote":                   "* %s %s",
	"chat.type.text":                    "<%s> %s",
	"command.context.here":              "<--[HERE]",
	"command.unknown.argument":          "Incorrect argument for command",
	"command.unknown.command":           "Unknown or incomplete command, see below for error",
	"commands.generic.permission":       "You do not have permission to use this command",
	"commands.generic.usage":            "Usage: %s",
	"commands.time.set":                 "Set the time to %s",
	"multiplayer.player.joined":         "%s joined the game",
	"multiplayer.player.left":           "%s left the game",
	"multiplayer.player.joined.renamed": "%s joined the game",
}

func formatJavaTextTranslation(key string, parameters []string) string {
	template, ok := javaTextTranslationFallbacks[key]
	if !ok {
		if len(parameters) == 0 {
			return key
		}
		return key + " " + strings.Join(parameters, " ")
	}
	for _, parameter := range parameters {
		index := strings.Index(template, "%")
		if index < 0 {
			break
		}
		end := index + 1
		for end < len(template) && (template[end] >= '0' && template[end] <= '9' || template[end] == '$') {
			end++
		}
		if end >= len(template) || (template[end] != 's' && template[end] != 'd') {
			continue
		}
		template = template[:index] + parameter + template[end+1:]
	}
	return template
}
