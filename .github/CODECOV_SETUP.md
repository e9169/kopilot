# Codecov Integration Setup

This guide will help you set up Codecov for code coverage tracking.

## Prerequisites

- GitHub repository published
- GitHub Actions CI running
- `.codecov.yml` configuration file (already created)

---

## Step 1: Sign Up for Codecov

### Option A: GitHub OAuth (Recommended)

1. Go to [codecov.io](https://about.codecov.io/)
2. Click **"Sign up with GitHub"**
3. Authorize Codecov to access your repositories
4. Select the `e9169/kopilot` repository

### Option B: Direct Installation

1. Go to [github.com/apps/codecov](https://github.com/apps/codecov)
2. Click **"Install"** or **"Configure"**
3. Select your account/organization
4. Choose "Only select repositories"
5. Select `kopilot`
6. Click **"Install"**

---

## Step 2: Get Your Upload Token

### For Public Repositories (Recommended)
Public repos don't need a token! Codecov automatically works with GitHub Actions.

### For Private Repositories
1. Go to [app.codecov.io](https://app.codecov.io/)
2. Navigate to your repository
3. Go to **Settings → General**
4. Copy the **Repository Upload Token**
5. Add as GitHub Secret:
   - Go to your GitHub repo → **Settings → Secrets and variables → Actions**
   - Click **"New repository secret"**
   - Name: `CODECOV_TOKEN`
   - Value: Paste your token
   - Click **"Add secret"**

---

## Step 3: Update GitHub Actions (Already Done ✅)

Your CI workflow already includes Codecov upload:

```yaml
- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    file: ./coverage.out
    flags: unittests
    name: codecov-umbrella
    fail_ci_if_error: false
```

### For Private Repos, Update to:

```yaml
- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    file: ./coverage.out
    token: ${{ secrets.CODECOV_TOKEN }}  # Add this line
    flags: unittests
    name: codecov-umbrella
    fail_ci_if_error: false
```

---

## Step 4: Configuration File (Already Created ✅)

The `.codecov.yml` file has been created with:

- **Target**: Auto-adjust based on previous coverage
- **Threshold**: 1% tolerance for fluctuations
- **Components**: Separate tracking for `agent` and `k8s` packages
- **Ignore patterns**: Test files, examples, website
- **Comment behavior**: PR comments with coverage diff

### Key Settings Explained

```yaml
coverage:
  status:
    project:
      target: auto        # Maintain current coverage
      threshold: 1%       # Allow 1% decrease without failing
    patch:
      target: auto        # New code should maintain coverage
      threshold: 1%       # Allow 1% tolerance
```

---

## Step 5: Add Codecov Badge to README

The badge is already in your README.md:

```markdown
[![codecov](https://codecov.io/gh/e9169/kopilot/branch/main/graph/badge.svg)](https://codecov.io/gh/e9169/kopilot)
```

This will automatically update once you push your first commit with coverage.

---

## Step 6: First Coverage Upload

### Push to GitHub and wait for CI

```bash
git add .codecov.yml .github/
git commit -m "ci: configure codecov integration"
git push origin main
```

### Verify Upload

1. CI runs and uploads coverage
2. Go to [app.codecov.io/gh/e9169/kopilot](https://app.codecov.io/gh/e9169/kopilot)
3. You should see your first coverage report!

---

## Expected Coverage

Based on your current tests:

```
Overall: 31.0%
├── main.go: 0.0% (entry point, integration tested)
├── pkg/agent: 19.5%
└── pkg/k8s: 79.3% ⭐
```

### Components in Codecov

You'll see two components:
- **Agent Package**: Currently ~19.5%
- **Kubernetes Package**: Currently ~79.3%

---

## Step 7: Configure PR Comments

Codecov will automatically comment on PRs with:

- Coverage change (increase/decrease)
- Diff coverage (coverage of changed lines)
- Component-level coverage changes

Example PR comment:
```
📊 Coverage: 31.00% (+0.50%) compared to base branch
✅ All checks passed!

Components:
- agent: 19.50% (+1.00%)
- k8s: 79.30% (no change)
```

---

## Step 8: Optional Enhancements

### Add Coverage to Status Checks

In branch protection rules, add:
- `codecov/project` - Overall coverage
- `codecov/patch` - Coverage of new code

### Set Coverage Goals

Update `.codecov.yml`:

```yaml
coverage:
  status:
    project:
      default:
        target: 50%      # Goal: 50% coverage
        threshold: 2%
```

### Add Sunburst Graph

Visit: `https://codecov.io/gh/e9169/kopilot`

You'll get beautiful visualizations showing:
- File-level coverage
- Package hierarchy
- Trend over time

---

## Troubleshooting

### No Coverage Upload

**Check:**
1. CI workflow ran successfully
2. `coverage.out` file was generated
3. Codecov action didn't fail (check logs)

**Fix:**
```bash
# Verify coverage locally
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage Shows 0%

**Causes:**
- Tests didn't run
- Coverage file is empty
- Wrong file path in workflow

**Fix:**
Check CI logs for test output.

### Private Repo Not Working

**Missing token:**
Add `CODECOV_TOKEN` to GitHub secrets (see Step 2).

### Badge Shows "unknown"

**Wait:**
- First upload takes a few minutes to process
- Badge updates after Codecov processes the data

---

## Monitoring

### Weekly Check
1. Coverage trend (should increase over time)
2. Untested files (shown in Codecov dashboard)
3. Most complex uncovered functions

### Goals
- **Short term**: Maintain 31%+
- **Medium term**: Reach 50%
- **Long term**: Target 80% (especially for core packages)

---

## Quick Reference

| What | Where |
|------|-------|
| Dashboard | https://app.codecov.io/gh/e9169/kopilot |
| Coverage Graph | https://codecov.io/gh/e9169/kopilot/branch/main/graph/badge.svg |
| Settings | https://app.codecov.io/gh/e9169/kopilot/settings |
| Docs | https://docs.codecov.io |

---

## Summary Checklist

- [x] `.codecov.yml` configuration created
- [x] CI workflow includes Codecov upload
- [x] Badge added to README
- [ ] Sign up for Codecov
- [ ] Authorize repository access
- [ ] Push changes to trigger first upload
- [ ] Verify coverage appears on Codecov dashboard
- [ ] (Optional) Add to branch protection rules
- [ ] (Optional) Set coverage goals

---

**Need Help?**
- Codecov Docs: https://docs.codecov.io
- Codecov Support: support@codecov.io
- Community: https://github.com/codecov/feedback/discussions
