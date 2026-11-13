package dotnet

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func assemblySeeds() [][]byte {
	out := [][]byte{{}, []byte("MZ"), {0x4d, 0x5a, 0x00, 0x00}}
	for _, p := range []string{
		"testdata/embedded.dll",
		"testdata/mod.dll",
		"testdata/portable/Mod.dll",
		"testdata/portable/Mod.pdb",
		"testdata/fuzz/FuzzAssembly/typeref-cycle",
		"testdata/fuzz/FuzzAssembly/typespec-self",
	} {
		b, err := os.ReadFile(p)
		if err != nil || len(b) > 1<<20 {
			continue
		}
		out = append(out, corpusBytes(b))
	}
	return out
}

// corpusBytes is the image a "go test fuzz v1" corpus file encodes, or b itself when it is a raw file.
func corpusBytes(b []byte) []byte {
	lit, ok := bytes.CutPrefix(b, []byte("go test fuzz v1\n[]byte("))
	if !ok {
		return b
	}
	img, err := strconv.Unquote(string(bytes.TrimSuffix(bytes.TrimSpace(lit), []byte(")"))))
	if err != nil {
		return b
	}
	return []byte(img)
}

func seedAssemblies(f *testing.F) {
	f.Helper()
	for _, b := range assemblySeeds() {
		f.Add(b)
	}
}

func fuzzPath(t *testing.T, data []byte, fn func(string)) {
	t.Helper()
	if len(data) > 1<<20 {
		data = data[:1<<20]
	}
	p := filepath.Join(t.TempDir(), "a.dll")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	fn(p)
}

func FuzzIdentity(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) { Identity(path) })
	})
}

func FuzzMembers(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) { _, _ = Members(path) })
	})
}

func FuzzUsages(f *testing.F) {
	for _, b := range assemblySeeds() {
		f.Add(b, "")
		f.Add(b, "N.T::M")
	}
	f.Fuzz(func(t *testing.T, data []byte, target string) {
		fuzzPath(t, data, func(path string) { _, _ = Usages(path, target) })
	})
}

func FuzzSettings(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) { _, _ = Settings(path) })
	})
}

func FuzzPatches(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) { _, _ = Patches(path) })
	})
}

func FuzzWrites(f *testing.F) {
	for _, b := range assemblySeeds() {
		f.Add(b, "", "")
		f.Add(b, "Game", "Wrapper")
	}
	f.Fuzz(func(t *testing.T, data []byte, game, wrapper string) {
		fuzzPath(t, data, func(path string) { _, _ = Writes(path, game, wrapper) })
	})
}

func FuzzScan(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) { _, _ = Scan(path) })
	})
}

func FuzzScanMelon(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) { _, _ = ScanMelon(path) })
	})
}

func FuzzDiagnostics(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("error CS0000: x"))
	f.Add([]byte("warning CS0000: x"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			data = data[:1<<20]
		}
		_ = Diagnostics(string(data))
	})
}

func FuzzSourceLine(f *testing.F) {
	for _, b := range assemblySeeds() {
		f.Add(b, "", -1, 0)
		f.Add(b, "N.T::M", 0, 0)
	}
	f.Fuzz(func(t *testing.T, data []byte, method string, params, il int) {
		fuzzPath(t, data, func(path string) { _ = SourceLine(path, method, params, il) })
	})
}

// FuzzSourceLineSiblingPDB pairs the portable fixture DLL with fuzzed bytes as its sibling PDB, which no other target
// reaches: they write the DLL alone.
func FuzzSourceLineSiblingPDB(f *testing.F) {
	dll, err := os.ReadFile("testdata/portable/Mod.dll")
	if err != nil {
		f.Fatal(err)
	}
	pdb, err := os.ReadFile("testdata/portable/Mod.pdb")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(pdb, 0)
	f.Add(pdb[:len(pdb)/2], 0x10)
	f.Add([]byte("BSJB"), 0)
	f.Fuzz(func(t *testing.T, data []byte, il int) {
		if len(data) > 1<<20 {
			data = data[:1<<20]
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "Mod.dll")
		if err := os.WriteFile(path, dll, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Mod.pdb"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, _ = Identity(path)
		_ = SourceLine(path, "Fixture.Patches.FarmerPatches::SkipUpdate", 0, il)
		_ = SourceLine(path, "Fixture.Patches.FarmerPatches::SkipUpdate", -1, il)
	})
}
