# Version Management Guide

## Overview

Kopilot uses **Git tags** as the single source of truth for versioning, following [Semantic Versioning 2.0.0](https://semver.org/).

## Version Format

**Semantic Versioning**: `vMAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

Examples:
- `v0.1.0` - Initial public release
- `v0.2.0` - Added new feature
- `v0.2.1` - Bug fix
- `v0.3.0` - Breaking API change

## How It Works

### 1. **Version Source: Git Tags**

The version is **automatically derived from Git tags**:

```bash
# Check current version
make version

# Output shows version from latest git tag
Current version: v0.1.0-3-g2a1b3c4
Git commit: 2a1b3c4
Build date: 2026-01-30_10:30:45
```

Version breakdown:
- `v0.1.0` - Latest tag
- `-3` - 3 commits since tag
- `-g2a1b3c4` - Git commit hash
- `-dirty` - Uncommitted changes (if any)

### 2. **Build-Time Injection**

Version info is injected at build time via `-ldflags`:

```bash
# Makefile automatically does this
go build -ldflags "-X main.version=v0.1.0 -X main.buildDate=... -X main.gitCommit=..."
```

### 3. **Development vs Release Builds**

| Scenario | Version String | Source |
|----------|---------------|--------|
| No git tags | `dev` | Default fallback |
| Tagged commit | `v0.1.0` | Exact tag |
| After tag | `v0.1.0-3-g2a1b3c4` | Tag + commits |
| Uncommitted changes | `v0.1.0-dirty` | Modified files |

## Release Process

### Step-by-Step Release

#### 1. **Prepare Release**

```bash
# Ensure you're on main branch
git checkout main
git pull origin main

# Check current status
make version
git status

# Run all tests
make test
```

#### 2. **Update CHANGELOG**

Update `CHANGELOG.md` manually:

```markdown
# Changelog

## [0.2.0] - 2026-02-15

### Added
- New feature X
- Support for Y

### Fixed
- Bug in Z component

### Changed
- Improved performance of cluster checks
```

#### 3. **Commit Changes**

```bash
git add CHANGELOG.md
git commit -m "chore: prepare v0.2.0 release"
```

#### 4. **Create Git Tag**

```bash
# Using make target (recommended)
make tag VERSION_TAG=v0.2.0

# Or manually
git tag -a v0.2.0 -m "Release v0.2.0"
```

#### 5. **Push to GitHub**

```bash
# Push commits
git push origin main

# Push tag
git push origin v0.2.0

# Or push all tags
git push origin --tags
```

#### 6. **Automated GitHub Release**

Once the tag is pushed, the Release workflow runs automatically and:

- Builds cross-platform binaries via GoReleaser
- **Generates automatic changelog** from conventional commits
- Publishes a GitHub Release with the changelog
- Generates SBOMs for release artifacts
- Signs checksums with Sigstore Cosign (OIDC keyless)
- Updates Homebrew tap automatically

**Note:** The GitHub Release will have an auto-generated changelog from your commit messages. The `CHANGELOG.md` file in the repo is manually maintained and serves as the official project history.

### Quick Commands

```bash
# Show current version
make version

# Create a new tag
make tag VERSION_TAG=v0.2.0

# Show release instructions
make release

# Build with version info
make build
```

## Version Numbering Guidelines

### When to Bump MAJOR (1.x.x → 2.x.x)

- Breaking API changes
- Removing features
- Incompatible changes to CLI flags
- Major architectural changes

Examples:
- Removing a tool from the agent
- Changing kubeconfig format
- Removing command flags

### When to Bump MINOR (x.1.x → x.2.x)

- New features (backward compatible)
- New tools added to agent
- New CLI flags (optional)
- Deprecations (not removals)

Examples:
- Adding new Kubernetes resource support
- New output formats
- Additional cluster comparison features

### When to Bump PATCH (x.x.1 → x.x.2)

- Bug fixes
- Security patches
- Performance improvements
- Documentation updates
- Dependency updates (security)

Examples:
- Fix crash in cluster status check
- Fix incorrect node count
- Performance optimization

## Pre-Release Versions

For pre-releases, use suffixes:

```bash
# Alpha release
git tag v0.2.0-alpha.1

# Beta release
git tag v0.2.0-beta.1

# Release candidate
git tag v0.2.0-rc.1
```

## Hotfix Process

For critical bugs in production:

```bash
# Create hotfix branch from tag
git checkout -b hotfix/v0.1.1 v0.1.0

# Make fix
git commit -m "fix: critical bug in cluster check"

# Tag hotfix
git tag v0.1.1

# Merge back to main
git checkout main
git merge hotfix/v0.1.1

# Push
git push origin main v0.1.1
```

## Automation Tips

### GitHub Actions Release

Add to `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Build binaries
        run: |
          make build
          
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: bin/*
          generate_release_notes: true
```

### GoReleaser (Advanced)

For multi-platform builds:

```bash
# Install GoReleaser
brew install goreleaser

# Create .goreleaser.yml
goreleaser init

# Test release locally
goreleaser release --snapshot --clean

# Actual release (triggered by CI on tag push)
goreleaser release
```

## Checking Versions

### In Development

```bash
# Without building
make version

# After building
./bin/kopilot --version
```

### In Production

```bash
# Installed binary
kopilot --version

# Output:
# kopilot version v0.1.0
#   build date: 2026-01-30_10:30:45
#   git commit: 2a1b3c4
```

### Programmatically

```go
// In code, version is available as:
fmt.Println(version)    // "v0.1.0"
fmt.Println(buildDate)  // "2026-01-30_10:30:45"
fmt.Println(gitCommit)  // "2a1b3c4"
```

## Best Practices

### ✅ DO

- Use annotated tags (`git tag -a`)
- Follow semantic versioning
- Update CHANGELOG for each release
- Test before tagging
- Use `v` prefix (v0.1.0, not 0.1.0)
- Keep main branch releasable

### ❌ DON'T

- Hardcode versions in source files
- Reuse or move tags
- Tag broken code
- Skip CHANGELOG updates
- Use lightweight tags (use `-a`)

## Troubleshooting

### Version shows "dev"

**Cause**: No git tags exist

**Solution**:
```bash
git tag -a v0.1.0 -m "Initial release"
make build
```

### Version shows "-dirty"

**Cause**: Uncommitted changes

**Solution**:
```bash
git status
git add .
git commit -m "fix: commit changes"
```

### Tag already exists

**Cause**: Tag already created

**Solution**:
```bash
# Delete local tag
git tag -d v0.1.0

# Delete remote tag
git push origin :refs/tags/v0.1.0

# Recreate tag
git tag -a v0.1.0 -m "Release v0.1.0"
```

## FAQ

**Q: Do I need to update version in multiple files?**  
A: No! Version is stored in Git tags only. Makefile extracts it automatically.

**Q: What if I forget to tag before releasing?**  
A: Tag the commit after the fact: `git tag -a v0.1.0 <commit-hash>`

**Q: Can I see version in development builds?**  
A: Yes, it shows "dev" or tag+commits like "v0.1.0-3-g2a1b3c4"

**Q: How do I version pre-releases?**  
A: Use suffixes: v0.2.0-alpha.1, v0.2.0-beta.1, v0.2.0-rc.1

**Q: Should I tag every commit?**  
A: No! Only tag releases. Use semantic versioning to decide when.

## Summary

**Single Source of Truth**: Git tags  
**Format**: Semantic Versioning (vMAJOR.MINOR.PATCH)  
**Workflow**: Code → Test → CHANGELOG → Commit → Tag → Push  
**Automation**: Makefile + GitHub Actions  

This approach ensures:
- ✅ No manual version updates in code
- ✅ Always accurate version information
- ✅ Clear release history via git tags
- ✅ Automated build versioning
- ✅ Easy rollback to any version
