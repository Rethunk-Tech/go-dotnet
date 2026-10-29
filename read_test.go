package dotnet

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAnAssemblyOverTheCapWrapsErrTooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.dll")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, MaxAssemblyBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(path); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Scan err = %v, want ErrTooLarge", err)
	}
}

func TestAFileThatIsNoAssemblyWrapsErrNotAssembly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.dll")
	if err := os.WriteFile(path, []byte("MZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(path); !errors.Is(err, ErrNotAssembly) {
		t.Fatalf("Scan err = %v, want ErrNotAssembly", err)
	}
}

func TestOpenOnATruncatedImageIsMalformedNotAPanic(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{len(data) / 2, 0x200} {
		if _, err := open(data[:n]); err == nil {
			t.Fatalf("open of %d bytes succeeded", n)
		}
	}
	if _, err := open(data[:len(data)/2]); !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrNotAssembly) {
		t.Fatalf("open err = %v, want ErrMalformed or ErrNotAssembly", err)
	}
}

func TestReadFileRefusesASymlinkThatLeavesItsDirectory(t *testing.T) {
	outside, err := os.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(t.TempDir(), "real.dll")
	if err := os.WriteFile(elsewhere, outside, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link.dll")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if _, err := Scan(link); err == nil {
		t.Fatal("Scan followed a symlink out of its directory")
	}
}

func TestASiblingPDBOverTheCapIsRefusedNotTruncated(t *testing.T) {
	dir := t.TempDir()
	dll := filepath.Join(dir, "Mod.dll")
	data, err := os.ReadFile(filepath.Join("testdata", "portable", "Mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dll, data, 0o600); err != nil {
		t.Fatal(err)
	}
	pdb := filepath.Join(dir, "Mod.pdb")
	if err := os.WriteFile(pdb, []byte(portablePDBMagic), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := readFile(dll)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(pdb, MaxPDBBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := a.pdbBytes(dll); err != nil {
		t.Fatalf("a PDB at the cap: %v", err)
	}
	if err := os.Truncate(pdb, MaxPDBBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := a.pdbBytes(dll); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("pdbBytes err = %v, want ErrTooLarge", err)
	}
}
