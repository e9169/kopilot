// Package main uses github.com/klauspost/compress/zstd directly in
// platform-specific generated files (e.g. zcopilot_darwin_arm64.go).
// This file keeps the dependency marked as direct in go.mod on every
// platform, preventing tools like Dependabot from demoting it to indirect
// when they run `go mod tidy` on a non-darwin/arm64 host.
package main

import _ "github.com/klauspost/compress/zstd"
