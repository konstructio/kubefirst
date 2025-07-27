# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Building and Running
- **Build**: `go build` (creates `kubefirst` binary)
- **Run from source**: `go run .` (e.g., `go run . civo create`)
- **Run compiled**: `./kubefirst` (after building)

### Testing
- **Run all tests**: `go test -v ./...`
- **Run short tests**: `go test -short -v ./...` (used in CI)
- **Run specific test**: `go test -v ./path/to/package -run TestName`

### Linting
- **Lint**: `golangci-lint run` (uses `.golangci.yaml` configuration with 70+ linters)
- **Format code**: `gofmt -w .` or `gofumpt -w .`

### Development with Dependencies
- **Use development gitops-template**: Add flags `--gitops-template-url https://github.com/konstructio/gitops-template --gitops-template-branch main`
- **Use local kubefirst-api**: Export `K1_CONSOLE_REMOTE_URL="http://localhost:3000"` and run both kubefirst-api and console locally
- **For k3d development**: Add replace directive in go.mod: `github.com/konstructio/kubefirst-api vX.X.XX => /path-to/kubefirst-api/`

## Architecture

### Command Structure
The CLI uses Cobra with modular commands for each cloud provider:
- `cmd/` contains provider-specific commands (aws, azure, civo, digitalocean, google, k3d, k3s, vultr, akamai)
- Each provider has its own package with create/destroy/root commands
- Common functionality is shared through `internal/` packages

### Key Internal Packages
- **catalog**: Application catalog management for marketplace apps
- **cluster**: Cluster creation and management logic
- **progress**: BubbleTea-based terminal UI for interactive installations
- **provision**: Core provisioning logic that orchestrates cluster creation
- **gitShim**: Git operations wrapper for repository management
- **segment**: Analytics tracking (can be disabled)
- **utilities**: Shared helpers and constants

### Core Dependencies
- **kubefirst-api**: External API package that handles most cluster operations
- **runtime**: Used for k3d local development (being phased out)
- **BubbleTea**: Terminal UI framework for interactive progress display
- **Viper/Cobra**: Configuration and CLI framework

### Error Handling Pattern
All errors should be wrapped with context using `fmt.Errorf("meaningful message: %w", err)`. The codebase follows a pattern of returning errors up the call stack rather than using log.Fatal.

### Logging
- Dual logging system: standard log package and zerolog
- Logs are written to `~/.k1/logs/`
- Console output uses formatted messages for user feedback
- Debug logging available with appropriate flags

### GitOps Integration
Kubefirst creates complete GitOps platforms with:
- ArgoCD for continuous deployment
- Metaphor repositories for application management
- GitHub/GitLab integration for repository management
- Terraform for infrastructure as code

### Configuration
- CLI configurations stored in `~/.k1/` directory
- Uses Viper for configuration management
- Environment variables prefixed with `K1_` or `KUBEFIRST_`