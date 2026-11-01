package dotnet

import (
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
