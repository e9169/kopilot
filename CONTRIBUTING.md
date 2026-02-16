# Contributing to Kopilot

Thank you for your interest in contributing to Kopilot! This document provides guidelines and instructions for contributing.

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive feedback
- Maintain professionalism in all interactions

## Getting Started

### Prerequisites

- Go 1.26 or later
- GitHub Copilot CLI installed and authenticated
- Access to a Kubernetes cluster for testing (optional)
- Git

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/e9169/kopilot.git
   cd kopilot
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/e9169/kopilot.git
   ```
4. Install dependencies:
   ```bash
   make deps
   ```
5. Set up git hooks (optional but recommended):
   ```bash
   make setup-hooks
   ```
   This installs pre-commit hooks that automatically check code formatting, run tests, and verify go.mod before each commit.

## Supported Platforms

Kopilot supports multiple platforms:

- **Tested in CI**: Ubuntu (linux/amd64) and macOS (darwin/arm64)
- **Compiled and released**: All 6 combinations of Linux/macOS/Windows × amd64/arm64
- **Cross-compilation verified**: All release targets are compiled in CI to catch build errors

If you're adding platform-specific code, ensure cross-compilation still works for all targets.

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

Use prefixes:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `test/` - Test improvements
- `refactor/` - Code refactoring

### 2. Make Changes

- Follow Go best practices and idioms
- Maintain code consistency with existing style
- Add tests for new functionality
- Update documentation as needed
- Keep commits focused and atomic

**Use Conventional Commits (for automatic changelog):**
- Format: `<type>: <description>`
- Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`
- Examples: 
  - `feat: add namespace filtering`
  - `fix: crash when kubeconfig is missing`
- Breaking changes: Use `!` like `feat!: remove old API`

Conventional commits enable automatic changelog generation in GitHub releases.

### 3. Testing

Run all tests before submitting:

```bash
# Unit tests
make test

# Integration tests (requires valid kubeconfig)
make test-integration

# Check formatting
go fmt ./...

# Run linter
go vet ./...
```

### 4. Commit Guidelines

Write clear, descriptive commit messages following [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `chore`: Maintenance tasks

**Examples:**
```
feat: add support for custom kubeconfig paths

- Allow KUBECONFIG environment variable
- Add --kubeconfig flag
- Update documentation

Closes #123
```

**Breaking Changes:**
```
feat!: remove deprecated API endpoint

BREAKING CHANGE: The /v1/old endpoint has been removed. Use /v2/new instead.
```

**Note:** Conventional Commits enable automatic CHANGELOG generation at release time.

### 5. Submit Pull Request

1. Push your branch:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create a Pull Request on GitHub

3. Ensure PR description includes:
   - What changes were made and why
   - How to test the changes
   - Any related issues (e.g., "Fixes #123")
   - Screenshots (if UI changes)

## Code Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Run `go vet` to catch common issues
- Keep functions focused and small
- Use meaningful variable names
- Add comments for exported functions

### Testing

- Write unit tests for all new functionality
- Aim for >80% code coverage
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error paths
- Ensure tests are deterministic

### Documentation

- Update README.md for user-facing changes
- Add package documentation comments
- Document exported functions and types
- Include code examples where helpful
- Update relevant .md files

## Project Structure

```
kopilot/
├── main.go              # Application entry point
├── pkg/
│   ├── agent/          # Copilot agent implementation
│   │   ├── agent.go
│   │   └── agent_test.go
│   └── k8s/            # Kubernetes provider
│       ├── provider.go
│       └── provider_test.go
├── bin/                # Compiled binaries
└── docs/               # Documentation
```

## Review Process

1. **Automated Checks**: CI runs tests and linters
2. **Code Review**: Maintainer reviews code quality and design
3. **Testing**: Changes are tested thoroughly
4. **Approval**: Once approved, PR will be merged

## Reporting Issues

When reporting bugs:

1. Check if issue already exists
2. Use the issue template
3. Provide:
   - Go version
   - OS and version
   - Steps to reproduce
   - Expected vs actual behavior
   - Error messages/logs
   - Kubeconfig details (sanitized)

## Feature Requests

For new features:

1. Open an issue first to discuss
2. Explain the use case
3. Describe proposed solution
4. Consider backward compatibility
5. Be open to feedback

## Questions?

- Open an issue for discussion
- Tag with `question` label

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing! 🎉
