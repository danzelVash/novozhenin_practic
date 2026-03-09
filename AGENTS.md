# AGENTS.md

## Project Overview
- Language: Go (`go 1.24.0`)
- Module: `github.com/novozhenin/practic`
- Purpose: voice-controlled servo system with two binaries:
  - `master`: records audio, applies VAD, calls neuroservice, publishes commands
  - `slave`: receives commands and moves the servo
- Transports: `grpc`, `websocket`, `mqtt`

## Repository Layout
- `cmd/master`, `cmd/slave`: entrypoints
- `internal/master`: master app, config, recorder, VAD, neuro gateway
- `internal/slave`: slave app, config, servo control
- `internal/transport`: transport interfaces and implementations
- `pkg/pb`: generated protobuf/gRPC code
- `proto`: protobuf sources
- `bench`: transport benchmarks and reporting helpers

## Key Runtime Flow
- `cmd/master/main.go` loads flags and runs `internal/master.App`
- `internal/master/app.go` pipeline: recorder -> VAD -> neuro gateway -> transport publisher
- `cmd/slave/main.go` loads flags and runs `internal/slave.App`
- `internal/slave/app.go` initializes servo, subscribes via selected transport, retries on disconnect

## Build And Validation
- Build all: `make build-all`
- Build master only: `make build-master`
- Build slave only: `make build-slave`
- Regenerate protobuf: `make proto`
- Run tests: `go test ./...`

## Implementation Notes
- Transport selection is flag-driven. Keep feature behavior aligned across `grpc`, `websocket`, and `mqtt`.
- `master` currently dials the neuroservice over gRPC in `internal/master/app.go`.
- `slave` reconnection behavior is implemented in `internal/slave/app.go`; preserve retry semantics when changing subscriber logic.
- Generated files under `pkg/pb` should not be edited manually. Update `proto/*.proto` and regenerate instead.
- Hardware/audio integrations may be environment-specific. Prefer isolating changes so package-level tests remain runnable without device access.

## Editing Guidelines
- Prefer focused changes that preserve the current CLI flags and transport names.
- When changing shared transport behavior, inspect both publisher and subscriber implementations.
- If servo or recorder APIs change, verify both initialization and shutdown paths.
