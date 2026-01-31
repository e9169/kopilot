# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Homebrew tap support for easy installation via `brew install e9169/tap/kopilot`
- Cross-platform compilation verification in CI for all 6 release targets
- Comprehensive platform support documentation (tested vs compiled-only)

## [0.1.0] - 2026-01-30

### Added
- Initial open-source release of Kopilot
- Interactive GitHub Copilot-powered agent for Kubernetes status queries
- Read-only and interactive execution modes for safety
- Parallel cluster health checks and reporting (5-10x faster)
- Kubernetes provider with cluster discovery and health diagnostics
- Support for multi-cluster kubeconfig files
- kubectl command execution through natural language
- Smart cost optimization with automatic model selection
- Real-time Copilot quota tracking with color indicators
- Type-safe tool definitions using Copilot SDK's DefineTool
- Comprehensive documentation and CI/CD setup
- Multi-platform release builds (Linux, macOS, Windows)
- Multi-architecture support (amd64, arm64)
- Security scanning with CodeQL, gosec, and govulncheck
- SBOM generation and artifact signing with cosign
- Docker support with multi-stage builds
- Pre-commit hooks for code quality
- Dependabot for automated dependency updates

### Documentation
- Detailed README with quick start guide
- Contributing guidelines for open source contributors
- Code of Conduct and Security Policy
- Comprehensive docs in `/docs` directory
- Architecture and design documentation

### CI/CD
- GitHub Actions workflows for testing and linting
- Automated release workflow with GoReleaser
- Security scanning and vulnerability detection
- Dependency review for pull requests
- Multi-OS testing (Ubuntu, macOS)
