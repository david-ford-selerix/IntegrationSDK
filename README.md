# IntegrationSDK (Modernized Bootstrap)

This repository originally centered around a legacy .NET/Visual Studio sample stack.
To make onboarding easier, we've added a **self-running executable app path** that:

- does **not** require Visual Studio,
- launches a UI automatically when run,
- bundles core project docs into a local knowledge repository.

> The legacy .NET sample code remains in place for reference while migration proceeds.

## Quick start (no build tooling required)

### 1. Download a prebuilt executable

Go to **GitHub Releases** and download the binary that matches your platform:

- `integrationsdk-windows-amd64.exe`
- `integrationsdk-linux-amd64`
- `integrationsdk-linux-arm64`
- `integrationsdk-darwin-amd64`
- `integrationsdk-darwin-arm64`

These files are produced automatically by the GitHub release workflow whenever a release is published.

### 2. Run

On macOS/Linux:

```bash
chmod +x ./integrationsdk-<platform>
./integrationsdk-<platform>
```

On Windows, double-click `integrationsdk-windows-amd64.exe` (or run it from PowerShell / Command Prompt).

When launched, the app starts a local web UI and opens your default browser automatically.

## Build from source (optional)

If you want to build locally instead:

```bash
make build
```

> Prerequisite: Go must be installed and available on your `PATH`.

This produces:

- `dist/integrationsdk` on Linux/macOS
- `dist/integrationsdk.exe` on Windows

## Why this path

- **No proprietary IDE dependency** for clients.
- **Single executable distribution model**.
- **Precompiled release assets** so users can run immediately.
- **Embedded UI + docs** so the app can be handed off as one runnable artifact.

## Project layout (new)

- `cmd/integrationsdk-app/` – executable entrypoint
- `cmd/integrationsdk-app/ui/` – embedded static UI
- `knowledge/` – essential docs repository for implementers
- `Makefile` – common local build/run commands
- `.github/workflows/release-build.yml` – automatic multi-platform release builds

## Next migration steps

1. Port required .NET integration flows into this executable backend.
2. Expand `knowledge/` with API compatibility notes and examples.
