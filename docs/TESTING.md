# Testing Documentation

## Overview

This document describes the comprehensive test suite for the kopilot project.

## Test Coverage

Current total coverage: **19.0%**

- `main.go`: 0.0% (entry point, tested via integration)
- `pkg/agent`: 15.4% coverage
- `pkg/k8s`: 29.9% coverage

## Test Structure

### Unit Tests

#### main_test.go

Tests for the application entry point and configuration:

- `TestRunWithValidKubeconfig`: Validates kubeconfig file handling
- `TestRunWithMissingKubeconfig`: Tests behavior with missing configuration
- `TestRunWithCustomKubeconfigPath`: Custom KUBECONFIG environment variable
- `TestKubeconfigPathResolution`: Path resolution logic
- `TestUserHomeDirResolution`: Home directory detection
- `TestApplicationConstants`: Basic compile-time validation

#### pkg/agent/agent_test.go

Tests for Copilot SDK integration and tool definitions:

- `TestDefineTools`: Validates all three tools are registered correctly
- `TestListClustersTool`: Tests list_clusters tool structure and execution
- `TestGetClusterStatusTool`: Tests get_cluster_status tool with various contexts
- `TestCompareClustersTool`: Tests compare_clusters tool definition
- `TestListClustersParams`: Parameter struct validation
- `TestGetClusterStatusParams`: Parameter struct with required fields
- `TestCompareClusterParams`: Array parameter validation
- `TestToolDescriptions`: Verifies tool descriptions contain key terminology
- `TestToolParameterValidation`: JSON tag validation
- `TestToolConcurrency`: Concurrent tool access safety
- Benchmarks: `BenchmarkDefineTools`, `BenchmarkListClustersTool`

#### pkg/k8s/provider_test.go

Tests for Kubernetes provider functionality:

- `TestNewProvider`: Provider initialization with valid/invalid kubeconfig
- `TestGetClusters`: Cluster listing functionality
- `TestGetClusterByContext`: Context-based cluster retrieval
- `TestGetCurrentContext`: Current context detection
- `TestGetClusterStatus_InvalidContext`: Error handling for invalid contexts
- `TestGetNodeRoles`: Node role detection logic
- `TestProviderConcurrency`: Concurrent access to provider methods
- `TestKubeconfigInvalidPath`: Invalid path handling

### Integration Tests

#### integration_test.go (build tag: integration)

Full end-to-end tests requiring actual kubeconfig:

- `TestIntegration_FullAgentFlow`: Complete agent initialization with real clusters

```bash
# Integration tests are gated to avoid accidental runs
KOPILOT_RUN_INTEGRATION_TESTS=1 go test -tags=integration ./...
```

## Running Tests

### Unit Tests Only (default)

```bash
make test
```

### With Coverage Report

```bash
make coverage
```

### Integration Tests (with real clusters)

```bash
KOPILOT_RUN_INTEGRATION_TESTS=1 make test-integration
```

### All Tests

```bash
KOPILOT_RUN_INTEGRATION_TESTS=1 make test-all
```

### Short Mode (fast)

```bash
make test-short
```

### Benchmarks

```bash
make bench
```

## Test Utilities

### Mock Providers

Tests use temporary kubeconfig files created with `createTestKubeconfig()` and `createMockProvider()` helper functions. These create minimal valid kubeconfig structures for testing.

### Test Data

- Temporary files are automatically cleaned up after tests
- No persistent test data is stored
- All test clusters use `https://127.0.0.1:6443` or localhost addresses

## Race Detection

All unit tests run with the `-race` flag enabled to detect data races.

## Test Best Practices

1. **Isolation**: Each test is independent and cleans up after itself
2. **Table-Driven**: Complex scenarios use table-driven test patterns
3. **Descriptive Names**: Test names clearly indicate what is being tested
4. **Error Handling**: All error paths are tested
5. **Concurrency**: Concurrent access patterns are validated
6. **Performance**: Benchmarks track tool initialization and execution

## Platform Testing

### CI Test Coverage

| Platform | Test Execution | Cross-Compilation |
|----------|----------------|-------------------|
| linux/amd64 | ✅ Full test suite | ✅ Verified |
| linux/arm64 | ❌ | ✅ Verified |
| darwin/amd64 (Intel) | ❌ | ✅ Verified |
| darwin/arm64 (Apple Silicon) | ✅ Full test suite | ✅ Verified |
| windows/amd64 | ❌ | ✅ Verified |
| windows/arm64 | ❌ | ✅ Verified |

**Test Execution**: Full test suite runs on Ubuntu and macOS runners.

**Cross-Compilation**: All 6 release platforms are compiled in CI to catch build errors early, even if tests don't run on those platforms.

### Why Not All Platforms?

- **GitHub Actions limitations**: No native ARM runners available for Linux/Windows
- **Cost considerations**: Running tests on all platforms would significantly increase CI time
- **Go's strong cross-compilation**: Go code is highly portable; compilation verification catches most platform issues

## CI/CD Considerations

Tests are designed to run in CI environments without external dependencies:

- No live Kubernetes clusters required for unit tests
- Integration tests are opt-in via KOPILOT_RUN_INTEGRATION_TESTS=1
- All tests complete in under 10 seconds
- No network calls in unit tests
- Cross-compilation for all release targets verified in every build

## Future Test Improvements

1. Increase coverage to >80% (especially main.go and agent.go)
2. Add more edge case testing for Kubernetes API responses
3. Mock Kubernetes clientset for deeper integration testing
4. Add property-based testing for complex scenarios
5. Performance regression testing
