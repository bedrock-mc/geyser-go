package translate

import (
	"encoding/json"
	"strings"
)

// projectJavaBlockEntityPayload applies the type-specific NBT projections
// needed by Bedrock's tile-entity readers. The generic identity, coordinates,
// and unknown fields are assembled by BedrockBlockEntityTag first so this
// helper can be shared by chunk payloads and standalone updates.
func projectJavaBlockEntityPayload(javaName string, tag map[string]any) {
	switch javaName {
	case "sign", "hanging_sign":
		projectJavaSign(tag)
	}
}

func projectJavaSign(tag map[string]any) {
	front, _ := tag["front_text"].(map[string]any)
	back, _ := tag["back_text"].(map[string]any)
	tag["FrontText"] = projectJavaSignSide(front)
	tag["BackText"] = projectJavaSignSide(back)
	tag["IsWaxed"] = javaNBTBool(tag["is_waxed"])
	delete(tag, "front_text")
	delete(tag, "back_text")
	delete(tag, "is_waxed")
}

func projectJavaSignSide(value map[string]any) map[string]any {
	result := map[string]any{
		"Text":           "",
		"IgnoreLighting": false,
	}
	if value == nil {
		return result
	}

	if rawMessages, ok := value["messages"]; ok {
		result["Text"] = projectJavaSignMessages(rawMessages)
	}
	if color, ok := javaNBTString(value, "color"); ok {
		result["SignTextColor"] = javaSignColor(color)
	}
	result["IgnoreLighting"] = javaNBTBool(value["has_glowing_text"])
	return result
}

func projectJavaSignMessages(value any) string {
	var messages []any
	switch value := value.(type) {
	case []any:
		messages = value
	case []string:
		for _, message := range value {
			messages = append(messages, message)
		}
	case []map[string]any:
		for _, message := range value {
			messages = append(messages, message)
		}
	default:
		return ""
	}

	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, projectJavaSignMessage(message))
	}
	return strings.Join(lines, "\n")
}

func projectJavaSignMessage(value any) string {
	if raw, ok := value.(string); ok {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			var component any
			if json.Unmarshal([]byte(trimmed), &component) == nil {
				return JavaTextComponentText(component)
			}
		}
		return raw
	}
	return JavaTextComponentText(value)
}

func javaNBTBool(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case int8:
		return value != 0
	case uint8:
		return value != 0
	case int16:
		return value != 0
	case uint16:
		return value != 0
	case int32:
		return value != 0
	case uint32:
		return value != 0
	case int64:
		return value != 0
	case uint64:
		return value != 0
	case int:
		return value != 0
	case uint:
		return value != 0
	default:
		return false
	}
}

var javaSignColors = map[string]int32{
	"white":      16383998,
	"orange":     16351261,
	"magenta":    13061821,
	"light_blue": 3847130,
	"yellow":     16701501,
	"lime":       8439583,
	"pink":       15961002,
	"gray":       4673362,
	"light_gray": 10329495,
	"cyan":       1481884,
	"purple":     8991416,
	"blue":       3949738,
	"brown":      8606770,
	"green":      6192150,
	"red":        11546150,
}

// javaSignColor returns the ARGB value used by Bedrock's SignTextColor tag.
// The values are Geyser's vanilla Java dye-color table, including the opaque
// alpha channel expected by Bedrock.
func javaSignColor(color string) int32 {
	value := javaSignColors[color]
	return int32(uint32(value) | 0xff000000)
}
