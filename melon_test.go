package dotnet

import (
	"encoding/binary"
	"path/filepath"
	"slices"
	"testing"
)

func ser(s string) []byte { return append([]byte{byte(min(len(s), 255))}, s...) }

// melonInfoSig is a MelonInfo ctor signature: an instance method returning void, then its parameter types.
func melonInfoSig(intVersion bool) []byte {
	if intVersion {
		return []byte{0x20, 7, 0x01, 0x12, 0x04, 0x0E, 0x08, 0x08, 0x08, 0x0E, 0x0E}
	}
	return []byte{0x20, 5, 0x01, 0x12, 0x04, 0x0E, 0x0E, 0x0E, 0x0E}
}

func TestMelonAttributesReadFromTheirBlobs(t *testing.T) {
	var m Melon
	// [assembly: MelonInfo(typeof(Core), "Better Hands", "1.2.0", "Ada", null)]
	info := append([]byte{1, 0}, ser("Better.Core")...)
	for _, s := range []string{"Better Hands", "1.2.0", "Ada"} {
		info = append(info, ser(s)...)
	}
	m.readAttribute("MelonInfo", append(info, 0xFF, 0), melonInfoSig(false))
	prio := binary.LittleEndian.AppendUint32([]byte{1, 0}, uint32(0xFFFFFFF6))
	m.readAttribute("MelonPriority", prio, nil)
	deps := binary.LittleEndian.AppendUint32([]byte{1, 0}, 2)
	m.readAttribute("MelonAdditionalDependencies", append(append(deps, ser("LibA")...), ser("LibB")...), nil)
	if m.Name != "Better Hands" || m.Version != "1.2.0" || m.Author != "Ada" || !m.HasInfo || m.Priority != -10 || !slices.Equal(m.Additional, []string{"LibA", "LibB"}) {
		t.Fatalf("got %+v", m)
	}
}

// The metadata carries the class's real name, and MelonLoader's end in Attribute: the blob and signature are
// otherwise the same, so only the lookup differs.
func TestMelonReadsTheAttributeSuffixedClassNames(t *testing.T) {
	var m Melon
	info := append([]byte{1, 0}, ser("Better.Core")...)
	for _, s := range []string{"Better Hands", "1.2.0", "Ada"} {
		info = append(info, ser(s)...)
	}
	m.readAttribute("MelonInfoAttribute", append(info, 0xFF, 0xFF), melonInfoSig(false))
	prio := binary.LittleEndian.AppendUint32([]byte{1, 0}, 7)
	m.readAttribute("MelonPriorityAttribute", prio, nil)
	deps := binary.LittleEndian.AppendUint32([]byte{1, 0}, 2)
	deps = append(append(deps, ser("LibA")...), ser("LibB")...)
	m.readAttribute("MelonAdditionalDependenciesAttribute", deps, nil)
	m.readAttribute("MelonOptionalDependenciesAttribute", deps, nil)
	if m.Name != "Better Hands" || m.Version != "1.2.0" || m.Author != "Ada" || !m.HasInfo || m.Priority != 7 ||
		!slices.Equal(m.Additional, []string{"LibA", "LibB"}) || !slices.Equal(m.Optional, []string{"LibA", "LibB"}) {
		t.Fatalf("got %+v", m)
	}
}

// [assembly: MelonInfo(typeof(Mod), "Better Hands", 1, 2, 0, "Ada")] has three int32s where the string ctor has a
// version string.
func TestMelonInfoReadsTheIntVersionCtor(t *testing.T) {
	var m Melon
	info := append([]byte{1, 0}, ser("Better.Core")...)
	info = append(info, ser("Better Hands")...)
	for _, v := range []uint32{1, 2, 0} {
		info = binary.LittleEndian.AppendUint32(info, v)
	}
	m.readAttribute("MelonInfoAttribute", append(info, ser("Ada")...), melonInfoSig(true))
	if m.Name != "Better Hands" || m.Version != "1.2.0" || m.Author != "Ada" || !m.HasInfo {
		t.Fatalf("got %+v", m)
	}
}

func TestMelonOrderPutsDependenciesFirstThenSortsByPriority(t *testing.T) {
	melon := func(asm string, prio int, refs ...string) Melon {
		return Melon{Assembly: asm, Name: asm, Priority: prio, Refs: refs, HasInfo: true}
	}
	got := MelonOrder([]Melon{
		melon("Zed", 0, "Alpha"),
		melon("Alpha", 0),
		melon("Mid", 0),
		melon("Early", -5),
		melon("LoopA", 0, "LoopB"),
		melon("LoopB", 0, "LoopA"),
	})
	var names []string
	for _, g := range got {
		names = append(names, g.Assembly)
	}
	// Alpha, Early, Mid are free (by name Alpha, Early, Mid); Zed follows Alpha; priority then lifts Early; cycles go last.
	if want := []string{"Early", "Alpha", "Mid", "Zed", "LoopA", "LoopB"}; !slices.Equal(names, want) {
		t.Fatalf("order %v, want %v", names, want)
	}
	if !got[4].Cycle || !got[5].Cycle || got[0].Cycle {
		t.Fatal("only the cycle is marked")
	}
}

func TestScanMelonReadsTheAssemblyNameAndReferences(t *testing.T) {
	m, err := ScanMelon(filepath.Join("testdata", "mod.dll"))
	if err != nil || m.Assembly != "Mod" || m.HasInfo || !slices.Contains(m.Refs, "System.Runtime") {
		t.Fatalf("got %+v, %v", m, err)
	}
}

func TestMelonInfoReadsANullAuthor(t *testing.T) {
	var m Melon
	info := append([]byte{1, 0}, ser("Better.Core")...)
	info = append(info, ser("Better Hands")...)
	info = append(info, ser("1.2.0")...)
	m.readAttribute("MelonInfo", append(info, 0xFF, 0xFF), melonInfoSig(false))
	if m.Name != "Better Hands" || m.Version != "1.2.0" || m.Author != "" || !m.HasInfo {
		t.Fatalf("got %+v", m)
	}
}
