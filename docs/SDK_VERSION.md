# GitHub Copilot SDK Version

## Current Version

Kopilot uses **GitHub Copilot SDK v0.1.24 (stable release)**.

```go
require (
    github.com/github/copilot-sdk/go v0.1.24
    // ...
)
```

## Version History

- **v0.1.24 (stable)** - Used since project initialization (January 2026)
  - First stable release of the Copilot SDK
  - No preview versions were ever used in this project
  - Compatible with GitHub Copilot CLI v0.0.409

## Compatibility

The SDK version v0.1.24 is compatible with:

- **GitHub Copilot CLI**: v0.0.409 (recommended)
- **Go**: 1.21 or later (project uses 1.26+)

## Notes

- This project was initialized with SDK v0.1.24 stable from the start
- No upgrade from preview versions occurred
- For SDK compatibility details, see the official [Copilot SDK documentation](https://github.com/github/copilot-sdk)

## Verification

To verify the SDK version:

```bash
# Check go.mod
grep copilot-sdk go.mod

# Check installed version
go list -m github.com/github/copilot-sdk/go
```
