// Package dotnet reads compiled .NET assemblies without running them: ECMA-335 metadata tables, IL method bodies,
// portable PDB sequence points, and the custom attributes that Harmony, BepInEx 5 and MelonLoader define. It parses
// PE files with debug/pe, does no reflection and loads no code.
//
// Every query takes a path. A crafted or truncated image yields an error wrapping ErrMalformed, never a panic, and a
// file over MaxAssemblyBytes yields ErrTooLarge without being read.
package dotnet
