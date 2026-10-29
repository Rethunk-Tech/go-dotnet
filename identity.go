package dotnet

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// What Identity finds of an assembly's debug symbols.
const (
	// SymbolsPortable is a portable PDB, embedded in the DLL or beside it, which Mono reads for source lines.
	SymbolsPortable = "portable"
	// SymbolsWindows is a Windows (MSF) PDB beside the DLL, which Mono cannot read.
	SymbolsWindows = "windows"
)

// Debug directory entry types (PE format, "Debug Type").
const (
	debugCodeView     = 2
	debugEmbeddedPDB  = 17
	debugEntrySize    = 28
	portablePDBMagic  = "BSJB"
	windowsPDBMagicV7 = "Microsoft C/C++ MSF 7.00"
)

// Identity reads the module version id of the assembly at path, as the runtime prints it in a stack frame it has no
// symbols for ("<mvid>:IL_002E", Guid.ToString("N")), and which symbols it ships: SymbolsPortable, SymbolsWindows or "".
// A file that is no assembly has neither.
func Identity(path string) (mvid, symbols string) {
	type ids struct{ mvid, symbols string }
	a, err := readFile(path)
	if err != nil {
		return "", ""
	}
	got, err := query(a, func(a *assembly) (ids, error) {
		out := ids{mvid: a.mvid()}
		if a.debugEntry(debugEmbeddedPDB) != nil {
			out.symbols = SymbolsPortable
		}
		return out, nil
	})
	if err != nil {
		return "", ""
	}
	mvid, symbols = got.mvid, got.symbols
	if symbols == "" {
		symbols = siblingPDB(path)
	}
	return mvid, symbols
}

// mvid formats the Module table's Mvid as Guid.ToString("N") does: the first three fields are little-endian.
func (m *assembly) mvid() string {
	i := m.cell(tModule, 1, 2)
	if i == 0 {
		return ""
	}
	return guidN(m.guids[(i-1)*16 : i*16])
}

func guidN(g []byte) string {
	b := []byte{g[3], g[2], g[1], g[0], g[5], g[4], g[7], g[6]}
	return hex.EncodeToString(append(b, g[8:16]...))
}

// debugEntry is the raw data of the first debug directory entry of the type, or nil.
func (m *assembly) debugEntry(typ uint32) []byte {
	for e := m.debug; len(e) >= debugEntrySize; e = e[debugEntrySize:] {
		if le32(e[12:]) == typ {
			size, ptr := le32(e[16:]), le32(e[24:])
			return m.img[ptr : ptr+size]
		}
	}
	return nil
}

// siblingPDB names the kind of PDB beside the DLL, by its magic.
func siblingPDB(dll string) string {
	path := strings.TrimSuffix(dll, filepath.Ext(dll)) + ".pdb"
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	path = filepath.Clean(path)
	f, err := os.OpenInRoot(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, len(windowsPDBMagicV7))
	n, _ := io.ReadFull(f, head)
	switch {
	case strings.HasPrefix(string(head[:n]), portablePDBMagic):
		return SymbolsPortable
	case string(head[:n]) == windowsPDBMagicV7:
		return SymbolsWindows
	}
	return ""
}
