# Architecture (Native Executable Path)

## Goal

Replace Visual Studio/.NET dependency for runtime clients with a self-running native executable that launches a local UI.

## Core principles

1. Single process executable for client usage.
2. Embedded UI assets to avoid separate web server setup.
3. Local HTTP loopback interface for UI/backend communication.
4. No dependency on proprietary IDE for running the app.

## Runtime flow

1. User launches executable.
2. App starts local server on `127.0.0.1` dynamic port.
3. App opens default browser to local UI.
4. UI calls internal endpoints (e.g., `/api/health`).
