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

func TestSettingsRejectsFilesThatAreNotAssemblies(t *testing.T) {
	if _, err := Settings("settings.go"); err == nil {
		t.Fatal("Settings read a Go file as an assembly")
	}
}
