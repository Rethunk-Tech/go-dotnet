package dotnet

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func assemblySeeds() [][]byte {
	out := [][]byte{{}, []byte("MZ"), {0x4d, 0x5a, 0x00, 0x00}}
	n := 0
	_ = filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || n >= 8 {
			return err
		}
		switch filepath.Ext(path) {
		case ".dll", ".exe", ".pdb", "":
		default:
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil || len(b) > 1<<20 {
			return nil
		}
		out = append(out, b)
		n++
		return nil
	})
	return out
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
	_ = filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(path) != ".txt" && filepath.Ext(path) != ".log" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil || len(b) > 1<<20 {
			return nil
		}
		f.Add(b)
		return nil
	})
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
