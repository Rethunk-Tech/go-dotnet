package dotnet

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// The expected lines are what System.Reflection.Metadata's sequence points give for testdata/src/Mod/Mod.cs: the
// last visible point at or before the offset, a lambda in a compiler-generated nested type, and none before the first
// visible point or in a method without points. A PDB whose id is not the DLL's is refused.
func TestSourceLineReadsPortableSymbols(t *testing.T) {
	stale := filepath.Join(t.TempDir(), "Mod.dll")
	dll, err := os.ReadFile(filepath.Join("testdata", "portable", "Mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	pdb, err := os.ReadFile(filepath.Join("testdata", "portable", "Mod.pdb"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := open(dll)
	if err != nil {
		t.Fatal(err)
	}
	id := m.debugEntry(debugCodeView)[4:20]
	at := bytes.Index(pdb, id)
	if at < 0 {
		t.Fatal("the PDB does not carry the DLL's id")
	}
	pdb = bytes.Clone(pdb)
	pdb[at] ^= 0xff
	if err := os.WriteFile(stale, dll, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(stale), "Mod.pdb"), pdb, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		dll, method string
		params, il  int
		want        string
	}{
		{"embedded.dll", "Fixture.Mod::SwitchTool", 2, 0x1c, "Mod.cs:18"},
		{"embedded.dll", "Fixture.Mod::SwitchTool", -1, 0x3d, "Mod.cs:24"},
		{"embedded.dll", "Fixture.Mod::SwitchTool", 1, 0x1c, ""},
		{"embedded.dll", "Fixture.Mod::SwitchTool", 2, 0x5, ""},
		{"portable/Mod.dll", "Fixture.Hooks.Hooks+<>c::<Apply>b__0_1", 1, 0, "Mod.cs:135"},
		{"portable/Mod.dll", "Fixture.Patches.FarmerPatches::SkipUpdate", 0, 0, "Mod.cs:85"},
		{"portable/Mod.dll", "Fixture.Plugins.Plugin::.ctor", 0, 0, ""},
		{"mod.dll", "Fixture.Mod::SwitchTool", 2, 0x1c, ""},
		{stale, "Fixture.Patches.FarmerPatches::SkipUpdate", 0, 0, ""},
	} {
		path := c.dll
		if !filepath.IsAbs(path) {
			path = filepath.Join("testdata", filepath.FromSlash(path))
		}
		got, err := SourceLine(path, c.method, c.params, c.il)
		if err != nil || got != c.want {
			t.Errorf("%s %s @%#x = %q %v, want %q", c.dll, c.method, c.il, got, err, c.want)
		}
	}
}

func TestSourceLineOverTheCapWrapsErrTooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.dll")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, MaxAssemblyBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := SourceLine(path, "T::M", 0, 0); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("SourceLine err = %v, want ErrTooLarge", err)
	}
}

func TestSourceLineAFileThatIsNoAssemblyWrapsErrNotAssembly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.dll")
	if err := os.WriteFile(path, []byte("MZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := SourceLine(path, "T::M", 0, 0); !errors.Is(err, ErrNotAssembly) {
		t.Fatalf("SourceLine err = %v, want ErrNotAssembly", err)
	}
}
