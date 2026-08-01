package data

import "testing"

func TestDragonflyItemEntriesCoverVanillaCatalog(t *testing.T) {
	entries, err := DragonflyItemEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 1000 {
		t.Fatalf("vanilla item table unexpectedly small: %d", len(entries))
	}
	if entries[0].Name != "minecraft:acacia_boat" {
		t.Fatalf("entries are not deterministic: first=%q", entries[0].Name)
	}
	found := false
	for _, entry := range entries {
		if entry.Name == "minecraft:brush" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("modern vanilla item missing")
	}
}
