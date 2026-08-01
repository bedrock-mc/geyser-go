package translate

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaRecipeDisplayEmpty int32 = iota
	javaRecipeDisplayAnyFuel
	javaRecipeDisplayItem
	javaRecipeDisplayItemStack
	javaRecipeDisplayTag
	javaRecipeDisplaySmithingTrim
	javaRecipeDisplayWithRemainder
	javaRecipeDisplayComposite
)

const (
	javaRecipeDisplayShapeless int32 = iota
	javaRecipeDisplayShaped
	javaRecipeDisplayFurnace
	javaRecipeDisplayStonecutter
	javaRecipeDisplaySmithing
)

const maxJavaRecipeDisplayDepth = 32

// JavaIDSet is the versioned Java HolderSet shape used by recipe packets. A
// named set is a server tag; an ID set contains registry IDs directly.
type JavaIDSet struct {
	Name string
	IDs  []int32
}

// JavaSlotDisplay is the Java 1.21.4 slot-display union used by recipe book
// packets. The fields are retained after decoding so conversion can reject a
// semantically unsupported display without losing packet alignment.
type JavaSlotDisplay struct {
	Kind        int32
	Item        JavaItemSlot
	Tag         string
	Children    []JavaSlotDisplay
	Unsupported bool
}

// JavaRecipeDisplay is the Java 1.21.4 recipe display union.
type JavaRecipeDisplay struct {
	Kind        int32
	Width       int32
	Height      int32
	Ingredients []JavaSlotDisplay
	Ingredient  JavaSlotDisplay
	Fuel        JavaSlotDisplay
	Template    JavaSlotDisplay
	Base        JavaSlotDisplay
	Addition    JavaSlotDisplay
	Result      JavaSlotDisplay
	Duration    int32
	Experience  float32
}

type JavaLegacyRecipe struct {
	Name  string
	Items []int32
}

type JavaStonecutterRecipe struct {
	Input  JavaIDSet
	Output JavaSlotDisplay
}

type JavaDeclareRecipes struct {
	Legacy      []JavaLegacyRecipe
	Stonecutter []JavaStonecutterRecipe
}

type JavaRecipeBookEntry struct {
	ID       int32
	Display  JavaRecipeDisplay
	Category int32
}

type JavaRecipeBookAdd struct {
	Entries []JavaRecipeBookEntry
	Replace bool
}

type JavaRecipeBookRemove struct {
	IDs []int32
}

// DecodeJavaDeclareRecipes reads the Java 1.21.4 declare-recipes packet. The
// legacy recipe array is retained and fully consumed even though 1.21.4
// servers normally leave it empty; stonecutter entries are the live payload.
func DecodeJavaDeclareRecipes(payload []byte) (JavaDeclareRecipes, error) {
	r := javaprotocol.NewReader(payload)
	legacyCount, err := readCollectionCountLimit(r, "declare recipes legacy recipe", maxJavaCollectionSize)
	if err != nil {
		return JavaDeclareRecipes{}, err
	}
	legacy := make([]JavaLegacyRecipe, 0, legacyCount)
	for i := 0; i < legacyCount; i++ {
		name, err := r.String()
		if err != nil {
			return JavaDeclareRecipes{}, fmt.Errorf("translate: declare recipes legacy recipe %d name: %w", i, err)
		}
		itemCount, err := readCollectionCountLimit(r, "declare recipes legacy item", maxJavaCollectionSize)
		if err != nil {
			return JavaDeclareRecipes{}, err
		}
		items := make([]int32, itemCount)
		for j := range items {
			items[j], err = r.VarInt()
			if err != nil {
				return JavaDeclareRecipes{}, fmt.Errorf("translate: declare recipes legacy recipe %d item %d: %w", i, j, err)
			}
		}
		legacy = append(legacy, JavaLegacyRecipe{Name: name, Items: items})
	}

	stonecutterCount, err := readCollectionCountLimit(r, "declare recipes stonecutter recipe", maxJavaCollectionSize)
	if err != nil {
		return JavaDeclareRecipes{}, err
	}
	stonecutter := make([]JavaStonecutterRecipe, 0, stonecutterCount)
	for i := 0; i < stonecutterCount; i++ {
		input, err := decodeJavaRecipeIDSet(r, fmt.Sprintf("declare recipes stonecutter %d input", i))
		if err != nil {
			return JavaDeclareRecipes{}, err
		}
		output, err := decodeJavaSlotDisplay(r, 0)
		if err != nil {
			return JavaDeclareRecipes{}, fmt.Errorf("translate: declare recipes stonecutter %d output: %w", i, err)
		}
		stonecutter = append(stonecutter, JavaStonecutterRecipe{Input: input, Output: output})
	}
	if r.Remaining() != 0 {
		return JavaDeclareRecipes{}, fmt.Errorf("translate: declare recipes has %d trailing bytes", r.Remaining())
	}
	return JavaDeclareRecipes{Legacy: legacy, Stonecutter: stonecutter}, nil
}

// DecodeJavaRecipeBookAdd reads ClientboundRecipeBookAddPacket for Java
// 1.21.4. Recipe displays are decoded completely before conversion so a
// display with an unsupported semantic projection cannot desynchronise the
// Java stream.
func DecodeJavaRecipeBookAdd(payload []byte) (JavaRecipeBookAdd, error) {
	r := javaprotocol.NewReader(payload)
	count, err := readCollectionCountLimit(r, "recipe book add entry", maxJavaCollectionSize)
	if err != nil {
		return JavaRecipeBookAdd{}, err
	}
	entries := make([]JavaRecipeBookEntry, 0, count)
	for i := 0; i < count; i++ {
		id, err := r.VarInt()
		if err != nil {
			return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add entry %d ID: %w", i, err)
		}
		display, err := decodeJavaRecipeDisplay(r)
		if err != nil {
			return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add entry %d display: %w", i, err)
		}
		if err := discardJavaOptionalVarInt(r, "recipe book add group"); err != nil {
			return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add entry %d group: %w", i, err)
		}
		category, err := r.VarInt()
		if err != nil {
			return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add entry %d category: %w", i, err)
		}
		hasRequirements, err := r.Bool()
		if err != nil {
			return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add entry %d requirements presence: %w", i, err)
		}
		if hasRequirements {
			requirementCount, err := readCollectionCountLimit(r, "recipe book add requirement", maxJavaCollectionSize)
			if err != nil {
				return JavaRecipeBookAdd{}, err
			}
			for requirement := 0; requirement < requirementCount; requirement++ {
				if _, err := decodeJavaRecipeIDSet(r, fmt.Sprintf("recipe book add entry %d requirement %d", i, requirement)); err != nil {
					return JavaRecipeBookAdd{}, err
				}
			}
		}
		if _, err := r.Byte(); err != nil { // notification/highlight bit flags
			return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add entry %d flags: %w", i, err)
		}
		entries = append(entries, JavaRecipeBookEntry{ID: id, Display: display, Category: category})
	}
	replace, err := r.Bool()
	if err != nil {
		return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add replace: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaRecipeBookAdd{}, fmt.Errorf("translate: recipe book add has %d trailing bytes", r.Remaining())
	}
	return JavaRecipeBookAdd{Entries: entries, Replace: replace}, nil
}

func DecodeJavaRecipeBookRemove(payload []byte) (JavaRecipeBookRemove, error) {
	r := javaprotocol.NewReader(payload)
	count, err := readCollectionCountLimit(r, "recipe book remove ID", maxJavaCollectionSize)
	if err != nil {
		return JavaRecipeBookRemove{}, err
	}
	ids := make([]int32, count)
	for i := range ids {
		ids[i], err = r.VarInt()
		if err != nil {
			return JavaRecipeBookRemove{}, fmt.Errorf("translate: recipe book remove ID %d: %w", i, err)
		}
	}
	if r.Remaining() != 0 {
		return JavaRecipeBookRemove{}, fmt.Errorf("translate: recipe book remove has %d trailing bytes", r.Remaining())
	}
	return JavaRecipeBookRemove{IDs: ids}, nil
}

func decodeJavaRecipeIDSet(r *javaprotocol.Reader, field string) (JavaIDSet, error) {
	marker, err := r.VarInt()
	if err != nil {
		return JavaIDSet{}, fmt.Errorf("translate: %s marker: %w", field, err)
	}
	if marker < 0 {
		return JavaIDSet{}, fmt.Errorf("translate: %s has negative marker %d", field, marker)
	}
	if marker == 0 {
		name, err := r.String()
		if err != nil {
			return JavaIDSet{}, fmt.Errorf("translate: %s tag: %w", field, err)
		}
		return JavaIDSet{Name: name}, nil
	}
	count := marker - 1
	if count > int32(maxJavaCollectionSize) {
		return JavaIDSet{}, fmt.Errorf("translate: %s ID count %d exceeds limit %d", field, count, maxJavaCollectionSize)
	}
	ids := make([]int32, count)
	for i := range ids {
		ids[i], err = r.VarInt()
		if err != nil {
			return JavaIDSet{}, fmt.Errorf("translate: %s ID %d: %w", field, i, err)
		}
	}
	return JavaIDSet{IDs: ids}, nil
}

func discardJavaOptionalVarInt(r *javaprotocol.Reader, field string) error {
	value, err := r.VarInt()
	if err != nil {
		return err
	}
	if value < 0 {
		return fmt.Errorf("%s has negative encoded value %d", field, value)
	}
	return nil
}

func decodeJavaSlotDisplay(r *javaprotocol.Reader, depth int) (JavaSlotDisplay, error) {
	if depth > maxJavaRecipeDisplayDepth {
		return JavaSlotDisplay{}, fmt.Errorf("slot display nesting exceeds limit %d", maxJavaRecipeDisplayDepth)
	}
	kind, err := r.VarInt()
	if err != nil {
		return JavaSlotDisplay{}, fmt.Errorf("slot display type: %w", err)
	}
	display := JavaSlotDisplay{Kind: kind}
	switch kind {
	case javaRecipeDisplayEmpty, javaRecipeDisplayAnyFuel:
		return display, nil
	case javaRecipeDisplayItem:
		itemID, err := r.VarInt()
		if err != nil {
			return JavaSlotDisplay{}, fmt.Errorf("slot display item ID: %w", err)
		}
		display.Item = javaRecipeItemFromID(itemID)
		return display, nil
	case javaRecipeDisplayItemStack:
		item, err := decodeJavaItemSlot(r, nil)
		if err != nil {
			if errors.Is(err, ErrUnsupportedJavaItemComponent) {
				display.Unsupported = true
				return display, nil
			}
			return JavaSlotDisplay{}, fmt.Errorf("slot display item stack: %w", err)
		}
		display.Item = item
		return display, nil
	case javaRecipeDisplayTag:
		display.Tag, err = r.String()
		if err != nil {
			return JavaSlotDisplay{}, fmt.Errorf("slot display tag: %w", err)
		}
		return display, nil
	case javaRecipeDisplaySmithingTrim:
		display.Children = make([]JavaSlotDisplay, 3)
		for i := range display.Children {
			display.Children[i], err = decodeJavaSlotDisplay(r, depth+1)
			if err != nil {
				return JavaSlotDisplay{}, fmt.Errorf("slot display smithing trim child %d: %w", i, err)
			}
		}
		return display, nil
	case javaRecipeDisplayWithRemainder:
		display.Children = make([]JavaSlotDisplay, 2)
		for i := range display.Children {
			display.Children[i], err = decodeJavaSlotDisplay(r, depth+1)
			if err != nil {
				return JavaSlotDisplay{}, fmt.Errorf("slot display remainder child %d: %w", i, err)
			}
		}
		return display, nil
	case javaRecipeDisplayComposite:
		count, err := readCollectionCountLimit(r, "slot display composite", maxJavaCollectionSize)
		if err != nil {
			return JavaSlotDisplay{}, err
		}
		display.Children = make([]JavaSlotDisplay, count)
		for i := range display.Children {
			display.Children[i], err = decodeJavaSlotDisplay(r, depth+1)
			if err != nil {
				return JavaSlotDisplay{}, fmt.Errorf("slot display composite child %d: %w", i, err)
			}
		}
		return display, nil
	default:
		return JavaSlotDisplay{}, fmt.Errorf("unsupported slot display type %d", kind)
	}
}

func decodeJavaRecipeDisplay(r *javaprotocol.Reader) (JavaRecipeDisplay, error) {
	kind, err := r.VarInt()
	if err != nil {
		return JavaRecipeDisplay{}, fmt.Errorf("recipe display type: %w", err)
	}
	display := JavaRecipeDisplay{Kind: kind}
	switch kind {
	case javaRecipeDisplayShapeless:
		if display.Ingredients, err = decodeJavaSlotDisplays(r, "shapeless ingredients"); err != nil {
			return JavaRecipeDisplay{}, err
		}
		if display.Result, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("shapeless result: %w", err)
		}
		if _, err := decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("shapeless station: %w", err)
		}
	case javaRecipeDisplayShaped:
		if display.Width, err = r.VarInt(); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("shaped width: %w", err)
		}
		if display.Height, err = r.VarInt(); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("shaped height: %w", err)
		}
		if display.Ingredients, err = decodeJavaSlotDisplays(r, "shaped ingredients"); err != nil {
			return JavaRecipeDisplay{}, err
		}
		if display.Result, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("shaped result: %w", err)
		}
		if _, err := decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("shaped station: %w", err)
		}
	case javaRecipeDisplayFurnace:
		if display.Ingredient, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("furnace ingredient: %w", err)
		}
		if display.Fuel, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("furnace fuel: %w", err)
		}
		if display.Result, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("furnace result: %w", err)
		}
		if _, err := decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("furnace station: %w", err)
		}
		if display.Duration, err = r.VarInt(); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("furnace duration: %w", err)
		}
		if display.Experience, err = r.Float32(); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("furnace experience: %w", err)
		}
	case javaRecipeDisplayStonecutter:
		if display.Ingredient, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("stonecutter ingredient: %w", err)
		}
		if display.Result, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("stonecutter result: %w", err)
		}
		if _, err := decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("stonecutter station: %w", err)
		}
	case javaRecipeDisplaySmithing:
		if display.Template, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("smithing template: %w", err)
		}
		if display.Base, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("smithing base: %w", err)
		}
		if display.Addition, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("smithing addition: %w", err)
		}
		if display.Result, err = decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("smithing result: %w", err)
		}
		if _, err := decodeJavaSlotDisplay(r, 0); err != nil {
			return JavaRecipeDisplay{}, fmt.Errorf("smithing station: %w", err)
		}
	default:
		return JavaRecipeDisplay{}, fmt.Errorf("unsupported recipe display type %d", kind)
	}
	return display, nil
}

func decodeJavaSlotDisplays(r *javaprotocol.Reader, field string) ([]JavaSlotDisplay, error) {
	count, err := readCollectionCountLimit(r, field, maxJavaCollectionSize)
	if err != nil {
		return nil, err
	}
	displays := make([]JavaSlotDisplay, count)
	for i := range displays {
		displays[i], err = decodeJavaSlotDisplay(r, 0)
		if err != nil {
			return nil, fmt.Errorf("translate: %s %d: %w", field, i, err)
		}
	}
	return displays, nil
}

func javaRecipeItemFromID(itemID int32) JavaItemSlot {
	runtimeID, known := data.JavaItemRuntimeID(itemID)
	if itemID == 0 {
		known = false
	}
	item := JavaItemSlot{ItemID: itemID, Known: known, Present: true}
	if known {
		item.Item = gtprotocol.ItemInstance{Stack: gtprotocol.ItemStack{
			ItemType: gtprotocol.ItemType{NetworkID: runtimeID},
			Count:    1,
		}}
	}
	return item
}

func (b *Basic) translateJavaDeclareRecipes(bedrock interface{ WritePacket(packet.Packet) error }, payload []byte) error {
	declare, err := DecodeJavaDeclareRecipes(payload)
	if err != nil {
		return err
	}
	crafting := &packet.CraftingData{}
	for recipeIndex, stonecutter := range declare.Stonecutter {
		output, ok := javaRecipeOutput(stonecutter.Output)
		if !ok {
			b.logSemanticAnomaly("skipping stonecutter recipe with unsupported output", "recipe", recipeIndex)
			continue
		}
		descriptors := javaRecipeIDSetDescriptors(stonecutter.Input)
		if len(descriptors) == 0 {
			b.logSemanticAnomaly("skipping stonecutter recipe with empty input set", "recipe", recipeIndex)
			continue
		}
		for inputIndex, descriptor := range descriptors {
			crafting.Recipes = append(crafting.Recipes, &gtprotocol.ShapelessRecipe{
				RecipeID:          fmt.Sprintf("stonecutter_%d_%d", recipeIndex, inputIndex),
				Input:             []gtprotocol.ItemDescriptorCount{descriptor},
				Output:            []gtprotocol.ItemStack{output},
				Block:             "stonecutter",
				UnlockRequirement: recipeAlwaysUnlocked(),
				RecipeNetworkID:   b.nextRecipeNetworkID(),
			})
		}
	}
	if len(crafting.Recipes) == 0 {
		return nil
	}
	return bedrock.WritePacket(crafting)
}

func (b *Basic) translateJavaRecipeBookAdd(bedrock interface{ WritePacket(packet.Packet) error }, payload []byte) error {
	update, err := DecodeJavaRecipeBookAdd(payload)
	if err != nil {
		return err
	}
	crafting := &packet.CraftingData{}
	unlocked := &packet.UnlockedRecipes{}
	if update.Replace {
		unlocked.UnlockType = packet.UnlockedRecipesTypeInitiallyUnlocked
	} else {
		unlocked.UnlockType = packet.UnlockedRecipesTypeNewlyUnlocked
	}
	for _, entry := range update.Entries {
		b.mu.Lock()
		known := len(b.recipeIDs[entry.ID]) != 0
		b.mu.Unlock()
		if known {
			continue
		}
		recipe, ok := b.javaRecipeDisplayToBedrock(entry.ID, entry.Display, entry.Category)
		if !ok {
			b.logSemanticAnomaly("skipping recipe with unsupported Bedrock projection", "recipe", entry.ID, "display", entry.Display.Kind)
			continue
		}
		crafting.Recipes = append(crafting.Recipes, recipe)
		if isJavaSmithingRecipe(recipe) {
			continue
		}
		recipeID := fmt.Sprintf("java_recipe_%d", entry.ID)
		b.mu.Lock()
		b.recipeIDs[entry.ID] = []string{recipeID}
		b.mu.Unlock()
		unlocked.Recipes = append(unlocked.Recipes, recipeID)
	}
	if len(crafting.Recipes) == 0 {
		return nil
	}
	if err := bedrock.WritePacket(crafting); err != nil {
		return err
	}
	if len(unlocked.Recipes) == 0 {
		return nil
	}
	return bedrock.WritePacket(unlocked)
}

func (b *Basic) translateJavaRecipeBookRemove(bedrock interface{ WritePacket(packet.Packet) error }, payload []byte) error {
	update, err := DecodeJavaRecipeBookRemove(payload)
	if err != nil {
		return err
	}
	unlocked := &packet.UnlockedRecipes{UnlockType: packet.UnlockedRecipesTypeRemoveUnlocked}
	for _, id := range update.IDs {
		b.mu.Lock()
		unlocked.Recipes = append(unlocked.Recipes, b.recipeIDs[id]...)
		delete(b.recipeIDs, id)
		b.mu.Unlock()
	}
	if len(unlocked.Recipes) == 0 {
		return nil
	}
	return bedrock.WritePacket(unlocked)
}

func (b *Basic) javaRecipeDisplayToBedrock(id int32, display JavaRecipeDisplay, category int32) (gtprotocol.Recipe, bool) {
	block := javaRecipeBlock(category, display.Kind)
	recipeID := fmt.Sprintf("java_recipe_%d", id)
	switch display.Kind {
	case javaRecipeDisplayShapeless:
		input, ok := javaRecipeDescriptors(display.Ingredients, false)
		if !ok {
			return nil, false
		}
		output, ok := javaRecipeOutput(display.Result)
		if !ok {
			return nil, false
		}
		return &gtprotocol.ShapelessRecipe{
			RecipeID:          recipeID,
			Input:             input,
			Output:            []gtprotocol.ItemStack{output},
			Block:             block,
			UnlockRequirement: recipeAlwaysUnlocked(),
			RecipeNetworkID:   b.nextRecipeNetworkID(),
		}, true
	case javaRecipeDisplayShaped:
		if display.Width <= 0 || display.Height <= 0 || int64(display.Width)*int64(display.Height) != int64(len(display.Ingredients)) {
			return nil, false
		}
		input, ok := javaRecipeDescriptors(display.Ingredients, true)
		if !ok {
			return nil, false
		}
		output, ok := javaRecipeOutput(display.Result)
		if !ok {
			return nil, false
		}
		return &gtprotocol.ShapedRecipe{
			RecipeID:          recipeID,
			Width:             display.Width,
			Height:            display.Height,
			Input:             input,
			Output:            []gtprotocol.ItemStack{output},
			Block:             block,
			UnlockRequirement: recipeAlwaysUnlocked(),
			RecipeNetworkID:   b.nextRecipeNetworkID(),
		}, true
	case javaRecipeDisplayFurnace, javaRecipeDisplayStonecutter:
		input, ok := javaRecipeDescriptor(display.Ingredient, false)
		if !ok {
			return nil, false
		}
		output, ok := javaRecipeOutput(display.Result)
		if !ok {
			return nil, false
		}
		return &gtprotocol.ShapelessRecipe{
			RecipeID:          recipeID,
			Input:             []gtprotocol.ItemDescriptorCount{input},
			Output:            []gtprotocol.ItemStack{output},
			Block:             block,
			UnlockRequirement: recipeAlwaysUnlocked(),
			RecipeNetworkID:   b.nextRecipeNetworkID(),
		}, true
	case javaRecipeDisplaySmithing:
		template, templateOK := javaRecipeDescriptor(display.Template, false)
		base, baseOK := javaRecipeDescriptor(display.Base, false)
		addition, additionOK := javaRecipeDescriptor(display.Addition, false)
		output, outputOK := javaRecipeOutput(display.Result)
		if !templateOK || !baseOK || !additionOK || !outputOK {
			return nil, false
		}
		return &gtprotocol.SmithingTransformRecipe{
			RecipeID:        recipeID,
			Template:        template,
			Base:            base,
			Addition:        addition,
			Result:          output,
			Block:           "smithing_table",
			RecipeNetworkID: b.nextRecipeNetworkID(),
		}, true
	default:
		return nil, false
	}
}

func isJavaSmithingRecipe(recipe gtprotocol.Recipe) bool {
	_, ok := recipe.(*gtprotocol.SmithingTransformRecipe)
	return ok
}

func javaRecipeBlock(category, displayKind int32) string {
	switch displayKind {
	case javaRecipeDisplayFurnace:
		switch category {
		case 7, 8:
			return "blast_furnace"
		case 9:
			return "smoker"
		case 12:
			return "campfire"
		default:
			return "furnace"
		}
	case javaRecipeDisplayStonecutter:
		return "stonecutter"
	case javaRecipeDisplaySmithing:
		return "smithing_table"
	default:
		return "crafting_table"
	}
}

func javaRecipeIDSetDescriptors(set JavaIDSet) []gtprotocol.ItemDescriptorCount {
	if set.Name != "" {
		name := strings.TrimPrefix(set.Name, "#")
		if name == "" {
			return nil
		}
		return []gtprotocol.ItemDescriptorCount{{
			Descriptor: &gtprotocol.ItemTagItemDescriptor{Tag: name},
			Count:      1,
		}}
	}
	descriptors := make([]gtprotocol.ItemDescriptorCount, 0, len(set.IDs))
	for _, id := range set.IDs {
		if descriptor, ok := javaRecipeDescriptor(javaRecipeItemDisplay(id), false); ok {
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors
}

func javaRecipeItemDisplay(itemID int32) JavaSlotDisplay {
	return JavaSlotDisplay{Kind: javaRecipeDisplayItem, Item: javaRecipeItemFromID(itemID)}
}

func javaRecipeDescriptors(displays []JavaSlotDisplay, allowEmpty bool) ([]gtprotocol.ItemDescriptorCount, bool) {
	descriptors := make([]gtprotocol.ItemDescriptorCount, 0, len(displays))
	for _, display := range displays {
		descriptor, ok := javaRecipeDescriptor(display, allowEmpty)
		if !ok {
			return nil, false
		}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, true
}

func javaRecipeDescriptor(display JavaSlotDisplay, allowEmpty bool) (gtprotocol.ItemDescriptorCount, bool) {
	if display.Unsupported {
		return gtprotocol.ItemDescriptorCount{}, false
	}
	switch display.Kind {
	case javaRecipeDisplayEmpty:
		if allowEmpty {
			return gtprotocol.ItemDescriptorCount{Descriptor: &gtprotocol.InvalidItemDescriptor{}, Count: 0}, true
		}
		return gtprotocol.ItemDescriptorCount{}, false
	case javaRecipeDisplayItem, javaRecipeDisplayItemStack:
		if !display.Item.Present || !display.Item.Known {
			return gtprotocol.ItemDescriptorCount{}, false
		}
		runtimeID := display.Item.Item.Stack.NetworkID
		if runtimeID == 0 || runtimeID < math.MinInt16 || runtimeID > math.MaxInt16 {
			return gtprotocol.ItemDescriptorCount{}, false
		}
		count := int32(display.Item.Item.Stack.Count)
		if count <= 0 {
			count = 1
		}
		metadata := display.Item.Item.Stack.MetadataValue
		if metadata > math.MaxInt16 {
			return gtprotocol.ItemDescriptorCount{}, false
		}
		return gtprotocol.ItemDescriptorCount{
			Descriptor: &gtprotocol.DefaultItemDescriptor{NetworkID: int16(runtimeID), MetadataValue: int16(metadata)},
			Count:      count,
		}, true
	case javaRecipeDisplayTag:
		name := strings.TrimPrefix(display.Tag, "#")
		if name == "" {
			return gtprotocol.ItemDescriptorCount{}, false
		}
		return gtprotocol.ItemDescriptorCount{
			Descriptor: &gtprotocol.ItemTagItemDescriptor{Tag: name},
			Count:      1,
		}, true
	case javaRecipeDisplayWithRemainder:
		if len(display.Children) != 2 {
			return gtprotocol.ItemDescriptorCount{}, false
		}
		return javaRecipeDescriptor(display.Children[0], allowEmpty)
	default:
		return gtprotocol.ItemDescriptorCount{}, false
	}
}

func javaRecipeOutput(display JavaSlotDisplay) (gtprotocol.ItemStack, bool) {
	if display.Unsupported || (display.Kind != javaRecipeDisplayItem && display.Kind != javaRecipeDisplayItemStack) {
		return gtprotocol.ItemStack{}, false
	}
	if !display.Item.Present || !display.Item.Known || display.Item.Item.Stack.NetworkID == 0 {
		return gtprotocol.ItemStack{}, false
	}
	output := display.Item.Item.Stack
	if output.Count == 0 {
		output.Count = 1
	}
	output.HasNetworkID = false
	return output, true
}

func recipeAlwaysUnlocked() gtprotocol.RecipeUnlockRequirement {
	return gtprotocol.RecipeUnlockRequirement{Context: gtprotocol.RecipeUnlockContextAlwaysUnlocked}
}
