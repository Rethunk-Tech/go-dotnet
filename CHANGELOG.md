# Changelog

Notable changes to `go-dotnet`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Queries over compiled .NET assemblies: `Scan`, `Patches`, `PatchKindByName`, `Writes`, `Members`, `Usages`, `Identity`,
  `SourceLine`, `ScanMelon`, `MelonOrder`.
- `ErrNotAssembly`, `ErrMalformed`, `ErrTooLarge` and the `MaxAssemblyBytes` and `MaxPDBBytes` caps.
- Fuzz targets for the metadata reader, the attribute blobs and the portable PDB reader.

[Unreleased]: https://github.com/Rethunk-Tech/go-dotnet/commits/main
