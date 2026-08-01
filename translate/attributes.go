package translate

import (
	"fmt"
	"math"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const maxJavaAttributeCount = 256

// JavaEntityAttributes is the bounded subset of Java's
// ClientboundUpdateAttributesPacket needed to expose server-controlled
// movement/combat values to Bedrock. Unknown registry keys are retained as
// skipped attributes rather than making an otherwise valid session fatal.
type JavaEntityAttributes struct {
	EntityID   int32
	Attributes []JavaAttribute
}

type JavaAttribute struct {
	Key       int32
	Value     float64
	Modifiers []JavaAttributeModifier
}

type JavaAttributeModifier struct {
	UUID      string
	Amount    float64
	Operation int8
}

func DecodeEntityAttributes(payload []byte) (JavaEntityAttributes, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityAttributes{}, fmt.Errorf("translate: attribute entity ID: %w", err)
	}
	count, err := readCollectionCountLimit(r, "attribute", maxJavaAttributeCount)
	if err != nil {
		return JavaEntityAttributes{}, err
	}
	attributes := make([]JavaAttribute, 0, count)
	for i := 0; i < count; i++ {
		key, err := r.VarInt()
		if err != nil {
			return JavaEntityAttributes{}, fmt.Errorf("translate: attribute %d key: %w", i, err)
		}
		value, err := r.Float64()
		if err != nil {
			return JavaEntityAttributes{}, fmt.Errorf("translate: attribute %d value: %w", i, err)
		}
		modifierCount, err := readCollectionCountLimit(r, "attribute modifier", maxJavaAttributeCount)
		if err != nil {
			return JavaEntityAttributes{}, err
		}
		modifiers := make([]JavaAttributeModifier, 0, modifierCount)
		for modifier := 0; modifier < modifierCount; modifier++ {
			uuid, err := r.String()
			if err != nil {
				return JavaEntityAttributes{}, fmt.Errorf("translate: attribute %d modifier %d UUID: %w", i, modifier, err)
			}
			amount, err := r.Float64()
			if err != nil {
				return JavaEntityAttributes{}, fmt.Errorf("translate: attribute %d modifier %d amount: %w", i, modifier, err)
			}
			operation, err := r.Int8()
			if err != nil {
				return JavaEntityAttributes{}, fmt.Errorf("translate: attribute %d modifier %d operation: %w", i, modifier, err)
			}
			modifiers = append(modifiers, JavaAttributeModifier{UUID: uuid, Amount: amount, Operation: operation})
		}
		attributes = append(attributes, JavaAttribute{Key: key, Value: value, Modifiers: modifiers})
	}
	if r.Remaining() != 0 {
		return JavaEntityAttributes{}, fmt.Errorf("translate: entity attributes have %d trailing bytes", r.Remaining())
	}
	return JavaEntityAttributes{EntityID: entityID, Attributes: attributes}, nil
}

func readCollectionCountLimit(r *javaprotocol.Reader, what string, limit int) (int, error) {
	count, err := r.VarInt()
	if err != nil {
		return 0, fmt.Errorf("translate: %s count: %w", what, err)
	}
	if count < 0 || count > int32(limit) {
		return 0, fmt.Errorf("translate: invalid %s count %d", what, count)
	}
	return int(count), nil
}

type bedrockAttributeDefinition struct {
	name                    string
	min, max, defaultValue  float32
	mapValueToHealthMaximum bool
}

var javaAttributeDefinitions = [...]bedrockAttributeDefinition{
	{name: "minecraft:armor", min: 0, max: 30, defaultValue: 0},
	{name: "minecraft:armor_toughness", min: 0, max: 20, defaultValue: 0},
	{name: "minecraft:attack_damage", min: 0, max: 2048, defaultValue: 1},
	{name: "minecraft:attack_knockback", min: 1.5, max: math.MaxFloat32, defaultValue: 0},
	{name: "minecraft:attack_speed", min: 0, max: 1024, defaultValue: 4},
	{name: "minecraft:block_break_speed", min: 0, max: 1024, defaultValue: 1},
	{name: "minecraft:block_interaction_range", min: 0, max: 64, defaultValue: 4.5},
	{name: "minecraft:entity_interaction_range", min: 0, max: 64, defaultValue: 3},
	{},
	{name: "minecraft:flying_speed", min: 0, max: 1024, defaultValue: 0.4},
	{name: "minecraft:follow_range", min: 0, max: 2048, defaultValue: 32},
	{},
	{name: "minecraft:horse.jump_strength", min: 0, max: 2, defaultValue: 0.7},
	{name: "minecraft:knockback_resistance", min: 0, max: 1, defaultValue: 0},
	{name: "minecraft:luck", min: -1024, max: 1024, defaultValue: 0},
	{name: "minecraft:absorption", min: 0, max: 1024, defaultValue: 0},
	{name: "minecraft:health", min: 0, max: 1024, defaultValue: 20, mapValueToHealthMaximum: true},
	{name: "minecraft:movement", min: 0, max: 1024, defaultValue: 0.1},
	{},
	{name: "minecraft:scale", min: 0.0625, max: 16, defaultValue: 1},
	{},
	{name: "minecraft:step_height", min: 0, max: 16, defaultValue: 0.6},
}

func (b *Basic) translateEntityAttributes(bedrock *minecraft.Conn, payload []byte) error {
	update, err := DecodeEntityAttributes(payload)
	if err != nil {
		return err
	}
	b.mu.Lock()
	entity := b.entities[update.EntityID]
	player := update.EntityID == int32(b.gameData.EntityUniqueID)
	b.mu.Unlock()
	if entity == nil && !player {
		return nil
	}
	runtimeID := b.gameData.EntityRuntimeID
	if entity != nil {
		runtimeID = entity.runtimeID
	}
	attributes := make([]gtprotocol.Attribute, 0, len(update.Attributes))
	for _, java := range update.Attributes {
		if java.Key < 0 || int(java.Key) >= len(javaAttributeDefinitions) {
			b.logSemanticAnomaly("skipping Java attribute outside generated definitions", "key", java.Key)
			continue
		}
		definition := javaAttributeDefinitions[java.Key]
		if definition.name == "" || !finiteFloat64(java.Value) {
			b.logSemanticAnomaly("skipping unsupported or non-finite Java attribute", "key", java.Key, "value", java.Value)
			continue
		}
		value := float32(java.Value)
		if !finiteFloat32(value) {
			b.logSemanticAnomaly("skipping Java attribute outside float32 range", "key", java.Key, "value", java.Value)
			continue
		}
		attribute := gtprotocol.Attribute{
			AttributeValue: gtprotocol.AttributeValue{
				Name: definition.name, Min: definition.min, Value: value, Max: definition.max,
			},
			DefaultMin: definition.min, DefaultMax: definition.max, Default: definition.defaultValue,
		}
		if definition.mapValueToHealthMaximum {
			attribute.AttributeValue.Max = clampFloat32(value, definition.min, definition.max)
		}
		for _, modifier := range java.Modifiers {
			if !finiteFloat64(modifier.Amount) || modifier.Operation < 0 || modifier.Operation > 2 {
				b.logSemanticAnomaly("skipping invalid Java attribute modifier", "uuid", modifier.UUID, "operation", modifier.Operation)
				continue
			}
			attribute.Modifiers = append(attribute.Modifiers, gtprotocol.AttributeModifier{
				ID: modifier.UUID, Name: modifier.UUID, Amount: float32(modifier.Amount), Operation: int32(modifier.Operation), Serializable: true,
			})
		}
		attributes = append(attributes, attribute)
	}
	if len(attributes) == 0 {
		return nil
	}
	return bedrock.WritePacket(&packet.UpdateAttributes{EntityRuntimeID: runtimeID, Attributes: attributes})
}

func finiteFloat64(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func clampFloat32(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
