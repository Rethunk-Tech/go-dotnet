package dotnet

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestOpsDecodesOperandsAndTwoByteOpcodes(t *testing.T) {
	// ldc.i4.s 5; ldloc 0x0102 (FE 0C); call 0x0A000001; ret
	code := []byte{0x1F, 0x05, 0xFE, 0x0C, 0x02, 0x01, 0x28, 0x01, 0x00, 0x00, 0x0A, 0x2A}
	var got []instr
	for in := range ops(code) {
		got = append(got, in)
	}
	want := []instr{
		{start: 0, op: 0x1F, at: 1, next: 2, arg: 5},
		{start: 2, op: 0xFE0C, at: 4, next: 6, arg: 0x0102},
		{start: 6, op: 0x28, at: 7, next: 11, arg: 0x0A000001},
		{start: 11, op: 0x2A, at: 12, next: 12},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ops = %+v", got)
	}
	if b := ilCode(append([]byte{12<<2 | 2}, code...)); !slices.Equal(b, code) {
		t.Fatalf("ilCode tiny = % x", b)
	}
}

func TestReturnTypeSkipsGenericAndParamCounts(t *testing.T) {
	if got := returnType([]byte("\x00\x00\x01")); got != elementVoid {
		t.Fatalf("void = %#x", got)
	}
	if got := returnType([]byte("\x10\x01\x02\x02")); got != elementBool {
		t.Fatalf("generic bool = %#x", got)
	}
	if got := returnType([]byte("\x00")); got != 0 {
		t.Fatalf("short = %#x", got)
	}
}

func mustPanicTooDeep(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if _, ok := recover().(tooDeep); !ok {
			t.Fatalf("%s did not refuse", name)
		}
	}()
	f()
}

func TestWalkBoundsBranchJoins(t *testing.T) {
	// n pushes, then n x (dup; brtrue.s +0): every branch retains a copy of the whole stack.
	var code []byte
	for range 900 {
		code = append(code, 0x1F, 0)
	}
	for range 900 {
		code = append(code, 0x25, 0x2D, 0x00)
	}
	mustPanicTooDeep(t, "join budget", func() {
		(&scan{m: &assembly{}, written: map[string]bool{}, refs: map[uint32]ref{}}).walk(code, map[int]bool{})
	})
	code = nil
	for range maxStack + 1 {
		code = append(code, 0x1F, 0)
	}
	mustPanicTooDeep(t, "stack depth", func() {
		(&scan{m: &assembly{}, written: map[string]bool{}, refs: map[uint32]ref{}}).walk(code, map[int]bool{})
	})
}

func TestSignatureCountsFitTheBytesLeft(t *testing.T) {
	mustPanicTooDeep(t, "parameter count", func() {
		sigNames{m: &assembly{}}.methodTypes([]byte{0x00, 0xC0, 0xFF, 0xFF, 0xFF, 0x01})
	})
	m, err := readFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	mustPanicTooDeep(t, "generic argument count", func() {
		sigNames{m: m}.csType([]byte{0x15, 0x11, 0x05, 0xC0, 0xFF, 0xFF, 0xFF}, 0)
	})
}

func TestSerStringCopiesNothing(t *testing.T) {
	blob := make([]byte, 1<<20)
	blob[0] = 1
	blob[1] = 'a'
	if n := testing.AllocsPerRun(10, func() { serString(blob) }); n > 1 {
		t.Fatalf("serString allocs = %v, want only the string", n)
	}
	if s, _, ok := serString(blob); !ok || s != "a" {
		t.Fatalf("serString = %q %v", s, ok)
	}
}

func TestILCodeStopsAtTheBodyEnd(t *testing.T) {
	if ilCode([]byte{10<<2 | 2, 0, 0}) != nil {
		t.Fatal("tiny body longer than its bytes")
	}
	fat := []byte{0x13, 0x30, 8, 0, 64, 0, 0, 0, 0, 0, 0, 0, 0x2A}
	if ilCode(fat) != nil {
		t.Fatal("fat body longer than its bytes")
	}
}
