package dotnet

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// The MVIDs are what System.Reflection.Metadata's ModuleVersionId.ToString("N") prints for each fixture, the form a
// Mono stack frame without symbols shows. A Windows PDB beside a DLL is told apart by its magic.
func TestIdentityReadsTheMVIDAndWhichSymbolsShip(t *testing.T) {
	dll, err := os.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	windows := filepath.Join(t.TempDir(), "Mod.dll")
	if err := os.WriteFile(windows, dll, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(windows), "Mod.pdb"), []byte("Microsoft C/C++ MSF 7.00\r\n\x1aDS\x00\x00\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ path, mvid, symbols string }{
		{filepath.Join("testdata", "mod.dll"), "6a29e3a0e29c447cb42ccbb83ef95d5b", ""},
		{filepath.Join("testdata", "embedded.dll"), "1689bfe579104a1b9a307c2e9fbe0fc2", SymbolsPortable},
		{windows, "6a29e3a0e29c447cb42ccbb83ef95d5b", SymbolsWindows},
	} {
		if mvid, symbols := Identity(c.path); mvid != c.mvid || symbols != c.symbols {
			t.Errorf("%s: %q %q, want %q %q", c.path, mvid, symbols, c.mvid, c.symbols)
		}
	}
}

func TestIdentityZeroSizeEmbeddedPDBIsNotPortable(t *testing.T) {
	path := patchDebugDir(t, filepath.Join("testdata", "mod.dll"), "mod.dll", func(e []byte) {
		binary.LittleEndian.PutUint32(e[12:], debugEmbeddedPDB)
		binary.LittleEndian.PutUint32(e[16:], 0)
	})
	mvid, symbols := Identity(path)
	if mvid != "6a29e3a0e29c447cb42ccbb83ef95d5b" || symbols != "" {
		t.Fatalf("Identity = %q %q, want MVID and no symbols", mvid, symbols)
	}
}

func TestIdentityBadDebugPointerKeepsMVID(t *testing.T) {
	path := patchDebugDir(t, filepath.Join("testdata", "embedded.dll"), "embedded.dll", func(e []byte) {
		if binary.LittleEndian.Uint32(e[12:]) == debugEmbeddedPDB {
			binary.LittleEndian.PutUint32(e[24:], 0x7fffffff)
		}
	})
	mvid, symbols := Identity(path)
	if mvid != "1689bfe579104a1b9a307c2e9fbe0fc2" || symbols != "" {
		t.Fatalf("Identity = %q %q, want MVID and no symbols", mvid, symbols)
	}
}

func patchDebugDir(t *testing.T, src, name string, patch func([]byte)) string {
	t.Helper()
	img, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	b := append([]byte(nil), img...)
	pe := int(binary.LittleEndian.Uint32(b[0x3C:]))
	opt := pe + 24
	dd := opt + 96
	if binary.LittleEndian.Uint16(b[opt:]) == 0x20B {
		dd = opt + 112
	}
	rva := binary.LittleEndian.Uint32(b[dd+48:])
	n := int(binary.LittleEndian.Uint32(b[dd+52:]))
	sec := opt + int(binary.LittleEndian.Uint16(b[pe+20:]))
	nsec := int(binary.LittleEndian.Uint16(b[pe+6:]))
	off := 0
	for i := range nsec {
		s := sec + i*40
		va := binary.LittleEndian.Uint32(b[s+12:])
		if rva < va || rva >= va+binary.LittleEndian.Uint32(b[s+8:]) {
			continue
		}
		off = int(binary.LittleEndian.Uint32(b[s+20:]) + (rva - va))
		break
	}
	for e := b[off : off+n]; len(e) >= debugEntrySize; e = e[debugEntrySize:] {
		patch(e[:debugEntrySize])
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
