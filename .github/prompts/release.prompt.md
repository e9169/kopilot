# Release Creation Workflow

You are a release automation assistant for the Kopilot project. Your task is to analyze the current state, determine the appropriate version bump, create a release with proper changelog, and verify the release artifacts.

## Workflow Steps

### 1. Check Current State
- Run `git status` to ensure working directory is clean
- Run `git log --oneline` to see recent commits
- Run `git tag` to see existing versions
- Ensure you're on the `main` branch: `git checkout main`
- Pull latest changes: `git pull --rebase origin main`

### 2. Determine Version Bump

Based on conventional commits since last release, determine the version bump following **Semantic Versioning**:

**Version Format: MAJOR.MINOR.PATCH (e.g., 1.2.3)**

- **MAJOR** (x.0.0) - Breaking changes
  - Commits with `!` (e.g., `feat!:`, `fix!:`)
  - Commits with `BREAKING CHANGE:` in footer
  - Incompatible API changes
  
- **MINOR** (0.x.0) - New features (backward compatible)
  - `feat:` commits
  - New functionality added
  - New tools or capabilities
  
- **PATCH** (0.0.x) - Bug fixes (backward compatible)
  - `fix:` commits
  - `perf:` commits
  - Bug fixes and performance improvements

**Examples:**
- Last version: `v0.0.1`, has `feat:` commits → Next: `v0.1.0`
- Last version: `v1.2.3`, has `fix:` commits → Next: `v1.2.4`
- Last version: `v1.2.3`, has `feat!:` commits → Next: `v2.0.0`

### 3. Review Commits Since Last Release

```bash
# Get last tag
LAST_TAG=$(git describe --tags --abbrev=0)

# See commits since last release
git log $LAST_TAG..HEAD --oneline

# Analyze commit types
git log $LAST_TAG..HEAD --pretty=format:"%s" | grep -E "^(feat|fix|perf|feat!|fix!):"
```

### 4. Create Release Tag

```bash
# Create annotated tag with version
git tag -a v<VERSION> -m "Release v<VERSION>"

# Example:
git tag -a v1.0.0 -m "Release v1.0.0"

# Verify tag
git tag -n
```

### 5. Push Tag to Trigger Release

```bash
# Push the tag to GitHub
git push origin v<VERSION>

# Example:
git push origin v1.0.0
```

### 6. Monitor Release Process

The GitHub Actions workflow will automatically:
1. ✅ Build binaries for all platforms (linux, darwin, windows × amd64, arm64)
2. ✅ Create release archives (.tar.gz for Unix, .zip for Windows)
3. ✅ Generate SBOM (Software Bill of Materials)
4. ✅ Sign artifacts with cosign
5. ✅ Generate changelog from conventional commits
6. ✅ Create GitHub release with all artifacts
7. ✅ **Automatically create/update Homebrew formula** in `e9169/homebrew-tap`

**Monitor at:** `https://github.com/e9169/kopilot/actions/workflows/release.yml`

### 7. Verify Release Artifacts

After workflow completes, check:

```bash
# View release on GitHub
open "https://github.com/e9169/kopilot/releases/tag/v<VERSION>"

# Expected artifacts:
# - kopilot_<VERSION>_linux_amd64.tar.gz
# - kopilot_<VERSION>_linux_arm64.tar.gz
# - kopilot_<VERSION>_darwin_amd64.tar.gz
# - kopilot_<VERSION>_darwin_arm64.tar.gz
# - kopilot_<VERSION>_windows_amd64.zip
# - kopilot_<VERSION>_windows_arm64.zip
# - checksums.txt
# - *.sbom files
# - *.sig files (signatures)
# - *.pem files (certificates)
```

### 8. Verify Homebrew Formula

Check the Homebrew tap repository:

```bash
# View formula
open "https://github.com/e9169/homebrew-tap/blob/main/Formula/kopilot.rb"

# The formula should be automatically updated with:
# - New version number
# - Download URLs for all binaries
# - SHA256 checksums
```

### 9. Test Installation

Test the release works:

```bash
# Test Homebrew installation (if tap exists)
brew tap e9169/tap
brew install kopilot
kopilot --version

# Or test direct download
curl -L https://github.com/e9169/kopilot/releases/download/v<VERSION>/kopilot_$(uname -s)_$(uname -m).tar.gz | tar xz
./kopilot --version
```

### 10. Report to User

Provide:
- Version released
- Link to GitHub release
- Link to Homebrew formula
- Summary of changes included
- Installation commands

## Version Strategy

### Pre-1.0 (0.x.x)
- Still in development/experimental phase
- `0.x.0` - New features
- `0.0.x` - Bug fixes
- Breaking changes allowed in minor versions

### Post-1.0 (1.x.x+)
- Stable production release
- Follow semantic versioning strictly
- Breaking changes only in major versions

## Important Rules

1. **Always work from clean main branch** - No uncommitted changes
2. **Use annotated tags** - `git tag -a` not `git tag`
3. **Never force-push tags** - Tags are immutable
4. **Test before releasing** - Run `make test` and `make build`
5. **Follow semantic versioning** - Be consistent with version bumps
6. **Let GoReleaser handle everything** - Don't manually create releases
7. **Verify Homebrew tap** - Check formula was updated correctly

## Changelog Generation

GoReleaser automatically generates changelog from commits:

**Included in changelog:**
- `feat:` → "Features" section
- `fix:` → "Bug Fixes" section
- `perf:` → "Performance Improvements" section
- `refactor:` → "Refactors" section

**Excluded from changelog:**
- `docs:` commits
- `test:` commits
- `chore:` commits
- Merge commits

## Rolling Back a Release

If a release has critical issues:

```bash
# Delete local tag
git tag -d v<VERSION>

# Delete remote tag
git push origin :refs/tags/v<VERSION>

# Delete GitHub release (manually via UI)
# Then fix issues and re-release
```

## Example: Creating First Production Release

```bash
# Scenario: Moving from v0.0.1 to v1.0.0 (first stable release)

# 1. Check status
git status
git checkout main
git pull --rebase origin main

# 2. Review changes since v0.0.1
git log v0.0.1..HEAD --oneline

# 3. Run tests
make test
make build

# 4. Create release tag
git tag -a v1.0.0 -m "Release v1.0.0 - First stable release

Major features:
- Interactive agent with natural language interface
- Multi-cluster support with parallel execution
- kubectl command integration
- Intelligent model selection for cost optimization
- Safe read-only mode by default

This is the first production-ready release of Kopilot."

# 5. Push tag
git push origin v1.0.0

# 6. Monitor release
open "https://github.com/e9169/kopilot/actions"

# 7. After completion, verify
open "https://github.com/e9169/kopilot/releases/tag/v1.0.0"
open "https://github.com/e9169/homebrew-tap"

# 8. Test Homebrew installation
brew tap e9169/tap
brew install kopilot
kopilot --version
```

## Pre-Release Checklist

Before creating a release, ensure:

- [ ] All tests pass: `make test`
- [ ] Code builds successfully: `make build`
- [ ] Documentation is up to date
- [ ] CHANGELOG.md reflects recent changes (if manually maintained)
- [ ] No known critical bugs
- [ ] All PRs for this release are merged
- [ ] Working directory is clean
- [ ] On main branch with latest changes

## Post-Release Tasks

After successful release:

- [ ] Verify all artifacts are present in GitHub release
- [ ] Verify Homebrew formula was updated
- [ ] Test installation via Homebrew
- [ ] Update documentation if needed
- [ ] Announce release (if applicable)
- [ ] Close related issues/milestones

## Troubleshooting

**Release workflow fails:**
- Check GitHub Actions logs
- Verify GoReleaser configuration
- Ensure secrets/tokens are valid

**Homebrew formula not updated:**
- Check if `homebrew-tap` repository exists
- Verify repository permissions
- Check GoReleaser logs for brew step

**Binary signing fails:**
- cosign signing requires GitHub OIDC
- Check workflow permissions
- Signing is optional, release continues without it

**Invalid version:**
- Must follow semantic versioning
- Must start with 'v' (e.g., v1.0.0)
- Cannot reuse existing tag

## Resources

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GoReleaser Documentation](https://goreleaser.com/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)

---

**Remember**: Releases are immutable. Once published, a version cannot be changed. Always verify before pushing tags!
