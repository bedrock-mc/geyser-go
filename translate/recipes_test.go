package translate

import (
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type recipePacketSink struct {
	packets []packet.Packet
}

func (s *recipePacketSink) WritePacket(pk packet.Packet) error {
	s.packets = append(s.packets, pk)
	return nil
}

func TestDecodeJavaDeclareRecipes(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(0) // legacy recipe list
	_ = w.VarInt(1) // stonecutter recipe list
	_ = w.VarInt(2) // one direct item ID
	_ = w.VarInt(1)
	_ = w.VarInt(javaRecipeDisplayItem)
	_ = w.VarInt(2)
	declare, err := DecodeJavaDeclareRecipes(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(declare.Legacy) != 0 || len(declare.Stonecutter) != 1 {
		t.Fatalf("declare recipes = %#v", declare)
	}
	stonecutter := declare.Stonecutter[0]
	if len(stonecutter.Input.IDs) != 1 || stonecutter.Input.IDs[0] != 1 {
		t.Fatalf("stonecutter input = %#v", stonecutter.Input)
	}
	if stonecutter.Output.Kind != javaRecipeDisplayItem || stonecutter.Output.Item.ItemID != 2 {
		t.Fatalf("stonecutter output = %#v", stonecutter.Output)
	}
}

func TestDecodeJavaRecipeBookAdd(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(1)  // entries
	_ = w.VarInt(42) // display ID
	_ = w.VarInt(javaRecipeDisplayShapeless)
	_ = w.VarInt(2) // ingredients
	_ = w.VarInt(javaRecipeDisplayItem)
	_ = w.VarInt(1)
	_ = w.VarInt(javaRecipeDisplayTag)
	_ = w.String("minecraft:planks")
	_ = w.VarInt(javaRecipeDisplayItemStack)
	_ = w.VarInt(1) // item count
	_ = w.VarInt(2) // item ID
	_ = w.VarInt(0) // added components
	_ = w.VarInt(0) // removed components
	_ = w.VarInt(javaRecipeDisplayEmpty)
	_ = w.VarInt(0) // absent group
	_ = w.VarInt(3) // crafting misc
	_ = w.Bool(false)
	_ = w.Byte(0) // notification/highlight flags
	_ = w.Bool(true)

	add, err := DecodeJavaRecipeBookAdd(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !add.Replace || len(add.Entries) != 1 {
		t.Fatalf("recipe book add = %#v", add)
	}
	entry := add.Entries[0]
	if entry.ID != 42 || entry.Category != 3 || entry.Display.Kind != javaRecipeDisplayShapeless {
		t.Fatalf("recipe book entry = %#v", entry)
	}
	if len(entry.Display.Ingredients) != 2 || entry.Display.Ingredients[1].Tag != "minecraft:planks" {
		t.Fatalf("recipe ingredients = %#v", entry.Display.Ingredients)
	}
	if entry.Display.Result.Item.ItemID != 2 || entry.Display.Result.Item.Item.Stack.Count != 1 {
		t.Fatalf("recipe result = %#v", entry.Display.Result)
	}
}

func TestDecodeJavaRecipeBookAddConsumesRemovedComponentsAfterUnsupportedDisplay(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(1)  // entries
	_ = w.VarInt(43) // display ID
	_ = w.VarInt(javaRecipeDisplayShapeless)
	_ = w.VarInt(1) // ingredients
	_ = w.VarInt(javaRecipeDisplayItemStack)
	_ = w.VarInt(1) // item count
	_ = w.VarInt(1) // item ID
	_ = w.VarInt(1) // added component count
	_ = w.VarInt(1) // removed component count
	_ = w.VarInt(javaItemComponentDamage)
	_ = w.VarInt(-1) // a well-formed but unsupported component value
	_ = w.VarInt(javaItemComponentDamage)
	_ = w.VarInt(javaRecipeDisplayItem)
	_ = w.VarInt(2)                      // result item ID
	_ = w.VarInt(javaRecipeDisplayEmpty) // station
	_ = w.VarInt(0)                      // absent group
	_ = w.VarInt(0)                      // crafting category
	_ = w.Bool(false)                    // no requirements
	_ = w.Byte(0)                        // notification/highlight flags
	_ = w.Bool(false)                    // replace

	add, err := DecodeJavaRecipeBookAdd(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(add.Entries) != 1 || !add.Entries[0].Display.Ingredients[0].Unsupported {
		t.Fatalf("decoded unsupported recipe display = %#v", add)
	}
}

func TestJavaRecipeDisplayProjectsBedrockRecipe(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	display := JavaRecipeDisplay{
		Kind: javaRecipeDisplayShapeless,
		Ingredients: []JavaSlotDisplay{
			javaRecipeItemDisplay(1),
			{Kind: javaRecipeDisplayTag, Tag: "#minecraft:planks"},
		},
		Result: javaRecipeItemDisplay(2),
	}
	recipe, ok := b.javaRecipeDisplayToBedrock(42, display, 3)
	if !ok {
		t.Fatal("recipe was not projected")
	}
	shapeless, ok := recipe.(*gtprotocol.ShapelessRecipe)
	if !ok {
		t.Fatalf("recipe type = %T", recipe)
	}
	if shapeless.Block != "crafting_table" || len(shapeless.Input) != 2 || len(shapeless.Output) != 1 {
		t.Fatalf("projected recipe = %#v", shapeless)
	}
	runtimeID, ok := data.JavaItemRuntimeID(1)
	if !ok || shapeless.Input[0].Descriptor.(*gtprotocol.DefaultItemDescriptor).NetworkID != int16(runtimeID) {
		t.Fatalf("projected item descriptor = %#v", shapeless.Input[0])
	}
	if shapeless.Input[1].Descriptor.(*gtprotocol.ItemTagItemDescriptor).Tag != "minecraft:planks" {
		t.Fatalf("projected tag descriptor = %#v", shapeless.Input[1])
	}
	if shapeless.Output[0].NetworkID == 0 || shapeless.RecipeNetworkID == 0 {
		t.Fatalf("projected output/network ID = %#v", shapeless)
	}
}

func TestJavaRecipeBookAddAndRemoveWritesBedrockPackets(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	sink := new(recipePacketSink)

	addWriter := javaprotocol.NewWriter()
	_ = addWriter.VarInt(1)
	_ = addWriter.VarInt(7)
	_ = addWriter.VarInt(javaRecipeDisplayShapeless)
	_ = addWriter.VarInt(1)
	_ = addWriter.VarInt(javaRecipeDisplayItem)
	_ = addWriter.VarInt(1)
	_ = addWriter.VarInt(javaRecipeDisplayItem)
	_ = addWriter.VarInt(2)
	_ = addWriter.VarInt(javaRecipeDisplayEmpty)
	_ = addWriter.VarInt(0)
	_ = addWriter.VarInt(0)
	_ = addWriter.Bool(false)
	_ = addWriter.Byte(0)
	_ = addWriter.Bool(false)
	if err := b.translateJavaRecipeBookAdd(sink, addWriter.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(sink.packets) != 2 {
		t.Fatalf("packets after add = %d, want CraftingData + UnlockedRecipes", len(sink.packets))
	}
	if _, ok := sink.packets[0].(*packet.CraftingData); !ok {
		t.Fatalf("first packet = %T", sink.packets[0])
	}
	unlocked, ok := sink.packets[1].(*packet.UnlockedRecipes)
	if !ok || len(unlocked.Recipes) != 1 || unlocked.Recipes[0] != "java_recipe_7" {
		t.Fatalf("unlock packet = %#v", sink.packets[1])
	}

	sink.packets = nil
	removeWriter := javaprotocol.NewWriter()
	_ = removeWriter.VarInt(1)
	_ = removeWriter.VarInt(7)
	if err := b.translateJavaRecipeBookRemove(sink, removeWriter.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(sink.packets) != 1 {
		t.Fatalf("packets after remove = %d, want one", len(sink.packets))
	}
	removed, ok := sink.packets[0].(*packet.UnlockedRecipes)
	if !ok || removed.UnlockType != packet.UnlockedRecipesTypeRemoveUnlocked || len(removed.Recipes) != 1 {
		t.Fatalf("remove packet = %#v", sink.packets[0])
	}
}
