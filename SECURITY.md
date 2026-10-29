# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- The assembly file given to any query: a crafted image must not panic (`ErrMalformed`), overflow the stack (TypeRef
  and TypeSpec chains are bounded), or allocate unboundedly (reads are capped at `MaxAssemblyBytes`, relations at 1024).
- The sibling or embedded portable PDB read by `SourceLine` and `Identity`: refused over `MaxPDBBytes`, and an embedded
  one must inflate to exactly its stated size.
- The path itself: opened inside its directory, so a final-element symlink cannot leave it.

## What is out of scope

- Deciding whether an assembly is trusted: the module reads, it does not load or execute, and says nothing about
  what the code does.
- A parent-directory symlink in the path: only the final element is confined.
- Wrong but well-formed metadata: results describe what the image claims.
