package translate

import (
	"math"
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestBedrockEntityTypeOverrides(t *testing.T) {
	cases := map[string]string{
		"minecraft:oak_boat":          "minecraft:boat",
		"minecraft:oak_chest_boat":    "minecraft:chest_boat",
		"minecraft:chest_minecart":    "minecraft:minecart",
		"minecraft:end_crystal":       "minecraft:ender_crystal",
		"minecraft:ender_crystal":     "minecraft:ender_crystal",
		"minecraft:evoker_fangs":      "minecraft:evocation_fang",
		"minecraft:experience_bottle": "minecraft:xp_bottle",
		"minecraft:eye_of_ender":      "minecraft:eye_of_ender_signal",
		"minecraft:firework_rocket":   "minecraft:fireworks_rocket",
		"minecraft:fishing_bobber":    "minecraft:fishing_hook",
		"minecraft:trident":           "minecraft:thrown_trident",
		"minecraft:text_display":      "minecraft:armor_stand",
		"minecraft:interaction":       "minecraft:armor_stand",
		"minecraft:villager":          "minecraft:villager_v2",
	}
	for javaType, want := range cases {
		if got := bedrockEntityType(javaType); got != want {
			t.Errorf("bedrockEntityType(%q) = %q, want %q", javaType, got, want)
		}
	}
	if got := bedrockEntityType("minecraft:pig"); got != "minecraft:pig" {
		t.Fatalf("ordinary entity identifier changed to %q", got)
	}
	if got := bedrockEntityType("minecraft:lingering_potion"); got != "minecraft:splash_potion" {
		t.Fatalf("lingering-potion Bedrock identifier = %q", got)
	}
	if got := bedrockEntityType("minecraft:potion"); got != "minecraft:splash_potion" {
		t.Fatalf("potion Bedrock identifier = %q", got)
	}
}

func TestJavaEntitySpawnProjectionDefaults(t *testing.T) {
	cloud, ok := javaSpawnEntityProjection("minecraft:area_effect_cloud", 0)
	if !ok || !cloud.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagFireImmune) || cloud[gtprotocol.EntityDataKeyDataDuration] != int32(2147483647) || cloud[gtprotocol.EntityDataKeyDataRadius] != float32(3) {
		t.Fatalf("area-cloud spawn metadata = %#v, ok=%t", cloud, ok)
	}

	crystal, ok := javaSpawnEntityProjection("minecraft:ender_crystal", 0)
	if !ok || !crystal.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagFireImmune) || crystal[gtprotocol.EntityDataKeyBlockTarget] != (gtprotocol.BlockPos{}) {
		t.Fatalf("end-crystal spawn metadata = %#v, ok=%t", crystal, ok)
	}

	textDisplay, ok := javaSpawnEntityProjection("minecraft:text_display", 0)
	if !ok || textDisplay[gtprotocol.EntityDataKeyScale] != float32(0) || textDisplay[gtprotocol.EntityDataKeyAlwaysShowNameTag] != byte(1) {
		t.Fatalf("text-display spawn metadata = %#v, ok=%t", textDisplay, ok)
	}
	if hitbox, ok := textDisplay[gtprotocol.EntityDataKeyHitBox].(map[string]any); !ok || len(hitbox) != 0 {
		t.Fatalf("text-display hitbox metadata = %#v", textDisplay[gtprotocol.EntityDataKeyHitBox])
	}

	interaction, ok := javaSpawnEntityProjection("minecraft:interaction", 0)
	if !ok || !interaction.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagInvisible) || interaction[gtprotocol.EntityDataKeyWidth] != float32(1) || interaction[gtprotocol.EntityDataKeyHeight] != float32(1) {
		t.Fatalf("interaction spawn metadata = %#v, ok=%t", interaction, ok)
	}
	boat, ok := javaSpawnEntityProjection("minecraft:pale_oak_boat", 0)
	if !ok || boat[gtprotocol.EntityDataKeyVariant] != int32(9) || boat[gtprotocol.EntityDataKeyIsBuoyant] != byte(1) || boat[gtprotocol.EntityDataKeyBuoyancyData] != javaBoatBuoyancyData || !boat.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagCollidable-64) {
		t.Fatalf("boat spawn metadata = %#v, ok=%t", boat, ok)
	}
	for _, entityType := range []string{
		"minecraft:egg", "minecraft:ender_pearl", "minecraft:experience_bottle",
		"minecraft:splash_potion", "minecraft:lingering_potion", "minecraft:snowball",
		"minecraft:xp_bottle", "minecraft:potion",
	} {
		projectile, ok := javaSpawnEntityProjection(entityType, 0)
		if !ok || projectile[gtprotocol.EntityDataKeyScale] != float32(0.5) || !projectile.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagInvisible) {
			t.Fatalf("throwable projectile %s spawn metadata = %#v, ok=%t", entityType, projectile, ok)
		}
	}
	eye, ok := javaSpawnEntityProjection("minecraft:eye_of_ender_signal", 0)
	if !ok || eye[gtprotocol.EntityDataKeyScale] != float32(0.5) || eye.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagInvisible) {
		t.Fatalf("eye-of-ender spawn metadata = %#v, ok=%t", eye, ok)
	}

	if got := javaEntitySpawnPosition("minecraft:leash_knot", mgl32.Vec3{10, 20, 30}); got != (mgl32.Vec3{10.5, 20.25, 30.5}) {
		t.Fatalf("leash-knot position = %v", got)
	}
}

func TestTranslateSpecialEntityMetadata(t *testing.T) {
	crystal := translateSpecialEntityMetadata("minecraft:ender_crystal", []JavaEntityMetadataEntry{
		{Index: 8, Type: 11, Value: gtprotocol.BlockPos{1, 64, -2}},
		{Index: 9, Type: 8, Value: true},
	})
	if crystal[gtprotocol.EntityDataKeyBlockTarget] != (gtprotocol.BlockPos{1, 64, -2}) || !crystal.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagShowBottom) {
		t.Fatalf("end-crystal metadata = %#v", crystal)
	}

	cloud := translateSpecialEntityMetadata("minecraft:area_effect_cloud", []JavaEntityMetadataEntry{{Index: 8, Type: 3, Value: float32(100)}})
	if cloud[gtprotocol.EntityDataKeyDataRadius] != float32(32) {
		t.Fatalf("area-cloud radius metadata = %#v", cloud)
	}

	tnt := translateSpecialEntityMetadata("minecraft:tnt", []JavaEntityMetadataEntry{{Index: 8, Type: 1, Value: int32(80)}})
	if tnt[gtprotocol.EntityDataKeyFuseTime] != int32(80) || !tnt.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagIgnited) {
		t.Fatalf("TNT metadata = %#v", tnt)
	}

	creeper := translateSpecialEntityMetadata("minecraft:creeper", []JavaEntityMetadataEntry{
		{Index: 16, Type: 1, Value: int32(1)},
		{Index: 17, Type: 8, Value: true},
		{Index: 18, Type: 8, Value: true},
	})
	if !creeper.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagIgnited) || !creeper.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagPowered) {
		t.Fatalf("creeper metadata = %#v", creeper)
	}
	clearedCreeper := translateSpecialEntityMetadata("minecraft:creeper", []JavaEntityMetadataEntry{
		{Index: 16, Type: 1, Value: int32(-1)},
		{Index: 17, Type: 8, Value: false},
		{Index: 18, Type: 8, Value: false},
	})
	if clearedCreeper.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagIgnited) || clearedCreeper.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagPowered) {
		t.Fatalf("cleared creeper metadata = %#v", clearedCreeper)
	}

	sheep := translateSpecialEntityMetadata("minecraft:sheep", []JavaEntityMetadataEntry{
		{Index: 16, Type: 8, Value: true},
		{Index: 17, Type: 0, Value: int8(0x15)},
	})
	if !sheep.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagBaby) || !sheep.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagSheared) || sheep[gtprotocol.EntityDataKeyColorIndex] != byte(5) {
		t.Fatalf("sheep metadata = %#v", sheep)
	}

	armorStand := translateSpecialEntityMetadata("minecraft:armor_stand", []JavaEntityMetadataEntry{{Index: 15, Type: 0, Value: int8(0x09)}})
	if !armorStand.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagBaby) || !armorStand.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagAngry) || !armorStand.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagAdmiring-64) {
		t.Fatalf("armor-stand metadata = %#v", armorStand)
	}

	cat := translateSpecialEntityMetadata("minecraft:cat", []JavaEntityMetadataEntry{
		{Index: 17, Type: 0, Value: int8(0x07)},
		{Index: 19, Type: 1, Value: int32(0)},
		{Index: 20, Type: 8, Value: true},
		{Index: 22, Type: 1, Value: int32(14)},
	})
	if !cat.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagSitting) || !cat.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagAngry) || !cat.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagTamed) || !cat.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagResting) || cat[gtprotocol.EntityDataKeyVariant] != int32(8) || cat[gtprotocol.EntityDataKeyColorIndex] != byte(14) {
		t.Fatalf("cat metadata = %#v", cat)
	}

	wolf := translateSpecialEntityMetadata("minecraft:wolf", []JavaEntityMetadataEntry{{Index: 22, Type: 23, Value: JavaWolfVariant{RegistryID: 3}}})
	if wolf[gtprotocol.EntityDataKeyVariant] != int32(0) {
		t.Fatalf("wolf metadata = %#v", wolf)
	}

	fox := translateSpecialEntityMetadata("minecraft:fox", []JavaEntityMetadataEntry{
		{Index: 17, Type: 1, Value: int32(1)},
		{Index: 18, Type: 0, Value: int8(0x2d)},
	})
	for _, flag := range []uint8{gtprotocol.EntityDataFlagSitting, gtprotocol.EntityDataFlagSneaking, gtprotocol.EntityDataFlagInterested} {
		if !fox.Flag(gtprotocol.EntityDataKeyFlags, flag) {
			t.Fatalf("fox metadata missing flag %d: %#v", flag, fox)
		}
	}
	if !fox.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagSleeping-64) {
		t.Fatalf("fox metadata missing sleeping flag: %#v", fox)
	}

	rabbit := translateSpecialEntityMetadata("minecraft:rabbit", []JavaEntityMetadataEntry{{Index: 17, Type: 1, Value: int32(99)}})
	if rabbit[gtprotocol.EntityDataKeyVariant] != int32(1) || !rabbit.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagBribed) {
		t.Fatalf("killer-rabbit metadata = %#v", rabbit)
	}

	bee := translateSpecialEntityMetadata("minecraft:bee", []JavaEntityMetadataEntry{
		{Index: 17, Type: 0, Value: int8(0x04)},
		{Index: 18, Type: 1, Value: int32(20)},
	})
	if bee[gtprotocol.EntityDataKeyMarkVariant] != int32(1) || !bee.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagAngry) {
		t.Fatalf("bee metadata = %#v", bee)
	}

	fish := translateSpecialEntityMetadata("minecraft:tropicalfish", []JavaEntityMetadataEntry{{Index: 17, Type: 1, Value: int32(0x0f030201)}})
	if fish[gtprotocol.EntityDataKeyVariant] != int32(1) || fish[gtprotocol.EntityDataKeyMarkVariant] != int32(2) || fish[gtprotocol.EntityDataKeyColorIndex] != byte(3) || fish[gtprotocol.EntityDataKeyColorTwoIndex] != byte(15) {
		t.Fatalf("tropical-fish metadata = %#v", fish)
	}

	sounds := javaLightningSounds(mgl32.Vec3{1, 2, 3})
	if len(sounds) != 2 || sounds[0].SoundName != "ambient.weather.thunder" || sounds[1].SoundName != "ambient.weather.lightning.impact" || sounds[0].Volume != 10000 || sounds[1].Volume != 2 || sounds[0].Pitch < 0.8 || sounds[0].Pitch >= 1 || sounds[1].Pitch < 0.5 || sounds[1].Pitch >= 0.7 {
		t.Fatalf("lightning sounds = %#v", sounds)
	}

	spectral, ok := javaSpawnEntityProjection("minecraft:spectral_arrow", 0)
	if !ok || !spectral.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagBribed) {
		t.Fatalf("spectral-arrow spawn metadata = %#v, ok=%t", spectral, ok)
	}

	arrow := translateSpecialEntityMetadata("minecraft:arrow", []JavaEntityMetadataEntry{
		{Index: 8, Type: 0, Value: int8(1)},
		{Index: 11, Type: 1, Value: int32(16762624)},
	})
	if !arrow.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagCritical) || arrow[gtprotocol.EntityDataKeyCustomDisplay] != byte(32) {
		t.Fatalf("arrow metadata = %#v", arrow)
	}

	trident := translateSpecialEntityMetadata("minecraft:trident", []JavaEntityMetadataEntry{
		{Index: 8, Type: 0, Value: int8(1)},
		{Index: 12, Type: 8, Value: true},
	})
	if !trident.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagCritical) || !trident.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagEnchanted) {
		t.Fatalf("trident metadata = %#v", trident)
	}
	if tippedArrowDisplayID(123456789) != 0 || tippedArrowDisplayID(-1) != 0 {
		t.Fatal("unknown tipped-arrow colors should use no display variant")
	}

	textDisplay := translateSpecialEntityMetadata("minecraft:text_display", []JavaEntityMetadataEntry{{Index: 23, Type: 5, Value: map[string]any{
		"text":  "line one",
		"extra": []any{"\nline two"},
	}}})
	if textDisplay[gtprotocol.EntityDataKeyName] != "line one\nline two" {
		t.Fatalf("text-display name metadata = %#v", textDisplay)
	}
	interaction := translateSpecialEntityMetadata("minecraft:interaction", []JavaEntityMetadataEntry{
		{Index: 8, Type: 3, Value: float32(2.5)},
		{Index: 9, Type: 3, Value: float32(100)},
	})
	if interaction[gtprotocol.EntityDataKeyWidth] != float32(2.5) || interaction[gtprotocol.EntityDataKeyHeight] != float32(64) {
		t.Fatalf("interaction metadata = %#v", interaction)
	}
	if math.Abs(float64(javaTextDisplayLineOffset("one")-float32(-0.4586))) > 0.00001 || math.Abs(float64(javaTextDisplayLineOffset("one\ntwo")-float32(-0.3172))) > 0.00001 || javaTextDisplayLineOffset("") != 0 {
		t.Fatalf("text-display line offsets are incorrect")
	}
}

func TestTranslateEntityMetadataMatrix(t *testing.T) {
	noAI := translateSpecialEntityMetadata("minecraft:zombie", []JavaEntityMetadataEntry{{Index: 15, Type: 0, Value: int8(1)}})
	if !noAI.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagNoAI) {
		t.Fatalf("mob flags = %#v", noAI)
	}

	axolotl := translateSpecialEntityMetadata("minecraft:axolotl", []JavaEntityMetadataEntry{
		{Index: 17, Type: 1, Value: int32(4)},
		{Index: 18, Type: 8, Value: true},
	})
	if axolotl[gtprotocol.EntityDataKeyVariant] != int32(4) || !axolotl.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagPlayingDead-64) {
		t.Fatalf("axolotl metadata = %#v", axolotl)
	}

	frog := translateSpecialEntityMetadata("minecraft:frog", []JavaEntityMetadataEntry{
		{Index: 6, Type: 21, Value: int32(8)},
		{Index: 17, Type: 24, Value: int32(2)},
	})
	if frog[gtprotocol.EntityDataKeyVariant] != int32(1) || !frog.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagCroaking-64) {
		t.Fatalf("frog metadata = %#v", frog)
	}

	horse := translateSpecialEntityMetadata("minecraft:horse", []JavaEntityMetadataEntry{
		{Index: 17, Type: 0, Value: int8(0x72)},
		{Index: 18, Type: 1, Value: int32(0x0302)},
	})
	if !horse.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagTamed) || !horse.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagEating) || !horse.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagStanding) || horse[gtprotocol.EntityDataKeyVariant] != int32(2) || horse[gtprotocol.EntityDataKeyMarkVariant] != int32(3) || horse[gtprotocol.EntityDataKeyContainerType] != byte(gtprotocol.ContainerTypeHorse) {
		t.Fatalf("horse metadata = %#v", horse)
	}
	if horse.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagSaddled) || horse.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagPowerJump) {
		t.Fatalf("horse metadata incorrectly derives saddle state from horse flags: %#v", horse)
	}

	camel := translateSpecialEntityMetadata("minecraft:camel", []JavaEntityMetadataEntry{
		{Index: 17, Type: 0, Value: int8(0x30)},
		{Index: 18, Type: 8, Value: true},
	})
	if !camel.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagTamed) || !camel.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagEating) || !camel.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagStanding) || !camel.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagHasDashTimeout-64) || camel[gtprotocol.EntityDataKeyContainerType] != byte(gtprotocol.ContainerTypeHorse) {
		t.Fatalf("camel metadata = %#v", camel)
	}

	mooshroom := translateSpecialEntityMetadata("minecraft:mooshroom", []JavaEntityMetadataEntry{{Index: 17, Type: 4, Value: "brown"}})
	if mooshroom[gtprotocol.EntityDataKeyVariant] != int32(1) {
		t.Fatalf("mooshroom metadata = %#v", mooshroom)
	}

	strider := translateSpecialEntityMetadata("minecraft:strider", []JavaEntityMetadataEntry{
		{Index: 18, Type: 8, Value: true},
		{Index: 19, Type: 8, Value: true},
	})
	if strider.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagBreathing) || !strider.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagShaking) || !strider.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagSaddled) {
		t.Fatalf("strider metadata = %#v", strider)
	}

	shulker := translateSpecialEntityMetadata("minecraft:shulker", []JavaEntityMetadataEntry{
		{Index: 16, Type: 12, Value: int32(2)},
		{Index: 17, Type: 0, Value: int8(5)},
		{Index: 18, Type: 0, Value: int8(14)},
	})
	if shulker[gtprotocol.EntityDataKeyAttachFace] != int32(2) || shulker[gtprotocol.EntityDataKeyPeekID] != int32(5) || shulker[gtprotocol.EntityDataKeyVariant] != int32(1) {
		t.Fatalf("shulker metadata = %#v", shulker)
	}

	sniffer := translateSpecialEntityMetadata("minecraft:sniffer", []JavaEntityMetadataEntry{{Index: 17, Type: 27, Value: int32(5)}})
	if !sniffer.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagDigging-64) {
		t.Fatalf("sniffer metadata = %#v", sniffer)
	}

	wither := translateSpecialEntityMetadata("minecraft:wither", []JavaEntityMetadataEntry{{Index: 19, Type: 1, Value: int32(200)}})
	if wither[gtprotocol.EntityDataKeyInvulnerableTicks] != int32(200) || wither[gtprotocol.EntityDataKeyAerialAttack] != int16(0) {
		t.Fatalf("wither metadata = %#v", wither)
	}

	boat := translateSpecialEntityMetadata("minecraft:oak_boat", []JavaEntityMetadataEntry{
		{Index: 8, Type: 1, Value: int32(4)},
		{Index: 9, Type: 1, Value: int32(2)},
		{Index: 10, Type: 3, Value: float32(5)},
		{Index: 11, Type: 8, Value: true},
		{Index: 12, Type: 8, Value: false},
		{Index: 13, Type: 1, Value: int32(7)},
	})
	if boat[gtprotocol.EntityDataKeyHurt] != int32(4) || boat[gtprotocol.EntityDataKeyHurtDirection] != int32(2) || boat[gtprotocol.EntityDataKeyStructuralIntegrity] != int32(35) || boat[gtprotocol.EntityDataKeyRowTimeLeft] != float32(0.04) || boat[gtprotocol.EntityDataKeyRowTimeRight] != float32(0) || boat[gtprotocol.EntityDataKeyBubbleTime] != int32(7) {
		t.Fatalf("boat metadata = %#v", boat)
	}

	minecart := translateSpecialEntityMetadata("minecraft:minecart", []JavaEntityMetadataEntry{
		{Index: 8, Type: 1, Value: int32(3)},
		{Index: 9, Type: 1, Value: int32(1)},
		{Index: 10, Type: 3, Value: float32(20)},
		{Index: 11, Type: 1, Value: int32(1)},
		{Index: 12, Type: 1, Value: int32(6)},
		{Index: 13, Type: 8, Value: true},
	})
	blockRuntime, known := JavaBlockRuntimeID(1)
	if !known || minecart[gtprotocol.EntityDataKeyStructuralIntegrity] != int32(3) || minecart[gtprotocol.EntityDataKeyHurtDirection] != int32(1) || minecart[gtprotocol.EntityDataKeyHurt] != int32(15) || minecart[gtprotocol.EntityDataKeyDisplayTileRuntimeID] != int32(blockRuntime) || minecart[gtprotocol.EntityDataKeyDisplayOffset] != int32(6) || minecart[gtprotocol.EntityDataKeyCustomDisplay] != byte(1) {
		t.Fatalf("minecart metadata = %#v", minecart)
	}
}

func TestJavaSpawnEntityProjection(t *testing.T) {
	xp, ok := javaSpawnEntityProjection("minecraft:xp_orb", 17)
	if !ok || xp[gtprotocol.EntityDataKeyTradeExperience] != int32(1) {
		t.Fatalf("experience-orb metadata = %#v, ok=%t", xp, ok)
	}

	falling, ok := javaSpawnEntityProjection("minecraft:falling_block", 1)
	if !ok {
		t.Fatal("known falling-block state was rejected")
	}
	want, known := JavaBlockRuntimeID(1)
	if !known || falling[gtprotocol.EntityDataKeyDisplayTileRuntimeID] != int32(want) {
		t.Fatalf("falling-block metadata = %#v, want runtime %d", falling, want)
	}

	unknown, ok := javaSpawnEntityProjection("minecraft:falling_block", data.Java1214BlockStateCount)
	if ok || unknown[gtprotocol.EntityDataKeyDisplayTileRuntimeID] != int32(data.Java1214ToBedrock[0]) {
		t.Fatalf("unknown falling-block projection = %#v, ok=%t", unknown, ok)
	}

	hook, ok := javaSpawnEntityProjection("minecraft:fishing_hook", 42)
	if !ok || hook[gtprotocol.EntityDataKeyOwner] != int64(42) {
		t.Fatalf("fishing-hook metadata = %#v, ok=%t", hook, ok)
	}
	if _, ok := javaSpawnEntityProjection("minecraft:fishing_hook", -1); ok {
		t.Fatal("ownerless fishing hook was projected")
	}
}

func TestFishingHookTargetProjection(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	b.entities[17] = &javaEntityState{runtimeID: 117}
	metadata := gtprotocol.NewEntityMetadata()
	b.mu.Lock()
	b.translateEntityTargetMetadataLocked("minecraft:fishing_bobber", []JavaEntityMetadataEntry{{Index: 8, Type: 1, Value: int32(18)}}, metadata)
	b.mu.Unlock()
	if got := metadata[gtprotocol.EntityDataKeyTarget]; got != int64(117) {
		t.Fatalf("hook target runtime ID = %#v, want 117", got)
	}

	metadata = gtprotocol.NewEntityMetadata()
	b.mu.Lock()
	b.translateEntityTargetMetadataLocked("minecraft:fishing_bobber", []JavaEntityMetadataEntry{{Index: 8, Type: 1, Value: int32(0)}}, metadata)
	b.mu.Unlock()
	if got := metadata[gtprotocol.EntityDataKeyTarget]; got != int64(0) {
		t.Fatalf("cleared hook target runtime ID = %#v, want 0", got)
	}
}

func TestThrowableProjectileVisibilityReveal(t *testing.T) {
	metadata, ok := javaSpawnEntityProjection("minecraft:egg", 0)
	if !ok {
		t.Fatal("egg projection rejected")
	}
	entity := &javaEntityState{entityType: "minecraft:egg", metadata: metadata, projectileHidden: true}
	visibility := revealJavaProjectileLocked(entity)
	if entity.projectileHidden || visibility == nil || visibility.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagInvisible) {
		t.Fatalf("projectile reveal state = hidden=%t metadata=%#v", entity.projectileHidden, visibility)
	}
	if metadata.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagInvisible) {
		t.Fatalf("stored projectile metadata kept invisible bit: %#v", metadata)
	}
}
