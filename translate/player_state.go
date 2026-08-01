package translate

import (
	"fmt"
	"math"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type JavaExperience struct {
	Progress float32
	Level    int32
	Total    int32
}

func DecodeExperience(payload []byte) (JavaExperience, error) {
	r := javaprotocol.NewReader(payload)
	progress, err := r.Float32()
	if err != nil {
		return JavaExperience{}, fmt.Errorf("translate: experience progress: %w", err)
	}
	level, err := r.VarInt()
	if err != nil {
		return JavaExperience{}, fmt.Errorf("translate: experience level: %w", err)
	}
	total, err := r.VarInt()
	if err != nil {
		return JavaExperience{}, fmt.Errorf("translate: total experience: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaExperience{}, fmt.Errorf("translate: experience has %d trailing bytes", r.Remaining())
	}
	return JavaExperience{Progress: progress, Level: level, Total: total}, nil
}

func (b *Basic) translateExperience(bedrock *minecraft.Conn, payload []byte) error {
	experience, err := DecodeExperience(payload)
	if err != nil {
		return err
	}
	if math.IsNaN(float64(experience.Progress)) || math.IsInf(float64(experience.Progress), 0) || experience.Progress < 0 || experience.Progress > 1 || experience.Level < 0 {
		b.logSemanticAnomaly("skipping Java experience update with invalid values", "progress", experience.Progress, "level", experience.Level)
		return nil
	}
	return bedrock.WritePacket(&packet.UpdateAttributes{
		EntityRuntimeID: b.gameData.EntityRuntimeID,
		Attributes: []gtprotocol.Attribute{
			{
				AttributeValue: gtprotocol.AttributeValue{
					Name: "minecraft:player.experience", Min: 0, Value: experience.Progress, Max: 1,
				},
				DefaultMin: 0, DefaultMax: 1, Default: 0,
			},
			{
				AttributeValue: gtprotocol.AttributeValue{
					Name: "minecraft:player.level", Min: 0, Value: float32(experience.Level), Max: 24791,
				},
				DefaultMin: 0, DefaultMax: 24791, Default: 0,
			},
		},
	})
}

type JavaPlayerAbilities struct {
	Flags        int8
	FlyingSpeed  float32
	WalkingSpeed float32
}

func DecodePlayerAbilities(payload []byte) (JavaPlayerAbilities, error) {
	r := javaprotocol.NewReader(payload)
	flags, err := r.Int8()
	if err != nil {
		return JavaPlayerAbilities{}, fmt.Errorf("translate: player abilities flags: %w", err)
	}
	flyingSpeed, err := r.Float32()
	if err != nil {
		return JavaPlayerAbilities{}, fmt.Errorf("translate: player abilities flying speed: %w", err)
	}
	walkingSpeed, err := r.Float32()
	if err != nil {
		return JavaPlayerAbilities{}, fmt.Errorf("translate: player abilities walking speed: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaPlayerAbilities{}, fmt.Errorf("translate: player abilities has %d trailing bytes", r.Remaining())
	}
	return JavaPlayerAbilities{Flags: flags, FlyingSpeed: flyingSpeed, WalkingSpeed: walkingSpeed}, nil
}

func (b *Basic) translatePlayerAbilities(bedrock *minecraft.Conn, payload []byte) error {
	abilities, err := DecodePlayerAbilities(payload)
	if err != nil {
		return err
	}
	if !finiteFloat32(abilities.FlyingSpeed) || !finiteFloat32(abilities.WalkingSpeed) || abilities.FlyingSpeed < 0 || abilities.WalkingSpeed < 0 {
		b.logSemanticAnomaly("skipping Java player abilities with invalid speeds")
		return nil
	}
	const (
		javaInvulnerable = 1 << 0
		javaFlying       = 1 << 1
		javaAllowFlying  = 1 << 2
		javaCreative     = 1 << 3
	)
	values := uint32(gtprotocol.AbilityBuild | gtprotocol.AbilityMine | gtprotocol.AbilityDoorsAndSwitches | gtprotocol.AbilityOpenContainers | gtprotocol.AbilityAttackPlayers | gtprotocol.AbilityAttackMobs)
	if abilities.Flags&javaAllowFlying != 0 {
		values |= gtprotocol.AbilityMayFly
	}
	if abilities.Flags&javaFlying != 0 {
		values |= gtprotocol.AbilityFlying
	}
	if abilities.Flags&javaInvulnerable != 0 {
		values |= gtprotocol.AbilityInvulnerable
	}
	if abilities.Flags&javaCreative != 0 {
		values |= gtprotocol.AbilityInstantBuild
	}
	base := gtprotocol.AbilityLayer{
		Type:             gtprotocol.AbilityLayerTypeBase,
		Abilities:        gtprotocol.AbilityCount - 1,
		Values:           values,
		FlySpeed:         abilities.FlyingSpeed,
		VerticalFlySpeed: gtprotocol.AbilityBaseVerticalFlySpeed,
		WalkSpeed:        abilities.WalkingSpeed,
	}
	if base.FlySpeed == 0 {
		base.FlySpeed = gtprotocol.AbilityBaseFlySpeed
	}
	if base.WalkSpeed == 0 {
		base.WalkSpeed = gtprotocol.AbilityBaseWalkSpeed
	}
	return bedrock.WritePacket(&packet.UpdateAbilities{AbilityData: gtprotocol.AbilityData{
		EntityUniqueID:     b.gameData.EntityUniqueID,
		PlayerPermissions:  byte(packet.PermissionLevelMember),
		CommandPermissions: byte(gtprotocol.CommandPermissionLevelAny),
		Layers:             []gtprotocol.AbilityLayer{base},
	}})
}

func finiteFloat32(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
