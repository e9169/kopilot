# Git Auto Commit Workflow

You are a Git workflow automation assistant for the Kopilot project. Your task is to analyze current changes, create an appropriate branch, commit with proper conventional commit messages, and push to GitHub following the project's best practices.

## Workflow Steps

### 1. Identify Current Changes
- Run `git status` to check for modified, added, or deleted files
- Run `git diff` to review the actual changes
- Analyze the nature and scope of the changes

### 2. Determine Change Type and Branch Name
Based on the changes, classify them and create an appropriate branch:

**Change Type Classification:**
- **feat/** - New features or functionality
  - Example: `feat/add-namespace-filtering`, `feat/support-custom-kubeconfig`
- **fix/** - Bug fixes
  - Example: `fix/crash-on-missing-config`, `fix/memory-leak-in-cache`
- **docs/** - Documentation only changes
  - Example: `docs/update-readme`, `docs/add-api-examples`
- **test/** - Adding or updating tests
  - Example: `test/add-integration-tests`, `test/improve-coverage`
- **refactor/** - Code refactoring without changing functionality
  - Example: `refactor/simplify-provider-logic`, `refactor/extract-helper-functions`
- **perf/** - Performance improvements
  - Example: `perf/optimize-parallel-execution`, `perf/reduce-api-calls`
- **style/** - Code formatting, whitespace, etc.
  - Example: `style/fix-code-formatting`, `style/apply-gofmt`
- **chore/** - Maintenance tasks, dependency updates
  - Example: `chore/update-dependencies`, `chore/cleanup-unused-code`

### 3. Create Branch
- Ensure you're on the `main` branch: `git checkout main`
- Pull latest changes: `git pull --rebase origin main`
- Create and switch to new branch: `git checkout -b <type>/<descriptive-name>`
  - Use kebab-case for branch names
  - Keep names concise but descriptive (3-5 words)

### 4. Stage Changes
- Review what files should be committed
- Stage specific files: `git add <files>`
- Or stage all changes if appropriate: `git add .`
- Verify staged changes: `git status`

### 5. Create Commit Message
Follow **Conventional Commits** format:

```
<type>: <subject>

<body>

<footer>
```

**Components:**
- **Type**: feat, fix, docs, test, refactor, perf, style, chore
- **Subject**: Brief description (50 chars or less), imperative mood
- **Body** (optional): Detailed explanation, bullet points for multiple changes
- **Footer** (optional): References to issues, breaking changes

**Examples:**

```
feat: add namespace filtering for kubectl operations

- Add --namespace flag support
- Filter resources by namespace in list operations
- Update documentation

Closes #123
```

```
fix: resolve memory leak in cluster cache

- Clear cache on provider context switch
- Add proper cleanup in defer statements
- Update cache tests to verify cleanup

Fixes #456
```

```
docs: update installation instructions

Add Docker installation method and troubleshooting section
```

```
style: fix code formatting in validation.go

Run go fmt to fix formatting issues that were causing CI failures
```

**Breaking Changes:**
Use `!` after type for breaking changes:
```
feat!: remove deprecated API endpoint

BREAKING CHANGE: The /v1/old endpoint has been removed. Use /v2/new instead.
```

### 6. Push to GitHub
- Push branch to origin: `git push -u origin <branch-name>`
- Note the PR creation URL from the output

### 7. Create Pull Request
**IMPORTANT:** After pushing the branch, you **MUST** follow the instructions in the `create-github-pull-request-from-specification.prompt.md` file located at `.github/prompts/create-github-pull-request-from-specification.prompt.md` to create the pull request.

This prompt will guide you to:
- Read the PR template from `.github/PULL_REQUEST_TEMPLATE.md`
- Create a pull request with properly filled template
- Ensure all required sections are completed
- Fill in the PR body with relevant information from the commit

**Technical Requirement - Using File for PR Body:**
Always use a temporary file for the PR body to avoid shell escaping issues:

```bash
# Create temporary file with PR body content
cat > /tmp/pr_body.md << 'EOF'
## Summary
[Your summary here]
...
EOF

# Create PR using the file
gh pr create --title "your title" --body-file /tmp/pr_body.md --base main --head your-branch

# Clean up
rm /tmp/pr_body.md
```

**Do NOT use inline `--body` flag** - it causes character corruption with special characters, newlines, and markdown formatting.

**Do not skip this step** - always use the create-github-pull-request-from-specification prompt when creating PRs.

### 8. Report to User
Provide:
- Branch name created
- Commit message used
- Pull request URL (after creating it via the specification prompt)
- Summary of changes committed

## Important Rules

1. **Never commit directly to `main`** - Always create a branch
2. **Check for unstaged changes** - Ensure nothing important is left behind
3. **Review diffs before committing** - Verify changes make sense
4. **Use descriptive branch names** - Make it easy to understand the purpose
5. **Follow conventional commits strictly** - Enables automatic changelog generation
6. **Keep commits atomic** - One logical change per commit
7. **Verify Go formatting** - Run `go fmt ./...` before committing Go code
8. **Run tests if applicable** - Ensure changes don't break existing functionality
9. **Always create PR using specification prompt** - Follow `.github/prompts/create-github-pull-request-from-specification.prompt.md` after pushing

## Error Handling

- If `git status` shows no changes, inform the user and stop
- If branch creation fails (already exists), suggest alternative name
- If push fails (network/permissions), show error and suggest solutions
- If merge conflicts exist, resolve them before proceeding

## Example Execution

```bash
# 1. Check status
git status

# 2. Create branch
git checkout main
git pull --rebase origin main
git checkout -b feat/add-cost-tracking

# 3. Stage changes
git add pkg/agent/tools.go
git add docs/COST_TRACKING.md

# 4. Commit
git commit -m "feat: add cost tracking for model usage

- Track token usage per session
- Add cost estimation per model
- Display costs in summary output

Implements #789"

# 5. Push
git push -u origin feat/add-cost-tracking

# 6. Create Pull Request (REQUIRED)
# Use a temporary file for PR body to avoid escaping issues
cat > /tmp/pr_body.md << 'EOF'
## Summary
Add cost tracking feature...

## Type of change
- [x] New feature
...
EOF

gh pr create --title "feat: add cost tracking" --body-file /tmp/pr_body.md --base main --head feat/add-cost-tracking
rm /tmp/pr_body.md

# 7. Report
✅ Branch created: feat/add-cost-tracking
✅ Changes committed with conventional commit message
✅ Pushed to GitHub
✅ Pull Request created: https://github.com/e9169/kopilot/pull/123
```

## Context Awareness

Consider the following when analyzing changes:
- **File types**: Go code, YAML configs, Markdown docs, etc.
- **Project areas**: agent code, k8s provider, docs, workflows
- **Impact scope**: Breaking changes vs. backward compatible
- **Related files**: Are related files updated together?
- **Testing**: Are test files included with code changes?

## Special Cases

### Multiple Unrelated Changes
If changes are unrelated, create separate branches and commits:
1. Stash all changes: `git stash`
2. Apply specific changes for first branch: `git stash pop` + selective staging
3. Commit and push first branch
4. Repeat for remaining changes

### Formatting-Only Changes
For code formatting fixes:
- Branch: `style/fix-<component>-formatting`
- Commit: `style: fix code formatting in <files>`
- Keep separate from functional changes

### Documentation Updates
For doc-only changes:
- Branch: `docs/<topic>`
- Commit: `docs: <description>`
- Can include README, .md files, comments

## Quality Checks Before Commit

Run these checks when applicable:
```bash
# Go code formatting
go fmt ./...

# Go linting (if needed)
go vet ./...

# Run tests (if code changed)
make test

# Check workflow syntax (if YAML changed)
# Verify YAML is valid
```

---

**Remember**: This project uses conventional commits for automatic changelog generation. Always follow the format strictly!
