package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

// Java 1.21.4 component IDs. Keep these versioned with the protocol profile:
// Mojang's registry is a wire-level ordered list, not a stable string enum.
const (
	javaItemComponentCustomData = iota
	javaItemComponentMaxStackSize
	javaItemComponentMaxDamage
	javaItemComponentDamage
	javaItemComponentUnbreakable
	javaItemComponentCustomName
	javaItemComponentItemName
	javaItemComponentItemModel
	javaItemComponentLore
	javaItemComponentRarity
	javaItemComponentEnchantments
	javaItemComponentCanPlaceOn
	javaItemComponentCanBreak
	javaItemComponentAttributeModifiers
	javaItemComponentCustomModelData
	javaItemComponentHideAdditionalTooltip
	javaItemComponentHideTooltip
	javaItemComponentRepairCost
	javaItemComponentCreativeSlotLock
	javaItemComponentEnchantmentGlintOverride
	javaItemComponentIntangibleProjectile
	javaItemComponentFood
	javaItemComponentConsumable
	javaItemComponentUseRemainder
	javaItemComponentUseCooldown
	javaItemComponentDamageResistant
	javaItemComponentTool
	javaItemComponentEnchantable
	javaItemComponentEquippable
	javaItemComponentRepairable
	javaItemComponentGlider
	javaItemComponentTooltipStyle
	javaItemComponentDeathProtection
	javaItemComponentStoredEnchantments
	javaItemComponentDyedColor
	javaItemComponentMapColor
	javaItemComponentMapID
	javaItemComponentMapDecorations
	javaItemComponentMapPostProcessing
	javaItemComponentChargedProjectiles
	javaItemComponentBundleContents
	javaItemComponentPotionContents
	javaItemComponentSuspiciousStewEffects
	javaItemComponentWritableBookContent
	javaItemComponentWrittenBookContent
	javaItemComponentTrim
	javaItemComponentDebugStickState
	javaItemComponentEntityData
	javaItemComponentBucketEntityData
	javaItemComponentBlockEntityData
	javaItemComponentInstrument
	javaItemComponentOminousBottleAmplifier
	javaItemComponentJukeboxPlayable
	javaItemComponentRecipes
	javaItemComponentLodestoneTracker
	javaItemComponentFireworkExplosion
	javaItemComponentFireworks
	javaItemComponentProfile
	javaItemComponentNoteBlockSound
	javaItemComponentBannerPatterns
	javaItemComponentBaseColor
	javaItemComponentPotDecorations
	javaItemComponentContainer
	javaItemComponentBlockState
	javaItemComponentBees
	javaItemComponentLock
	javaItemComponentContainerLoot
)

const javaItemComponentCount = javaItemComponentContainerLoot + 1

type javaItemComponentState struct {
	nbt               map[string]any
	metadata          uint32
	hasMetadata       bool
	customName        bool
	displayName       string
	lore              []string
	enchantments      []map[string]any
	glint             bool
	potionID          int32
	hasPotionID       bool
	fireworks         *JavaFireworksData
	fireworkExplosion *JavaFireworkExplosion
	unsupported       []int32
}

func decodeJavaItemComponents(r *javaprotocol.Reader, count int) (javaItemComponentState, error) {
	state := javaItemComponentState{}
	for i := 0; i < count; i++ {
		componentType, err := r.VarInt()
		if err != nil {
			return javaItemComponentState{}, fmt.Errorf("component %d type: %w", i, err)
		}
		if componentType < 0 || componentType >= javaItemComponentCount {
			// There is no length prefix on a component payload. An unknown
			// component is therefore a semantic skip for this fixed profile;
			// trying to guess its shape would turn valid future data into a
			// false wire decode.
			state.unsupported = append(state.unsupported, componentType)
			return state, nil
		}
		supported, err := decodeJavaItemComponent(r, componentType, &state)
		if err != nil {
			return javaItemComponentState{}, fmt.Errorf("component %d type %d: %w", i, componentType, err)
		}
		if !supported {
			state.unsupported = append(state.unsupported, componentType)
		}
	}
	state.finish()
	return state, nil
}

func (s *javaItemComponentState) finish() {
	if s.nbt == nil {
		s.nbt = make(map[string]any)
	}
	if s.displayName != "" || len(s.lore) != 0 {
		display, _ := s.nbt["display"].(map[string]any)
		if display == nil {
			display = make(map[string]any)
		}
		if s.displayName != "" {
			display["Name"] = s.displayName
		}
		if len(s.lore) != 0 {
			display["Lore"] = s.lore
		}
		s.nbt["display"] = display
	}
	if len(s.enchantments) != 0 {
		s.nbt["ench"] = s.enchantments
	} else if s.glint {
		// An empty compound list is the Bedrock convention used by Geyser
		// for a glint-only item.
		s.nbt["ench"] = []map[string]any{}
	}
	if len(s.nbt) == 0 {
		s.nbt = nil
	}
}

func (s *javaItemComponentState) put(name string, value any) {
	if s.nbt == nil {
		s.nbt = make(map[string]any)
	}
	s.nbt[name] = value
}

func (s *javaItemComponentState) mergeCustomData(value any) bool {
	compound, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if s.nbt == nil {
		s.nbt = make(map[string]any, len(compound))
	}
	for key, value := range compound {
		s.nbt[key] = value
	}
	return true
}

func (s *javaItemComponentState) setName(value any, custom bool) {
	name := JavaTextComponentText(value)
	if name == "" {
		return
	}
	if custom || !s.customName {
		s.displayName = name
	}
	if custom {
		s.customName = true
	}
}

func decodeJavaItemComponent(r *javaprotocol.Reader, componentType int32, state *javaItemComponentState) (bool, error) {
	switch componentType {
	case javaItemComponentCustomData:
		value, err := decodeJavaNBTValue(r)
		return state.mergeCustomData(value), err
	case javaItemComponentMaxStackSize, javaItemComponentMaxDamage:
		_, err := r.VarInt()
		return false, err
	case javaItemComponentDamage:
		value, err := r.VarInt()
		if err == nil && value >= 0 {
			state.metadata = uint32(value)
			state.hasMetadata = true
		}
		return err == nil && value >= 0, err
	case javaItemComponentUnbreakable:
		_, err := r.Bool()
		if err == nil {
			state.put("Unbreakable", byte(1))
		}
		return true, err
	case javaItemComponentCustomName, javaItemComponentItemName:
		value, err := decodeJavaNBTValue(r)
		if err != nil {
			return false, err
		}
		state.setName(value, componentType == javaItemComponentCustomName)
		return true, nil
	case javaItemComponentItemModel:
		_, err := r.String()
		return false, err
	case javaItemComponentLore:
		count, err := boundedJavaCount(r, "item lore count")
		if err != nil {
			return false, err
		}
		for i := 0; i < count; i++ {
			value, err := decodeJavaNBTValue(r)
			if err != nil {
				return false, fmt.Errorf("lore line %d: %w", i, err)
			}
			if value != nil {
				state.lore = append(state.lore, JavaTextComponentText(value))
			}
		}
		return true, nil
	case javaItemComponentRarity:
		_, err := r.VarInt()
		return false, err
	case javaItemComponentEnchantments, javaItemComponentStoredEnchantments:
		if err := decodeJavaEnchantments(r, state); err != nil {
			return false, err
		}
		return true, nil
	case javaItemComponentCanPlaceOn, javaItemComponentCanBreak:
		if err := decodeJavaBlockPredicates(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentAttributeModifiers:
		if err := decodeJavaAttributeModifiers(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentCustomModelData:
		if err := decodeJavaCustomModelData(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentHideAdditionalTooltip, javaItemComponentHideTooltip,
		javaItemComponentCreativeSlotLock, javaItemComponentGlider:
		return true, nil
	case javaItemComponentRepairCost:
		value, err := r.VarInt()
		if err == nil && value != 0 {
			state.put("RepairCost", value)
		}
		return true, err
	case javaItemComponentEnchantmentGlintOverride:
		value, err := r.Bool()
		if err == nil && value {
			state.glint = true
		}
		return true, err
	case javaItemComponentIntangibleProjectile, javaItemComponentMapDecorations,
		javaItemComponentRecipes, javaItemComponentDebugStickState,
		javaItemComponentEntityData, javaItemComponentBucketEntityData,
		javaItemComponentBlockEntityData, javaItemComponentLock,
		javaItemComponentContainerLoot:
		_, err := decodeJavaNBTValue(r)
		return true, err
	case javaItemComponentFood:
		if err := decodeJavaFood(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentConsumable:
		if err := decodeJavaConsumable(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentUseRemainder:
		if _, err := decodeJavaItemSlot(r, nil); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentUseCooldown:
		if err := decodeJavaUseCooldown(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentDamageResistant, javaItemComponentTooltipStyle, javaItemComponentNoteBlockSound:
		_, err := r.String()
		return false, err
	case javaItemComponentTool:
		if err := decodeJavaTool(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentEnchantable, javaItemComponentOminousBottleAmplifier,
		javaItemComponentMapPostProcessing, javaItemComponentBaseColor:
		_, err := r.VarInt()
		return false, err
	case javaItemComponentEquippable:
		if err := decodeJavaEquippable(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentRepairable:
		if err := decodeJavaIDSet(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentDeathProtection:
		if err := decodeJavaConsumeEffects(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentDyedColor:
		value, err := r.Int32()
		if err == nil {
			state.put("customColor", value)
		}
		if tooltip, tooltipErr := r.Bool(); err == nil {
			_ = tooltip
		} else {
			err = tooltipErr
		}
		return true, err
	case javaItemComponentMapColor:
		_, err := r.Int32()
		return false, err
	case javaItemComponentMapID:
		value, err := r.VarInt()
		if err == nil {
			state.put("map_uuid", int64(value))
			state.put("map_name_index", value)
			state.put("map_display_players", byte(1))
		}
		return true, err
	case javaItemComponentChargedProjectiles:
		if err := decodeJavaItemSlotArray(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentBundleContents, javaItemComponentContainer:
		if err := decodeJavaItemSlotArray(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentPotionContents:
		if err := decodeJavaPotionContents(r, state); err != nil {
			return false, err
		}
		return true, nil
	case javaItemComponentSuspiciousStewEffects:
		count, err := boundedJavaCount(r, "suspicious stew effect count")
		if err != nil {
			return false, err
		}
		for i := 0; i < count; i++ {
			if _, err := r.VarInt(); err != nil {
				return false, err
			}
			if _, err := r.VarInt(); err != nil {
				return false, err
			}
		}
		return false, nil
	case javaItemComponentWritableBookContent:
		if err := decodeJavaWritableBook(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentWrittenBookContent:
		if err := decodeJavaWrittenBook(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentTrim:
		if err := decodeJavaTrim(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentInstrument:
		if err := decodeJavaRegistryHolder(r, decodeJavaInstrumentData); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentJukeboxPlayable:
		if err := decodeJavaJukeboxPlayable(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentLodestoneTracker:
		if err := decodeJavaLodestoneTracker(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentFireworkExplosion:
		explosion, err := decodeJavaFireworkExplosion(r)
		if err != nil {
			return false, err
		}
		state.fireworkExplosion = &explosion
		return true, nil
	case javaItemComponentFireworks:
		fireworks, err := decodeJavaFireworks(r)
		if err != nil {
			return false, err
		}
		state.fireworks = &fireworks
		return true, nil
	case javaItemComponentProfile:
		if err := decodeJavaProfile(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentBannerPatterns:
		if err := decodeJavaBannerPatterns(r); err != nil {
			return false, err
		}
		return false, nil
	case javaItemComponentPotDecorations:
		count, err := boundedJavaCount(r, "pot decoration count")
		if err != nil {
			return false, err
		}
		for i := 0; i < count; i++ {
			if _, err := r.String(); err != nil {
				return false, err
			}
		}
		return false, nil
	case javaItemComponentBlockState:
		count, err := boundedJavaCount(r, "item block state property count")
		if err != nil {
			return false, err
		}
		for i := 0; i < count; i++ {
			if _, err := r.String(); err != nil {
				return false, err
			}
			if _, err := r.String(); err != nil {
				return false, err
			}
		}
		return false, nil
	case javaItemComponentBees:
		count, err := boundedJavaCount(r, "item bee count")
		if err != nil {
			return false, err
		}
		for i := 0; i < count; i++ {
			if _, err := decodeJavaNBTValue(r); err != nil {
				return false, err
			}
			if _, err := r.VarInt(); err != nil {
				return false, err
			}
			if _, err := r.VarInt(); err != nil {
				return false, err
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unhandled Java item component type %d", componentType)
	}
}

func decodeJavaEnchantments(r *javaprotocol.Reader, state *javaItemComponentState) error {
	count, err := boundedJavaCount(r, "enchantment count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		id, err := r.VarInt()
		if err != nil {
			return err
		}
		level, err := r.VarInt()
		if err != nil {
			return err
		}
		if bedrockID, ok := bedrockEnchantmentID(id); ok && level >= 0 && level <= 32767 {
			state.enchantments = append(state.enchantments, map[string]any{
				"id":  int16(bedrockID),
				"lvl": int16(level),
			})
		} else {
			// Java-only enchantments still need the Bedrock glint so the
			// item does not lose its most visible property.
			state.glint = true
		}
	}
	_, err = r.Bool() // showTooltip/showInTooltip
	return err
}

// Java's registry IDs are alphabetically ordered in minecraft-data, while
// Bedrock's enchantment IDs follow Geyser's BedrockEnchantment order.
func bedrockEnchantmentID(javaID int32) (int32, bool) {
	ids := [...]int32{
		8, 11, 27, 3, 40, 32, 39, 7, 15, 2,
		13, 1, 21, 18, 25, 29, 22, 12, 14, 31,
		23, 24, 26, 33, 34, 19, 4, 0, 20, 35,
		6, 30, 9, 16, 10, 17, -1, 37, 5, 38,
		28, 36,
	}
	if javaID < 0 || int(javaID) >= len(ids) || ids[javaID] < 0 {
		return 0, false
	}
	return ids[javaID], true
}

func decodeJavaBlockPredicates(r *javaprotocol.Reader) error {
	count, err := boundedJavaCount(r, "item block predicate count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if err := decodeJavaOptional(r, decodeJavaIDSet); err != nil {
			return err
		}
		if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
			propertyCount, err := boundedJavaCount(r, "item block property count")
			if err != nil {
				return err
			}
			for j := 0; j < propertyCount; j++ {
				if _, err := r.String(); err != nil {
					return err
				}
				exact, err := r.Bool()
				if err != nil {
					return err
				}
				if _, err := r.String(); err != nil {
					return err
				}
				if !exact {
					if _, err := r.String(); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
		if _, err := decodeJavaNBTValue(r); err != nil {
			return err
		}
	}
	_, err = r.Bool() // showTooltip
	return err
}

func decodeJavaAttributeModifiers(r *javaprotocol.Reader) error {
	count, err := boundedJavaCount(r, "item attribute modifier count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := r.VarInt(); err != nil {
			return err
		}
		if _, err := r.String(); err != nil {
			return err
		}
		if _, err := r.Float64(); err != nil {
			return err
		}
		if _, err := r.VarInt(); err != nil {
			return err
		}
		if _, err := r.VarInt(); err != nil {
			return err
		}
	}
	_, err = r.Bool()
	return err
}

func decodeJavaCustomModelData(r *javaprotocol.Reader) error {
	floatCount, err := boundedJavaCount(r, "item custom model float count")
	if err != nil {
		return err
	}
	for i := 0; i < floatCount; i++ {
		if _, err := r.Float32(); err != nil {
			return err
		}
	}
	flagCount, err := boundedJavaCount(r, "item custom model flag count")
	if err != nil {
		return err
	}
	for i := 0; i < flagCount; i++ {
		if _, err := r.Bool(); err != nil {
			return err
		}
	}
	stringCount, err := boundedJavaCount(r, "item custom model string count")
	if err != nil {
		return err
	}
	for i := 0; i < stringCount; i++ {
		if _, err := r.String(); err != nil {
			return err
		}
	}
	colorCount, err := boundedJavaCount(r, "item custom model color count")
	if err != nil {
		return err
	}
	for i := 0; i < colorCount; i++ {
		if _, err := r.Int32(); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaFood(r *javaprotocol.Reader) error {
	if _, err := r.VarInt(); err != nil {
		return err
	}
	if _, err := r.Float32(); err != nil {
		return err
	}
	_, err := r.Bool()
	return err
}

func decodeJavaConsumable(r *javaprotocol.Reader) error {
	if _, err := r.Float32(); err != nil {
		return err
	}
	animation, err := r.VarInt()
	if err != nil {
		return err
	}
	if animation < 0 || animation > 9 {
		return fmt.Errorf("invalid consumable animation %d", animation)
	}
	if err := decodeJavaItemSoundHolder(r); err != nil {
		return err
	}
	if _, err := r.Bool(); err != nil {
		return err
	}
	return decodeJavaConsumeEffects(r)
}

func decodeJavaUseCooldown(r *javaprotocol.Reader) error {
	if _, err := r.Float32(); err != nil {
		return err
	}
	return decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.String()
		return err
	})
}

func decodeJavaTool(r *javaprotocol.Reader) error {
	count, err := boundedJavaCount(r, "item tool rule count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if err := decodeJavaIDSet(r); err != nil {
			return err
		}
		if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
			_, err := r.Float32()
			return err
		}); err != nil {
			return err
		}
		if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
			_, err := r.Bool()
			return err
		}); err != nil {
			return err
		}
	}
	if _, err := r.Float32(); err != nil {
		return err
	}
	_, err = r.VarInt()
	return err
}

func decodeJavaEquippable(r *javaprotocol.Reader) error {
	if _, err := r.VarInt(); err != nil {
		return err
	}
	if err := decodeJavaItemSoundHolder(r); err != nil {
		return err
	}
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.String()
		return err
	}); err != nil {
		return err
	}
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.String()
		return err
	}); err != nil {
		return err
	}
	if err := decodeJavaOptional(r, decodeJavaIDSet); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		if _, err := r.Bool(); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaConsumeEffects(r *javaprotocol.Reader) error {
	count, err := boundedJavaCount(r, "item consume effect count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if err := decodeJavaConsumeEffect(r); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaConsumeEffect(r *javaprotocol.Reader) error {
	effectType, err := r.VarInt()
	if err != nil {
		return err
	}
	switch effectType {
	case 0:
		count, err := boundedJavaCount(r, "apply-effects potion count")
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			if err := decodeJavaPotionEffect(r); err != nil {
				return err
			}
		}
		_, err = r.Float32()
		return err
	case 1:
		return decodeJavaIDSet(r)
	case 2:
		return nil
	case 3:
		_, err = r.Float32()
		return err
	case 4:
		return decodeJavaItemSoundHolder(r)
	default:
		return fmt.Errorf("invalid item consume effect type %d", effectType)
	}
}

func decodeJavaPotionEffect(r *javaprotocol.Reader) error {
	if _, err := r.VarInt(); err != nil {
		return err
	}
	return decodeJavaEffectDetail(r, 0)
}

func decodeJavaEffectDetail(r *javaprotocol.Reader, depth int) error {
	if depth > 32 {
		return fmt.Errorf("item effect detail nesting exceeds 32")
	}
	if _, err := r.VarInt(); err != nil {
		return err
	}
	if _, err := r.VarInt(); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		if _, err := r.Bool(); err != nil {
			return err
		}
	}
	return decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		return decodeJavaEffectDetail(r, depth+1)
	})
}

func decodeJavaItemSoundHolder(r *javaprotocol.Reader) error {
	return decodeJavaRegistryHolder(r, func(r *javaprotocol.Reader) error {
		if _, err := r.String(); err != nil {
			return err
		}
		return decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
			_, err := r.Float32()
			return err
		})
	})
}

func decodeJavaRegistryHolder(r *javaprotocol.Reader, custom func(*javaprotocol.Reader) error) error {
	selector, err := r.VarInt()
	if err != nil {
		return err
	}
	if selector < 0 {
		return fmt.Errorf("negative registry holder selector %d", selector)
	}
	if selector == 0 {
		return custom(r)
	}
	return nil
}

func decodeJavaIDSet(r *javaprotocol.Reader) error {
	selector, err := r.VarInt()
	if err != nil {
		return err
	}
	if selector < 0 {
		return fmt.Errorf("negative holder set selector %d", selector)
	}
	if selector == 0 {
		_, err := r.String()
		return err
	}
	count := selector - 1
	if count > int32(maxJavaCollectionSize) {
		return fmt.Errorf("holder set count %d exceeds limit %d", count, maxJavaCollectionSize)
	}
	for i := int32(0); i < count; i++ {
		if _, err := r.VarInt(); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaItemSlotArray(r *javaprotocol.Reader) error {
	count, err := boundedJavaCount(r, "nested item count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := decodeJavaItemSlot(r, nil); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaPotionContents(r *javaprotocol.Reader, state *javaItemComponentState) error {
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		potionID, err := r.VarInt()
		if err == nil {
			state.potionID = potionID
			state.hasPotionID = true
		}
		return err
	}); err != nil {
		return err
	}
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.Int32()
		return err
	}); err != nil {
		return err
	}
	count, err := boundedJavaCount(r, "potion custom effect count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if err := decodeJavaPotionEffect(r); err != nil {
			return err
		}
	}
	return decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.String()
		return err
	})
}

func decodeJavaWritableBook(r *javaprotocol.Reader) error {
	count, err := boundedJavaCount(r, "writable book page count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := r.String(); err != nil {
			return err
		}
		if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
			_, err := r.String()
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaWrittenBook(r *javaprotocol.Reader) error {
	if _, err := r.String(); err != nil {
		return err
	}
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.String()
		return err
	}); err != nil {
		return err
	}
	if _, err := r.String(); err != nil {
		return err
	}
	if _, err := r.VarInt(); err != nil {
		return err
	}
	count, err := boundedJavaCount(r, "written book page count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := decodeJavaNBTValue(r); err != nil {
			return err
		}
		if _, err := decodeJavaNBTValue(r); err != nil {
			return err
		}
	}
	_, err = r.Bool()
	return err
}

func decodeJavaTrim(r *javaprotocol.Reader) error {
	if err := decodeJavaRegistryHolder(r, decodeJavaTrimMaterial); err != nil {
		return err
	}
	if err := decodeJavaRegistryHolder(r, decodeJavaTrimPattern); err != nil {
		return err
	}
	_, err := r.Bool()
	return err
}

func decodeJavaTrimMaterial(r *javaprotocol.Reader) error {
	if _, err := r.String(); err != nil {
		return err
	}
	if _, err := r.VarInt(); err != nil {
		return err
	}
	count, err := boundedJavaCount(r, "trim material override count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := r.String(); err != nil {
			return err
		}
		if _, err := r.String(); err != nil {
			return err
		}
	}
	_, err = decodeJavaNBTValue(r)
	return err
}

func decodeJavaTrimPattern(r *javaprotocol.Reader) error {
	if _, err := r.String(); err != nil {
		return err
	}
	if _, err := r.VarInt(); err != nil {
		return err
	}
	if _, err := decodeJavaNBTValue(r); err != nil {
		return err
	}
	_, err := r.Bool()
	return err
}

func decodeJavaInstrumentData(r *javaprotocol.Reader) error {
	if err := decodeJavaItemSoundHolder(r); err != nil {
		return err
	}
	if _, err := r.Float32(); err != nil {
		return err
	}
	if _, err := r.Float32(); err != nil {
		return err
	}
	_, err := decodeJavaNBTValue(r)
	return err
}

func decodeJavaJukeboxPlayable(r *javaprotocol.Reader) error {
	hasHolder, err := r.Bool()
	if err != nil {
		return err
	}
	if hasHolder {
		if err := decodeJavaRegistryHolder(r, decodeJavaJukeboxSongData); err != nil {
			return err
		}
	} else if _, err := r.String(); err != nil {
		return err
	}
	_, err = r.Bool()
	return err
}

func decodeJavaJukeboxSongData(r *javaprotocol.Reader) error {
	if err := decodeJavaItemSoundHolder(r); err != nil {
		return err
	}
	if _, err := decodeJavaNBTValue(r); err != nil {
		return err
	}
	if _, err := r.Float32(); err != nil {
		return err
	}
	_, err := r.VarInt()
	return err
}

func decodeJavaLodestoneTracker(r *javaprotocol.Reader) error {
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		if _, err := r.String(); err != nil {
			return err
		}
		_, err := r.Int64() // packed Java BlockPos
		return err
	}); err != nil {
		return err
	}
	_, err := r.Bool()
	return err
}

func decodeJavaFireworkExplosion(r *javaprotocol.Reader) (JavaFireworkExplosion, error) {
	shape, err := r.VarInt()
	if err != nil {
		return JavaFireworkExplosion{}, err
	}
	if shape < 0 || shape > 4 {
		return JavaFireworkExplosion{}, fmt.Errorf("invalid firework explosion shape %d", shape)
	}
	explosion := JavaFireworkExplosion{Shape: shape}
	for i := 0; i < 2; i++ {
		count, err := boundedJavaCount(r, "firework color count")
		if err != nil {
			return JavaFireworkExplosion{}, err
		}
		colors := make([]int32, count)
		for j := 0; j < count; j++ {
			color, err := r.Int32()
			if err != nil {
				return JavaFireworkExplosion{}, err
			}
			colors[j] = color
		}
		if i == 0 {
			explosion.Colors = colors
		} else {
			explosion.FadeColors = colors
		}
	}
	explosion.Flicker, err = r.Bool()
	if err != nil {
		return JavaFireworkExplosion{}, err
	}
	explosion.Trail, err = r.Bool()
	if err != nil {
		return JavaFireworkExplosion{}, err
	}
	return explosion, nil
}

func decodeJavaFireworks(r *javaprotocol.Reader) (JavaFireworksData, error) {
	flightDuration, err := r.VarInt()
	if err != nil {
		return JavaFireworksData{}, err
	}
	count, err := boundedJavaCount(r, "firework explosion count")

	if err != nil {
		return JavaFireworksData{}, err
	}
	explosions := make([]JavaFireworkExplosion, count)
	for i := 0; i < count; i++ {
		explosions[i], err = decodeJavaFireworkExplosion(r)
		if err != nil {
			return JavaFireworksData{}, err
		}
	}
	return JavaFireworksData{FlightDuration: flightDuration, Explosions: explosions}, nil
}

func decodeJavaProfile(r *javaprotocol.Reader) error {
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.String()
		return err
	}); err != nil {
		return err
	}
	if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
		_, err := r.Bytes(16)
		return err
	}); err != nil {
		return err
	}
	count, err := boundedJavaCount(r, "profile property count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := r.String(); err != nil {
			return err
		}
		if _, err := r.String(); err != nil {
			return err
		}
		if err := decodeJavaOptional(r, func(r *javaprotocol.Reader) error {
			_, err := r.String()
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaBannerPatterns(r *javaprotocol.Reader) error {
	count, err := boundedJavaCount(r, "banner pattern count")
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if err := decodeJavaRegistryHolder(r, decodeJavaBannerPattern); err != nil {
			return err
		}
		if _, err := r.VarInt(); err != nil {
			return err
		}
	}
	return nil
}

func decodeJavaBannerPattern(r *javaprotocol.Reader) error {
	if _, err := r.String(); err != nil {
		return err
	}
	_, err := r.String()
	return err
}

func decodeJavaOptional(r *javaprotocol.Reader, decode func(*javaprotocol.Reader) error) error {
	present, err := r.Bool()
	if err != nil {
		return err
	}
	if !present {
		return nil
	}
	return decode(r)
}
