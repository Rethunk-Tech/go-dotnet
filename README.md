<h1 align="center">go-dotnet</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

A reader for compiled .NET assemblies: ECMA-335 metadata, IL method bodies, portable PDB sequence points and the Harmony, BepInEx and MelonLoader attributes, with no runtime and no reflection. Package `dotnet`, module `github.com/Rethunk-Tech/go-dotnet`.

Every query takes a path and returns an error, never a panic, for a truncated or crafted image. Reads are capped and
never truncated. Nothing runs and nothing is loaded.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-dotnet
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- BepInEx 5 plugin identity and dependencies (`Scan`), Harmony patches (`Patches`), MelonLoader info and order.
- Which game members a mod assigns (`Writes`), its members (`Members`) and who calls one (`Usages`).
- The BepInEx settings a plugin binds, with their defaults, ranges and lobby sync, before it ever runs (`Settings`).
- Module id and source line for a stack frame (`Identity`, `SourceLine`), from embedded or sibling portable PDBs.
- Crafted-input safe: panics become `ErrMalformed`, type-chain cycles are bounded, size caps are errors.
- Standard library only.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
