# Pulse

A fast, keyboard-first project management tool inspired by Linear.

## Quick Start

```bash
# Build from source
go build -o pulse ./cmd/pulse

# Start the server
./pulse start --addr localhost:3002

# Open in browser
open http://localhost:3002
```

## Features

- **Issue Management**: Create, update, and track issues with priorities, labels, and estimates
- **Kanban Board**: Visual workflow with drag-and-drop (coming soon)
- **Cycles**: Sprint planning with time-boxed iterations
- **Velocity Metrics**: Track team performance with cycle time, lead time, and completion rate
- **Keyboard Shortcuts**: Full keyboard navigation for power users
- **REST API**: Programmatic access for integrations

## Documentation

See [PRD.md](PRD.md) for detailed product requirements and roadmap.

## Project Structure

```
pulse/
├── cmd/
│   └── pulse/
│       └── main.go           # CLI entry point
├── internal/
│   └── server/
│       └── server.go         # Web server & API
├── docs/                      # Documentation
├── PRD.md                     # Product requirements
├── go.mod                     # Go module definition
└── Makefile                   # Build automation
```

## Building

```bash
# Build binary
make build

# Run tests
make test

# Run linter
make lint
```

## License

MIT
