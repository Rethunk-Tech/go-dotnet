package dotnet

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func queryOK(t *testing.T, err error) bool {
	t.Helper()
	if err == nil {
		return true
	}
	if errors.Is(err, ErrMalformed) || errors.Is(err, ErrNotAssembly) || errors.Is(err, ErrTooLarge) {
		return false
	}
	t.Fatalf("error %v", err)
	return false
}

func FuzzIdentity(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) {
			mvid, symbols, err := Identity(path)
			if !queryOK(t, err) {
				return
			}
			if mvid != "" {
				if _, decErr := hex.DecodeString(mvid); decErr != nil || len(mvid) != 32 {
					t.Fatalf("mvid %q", mvid)
				}
			}
			switch symbols {
			case "", SymbolsPortable, SymbolsWindows:
			default:
				t.Fatalf("symbols %q", symbols)
			}
		})
	})
}

func FuzzMembers(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) {
			got, err := Members(path)
			if !queryOK(t, err) {
				return
			}
			for _, m := range got {
				if m.Kind != "method" && m.Kind != "field" {
					t.Fatalf("kind %q", m.Kind)
				}
			}
		})
	})
}

func FuzzUsages(f *testing.F) {
	for _, b := range assemblySeeds() {
		f.Add(b, "")
		f.Add(b, "N.T::M")
	}
	f.Fuzz(func(t *testing.T, data []byte, target string) {
		fuzzPath(t, data, func(path string) { queryOK(t, errOf(Usages(path, target))) })
	})
}

func FuzzSettings(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) {
			got, err := Settings(path)
			if !queryOK(t, err) {
				return
			}
			for _, s := range got {
				switch s.Type {
				case "bool", "int", "uint", "long", "ulong", "short", "ushort", "byte", "sbyte", "float", "double", "string", "enum":
				default:
					t.Fatalf("type %q", s.Type)
				}
				switch s.Sync {
				case "", "host-enforced", "host-recommended":
				default:
					t.Fatalf("sync %q", s.Sync)
				}
			}
		})
	})
}

func FuzzPatches(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) {
			got, err := Patches(path)
			if !queryOK(t, err) {
				return
			}
			for _, p := range got {
				switch p.Kind {
				case PatchSkip, PatchPrefix, PatchPostfix, PatchTranspiler, PatchFinalizer, PatchHook:
				default:
					t.Fatalf("kind %q", p.Kind)
				}
			}
		})
	})
}

func FuzzWrites(f *testing.F) {
	for _, b := range assemblySeeds() {
		f.Add(b, "", "")
		f.Add(b, "Game", "Wrapper")
	}
	f.Fuzz(func(t *testing.T, data []byte, game, wrapper string) {
		fuzzPath(t, data, func(path string) { queryOK(t, errOf(Writes(path, game, wrapper))) })
	})
}

func FuzzScan(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) {
			got, err := Scan(path)
			if !queryOK(t, err) {
				return
			}
			if len(got.Plugins) > maxRelations || len(got.Relations) > maxRelations {
				t.Fatalf("plugins %d relations %d", len(got.Plugins), len(got.Relations))
			}
		})
	})
}

func FuzzScanMelon(f *testing.F) {
	seedAssemblies(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzPath(t, data, func(path string) {
			got, err := ScanMelon(path)
			if !queryOK(t, err) {
				return
			}
			if len(got.Additional) > maxRelations || len(got.Optional) > maxRelations {
				t.Fatalf("additional %d optional %d", len(got.Additional), len(got.Optional))
			}
		})
	})
}

func FuzzDiagnostics(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("error CS0000: x"))
	f.Add([]byte("warning CS0000: x"))
	f.Add([]byte("Plugin.cs(12,5): error CS1002: ; expected [/p/src/A/A.csproj]\n"))
	f.Add([]byte("/p/src/A/Plugin.cs(3,1,3,9): warning CS0168: The variable 'x' is declared but never used [/p/src/A/A.csproj]\n"))
	f.Add([]byte("Plugin.cs(1,1): fatal error CS0001: msg\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			data = data[:1<<20]
		}
		in := string(data)
		got := Diagnostics(in)
		if len(got) > strings.Count(in, "\n")+1 {
			t.Fatalf("diagnostics %d", len(got))
		}
		for _, d := range got {
			if d.Severity != "error" && d.Severity != "warning" {
				t.Fatalf("severity %q", d.Severity)
			}
			if d.Line < 0 || d.Column < 0 {
				t.Fatalf("line/col %d %d", d.Line, d.Column)
			}
		}
	})
}

func FuzzSourceLine(f *testing.F) {
	for _, b := range assemblySeeds() {
		f.Add(b, "", -1, 0)
		f.Add(b, "N.T::M", 0, 0)
		f.Add(b, "Fixture.Patches.FarmerPatches::SkipUpdate", 0, 0)
	}
	f.Fuzz(func(t *testing.T, data []byte, method string, params, il int) {
		fuzzPath(t, data, func(path string) {
			got, err := SourceLine(path, method, params, il)
			if !queryOK(t, err) {
				return
			}
			if got != "" && !strings.Contains(got, ":") {
				t.Fatalf("source line %q", got)
			}
		})
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
		mvid, symbols, err := Identity(path)
		if queryOK(t, err) {
			if mvid != "" {
				if _, decErr := hex.DecodeString(mvid); decErr != nil || len(mvid) != 32 {
					t.Fatalf("mvid %q", mvid)
				}
			}
			switch symbols {
			case "", SymbolsPortable, SymbolsWindows:
			default:
				t.Fatalf("symbols %q", symbols)
			}
		}
		got, err := SourceLine(path, "Fixture.Patches.FarmerPatches::SkipUpdate", 0, il)
		if queryOK(t, err) && got != "" && !strings.Contains(got, ":") {
			t.Fatalf("source line %q", got)
		}
		_, err = SourceLine(path, "Fixture.Patches.FarmerPatches::SkipUpdate", -1, il)
		queryOK(t, err)
	})
}

func errOf[T any](_ T, err error) error { return err }
