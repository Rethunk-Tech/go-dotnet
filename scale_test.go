package dotnet

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"
	"time"
)

// image builds an assembly in memory: each table gets rows rows of width-4 columns, and strings holds the names the
// cells point at. It reaches the readers the way a crafted file would, without a PE around it.
type image struct {
	m       *assembly
	strings []byte
}

func newImage() *image {
	return &image{m: &assembly{}, strings: []byte{0}}
}

// str adds a name to the string heap and returns its index.
func (im *image) str(s string) uint32 {
	at := uint32(len(im.strings) & math.MaxUint32)
	im.strings = append(im.strings, s...)
	im.strings = append(im.strings, 0)
	return at
}

func (im *image) table(t, rows, cols int) {
	m := im.m
	m.rows[t], m.start[t], m.rowSize[t] = rows, len(m.tables), 4*cols
	for c := 0; c <= cols; c++ {
		m.cols[t] = append(m.cols[t], 4*c)
	}
	for range cols {
		m.widths[t] = append(m.widths[t], 4)
	}
	m.tables = append(m.tables, make([]byte, rows*4*cols)...)
}

func (im *image) set(t, row, col int, v uint32) {
	m := im.m
	binary.LittleEndian.PutUint32(m.tables[m.start[t]+(row-1)*m.rowSize[t]+m.cols[t][col]:], v)
}

func (im *image) done() *assembly {
	im.m.strings, im.m.blobs = im.strings, []byte{0}
	return im.m
}

// within fails the test when f takes longer than limit. The inputs are sized so a quadratic reader needs many times
// the limit while a linear one needs a small fraction of it.
func within(t *testing.T, limit time.Duration, f func()) {
	t.Helper()
	start := time.Now()
	f()
	if took := time.Since(start); took > limit {
		t.Fatalf("took %v, want under %v", took, limit)
	}
}

func TestScanMelonDedupesReferencesInLinearTime(t *testing.T) {
	const refs = 150_000
	im := newImage()
	im.table(tAssemblyRef, refs, 7)
	for row := 1; row <= refs; row++ {
		im.set(tAssemblyRef, row, 6, im.str(fmt.Sprintf("Lib%d", row)))
	}
	m := im.done()
	var got Melon
	within(t, 2*time.Second, func() {
		var err error
		if got, err = scanMelon(m); err != nil {
			t.Fatal(err)
		}
	})
	if len(got.Refs) != refs {
		t.Fatalf("refs = %d, want %d", len(got.Refs), refs)
	}
}

func TestTypeDefNameReadsTheNestingOnce(t *testing.T) {
	const types = 120_000
	im := newImage()
	im.table(tTypeDef, types, 6)
	im.table(tNested, types/2, 2)
	for row := 1; row <= types; row++ {
		im.set(tTypeDef, row, 1, im.str(fmt.Sprintf("T%d", row)))
	}
	// Every even type is nested in the odd one before it.
	for i := 1; i <= types/2; i++ {
		im.set(tNested, i, 0, uint32(2*i))
		im.set(tNested, i, 1, uint32(2*i-1))
	}
	m := im.done()
	var name string
	within(t, 2*time.Second, func() {
		for row := 1; row <= types; row++ {
			name = m.typeDefName(row)
		}
	})
	if name != "T119999+T120000" {
		t.Fatalf("name = %q", name)
	}
}

func TestEnumOfReadsTheConstantsOnce(t *testing.T) {
	const enums, constants = 3000, 200_000
	im := newImage()
	im.table(tTypeRef, 1, 3)
	im.set(tTypeRef, 1, 1, im.str("Enum"))
	im.set(tTypeRef, 1, 2, im.str("System"))
	im.table(tTypeDef, enums, 6)
	for row := 1; row <= enums; row++ {
		im.set(tTypeDef, row, 3, 1<<2|1) // extends TypeRef row 1
		im.set(tTypeDef, row, 4, 1)      // no fields: the next type's list starts at 1 as well
	}
	im.table(tField, 0, 3)
	im.table(tConstant, constants, 3)
	m := im.done()
	within(t, 2*time.Second, func() {
		for row := 1; row <= enums; row++ {
			if _, _, ok := m.enumOf(row); !ok {
				t.Fatalf("type %d is no enum", row)
			}
		}
	})
}

func TestScanDeclaredCapsThePluginsAnAssemblyCanDeclare(t *testing.T) {
	const attrs = maxRelations + 100
	im := newImage()
	im.table(tTypeRef, 1, 3)
	im.set(tTypeRef, 1, 1, im.str("BepInPlugin"))
	im.set(tTypeRef, 1, 2, im.str("BepInEx"))
	im.table(tMemberRef, 1, 3)
	im.set(tMemberRef, 1, 0, 1<<3|1) // parent: TypeRef row 1
	im.table(tTypeDef, 1, 6)
	im.table(tCustomAttr, attrs, 3)
	for row := 1; row <= attrs; row++ {
		im.set(tCustomAttr, row, 0, 1<<5|3) // owner: TypeDef row 1
		im.set(tCustomAttr, row, 1, 1<<3|3) // constructor: MemberRef row 1
		im.set(tCustomAttr, row, 2, 1)      // value: the blob below
	}
	m := im.done()
	value := append([]byte{1, 0}, ser("guid")...)
	value = append(value, ser("name")...)
	value = append(value, ser("1.0.0")...)
	m.blobs = append([]byte{0, byte(len(value) & 0xFF)}, value...)
	got, err := scanDeclared(m)
	if err != nil || len(got.Plugins) != maxRelations {
		t.Fatalf("plugins = %d, %v; want %d", len(got.Plugins), err, maxRelations)
	}
}
