# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the **Panel** server and VM infrastructure for deploying Treasury nodes. It provides a "1-click" deployment solution that handles initial Treasury installation, pairing with other nodes, and backup/restore procedures through a built-in UI. The system is designed to run on VMs in cloud environments (AWS, GCP) with no SSH access or manual configuration required.

The project consists of:

- **Go-based Panel Server**: HTTP API server that manages Treasury lifecycle
- **Next.js Web UI**: Static single-page application for activation and management
- **Bootable Container Image**: VM images built as bootable containers for various cloud platforms

## Development Commands

### Go Backend

```bash
# Run panel server in development mode
just dev

# Install panel binary
just install

# Run tests
just test

# Run linter and formatter
just lint

# Tidy Go modules (GOWORK=off for this repo)
just tidy
```

### Web Frontend (Next.js)

```bash
cd web

# Install dependencies
pnpm install

# Development server (hot reload on http://localhost:3000)
pnpm run dev

# Build static export
pnpm run build

# Run tests
pnpm test

# Run tests in watch mode
pnpm run test:watch
```

### Testing Individual Components

```bash
# Run a specific Go test
GOWORK=off go test -v ./pkg/bak/...

# Run a single test function
GOWORK=off go test -v -run TestSpecificFunction ./server/panel/...
```

### Docker/VM Development

```bash
# Build dev container image
just build-dev

# Run first dev VM instance
just run-dev-1

# Run second dev VM instance (for testing multi-node setups)
just run-dev-2

# Execute into running container
just exec-1  # or just exec-2

# Run panel server inside container
just dev-container n="1"
```

## Architecture Overview

### Panel Server (Go)

The Panel server is a Fiber-based HTTP API server that orchestrates the Treasury deployment lifecycle:

**Key Components:**

- `cmd/panel/main.go`: CLI entrypoint with cobra commands (`start`, `activate`, `reset`, etc.)
- `server/server.go`: Fiber app setup and server initialization
- `server/endpoints/`: API endpoint handlers organized by domain (activate, treasury, backup, service, etc.)
- `server/panel/`: Panel state management and persistence
- `pkg/`: Reusable packages for admin client, secrets, paths, genesis, etc.

**Activation Flow:**

1. API key activation (`/v1/activate/api-key`)
2. Binary installation (`/v1/activate/binaries`)
3. Network setup via Netbird (`/v1/activate/network`)
4. Backup key configuration (`/v1/activate/backup`)
5. Treasury generation and peer synchronization

**Service Management:**
The panel manages systemd services (`treasury.service`, `start-treasury.service`) and can start/stop/restart them via the `/v1/services/*` endpoints.

**Backup System:**
Uses age encryption with BIP39 mnemonic-based keys. Backups are stored in S3-compatible storage and can be restored through the UI.

### Web Frontend (Next.js + TypeScript)

The web UI is a static Next.js application that exports to `web/out/` and is served by the Go server at the root path.

**Key Components:**

- `pages/index.tsx`: Main entry point with tab-based UI
- `components/`: React components for each major feature
  - `ActivationPanel.tsx`: Step-by-step activation wizard
  - `BackupRestoreTab.tsx`: Backup management and restore
  - `StatusTab.tsx`: Treasury health and service status
  - `EncryptionAtRestTab.tsx`: Encryption-at-rest configuration
  - `InitialUsersTab.tsx`: User management
  - `TreasuryLogsTab.tsx`: Real-time logs viewer

**API Communication:**
The frontend calls the Panel server API at `NEXT_PUBLIC_API_HOST` (defaults to same host). During development, set this in `.env.local` to point to `http://localhost:7666` while running the Next.js dev server on port 3000.

**Static Export:**
The app uses `output: 'export'` in `next.config.js` to generate static HTML/JS/CSS files that are served by the Go server.

### State Management

**Panel State** (`/etc/panel/panel.json`):

- API key and node configuration
- Treasury ID and node ID (assigned during activation)
- Backup keys (age public keys)
- Encryption-at-rest secret reference
- Connector and API node flags

### VM and Container Architecture

The project builds bootable container images using [bootc](https://docs.fedoraproject.org/en-US/bootc/):

1. Base image built with platform-specific config (AWS/GCP)
2. Panel binary and web UI embedded in the image
3. Systemd services configured for automatic startup
4. Container converted to VM image using `bootc-image-builder`
5. VM image published to cloud provider marketplaces

**Build Process:**

```bash
# Build for AWS
BASE=aws docker buildx bake

# Build for GCP
BASE=gcp docker buildx bake
```

The VM runs the panel server on port 7666 and the user accesses it through a port forward for initial activation.

### GOWORK Environment

This repository uses `GOWORK=off` for all Go commands because the Panel is intended to eventually be split into a separate repository from the main Treasury codebase.

## API Endpoint Structure

All activation endpoints follow a POST-based flow:

- `/v1/activate/api-key` - Activates API key and fetches node assignment
- `/v1/activate/binaries` - Downloads Treasury, Signer, and Cord binaries
- `/v1/activate/network` - Joins Netbird network for peer connectivity
- `/v1/activate/backup` - Configures backup keys
- `/v1/activate/otel` - Configures observability collection

Treasury management endpoints:

- `POST /v1/treasury` - Generate new Treasury node
- `POST /v1/treasury/complete` - Complete Treasury setup after peers are synced
- `POST /v1/treasury/image` - Use specific Treasury container image
- `DELETE /v1/treasury` - Delete Treasury node (with confirmation)
- `GET /v1/treasury/healthy` - Health check with optional verbose output

Service control endpoints:

- `GET /v1/services` - List all systemd services
- `POST /v1/services/{service}/{action}` - Perform action (start/stop/restart/enable/disable)

Backup endpoints:

- `POST /v1/backup/snapshot` - Create snapshot backup
- `POST /v1/backup/restore` - Restore from snapshot
- `GET /v1/backup/list` - List available backups

## Testing Strategy

**Go Tests:**

- Unit tests in `*_test.go` files
- Use testify for assertions
- Mock systemd interactions where needed
- Test secret loading with different providers

**Frontend Tests:**

- Vitest for component tests
- React Testing Library for component interaction
- `vitest.setup.ts` configures jsdom environment

## Common Gotchas

- Always use `GOWORK=off` when running Go commands
- The panel server serves static files from `web/out/` - rebuild the frontend when making UI changes
- The `--treasury-user` flag defaults to `cordial` - this is the Linux user that runs Treasury
- Binary downloads are verified with Sigstore cosign signatures
- Multi-node activation requires all nodes to exchange peer information before completing
- The `start-treasury.service` keeps retrying if peer information isn't ready yet
