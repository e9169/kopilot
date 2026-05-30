//go:build tools

package main

import _ "github.com/securego/gosec/v2/cmd/gosec" // pin gosec in go.sum so the CI action uses a verified version
