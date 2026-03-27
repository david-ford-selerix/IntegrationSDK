# IntegrationSDK (Modernized Bootstrap)

This repository originally centered around a legacy .NET/Visual Studio sample stack.
To make onboarding easier, we've added a **self-running executable app path** that:

- does **not** require Visual Studio,
- launches a UI automatically when run,
- bundles core project docs into a local knowledge repository.

> The legacy .NET sample code remains in place for reference while migration proceeds.

## Quick start (new path)

### 1. Build a native executable

From repo root:

```bash
make build
```

> Prerequisite: Go must be installed and available on your `PATH`.  
> Quick check: `go version`

This produces:

- `dist/integrationsdk` on Linux/macOS
- `dist/integrationsdk.exe` on Windows

If `make build` fails with `CreateProcess ... failed` on Windows, Git Bash could not find `go` in `PATH`. Install Go from <https://go.dev/dl/>, then restart your terminal.

### 2. Run

```bash
make run
```

When launched, the app starts a local web UI and opens your default browser automatically.

## Why this path

- **No proprietary IDE dependency** for clients.
- **Single executable distribution model**.
- **Embedded UI + docs** so the app can be handed off as one runnable artifact.

## Project layout (new)

- `cmd/integrationsdk-app/` – executable entrypoint
- `cmd/integrationsdk-app/ui/` – embedded static UI
- `knowledge/` – essential docs repository for implementers
- `Makefile` – common build/run commands

## Next migration steps

1. Port required .NET integration flows into this executable backend.
2. Add platform-targeted release packaging in CI.
3. Expand `knowledge/` with API compatibility notes and examples.
