package data

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCloudburstAndGenerate(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "runtime_item_states.json"), []byte(`[
  {"name":"minecraft:z_item","id":2,"version":2,"componentBased":true},
  {"name":"minecraft:a_item","id":1,"version":1,"componentBased":false}
]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "block_properties.json"), []byte(`[
  {"name":"minecraft:z_block","blockStateHash":2,"translationKey":"tile.z","isSolid":true,"hardness":1.0,"explosionResistance":2.0,"friction":0.6,"requiresCorrectToolForDrops":true,"lightEmission":0},
  {"name":"minecraft:a_block","blockStateHash":1,"translationKey":"tile.a","isSolid":false,"hardness":0,"explosionResistance":0,"friction":0.6,"requiresCorrectToolForDrops":false,"lightEmission":0}
]`), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadCloudburst(directory, "test-version")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Items) != 2 || catalog.Items[0].Name != "minecraft:a_item" {
		t.Fatalf("items were not sorted: %#v", catalog.Items)
	}
	if len(catalog.Blocks) != 2 || catalog.Blocks[0].Name != "minecraft:a_block" {
		t.Fatalf("blocks were not sorted: %#v", catalog.Blocks)
	}
	var output bytes.Buffer
	if err := WriteGo(&output, "data", catalog); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "GeneratedCatalog") || !strings.Contains(output.String(), "minecraft:a_item") {
		t.Fatalf("generated source missing catalog: %s", output.String())
	}
}

func TestGeneratedJavaLookups(t *testing.T) {
	if got, ok := JavaItemID("minecraft:stone"); !ok || got != 1 {
		t.Fatalf("stone item name lookup = id=%d ok=%v", got, ok)
	}
	if got, ok := JavaItemID("minecraft:chain"); !ok || got == 0 {
		t.Fatalf("chain item name lookup = id=%d ok=%v", got, ok)
	}
	if _, ok := JavaItemID("minecraft:not_an_item"); ok {
		t.Fatal("unknown Java item name unexpectedly resolved")
	}
	if got, ok := JavaItemRuntimeID(0); !ok || got != 0 {
		t.Fatalf("air item lookup is not stable: id=%d ok=%v", got, ok)
	}
	if got, ok := JavaItemRuntimeID(Java1214ItemCount - 1); !ok || got == 0 {
		t.Fatalf("last generated item lookup missing: id=%d ok=%v", got, ok)
	}
	if _, ok := JavaItemRuntimeID(-1); ok {
		t.Fatal("negative Java item ID unexpectedly reported as known")
	}
	if got, ok := JavaEntityTypeName(47); !ok || got != "minecraft:xp_orb" {
		t.Fatalf("experience orb mapping = %q, ok=%v", got, ok)
	}
	if _, ok := JavaEntityTypeName(9999); ok {
		t.Fatal("out-of-range Java entity ID unexpectedly reported as known")
	}
}

func TestGeneratedJavaHeadStatesDoNotFallBackToAir(t *testing.T) {
	// Java 1.21.4 places the seven skull/head families contiguously in this
	// registry range: standing rotations followed by wall attachments.
	airRuntimeID := Java1214ToBedrock[0]
	for stateID := 9626; stateID <= 9905; stateID++ {
		if Java1214ToBedrock[stateID] == airRuntimeID {
			t.Fatalf("Java head state %d still maps to Bedrock air", stateID)
		}
	}
}

func TestBedrockItemRuntimeIDRoundTripsGeneratedItem(t *testing.T) {
	for itemID := int32(1); itemID < Java1214ItemCount; itemID++ {
		runtimeID, ok := JavaItemRuntimeID(itemID)
		if !ok {
			t.Fatalf("Java item %d is missing from generated mapping", itemID)
		}
		alias, ok := BedrockItemRuntimeID(runtimeID)
		if !ok || Java1214ToBedrockItem[alias] != runtimeID {
			t.Fatalf("Bedrock runtime %d did not resolve to a generated Java alias", runtimeID)
		}
	}
}
