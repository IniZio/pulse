# Pulse Development Roadmap

This document tracks parallel development tracks for rapid iteration and dogfooding.

## Development Philosophy

**Ship fast, iterate often.** Every commit should be usable. No big-bang releases.

---

## Parallel Tracks

### Track A: Core Issue Engine (Priority: 🔴 P0)
- [ ] SQLite persistence layer
- [ ] Issue CRUD with optimistic UI
- [ ] Status workflow transitions
- [ ] Labels and priorities
- [ ] Search and filtering

### Track B: Analytics Engine (Priority: 🟡 P1)
- [ ] Velocity calculation
- [ ] Cycle time tracking
- [ ] Lead time metrics
- [ ] Quality dashboards

### Track C: UX Polish (Priority: 🟡 P1)
- [ ] Keyboard shortcuts
- [ ] Drag-and-drop board
- [ ] Issue detail modal
- [ ] Mobile responsive

### Track D: Infrastructure (Priority: 🟢 P2)
- [ ] Docker container
- [ ] CI/CD pipeline
- [ ] Automated testing
- [ ] Performance benchmarks

---

## Sprint 0: Usable Core (Week 1)

**Goal**: Track Pulse development in Pulse itself

### Milestone: Track 1 Issues
```bash
# Track this sprint's work
./pulse issues create --title "Implement SQLite persistence" --labels infrastructure --estimate 3
./pulse issues create --title "Add keyboard shortcuts" --labels ux --estimate 2
./pulse issues create --title "Create CI/CD pipeline" --labels devops --estimate 2
```

### Deliverables
| Feature | Status | Notes |
|---------|--------|-------|
| SQLite persistence | ⏳ | In progress |
| Issue create/read | ✅ | Working |
| Status workflow | ⏳ | Pending |
| Search | ⏳ | Pending |
| Local storage | ✅ | Working |

---

## Track A: Core Issue Engine

### A.1 Data Layer
```
pulse/internal/db/
├── db.go              # SQLite wrapper
├── migrations.go      # Schema migrations
├── issues.go          # Issue queries
├── workspaces.go      # Workspace queries
└── cycles.go          # Cycle queries
```

### A.2 Schema
```sql
-- Core tables
CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    settings TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE issues (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'backlog',
    priority INTEGER DEFAULT 0,
    assignee_id TEXT,
    estimate INTEGER,
    cycle_id TEXT,
    labels TEXT, -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE cycles (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL,
    start_date DATETIME,
    end_date DATETIME,
    status TEXT DEFAULT 'upcoming',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE INDEX idx_issues_workspace ON issues(workspace_id);
CREATE INDEX idx_issues_status ON issues(status);
CREATE INDEX idx_issues_assignee ON issues(assignee_id);
```

### A.3 API Endpoints
```
GET    /api/v1/workspaces
POST   /api/v1/workspaces
GET    /api/v1/workspaces/:id
PUT    /api/v1/workspaces/:id
DELETE /api/v1/workspaces/:id

GET    /api/v1/issues
POST   /api/v1/issues
GET    /api/v1/issues/:id
PUT    /api/v1/issues/:id
DELETE /api/v1/issues/:id
PATCH  /api/v1/issues/:id/status  -- Move between states

GET    /api/v1/cycles
POST   /api/v1/cycles
GET    /api/v1/cycles/:id
PATCH  /api/v1/cycles/:id/complete
```

---

## Track B: Analytics Engine

### B.1 Velocity Calculator
```go
type VelocityMetrics struct {
    CycleID           string
    PointsPlanned     int
    PointsCompleted   int
    CompletionRate    float64  // 0-100
    CarryoverPoints   int
}

func CalculateVelocity(cycleID string) (*VelocityMetrics, error) {
    // Sum estimates for all issues in cycle
    // Count completed issues
    // Calculate completion rate
}
```

### B.2 Cycle Time Tracker
```go
type CycleTimeMetrics struct {
    AverageHours  float64
    P50Hours     float64
    P90Hours     float64
    P99Hours     float64
}

func CalculateCycleTime(workspaceID string) (*CycleTimeMetrics, error) {
    // For each issue: completed_at - started_at (when status became 'in_progress')
    // Calculate percentiles
}
```

### B.3 Quality Metrics
```go
type QualityMetrics struct {
    BugCount           int
    BugRate            float64  // bugs per 100 issues
    ReopenRate         float64
    EscapedBugs        int     // found in production
}
```

---

## Track C: UX Enhancements

### C.1 Keyboard Shortcuts
| Shortcut | Action | Context |
|----------|--------|---------|
| `c` | Create issue | Global |
| `/` | Search | Global |
| `j/k` | Navigate issues | Issue list |
| `n` | Next status | Issue selected |
| `p` | Previous status | Issue selected |
| `space` | Assign to me | Issue selected |
| `e` | Edit estimate | Issue selected |
| `l` | Add label | Issue selected |
| `enter` | Open detail | Issue selected |
| `escape` | Close modal | Modal open |

### C.2 Drag-and-Drop
Use standard HTML5 drag-and-drop for Kanban board:
- Drag issue card to change status
- Visual feedback on valid drop targets
- Animated reordering

### C.3 Issue Detail Modal
```
┌─────────────────────────────────────────┐
│  [P1] Fix login authentication          │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                         │
│  Status:    [ In Progress     ▼]        │
│  Assignee:  [ @user           ▼]        │
│  Estimate:  [ 3               ]        │
│  Cycle:     [ Cycle 12        ▼]        │
│                                         │
│  Labels:    [bug] [security] [+ add]    │
│                                         │
│  ┌─ Description ──────────────────────┐  │
│  │ The login flow fails when...     │  │
│  └──────────────────────────────────┘  │
│                                         │
│  ┌─ Sub-issues ───────────────────────┐  │
│  │ [ ] Fix validation logic          │  │
│  │ [ ] Add unit tests               │  │
│  └──────────────────────────────────┘  │
│                                         │
│  [Move to Next] [Archive] [Delete]     │
└─────────────────────────────────────────┘
```

---

## Track D: Infrastructure

### D.1 Docker Container
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /pulse ./cmd/pulse

FROM alpine:latest
RUN addgroup -g 1000 app && adduser -u 1000 -G app -s /bin/sh -D app
USER app
COPY --from=builder /pulse /usr/local/bin/
EXPOSE 3002
CMD ["pulse", "start", "--addr", "0.0.0.0:3002"]
```

### D.2 CI/CD Pipeline (.github/workflows/ci.yml)
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: go-version: '1.24'
      - run: go test ./... -race -cover
      - run: golangci-lint run
  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go build -o pulse ./cmd/pulse
      - uses: actions/upload-artifact@v4
        with:
          name: pulse
          path: pulse
  release:
    needs: build
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v4
      - uses: softprops/action-gh-release@v1
        with:
          files: pulse
```

### D.3 Performance Benchmarks
```go
// benchmarks/benchmarks_test.go
func BenchmarkIssueCreate(b *testing.B) {
    db := setupTestDB()
    for b.N {
        createIssue(db)
    }
}

func BenchmarkIssueList(b *testing.B) {
    db := setupTestDB()
    for b.N {
        listIssues(db, statusFilter)
    }
}

// Target: < 10ms for issue list, < 50ms for board render
```

---

## Development Workflow

### Daily Cycle
1. **Morning**: Check Pulse board for assigned issues
2. **During dev**: Create issues for tasks, update status
3. **Review**: Move completed items to Done
4. **End of day**: Review velocity metrics

### Creating Issues
```bash
# Quick create for bug
./pulse issues create --title "Login fails on mobile" --priority urgent --labels bug --estimate 2

# Create from branch
git checkout -b fix/login-mobile
./pulse issues create --title "Fix login on mobile" --link-branch
```

### Tracking Progress
```bash
# Show my issues
./pulse issues list --assignee=me

# Show velocity this cycle
./pulse metrics velocity

# Show cycle time trend
./pulse metrics cycle-time --last-30-days
```

---

## Roadmap Timeline

```
Week 1          Week 2          Week 3          Week 4
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Sprint 0: Usable Core
├── Issue CRUD ✅├── Status workflow ─┤
├── SQLite 📦 ──┤                ├── Analytics Engine ─┤
└── Search ─────┴─────────────────┴── Dashboard ───────┘
                                            │
Sprint 1: First Release ───────────────────┤
├── Keyboard shortcuts ◌    ◌───────────────┤
├── CI/CD pipeline ────────◌───────────────┤
└── Docker image ─────────◌───────────────┘

◌ = Parallel work
```

---

## Success Metrics

### v0.1 (Usable Core)
- [ ] Can create and track issues
- [ ] Status workflow works
- [ ] Local SQLite storage
- [ ] Issues load in <100ms

### v0.5 (First Release)
- [ ] Keyboard shortcuts complete
- [ ] Velocity metrics working
- [ ] CI/CD passing
- [ ] Self-hosting works

### v1.0 (Production Ready)
- [ ] 80% test coverage
- [ ] <50ms p95 for all operations
- [ ] Documentation complete
- [ ] Performance benchmarks passing
