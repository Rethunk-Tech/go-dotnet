# HUMANS.md

Run and use `go-dotnet`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-dotnet
```

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Rethunk-Tech/go-dotnet"
)

func main() {
	d, err := dotnet.Scan(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range d.Plugins {
		fmt.Println(p.GUID, p.Name, p.Version)
	}
	for _, r := range d.Relations {
		fmt.Println(r.Plugin, r.Kind, r.GUID, r.MinVersion)
	}
}
```

## Contracts

| Entry point | Contract |
| --- | --- |
| every query | takes a path; a crafted image returns an error wrapping `ErrMalformed`, never a panic |
| every query | refuses a file over `MaxAssemblyBytes` unread (`ErrTooLarge`); a non-assembly is `ErrNotAssembly` |
| `Scan` | BepInEx 5 `[BepInPlugin]` and `[BepInDependency]`/`[BepInIncompatibility]`, at most 1024 relations |
| `Patches`, `PatchKindByName` | Harmony patch targets and kinds from attributes and naming convention |
| `Writes` | members of the `game` namespace the IL assigns; the output text is stable |
| `Members`, `Usages` | method and field listing with parameter, return and field types as C# writes them; the call sites of one member |
| `Settings` | the BepInEx config entries a plugin binds with constants: default as the .cfg writes it, description, range, enum names, ConfigSync or CSync level |
| `Identity`, `SourceLine` | MVID and symbol kind; `file.cs:line` from a portable PDB, `""` when stale or absent; a sibling PDB over `MaxPDBBytes` is an error |
| `ScanMelon`, `MelonOrder` | MelonLoader attributes; load order, pure on its input |
| any path | opened inside its own directory: a final-element symlink out of it is refused |

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
