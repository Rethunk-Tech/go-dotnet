package dotnet

import (
	"reflect"
	"testing"
)

// testdata/mod.dll's Fixture.Settings.Binds binds a setting each way a plugin does: a description string, a
// ConfigDescription with a range, a long written as ldc.i4 then conv.i8, a double, a negative int, a module enum shared
// through ConfigSync at its default level and a bool at HostRecommended, one through CSync, and one whose default is a
// parameter, which is left out.
func TestSettingsReadsConstantBinds(t *testing.T) {
	got, err := Settings("testdata/mod.dll")
	if err != nil {
		t.Fatal(err)
	}
	want := []Setting{
		{Section: "General", Key: "Enabled", Type: "bool", Default: "true", Description: "Turns it on."},
		{Section: "General", Key: "Name", Type: "string", Default: "Fixture", Description: "A name."},
		{Section: "Numbers", Key: "Count", Type: "int", Default: "12", Description: "How many.", Min: "1", Max: "100"},
		{Section: "Numbers", Key: "Speed", Type: "float", Default: "1.5", Description: "How fast.", Min: "0.5", Max: "10"},
		{Section: "Numbers", Key: "Big", Type: "long", Default: "3000000000", Description: "A long."},
		{Section: "Numbers", Key: "Ratio", Type: "double", Default: "0.25", Description: "A double."},
		{Section: "Numbers", Key: "Below", Type: "int", Default: "-3", Description: "Negative."},
		{Section: "Shared", Key: "Mode", Type: "enum", Default: "Wild", Description: "Picked by the host.", Values: []string{"Calm", "Wild"}, Sync: "host-enforced"},
		{Section: "Shared", Key: "Hint", Type: "bool", Default: "false", Description: "The host suggests it.", Sync: "host-recommended"},
		{Section: "Shared", Key: "Through CSync", Type: "int", Default: "7", Description: "Synced by CSync.", Sync: "host-enforced"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Settings:\n got %+v\nwant %+v", got, want)
	}
}

func TestSettingsReadsLdnullStringDefault(t *testing.T) {
	im := newImage()
	im.table(tTypeRef, 1, 3)
	im.set(tTypeRef, 1, 1, im.str("ConfigFile"))
	im.set(tTypeRef, 1, 2, im.str("BepInEx.Configuration"))
	im.table(tMemberRef, 1, 3)
	im.set(tMemberRef, 1, 0, 1<<3|1)
	im.set(tMemberRef, 1, 1, im.str("Bind"))
	im.table(tMethodSpec, 1, 2)
	im.set(tMethodSpec, 1, 0, 1<<1|1)
	im.set(tMethodSpec, 1, 1, 1)
	m := im.done()
	inst := []byte{0x0A, 0x01, 0x0E}
	m.blobs = append([]byte{0, byte(len(inst))}, inst...)
	us := func(s string) []byte {
		b := make([]byte, 1+2*len(s)+1)
		b[0] = byte(2*len(s) + 1)
		for i := range s {
			b[1+2*i] = s[i]
		}
		return b
	}
	m.us = append(append([]byte{0}, us("General")...), us("Name")...)
	code := []byte{
		0x72, 0x01, 0x00, 0x00, 0x70,
		0x72, 0x11, 0x00, 0x00, 0x70,
		0x14,
		0x28, 0x01, 0x00, 0x00, 0x2B,
	}
	r := settingReader{m: m, u: m.usageNames(), seen: map[string]bool{}}
	r.body(code)
	want := []Setting{{Section: "General", Key: "Name", Type: "string"}}
	if !reflect.DeepEqual(r.out, want) {
		t.Fatalf("Settings: got %+v", r.out)
	}
}

func TestSettingsRejectsFilesThatAreNotAssemblies(t *testing.T) {
	if _, err := Settings("settings.go"); err == nil {
		t.Fatal("Settings read a Go file as an assembly")
	}
}
