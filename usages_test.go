package dotnet

import (
	"path/filepath"
	"slices"
	"testing"
)

// testdata/mod.dll: a field read and written in two methods, a property setter, and a game member referenced from
// another assembly.
func TestUsagesNamesOwnGenericMemberRef(t *testing.T) {
	im := newImage()
	im.table(tTypeDef, 2, 6)
	im.set(tTypeDef, 1, 1, im.str("Box"))
	im.set(tTypeDef, 1, 2, im.str("Ns"))
	im.set(tTypeDef, 1, 4, 1)
	im.set(tTypeDef, 1, 5, 1)
	im.set(tTypeDef, 2, 1, im.str("C"))
	im.set(tTypeDef, 2, 2, im.str("Ns"))
	im.set(tTypeDef, 2, 4, 1)
	im.set(tTypeDef, 2, 5, 2)
	im.table(tMethodDef, 2, 6)
	im.set(tMethodDef, 1, 3, im.str("Set"))
	im.set(tMethodDef, 2, 3, im.str("M"))
	im.table(tTypeSpec, 1, 1)
	im.set(tTypeSpec, 1, 0, 1)
	im.table(tMemberRef, 1, 3)
	im.set(tMemberRef, 1, 0, 1<<3|4)
	im.set(tMemberRef, 1, 1, im.str("Set"))
	m := im.done()
	spec := []byte{0x15, 0x12, 0x04, 0x01, 0x08}
	m.blobs = append([]byte{0, byte(len(spec))}, spec...)
	u := m.usageNames()
	if got := u.name(m, tMemberRef<<24|1); got != "Ns.Box::Set" {
		t.Fatalf("name = %q", got)
	}
	il := []byte{byte(6<<2 | 2), 0x28, 0x01, 0x00, 0x00, 0x0A, 0x2A}
	if !slices.Contains(u.references(m, il), "Ns.Box::Set") {
		t.Fatal("references missed Ns.Box::Set")
	}
	if u.methods[2] != "Ns.C::M" {
		t.Fatalf("caller = %q", u.methods[2])
	}
}

func TestUsagesFindsCallersAndFieldUses(t *testing.T) {
	path := filepath.Join("testdata", "mod.dll")
	for target, want := range map[string][]string{
		"Fixture.Mod::own":                           {"Fixture.Mod::SwitchTool", "Fixture.Mod::Read"},
		"Fixture.Mod::set_Own":                       {"Fixture.Mod::SwitchTool"},
		"StardewValley.Farmer::set_CurrentToolIndex": {"Fixture.Mod::SwitchTool"},
		"StardewValley.Game1::Update":                {},
	} {
		got, err := Usages(path, target)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: %v %v", target, got, err)
		}
	}
}
