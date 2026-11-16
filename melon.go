package dotnet

import (
	"cmp"
	"encoding/binary"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Melon is what a MelonLoader mod or plugin assembly says about itself: its [assembly: MelonInfo], [MelonPriority],
// [MelonAdditionalDependencies] and [MelonOptionalDependencies] attributes (MelonLoader/Attributes), and the assemblies
// it references, which MelonLoader treats as dependencies too.
type Melon struct {
	// Assembly is the assembly's simple name, which other melons name as a dependency.
	Assembly              string
	Name, Version, Author string
	// Priority is MelonPriority; lower loads first.
	Priority                   int
	Refs, Additional, Optional []string
	HasInfo                    bool
	// Cycle is a melon MelonLoader leaves out because its dependencies depend on it.
	Cycle bool
}

// ScanMelon reads the MelonLoader attributes of the assembly at path; HasInfo is false for a library that is no melon.
func ScanMelon(path string) (Melon, error) {
	return withFile(path, scanMelon)
}

func scanMelon(m *assembly) (out Melon, err error) {
	if m.rows[tAssembly] > 0 {
		out.Assembly = m.str(m.cell(tAssembly, 1, 7))
	}
	seen := map[string]bool{}
	for row := 1; row <= m.rows[tAssemblyRef]; row++ {
		if name := m.str(m.cell(tAssemblyRef, row, 6)); !seen[name] {
			seen[name] = true
			out.Refs = append(out.Refs, name)
		}
	}
	for a := range m.attributes("MelonLoader") {
		if a.owner == tAssembly {
			out.readAttribute(a.name, m.value(a), m.ctorSig(a))
		}
	}
	return out, nil
}

// ctorSig is the signature of the constructor a custom attribute names; the blob records no argument types of its own.
func (m *assembly) ctorSig(a attribute) []byte {
	switch a.kind {
	case tMemberRef:
		return m.blob(m.cell(tMemberRef, a.ctor, 2))
	case tMethodDef:
		return m.blob(m.cell(tMethodDef, a.ctor, 4))
	}
	return nil
}

func (mel *Melon) readAttribute(name string, blob, sig []byte) {
	if len(blob) < 2 || blob[0] != 1 || blob[1] != 0 {
		return
	}
	blob = blob[2:]
	// A class is declared as MelonInfoAttribute and written as MelonInfo, but metadata keeps the declared name.
	switch strings.TrimSuffix(name, "Attribute") {
	case "MelonInfo":
		// MelonInfo(Type, name, version, author, downloadLink): the type is serialized as its name string. The version
		// is a string in one ctor and major, minor and patch ints in the other.
		_, rest, ok := serString(blob)
		if !ok {
			return
		}
		var mod, version string
		if mod, rest, ok = serString(rest); !ok {
			return
		}
		if melonInfoIntVersion(sig) {
			if len(rest) < 12 {
				return
			}
			version = strconv.Itoa(int(binary.LittleEndian.Uint32(rest))) + "." +
				strconv.Itoa(int(binary.LittleEndian.Uint32(rest[4:]))) + "." +
				strconv.Itoa(int(binary.LittleEndian.Uint32(rest[8:])))
			rest = rest[12:]
		} else if version, rest, ok = serString(rest); !ok {
			return
		}
		author, _, ok := serString(rest)
		if !ok {
			return
		}
		mel.Name, mel.Version, mel.Author, mel.HasInfo = mod, version, author, true
	case "MelonPriority":
		if len(blob) >= 4 {
			// The blob is a signed 32-bit integer.
			u := binary.LittleEndian.Uint32(blob)
			mel.Priority = int(u&math.MaxInt32) - int(u>>31)<<31
		}
	case "MelonAdditionalDependencies":
		mel.Additional = append(mel.Additional, stringArray(blob)...)
	case "MelonOptionalDependencies":
		mel.Optional = append(mel.Optional, stringArray(blob)...)
	}
}

// melonInfoIntVersion reports whether a MelonInfo ctor takes the version as major, minor and patch ints: the blob
// records no argument types, so the shape it was written with has to come from the signature.
func melonInfoIntVersion(sig []byte) bool {
	if len(sig) < 3 {
		return false
	}
	p := 1
	if sig[0]&0x10 != 0 { // a generic method's type parameter count precedes its parameter count
		_, n := compressed(sig[p:])
		p += n
	}
	if p >= len(sig) {
		return false
	}
	_, n := compressed(sig[p:])
	p += n + 1    // past the parameter count and the void return
	for range 2 { // System.Type for typeof(Melon), then the melon's name
		if p >= len(sig) {
			return false
		}
		el := sig[p]
		p++
		if el == 0x11 || el == 0x12 { // a TypeDefOrRef index follows
			if p >= len(sig) {
				return false
			}
			_, n := compressed(sig[p:])
			if p+n > len(sig) {
				return false
			}
			p += n
		}
	}
	return p < len(sig) && sig[p] == 0x08
}

// stringArray reads a params string[] argument: a 4-byte count (all ones for null), then each string.
func stringArray(b []byte) []string {
	if len(b) < 4 {
		return nil
	}
	n := binary.LittleEndian.Uint32(b)
	if n == 0xFFFFFFFF || n > maxRelations {
		return nil
	}
	b = b[4:]
	var out []string
	for range n {
		s, rest, ok := serString(b)
		if !ok {
			break
		}
		out, b = append(out, s), rest
	}
	return out
}

// MelonOrder is the order MelonLoader registers melons of one kind (plugins, or mods) in (MelonBase.SortMelons): a
// topological sort in which a melon follows the melons whose assemblies it references or names in
// MelonAdditionalDependencies, the loadable melon with the smallest name going first, then a stable sort by priority.
// A melon in a dependency cycle is not loaded; the ones MelonLoader reports are returned last with Cycle set.
//
// ponytail: names compare lower-cased ordinal where MelonLoader's SortedList uses the current culture.
func MelonOrder(melons []Melon) []Melon {
	index := map[string]int{}
	for i, m := range melons {
		index[m.Assembly] = i
	}
	deps := make([][]int, len(melons))
	dependents := make([][]int, len(melons))
	for i, m := range melons {
		for _, name := range slices.Concat(m.Refs, m.Additional) {
			if j, ok := index[name]; ok && j != i && !slices.Contains(deps[i], j) {
				deps[i] = append(deps[i], j)
				dependents[j] = append(dependents[j], i)
			}
		}
	}
	waiting := make([]int, len(melons))
	var ready []int
	for i := range melons {
		if waiting[i] = len(deps[i]); waiting[i] == 0 {
			ready = append(ready, i)
		}
	}
	var order []Melon
	done := make([]bool, len(melons))
	for len(ready) > 0 {
		slices.SortStableFunc(ready, func(a, b int) int {
			return cmp.Or(strings.Compare(strings.ToLower(melons[a].Name), strings.ToLower(melons[b].Name)), a-b)
		})
		next := ready[0]
		ready = ready[1:]
		order, done[next] = append(order, melons[next]), true
		for _, d := range dependents[next] {
			if waiting[d]--; waiting[d] == 0 {
				ready = append(ready, d)
			}
		}
	}
	for i, m := range melons {
		if !done[i] {
			m.Cycle = true
			order = append(order, m)
		}
	}
	slices.SortStableFunc(order, func(a, b Melon) int {
		if a.Cycle != b.Cycle {
			if a.Cycle {
				return 1
			}
			return -1
		}
		return cmp.Compare(a.Priority, b.Priority)
	})
	return order
}
