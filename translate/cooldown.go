package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaCooldown is the wire shape of Java's ClientboundCooldownPacket.
// cooldownGroup is a resource-location-like string identifying the item group;
// cooldownTicks is measured in Java server ticks.
type JavaCooldown struct {
	Group string
	Ticks int32
}

// DecodeJavaCooldown decodes one complete ClientboundCooldownPacket. A
// truncated or otherwise malformed packet is fatal to the packet pump; an
// unusual but well-formed group or duration is handled by the translator.
func DecodeJavaCooldown(data []byte) (JavaCooldown, error) {
	r := javaprotocol.NewReader(data)
	group, err := r.String()
	if err != nil {
		return JavaCooldown{}, fmt.Errorf("translate: cooldown group: %w", err)
	}
	ticks, err := r.VarInt()
	if err != nil {
		return JavaCooldown{}, fmt.Errorf("translate: cooldown ticks: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaCooldown{}, fmt.Errorf("translate: cooldown has %d trailing bytes", r.Remaining())
	}
	return JavaCooldown{Group: group, Ticks: ticks}, nil
}

// bedrockCooldownCategory applies the vanilla Bedrock category names used by
// Geyser for the two server-driven Java item cooldown groups. Other groups are
// preserved so custom and modded servers retain a useful best-effort signal.
func bedrockCooldownCategory(group string) string {
	switch group {
	case "minecraft:goat_horn":
		return "goat_horn"
	case "minecraft:shield":
		return "shield"
	default:
		return group
	}
}

func (b *Basic) translateJavaCooldown(bedrock *minecraft.Conn, data []byte) error {
	cooldown, err := DecodeJavaCooldown(data)
	if err != nil {
		return err
	}
	if cooldown.Group == "" {
		b.logSemanticAnomaly("skipping Java cooldown with an empty group")
		return nil
	}
	if cooldown.Ticks < 0 {
		b.logSemanticAnomaly("skipping Java cooldown with a negative duration", "group", cooldown.Group, "ticks", cooldown.Ticks)
		return nil
	}
	return bedrock.WritePacket(&packet.ClientStartItemCooldown{
		Category: bedrockCooldownCategory(cooldown.Group),
		Duration: cooldown.Ticks,
	})
}
