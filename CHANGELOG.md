# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Documentation
- Add interface screenshot to README showing kopilot startup screen

## [0.2.2] - 2026-02-19

### Fixed
- Improve compatibility between the Copilot SDK and Copilot CLI integrations
- Minor housekeeping and release preparation


## [0.2.1] - 2026-02-17

### Fixed
- Resolve compatibility issues with Copilot CLI v0.0.410

### Added
- System Copilot CLI integration with version checking

### Documentation
- Update CLI version requirements documentation

## [0.2.0] - 2026-02-17

### Added
- Modernize CLI interface with GitHub Copilot-inspired UX
- Cross-platform compilation verification in CI for all 6 release targets
- Comprehensive platform support documentation (tested vs compiled-only)

### Changed
- Update to Go 1.26 across CI workflows and codebase
- Upgrade Copilot SDK from v0.1.20 to v0.1.23
- Update k8s.io/client-go from 0.35.0 to 0.35.1
- Update k8s.io/apimachinery from 0.35.0 to 0.35.1
- Update GitHub Actions: checkout (4→6), setup-go (5→6), upload-artifact (4→6)
- Update GitHub Actions: codeql-action (3→4), codecov-action (4→5)

### Fixed
- Configure custom domain for website deployment
- Correct logo path for GitHub Pages deployment
- Correct codecov action parameter from file to files
- Use GITHUB_TOKEN for release workflow
- Disable homebrew publishing until PAT is configured
- CI lint failure by pinning Go version to 1.26

### Style
- Improve website UI with transparent navigation and consistent icons
- Update AI Output Disclaimer styling for improved visibility
- Cleanup unused code and improve UI output formatting

### Documentation
- Add GitHub Copilot instructions and workflow prompts
- Fix startup behavior description
- Integrate PR specification prompt into auto-commit workflow
- Add release workflow automation prompt

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
