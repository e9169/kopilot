# Branch Protection Rules Configuration

This document provides the recommended branch protection rules for the `main` branch.

## GitHub Repository Settings Path

**Settings → Branches → Branch protection rules → Add rule**

Branch name pattern: `main`

## Required Settings

### 🔒 **Protect matching branches**

#### ✅ **Require a pull request before merging**
- ✅ **Require approvals**: 1 (or 2 for critical changes)
- ✅ **Dismiss stale pull request approvals when new commits are pushed**
- ✅ **Require review from Code Owners** (if you add a CODEOWNERS file)
- ⬜ Require approval of the most recent reviewable push (optional, for extra security)

#### ✅ **Require status checks to pass before merging**
- ✅ **Require branches to be up to date before merging**

**Required status checks:**
1. `Test (ubuntu-latest, 1.25)` - Ubuntu CI tests
2. `Test (macos-latest, 1.25)` - macOS CI tests
3. `Build` - Build verification
4. `CodeQL-Go` - Security analysis
5. `Gosec` - Security scanning
6. `codecov/patch` - Coverage for new code (optional but recommended)
7. `codecov/project` - Overall coverage tracking (optional but recommended)

#### ✅ **Require conversation resolution before merging**
Ensures all PR comments are addressed before merging.

#### ⬜ **Require signed commits** (Optional but recommended for security)
Requires GPG-signed commits.

#### ✅ **Require linear history**
Prevents merge commits, keeps history clean (choose merge strategy: squash or rebase).

#### ✅ **Require deployments to succeed before merging** (Skip for now)
Not applicable for CLI tool.

#### ⬜ **Lock branch** (Only for emergencies)
Don't enable unless you need to freeze the branch.

### 🔧 **Rules applied to everyone including administrators**

#### ✅ **Include administrators**
Administrators must follow the same rules (recommended for consistency).

### 🚫 **Restrict who can push to matching branches**

#### ⬜ **Restrict pushes that create matching branches** (Optional)
Limits who can create branches matching the pattern.

### 🔄 **Rules applied to pull requests**

#### ⬜ **Allow force pushes** - **DO NOT CHECK**
Never allow force pushes to main.

#### ⬜ **Allow deletions** - **DO NOT CHECK**
Never allow branch deletion.

---

## Quick Setup Checklist

**Phase 1: Basic Protection (Minimum)**
- [x] Require pull request before merging
- [x] Require 1 approval
- [x] Require status checks: Test, Build
- [x] Require conversation resolution

**Phase 2: Enhanced Security (Recommended)**
- [x] Dismiss stale reviews
- [x] Require branches up to date
- [x] Add CodeQL and Gosec checks
- [x] Include administrators
- [x] Require linear history

**Phase 3: Maximum Security (Optional)**
- [ ] Require signed commits
- [ ] Require 2 approvals for critical changes
- [ ] Add CODEOWNERS file
- [ ] Require code owner review

---

## Additional Configurations

### 1. **Add CODEOWNERS file** (Optional)

Create `.github/CODEOWNERS`:

```
# Global owners
* @e9169

# Package-specific owners (if you have contributors)
/pkg/agent/ @e9169
/pkg/k8s/ @e9169

# Documentation
/docs/ @e9169
*.md @e9169

# CI/CD
/.github/ @e9169
```

### 2. **Rulesets** (New GitHub Feature - Alternative)

GitHub now offers "Rulesets" as a more flexible alternative to branch protection rules:

**Settings → Rules → Rulesets → New ruleset → New branch ruleset**

Rulesets allow:
- More granular control
- Multiple branch patterns
- Better permission management
- Enforcement across forks

Consider using Rulesets if you want more advanced control.

---

## Codecov Integration

See separate instructions in the main documentation for Codecov setup.

---

## Testing Branch Protection

After setup, test by:

1. Creating a test branch
2. Making a change
3. Opening a PR
4. Verify all checks must pass
5. Verify approval is required
6. Merge and confirm protections worked

---

## Maintenance

**Review quarterly:**
- Are rules still appropriate?
- Do new checks need to be required?
- Should approval count increase as team grows?

**Update when:**
- Adding new CI workflows
- Adding new security tools
- Team size changes
- Project maturity increases
