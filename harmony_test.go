package dotnet

import (
	"path/filepath"
	"slices"
	"testing"
)

// testdata/mod.dll's Fixture.Patches covers each way a patch names its target: class and method attributes merged, a
// method found by name, a getter and a constructor, argument types, and HarmonyX's type name as a string; Fixture.Hooks
// subscribes a HookGen On hook and an IL hook. Each attribute patch names its own method as a stack frame would.
func TestPatchesReadsHarmonyAttributes(t *testing.T) {
	got, err := Patches(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Patch{
		{"StardewValley.Farmer::.ctor", PatchPostfix, "Fixture.Patches.Built::Postfix"},
		{"StardewValley.Farmer::Update", PatchHook, ""},
		{"StardewValley.Farmer::Update", PatchSkip, "Fixture.Patches.FarmerPatches::SkipUpdate"},
		{"StardewValley.Farmer::get_CurrentToolIndex", PatchPostfix, "Fixture.Patches.FarmerPatches::Postfix"},
		{"StardewValley.Game1::Draw", PatchPrefix, "Fixture.Patches.DrawPatch::Prefix"},
		{"StardewValley.Game1::Draw", PatchTranspiler, "Fixture.Patches.DrawPatch::Transpiler"},
		{"StardewValley.Game1::Update", PatchTranspiler, ""},
		{"StardewValley.Menus.ClickableComponent::snap", PatchFinalizer, "Fixture.Patches.ByName::Done"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("patches = %v\nwant      %v", got, want)
	}
}

func TestPatchesReadsMethodDefHarmonyCtor(t *testing.T) {
	im := newImage()
	im.table(tTypeDef, 2, 6)
	im.set(tTypeDef, 1, 1, im.str("HarmonyPatch"))
	im.set(tTypeDef, 1, 2, im.str("HarmonyLib"))
	im.set(tTypeDef, 1, 4, 1)
	im.set(tTypeDef, 1, 5, 1)
	im.set(tTypeDef, 2, 1, im.str("Built"))
	im.set(tTypeDef, 2, 2, im.str("Fixture"))
	im.set(tTypeDef, 2, 4, 1)
	im.set(tTypeDef, 2, 5, 2)
	im.table(tMethodDef, 2, 6)
	im.set(tMethodDef, 1, 3, im.str(".ctor"))
	im.set(tMethodDef, 1, 4, 1)
	im.set(tMethodDef, 1, 5, 1)
	im.set(tMethodDef, 2, 3, im.str("Prefix"))
	im.set(tMethodDef, 2, 4, 7)
	im.set(tMethodDef, 2, 5, 1)
	im.table(tCustomAttr, 1, 3)
	im.set(tCustomAttr, 1, 0, 2<<5|3)
	im.set(tCustomAttr, 1, 1, 1<<3|2)
	im.set(tCustomAttr, 1, 2, 11)
	m := im.done()
	ctorSig := []byte{0x20, 0x02, 0x01, 0x0e, 0x0e}
	prefixSig := []byte{0x00, 0x00, 0x01}
	value := append(append([]byte{1, 0}, ser("Game.Type")...), ser("Update")...)
	blobs := []byte{0}
	blobs = append(append(blobs, byte(len(ctorSig)&0xFF)), ctorSig...)
	blobs = append(append(blobs, byte(len(prefixSig)&0xFF)), prefixSig...)
	blobs = append(append(blobs, byte(len(value)&0xFF)), value...)
	m.blobs = blobs
	got, err := patches(m)
	if err != nil {
		t.Fatal(err)
	}
	want := []Patch{{Target: "Game.Type::Update", Kind: PatchPrefix, Method: "Fixture.Built::Prefix"}}
	if !slices.Equal(got, want) {
		t.Fatalf("patches = %v", got)
	}
}

func TestTypeNameKeepsGenericArguments(t *testing.T) {
	in := "System.Collections.Generic.List`1[[System.Int32, mscorlib, Version=4.0.0.0]], mscorlib, Version=4.0.0.0"
	if got := typeName(in); got != "System.Collections.Generic.List`1[[System.Int32]]" {
		t.Fatalf("typeName = %q", got)
	}
}

func TestPatchKindByName(t *testing.T) {
	for name, want := range map[string]string{"Prefix": PatchPrefix, "HarmonyFinalizer": "", "Other": ""} {
		if got := PatchKindByName(name); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
}
