# Pulse - Product Requirements Document

**Version:** 1.0.0
**Last Updated:** 2026-02-07
**Status:** Draft

---

## 1. Executive Summary

Pulse is a fast, keyboard-first project management tool inspired by Linear. Designed for development teams who need efficient issue tracking, sprint management, and velocity analytics without the complexity of enterprise tools.

### 1.1 Goals
- Provide sub-100ms UI interactions
- Enable complete keyboard-driven workflow
- Deliver real-time velocity metrics
- Support offline-first operation

### 1.2 Success Metrics
| Metric | Target | Timeline |
|--------|--------|----------|
| Time to create issue | < 2 seconds | v1.0 |
| Issue status change | < 100ms | v1.0 |
| Board render time | < 50ms | v1.0 |
| User retention (30d) | > 60% | v1.0 |

---

## 2. Core Features

### 2.1 Issue Management

#### 2.1.1 Issue Properties
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| ID | string | Auto | Unique identifier |
| Title | string | Yes | Short issue description |
| Description | string | No | Markdown-enabled details |
| Status | enum | Yes | backlog, todo, in_progress, done, canceled |
| Priority | enum | No | urgent(1), high(2), medium(3), low(4), none(0) |
| Assignee | string | No | User ID |
| Labels | array | No | List of label strings |
| Estimate | int | No | Story points (Fibonacci: 1,2,3,5,8,13,21) |
| Cycle | string | No | Associated sprint cycle |
| Parent | string | No | Parent issue ID for sub-issues |
| CreatedAt | timestamp | Auto | Creation time |
| UpdatedAt | timestamp | Auto | Last modification time |
| CompletedAt | timestamp | Auto | When status became "done" |

#### 2.1.2 Issue Relations
- **Blocks**: Issue A blocks Issue B (A must complete before B)
- **Blocked By**: Inverse of blocks
- **Duplicate**: Issue A is duplicate of Issue B
- **Relates To**: Related but not blocking

#### 2.1.3 Issue States
```
backlog    → Issues not yet planned
todo       → Ready to work on
in_progress→ Currently being worked
done       → Completed
canceled   → Won't be done
```

### 2.2 Workspaces

#### 2.2.1 Workspace Properties
| Property | Type | Description |
|----------|------|-------------|
| ID | string | Unique identifier |
| Name | string | Display name |
| Description | string | Optional description |
| Settings | object | Workspace configuration |
| Members | array | User IDs with access |

#### 2.2.2 Workspace Settings
```typescript
interface WorkspaceSettings {
  defaultAssignee: string | null
  autoCloseDays: number
  requireEstimate: boolean
  requireLabels: boolean
  cycleDuration: number // weeks
  issuePrefix: string
}
```

### 2.3 Cycles (Sprints)

#### 2.3.1 Cycle Properties
| Property | Type | Description |
|----------|------|-------------|
| ID | string | Unique identifier |
| Name | string | Display name (e.g., "Cycle 12") |
| StartDate | date | Sprint start |
| EndDate | date | Sprint end |
| IssueIDs | array | Planned issues |
| Status | enum | upcoming, active, completed |

#### 2.3.2 Cycle Scope Management
- Add/remove issues from active cycle
- Bulk update estimates
- Auto-balance capacity

### 2.4 Labels

#### 2.4.1 Default Labels
| Label | Color | Usage |
|-------|-------|-------|
| bug | #F85149 | Defects and issues |
| feature | #A371F7 | New functionality |
| improvement | #3FB950 | Enhancements |
| docs | #58A6FF | Documentation |
| urgent | #F85149 | High priority |
| breaking | #F85149 | Breaking changes |

#### 2.4.2 Label Properties
```typescript
interface Label {
  name: string
  color: string  // Hex color code
  description?: string
}
```

---

## 3. Analytics & Metrics

### 3.1 Velocity Metrics

#### 3.1.1 Cycle Velocity
```typescript
interface CycleVelocity {
  cycleId: string
  pointsPlanned: number      // Sum of estimates in cycle
  pointsCompleted: number    // Sum of completed estimates
  issuesPlanned: number
  issuesCompleted: number
  completionRate: number     // percentage
  carryover: number          // Points not completed
}
```

#### 3.1.2 Cycle Time
```
Cycle Time = CompletedAt - StartedAt (In Progress)
```
- Track per-issue for continuous improvement
- Average, P50, P90, P99 distributions

#### 3.1.3 Lead Time
```
Lead Time = CompletedAt - CreatedAt
```
- Measures time from request to delivery
- Key metric for customer satisfaction

### 3.2 Quality Metrics

#### 3.2.1 Bug Metrics
- Bug count by cycle
- Bug escape rate (bugs found in production)
- Bug reopen rate

#### 3.2.2 Aging Report
- Issues open > N days
- Issues in status > N days

---

## 4. User Experience

### 4.1 Keyboard Shortcuts

#### 4.1.1 Global
| Shortcut | Action |
|----------|--------|
| `C` | Create new issue |
| `/` | Quick search |
| `Esc` | Close modal / clear selection |
| `?` | Show all shortcuts |

#### 4.1.2 Navigation
| Shortcut | Action |
|----------|--------|
| `J` | Next issue in list |
| `K` | Previous issue in list |
| `G then G` | Go to board |
| `G then I` | Go to issues |
| `G then C` | Go to cycles |

#### 4.1.3 Issue Actions
| Shortcut | Action |
|----------|--------|
| `Space` | Assign to me |
| `P` | Toggle priority menu |
| `L` | Toggle labels menu |
| `N` | Next status |
| `P` | Previous status |
| `M` | Assign to... |
| `E` | Edit estimate |
| `D` | Set due date |

### 4.2 Views

#### 4.2.1 Board View
- Horizontal columns by status
- Drag-and-drop issue movement
- Swimlanes by assignee or label
- Compact/expanded card modes

#### 4.2.2 List View
- Sortable columns
- Bulk actions
- Column configuration
- Saved filters

#### 4.2.3 Timeline View
- Gantt-style visualization
- Issue dependencies
- Milestone tracking

---

## 5. API Specification

### 5.1 REST API

#### 5.1.1 Workspaces
```
GET    /api/workspaces           List all workspaces
POST   /api/workspaces           Create workspace
GET    /api/workspaces/{id}      Get workspace
PUT    /api/workspaces/{id}      Update workspace
DELETE /api/workspaces/{id}      Delete workspace
```

#### 5.1.2 Issues
```
GET    /api/issues               List issues (with filters)
POST   /api/issues               Create issue
GET    /api/issues/{id}          Get issue
PUT    /api/issues/{id}          Update issue
DELETE /api/issues/{id}          Delete issue
POST   /api/issues/{id}/move     Change status
```

#### 5.1.3 Cycles
```
GET    /api/cycles               List cycles
POST   /api/cycles               Create cycle
GET    /api/cycles/{id}          Get cycle
PUT    /api/cycles/{id}          Update cycle
POST   /api/cycles/{id}/complete Complete cycle
```

#### 5.1.4 Analytics
```
GET    /api/metrics?workspace_id={id}  Get velocity metrics
GET    /api/metrics/cycle-time          Get cycle time distribution
GET    /api/metrics/lead-time           Get lead time distribution
```

### 5.2 WebSocket Events

```typescript
// Issue updates
EVENT: issue:updated
DATA: { id, changes, updatedBy }

// Issue moved between states
EVENT: issue:moved
DATA: { id, fromStatus, toStatus, movedBy }

// New issue created
EVENT: issue:created
DATA: { issue, createdBy }

// Comment added
EVENT: comment:added
DATA: { issueId, comment, author }
```

---

## 6. Data Model

### 6.1 Entity Relationships

```
Workspace
├── Issues
│   └── (can have parent issue)
├── Cycles
│   └── Issues (many-to-many)
├── Labels
├── Members
└── Settings

Issue
├── Labels (many-to-many)
├── Comments
├── Relations (blocks, blocked-by, duplicates)
├── Cycle (optional)
└── SubIssues
```

### 6.2 Storage

#### 6.2.1 Local-First (v1.0)
- SQLite for local storage
- IndexedDB for browser cache
- Conflict-free replication for sync

#### 6.2.2 Server Storage (future)
- PostgreSQL for persistence
- Redis for real-time events
- S3 for attachments

---

## 7. Technical Architecture

### 7.1 System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      Client (SPA)                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐   │
│  │ Board   │  │ Issues  │  │ Cycles  │  │ Analytics   │   │
│  │ View    │  │ List    │  │ View    │  │ Dashboard   │   │
│  └─────────┘  └─────────┘  └─────────┘  └─────────────┘   │
└────────────────────────┬──────────────────────────────────┘
                         │ WebSocket / HTTP
┌────────────────────────┴──────────────────────────────────┐
│                      API Gateway                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐    │
│  │ Auth        │  │ REST        │  │ WebSocket       │    │
│  │ Middleware  │  │ Handler     │  │ Handler         │    │
│  └─────────────┘  └─────────────┘  └─────────────────┘    │
└────────────────────────┬──────────────────────────────────┘
                         │
┌────────────────────────┴──────────────────────────────────┐
│                    Services                                 │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐   │
│  │ Issue      │  │ Cycle      │  │ Analytics          │   │
│  │ Service    │  │ Service    │  │ Service            │   │
│  └────────────┘  └────────────┘  └────────────────────┘   │
└────────────────────────┬──────────────────────────────────┘
                         │
┌────────────────────────┴──────────────────────────────────┐
│                    Data Layer                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐   │
│  │ SQLite     │  │ Redis      │  │ File Storage       │   │
│  │ (local)    │  │ (cache)    │  │ (attachments)      │   │
│  └────────────┘  └────────────┘  └────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 Tech Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Frontend | React + TypeScript | Strong typing, large ecosystem |
| Backend | Go | Performance, simplicity |
| Database | SQLite (local), PostgreSQL (server) | Reliable, well-understood |
| State | Zustand | Minimal, fast |
| Styling | Tailwind CSS | Rapid development |
| Testing | Playwright | E2E coverage |

---

## 8. Milestones

### 8.1 v0.1 - MVP (Week 1)
- [ ] Issue CRUD operations
- [ ] Basic board view
- [ ] Local SQLite storage
- [ ] Keyboard shortcuts (basic)

### 8.2 v0.2 - Core Features (Week 2)
- [ ] Cycle management
- [ ] Labels and filtering
- [ ] Velocity metrics
- [ ] Search functionality

### 8.3 v0.3 - Polish (Week 3)
- [ ] Full keyboard navigation
- [ ] Drag-and-drop board
- [ ] Issue relations
- [ ] Performance optimization

### 8.4 v1.0 - Release (Week 4)
- [ ] User accounts
- [ ] Team workspaces
- [ ] API v1
- [ ] Documentation
- [ ] E2E test coverage > 80%

---

## 9. Open Questions

1. **Authentication**: Should we support SSO from day one?
2. **Self-hosting**: Is this a priority for v1.0?
3. **Integrations**: GitHub sync - when to implement?
4. **Offline**: Sync conflicts - what's the strategy?

---

## 10. Appendix

### 10.1 Competitor Analysis

| Feature | Pulse | Linear | Jira | Asana |
|---------|-------|--------|------|-------|
| Keyboard-first | Yes | Yes | Partial | No |
| Cycle metrics | Yes | Yes | Complex | Basic |
| Sub-issues | Yes | Yes | Yes | No |
| Self-hosted | Future | No | Yes | No |
| Free tier | Yes (self) | Limited | Free | Limited |
| Setup time | <5min | <5min | Hours | 30min |

### 10.2 References
- [Linear Documentation](https://linear.app/docs)
- [Atlassian Design](https://atlassian.design)
- [Figma keyboard shortcuts](https://shortcuts.figma.dev/)

---

**Document Owner:** Product Team
**Reviewers:** Engineering, Design
**Next Review:** 2026-02-14
