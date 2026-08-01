package translate

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

const javaChatMessageLimit = 256

var errInvalidChatCommand = errors.New("translate: empty or oversized Java chat command")

// normalizeIncomingChat matches the whitespace normalization performed by a
// vanilla Java client before it chooses the chat-message or command packet.
// Bedrock's Text and CommandRequest packets are both accepted here because
// clients use either path depending on the UI and protocol revision.
func normalizeIncomingChat(message string) string {
	if message == "" {
		return ""
	}
	var normalized strings.Builder
	normalized.Grow(len(message))
	hasContent := false
	whitespacePending := false
	for _, r := range message {
		if r == '\u00a0' {
			r = ' '
		}
		if isJavaChatWhitespace(r) {
			if hasContent {
				whitespacePending = true
			}
			continue
		}
		if whitespacePending {
			normalized.WriteByte(' ')
			whitespacePending = false
		}
		normalized.WriteRune(r)
		hasContent = true
	}
	return normalized.String()
}

func isJavaChatWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}

func tooLongChat(message string) bool {
	return utf8.RuneCountInString(message) > javaChatMessageLimit
}

// encodeChatMessage writes the unsigned 1.21.4 ServerboundChatPacket. The
// bridge has no Java secure-chat key, so it intentionally sends a null
// signature and an empty 20-message acknowledgement bitset.
func encodeChatMessage(message string) ([]byte, error) {
	w := javaprotocol.NewWriter()
	if err := w.String(message); err != nil {
		return nil, err
	}
	if err := w.Int64(time.Now().UnixMilli()); err != nil {
		return nil, err
	}
	if err := w.Int64(0); err != nil {
		return nil, err
	}
	if err := w.Bool(false); err != nil { // no secure-chat signature
		return nil, err
	}
	if err := w.VarInt(0); err != nil { // offset
		return nil, err
	}
	if err := w.BytesValue([]byte{0, 0, 0}); err != nil { // 20-bit acknowledgement set
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

// encodeChatCommandSigned matches Geyser's current downstream command path:
// an unsigned ServerboundChatCommandSignedPacket with no argument
// signatures and an empty acknowledgement set.
func encodeChatCommandSigned(command string) ([]byte, error) {
	if command == "" || tooLongChat(command) {
		return nil, errInvalidChatCommand
	}
	w := javaprotocol.NewWriter()
	if err := w.String(command); err != nil {
		return nil, err
	}
	if err := w.Int64(time.Now().UnixMilli()); err != nil {
		return nil, err
	}
	if err := w.Int64(0); err != nil {
		return nil, err
	}
	if err := w.VarInt(0); err != nil { // argument signature count
		return nil, err
	}
	if err := w.VarInt(0); err != nil { // message count
		return nil, err
	}
	if err := w.BytesValue([]byte{0, 0, 0}); err != nil { // 20-bit acknowledgement set
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}
