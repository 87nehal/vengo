# Compatibility Policy

This document outlines the versioning scheme, compatibility expectations, and deprecation process for the Vengo framework.

## Semantic Versioning (SemVer)

Vengo follows [Semantic Versioning 2.0.0](https://semver.org/). Version numbers are structured as `MAJOR.MINOR.PATCH`, where:

1. **MAJOR** version bumps indicate incompatible API changes.
2. **MINOR** version bumps add functionality in a backwards-compatible manner.
3. **PATCH** version bumps introduce backwards-compatible bug fixes.

### Pre-1.0.0 Stability

While Vengo is in the `0.x.y` developmental phase:
- Breaking API modifications may occur in **MINOR** releases (e.g. `0.2.0` to `0.3.0`).
- **PATCH** versions (e.g. `0.2.1` to `0.2.2`) will remain backwards-compatible and represent bug fixes or non-breaking feature additions.

---

## Deprecation Process

To evolve the framework while minimizing disruption to applications, Vengo uses a structured deprecation process:

1. **Phase 1: Mark Deprecated**
   - An API component (function, struct, method, interface) is marked as deprecated in code using standard Go comment conventions: `// Deprecated: Use NewAPI instead.`
   - A compiler warning may trigger if the IDE or linter supports it.
   - The deprecation is documented in the release notes of the version where it was deprecated.
   
2. **Phase 2: Maintain Compatibility**
   - The deprecated API continues to function as expected for at least one minor release cycle (e.g. if deprecated in `0.3.0`, it will be supported throughout `0.3.x`).

3. **Phase 3: Removal**
   - The deprecated API is removed in the next minor version (or major version once `1.0.0` is reached).

---

## What is a Breaking Change?

The following are considered **breaking changes** and require a version bump (Minor in `0.x`, Major in `1.x`):
- Removing or renaming any exported function, type, struct field, or interface method.
- Changing the signature of an exported function or method.
- Adding a new method to an exported interface (since it breaks third-party implementations).
- Changing default behaviors in a way that alters application routing, database transaction mechanics, or security constraints.

The following are **non-breaking changes**:
- Adding new exported packages, functions, types, or methods.
- Adding fields to exported structs (provided they don't change instantiation requirements).
- Changing internal implementations (as long as public contracts are maintained).
