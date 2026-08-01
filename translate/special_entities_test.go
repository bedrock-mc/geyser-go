package translate

import (
	"math"
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestBedrockEntityTypeOverrides(t *testing.T) {
	cases := map[string]string{
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
