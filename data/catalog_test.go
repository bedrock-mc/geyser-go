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
