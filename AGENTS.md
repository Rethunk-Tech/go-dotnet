# AGENTS.md

Package `dotnet` reads compiled .NET assemblies by path: ECMA-335 metadata, IL, portable PDB sequence points and the
Harmony, BepInEx 5 and MelonLoader attributes. It imports only the standard library (no consumer package, no file-safety
or version-ordering module), because it must stay a leaf any mod manager or analyser can take.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH. No code is OS-specific; the
`windows-latest` job runs the same tests, and the symlink test skips where symlinks need privilege.

## Shape

- Only the queries are exported; the row layer (`assembly`, `open`, `readFile`, `query`, the table schema, `attributes`)
  is not. It changed often and its accessors index the image unchecked, so an exported one would hand every caller a
  panic to recover, and `attributes` skips MethodDef constructors and TypeSpec parents, which would silently drop data.
- Every query reads through `readFile` and runs under `query`, and `open` recovers its own panics: a runtime panic on a
  crafted image becomes an error wrapping `ErrMalformed`, never a crash. A non-runtime panic is a bug and propagates.
- `maxDepth` bounds TypeRef scopes and TypeSpec instantiations because a crafted cycle overflows the stack, which no
  `recover` catches; `tooDeep` is a `runtime.Error` so it takes the same path. `typeDefName` and `nestedName` stop at 32
  levels for the same reason.
- File opens go through `os.OpenInRoot` on the file's directory, so a final-element symlink cannot leave it, and reads
  are capped on the handle (`readCapped`): over the cap is `ErrTooLarge`, never a truncated image, which would parse as
  a smaller valid one.
- `Writes` output is a persisted string format in callers' caches: a change to its text invalidates them.
- `Scan` and `stringArray` cap relations at `maxRelations` so one attribute list cannot allocate without bound.
- `MelonOrder` and `PatchKindByName` are pure; `Member` keeps its JSON tags because callers serialise it.

## Files

| File | Holds |
| --- | --- |
| `doc.go` | Package documentation |
| `assembly.go` | Errors and caps, file read, `open`/`query`, PE and metadata layout, heaps, signatures, IL walk, `Writes`, `Scan` |
| `harmony.go` | `Patches`, `PatchKindByName`, Harmony attribute blobs |
| `members.go` | `Members` |
| `usages.go` | `Usages` |
| `identity.go` | `Identity`, sibling PDB kind |
| `pdb.go` | `SourceLine`, embedded and sibling portable PDB, sequence points |
| `melon.go` | `ScanMelon`, `MelonOrder` |
