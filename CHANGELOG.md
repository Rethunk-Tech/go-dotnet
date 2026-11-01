# Changelog

Notable changes to `go-dotnet`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-11-01

### Added

- `Settings`: the BepInEx config entries a plugin binds with a constant section, key and default, read from its IL,
  with the description, range, enum names and any ConfigSync or CSync sync level.
- `Member.Types` and `Member.Returns`: a method's parameter types and return type, or a field's type, as C# writes
  them.

## [0.1.0] - 2026-10-29

### Added

- Queries over compiled .NET assemblies: `Scan`, `Patches`, `PatchKindByName`, `Writes`, `Members`, `Usages`, `Identity`,
  `SourceLine`, `ScanMelon`, `MelonOrder`.
- `ErrNotAssembly`, `ErrMalformed`, `ErrTooLarge` and the `MaxAssemblyBytes` and `MaxPDBBytes` caps.
- Fuzz targets for the metadata reader, the attribute blobs and the portable PDB reader.

[Unreleased]: https://github.com/Rethunk-Tech/go-dotnet/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Rethunk-Tech/go-dotnet/compare/v0.1.1...v0.2.0
[0.1.0]: https://github.com/Rethunk-Tech/go-dotnet/releases/tag/v0.1.0
