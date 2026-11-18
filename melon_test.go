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

func TestScanMelonReadsMelonLoaderAssemblyAttributes(t *testing.T) {
	im := newImage()
	im.table(tAssembly, 1, 9)
	im.set(tAssembly, 1, 7, im.str("BetterHands"))
	im.table(tTypeRef, 1, 3)
	im.set(tTypeRef, 1, 1, im.str("MelonInfo"))
	im.set(tTypeRef, 1, 2, im.str("MelonLoader"))
	im.table(tTypeDef, 1, 6)
	im.set(tTypeDef, 1, 1, im.str("MelonPriority"))
	im.set(tTypeDef, 1, 2, im.str("MelonLoader"))
	im.set(tTypeDef, 1, 5, 1)
	im.table(tMethodDef, 1, 6)
	im.set(tMethodDef, 1, 3, im.str(".ctor"))
	im.set(tMethodDef, 1, 4, 0) // filled after the blob heap is laid out
	im.table(tMemberRef, 1, 3)
	im.set(tMemberRef, 1, 0, 1<<3|1) // parent: TypeRef row 1
	im.set(tMemberRef, 1, 1, im.str(".ctor"))
	im.table(tCustomAttr, 3, 3)
	im.set(tCustomAttr, 1, 0, 1<<5|14) // owner: Assembly row 1
	im.set(tCustomAttr, 1, 1, 1<<3|3)  // constructor: MemberRef row 1
	im.set(tCustomAttr, 2, 0, 1<<5|14)
	im.set(tCustomAttr, 2, 1, 1<<3|2) // constructor: MethodDef row 1
	im.set(tCustomAttr, 3, 0, 1<<5|3) // owner: TypeDef row 1, ignored
	im.set(tCustomAttr, 3, 1, 1<<3|3)
	m := im.done()
	info := append([]byte{1, 0}, ser("Better.Core")...)
	for _, s := range []string{"Better Hands", "1.2.0", "Ada"} {
		info = append(info, ser(s)...)
	}
	prio := binary.LittleEndian.AppendUint32([]byte{1, 0}, uint32(0xFFFFFFF6))
	infoSig, prioSig := melonInfoSig(false), []byte{0x20, 1, 0x01, 0x08}
	blobs := []byte{0}
	put := func(b []byte) uint32 {
		at := uint32(len(blobs))
		blobs = append(append(blobs, byte(len(b))), b...)
		return at
	}
	infoSigAt, prioSigAt := put(infoSig), put(prioSig)
	infoAt, prioAt := put(info), put(prio)
	m.blobs = blobs
	im.set(tMemberRef, 1, 2, infoSigAt)
	im.set(tMethodDef, 1, 4, prioSigAt)
	im.set(tCustomAttr, 1, 2, infoAt)
	im.set(tCustomAttr, 2, 2, prioAt)
	im.set(tCustomAttr, 3, 2, infoAt)
	got, err := scanMelon(m)
	if err != nil || got.Assembly != "BetterHands" || got.Name != "Better Hands" || got.Version != "1.2.0" ||
		got.Author != "Ada" || !got.HasInfo || got.Priority != -10 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if sig := m.ctorSig(attribute{kind: tTypeRef, ctor: 1}); sig != nil {
		t.Fatalf("unknown ctor table = %x", sig)
	}
}

func TestMelonInfoIgnoresATruncatedBlob(t *testing.T) {
	noAuthor := append([]byte{1, 0}, ser("T")...)
	noAuthor = append(noAuthor, ser("M")...)
	noAuthor = append(noAuthor, ser("1")...)
	shortInts := append([]byte{1, 0}, ser("T")...)
	shortInts = append(shortInts, ser("M")...)
	shortInts = append(shortInts, 1, 2, 3)
	for name, c := range map[string]struct {
		blob, sig []byte
	}{
		"short prolog": {[]byte{1}, nil},
		"no type":      {[]byte{1, 0}, melonInfoSig(false)},
		"no name":      {append([]byte{1, 0}, ser("T")...), melonInfoSig(false)},
		"no version":   {append(append([]byte{1, 0}, ser("T")...), ser("M")...), melonInfoSig(false)},
		"no author":    {noAuthor, melonInfoSig(false)},
		"short ints":   {shortInts, melonInfoSig(true)},
	} {
		var m Melon
		m.readAttribute("MelonInfo", c.blob, c.sig)
		if m.HasInfo {
			t.Errorf("%s: %+v", name, m)
		}
	}
}

func TestMelonInfoIntVersionFollowsTheCtorSignature(t *testing.T) {
	generic := []byte{0x30, 0x01, 7, 0x01, 0x12, 0x04, 0x0E, 0x08, 0x08, 0x08, 0x0E, 0x0E}
	for name, c := range map[string]struct {
		sig  []byte
		want bool
	}{
		"too short":        {[]byte{0x20}, false},
		"generic int ctor": {generic, true},
		// 2-byte generic-arity so the <3-byte guard does not fire first
		"generic truncated": {[]byte{0x10, 0x80, 0x01}, false},
		"no type args":      {[]byte{0x20, 5, 0x01}, false},
		"class without idx": {[]byte{0x20, 5, 0x01, 0x12}, false},
	} {
		if got := melonInfoIntVersion(c.sig); got != c.want {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
}

func TestAStringArrayStopsOnANullOrTruncatedList(t *testing.T) {
	one := binary.LittleEndian.AppendUint32(nil, 1)
	null := binary.LittleEndian.AppendUint32(nil, 0xFFFFFFFF)
	over := binary.LittleEndian.AppendUint32(nil, maxRelations+1)
	truncated := append(binary.LittleEndian.AppendUint32(nil, 2), ser("A")...)
	if stringArray([]byte{1, 2, 3}) != nil || stringArray(null) != nil || stringArray(over) != nil {
		t.Fatal("short, null and oversized lists are empty")
	}
	if got := stringArray(append(one, 9, 'x')); got != nil {
		t.Fatalf("truncated element = %v", got)
	}
	if got := stringArray(truncated); !slices.Equal(got, []string{"A"}) {
		t.Fatalf("partial = %v", got)
	}
}
