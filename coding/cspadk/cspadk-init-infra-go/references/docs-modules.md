# Per-Module Documentation Template

## modules/\<module-name\>.md — Module Documentation

| Field | Content |
|-------|---------|
| Purpose | In-depth view of a single module: internal architecture, sub-module breakdown, core flows, complete code index |
| When to Create | For every module/sub-project/package/service with its own directory |

### Section Structure

Every module documentation file must include all 8 sections below:

#### Section 1: Module Overview

- Must be ≤ 3 sentences
- States what this module does and its role in the system

```markdown
## Module Overview

ticket-service handles ticket routing and lifecycle management in the CSP platform. It provides CRUD operations for tickets and implements rule-based routing to assign tickets to the most appropriate agent.
```

#### Section 2: Internal Architecture

- Directory tree with annotations for every directory
- Architectural pattern description
- DI/wiring mechanism if applicable

```markdown
## Internal Architecture

internal/ticket-service/
├── handler/     # HTTP/RPC request handlers
├── service/     # Business logic layer
├── repository/  # Data access (MySQL + Redis)
├── model/       # Data models and DTOs
└── config/      # Configuration loading

**Pattern**: Layered (handler → service → repository).

**Wiring**: Dependencies are injected via constructor functions. `NewTicketService(repo TicketRepository) *TicketService`.
```

#### Section 3: Sub-Module Breakdown

- Table: Sub-Module / Path / Responsibility / Key Exports

```markdown
## Sub-Module Breakdown

| Sub-Module | Path | Responsibility | Key Exports |
|-----------|------|---------------|-------------|
| Handler | handler/ | HTTP/RPC request handlers | CreateTicket, GetTicket, ListTickets |
| Service | service/ | Business logic and routing | TicketService, RouteTicket |
| Repository | repository/ | Data access (MySQL + Redis) | MySQLRepo, RedisCache |
| Model | model/ | Data models and DTOs | Ticket, TicketRequest, TicketResponse |
| Config | config/ | Configuration loading | LoadConfig |
```

#### Section 4: Core Execution Flows

- For each flow: name, entry point (file:function), step-by-step call chain, error handling path

```markdown
## Core Execution Flows

### Flow: Ticket Creation

- **Entry Point**: `handler/ticket.go:CreateTicket()`
- **Call Chain**:
  1. `CreateTicket()` validates request and extracts parameters
  2. Delegates to `service/ticket.go:Create()`
  3. `Create()` applies business rules and calls `service/routing.go:RouteTicket()`
  4. `RouteTicket()` evaluates routing rules and selects agent queue
  5. `repository/mysql.go:Save()` persists the ticket
  6. `repository/redis.go:Set()` caches the ticket state
- **Error Handling**: Validation errors return 400; routing failures log warning and assign to default queue; database errors return 500 with retry suggestion.

### Flow: Ticket Routing

- **Entry Point**: `service/routing.go:RouteTicket()`
- **Call Chain**:
  1. `RouteTicket()` loads routing rules from config
  2. Evaluates rules against ticket metadata (channel, language, category)
  3. Matches to agent queue by skill and availability
  4. Returns the top available agent
- **Error Handling**: No matching rules → default queue. All agents busy → pending queue with timeout.
```

#### Section 5: Code Location Index

- Table: File Path / Responsibility / Key Exports
- Must cover ALL source files in the module

```markdown
## Code Location Index

| File Path | Responsibility | Key Exports |
|-----------|---------------|-------------|
| handler/ticket.go | CRUD handlers | CreateTicket, GetTicket, ListTickets, UpdateTicket |
| service/ticket.go | Ticket business logic | Create, Get, List, Update |
| service/routing.go | Routing algorithm | RouteTicket, EvaluateRules |
| repository/mysql.go | MySQL data access | Save, FindByID, FindByStatus |
| repository/redis.go | Redis caching | Set, Get, Delete |
| model/ticket.go | Data models | Ticket, TicketRequest, TicketResponse |
| model/routing.go | Routing models | RoutingRule, AgentQueue |
| config/config.go | Configuration loading | LoadConfig, Config |
```

#### Section 6: Configuration Reference

- Config files, env vars, feature flags

```markdown
## Configuration Reference

| Config | Location | Description | Default |
|--------|----------|-------------|---------|
| config.yaml | config/ | Application configuration file | Auto-generated |
| DB_HOST | Environment | MySQL host address | localhost:3306 |
| REDIS_ADDR | Environment | Redis address | localhost:6379 |
| ROUTING_TIMEOUT | Environment | Max routing evaluation time (seconds) | 5 |
| LOG_LEVEL | Environment | Logging verbosity | info |
```

#### Section 7: Testing Guide

- Test directory structure
- Exact command to run this module's tests in isolation
- Test conventions

```markdown
## Testing Guide

**Test Directory**: Co-located `*_test.go` files alongside source files

**Run in Isolation**:
```bash
go test ./internal/ticket-service/...
```

**Coverage**:
```bash
go test ./internal/ticket-service/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Conventions**:
- Test files named `*_test.go` in the same package directory
- Use `xxx_test` package for black-box testing
- Table-driven tests for multiple scenarios
- Use `testify/assert` for assertions
- Use `go.uber.org/mock` for interface mocks
```

#### Section 8: Dependencies

- Table: Dependency / Version / Purpose / Internal or External

```markdown
## Dependencies

| Dependency | Version | Purpose | Type |
|-----------|---------|---------|------|
| code.byted.org/kitex | latest | RPC framework | External (Internal registry) |
| code.byted.org/hertz | latest | HTTP framework | External (Internal registry) |
| github.com/stretchr/testify | latest | Test assertions | External |
| github.com/go-sql-driver/mysql | latest | MySQL driver | External |
| github.com/redis/go-redis | latest | Redis client | External |
```

---

### Content Rules

1. Core execution flows must reference actual file paths and function names (e.g., `handler/ticket.go:CreateTicket()`)
2. Code location index must cover ALL source files in the module — no file may be omitted
3. Testing guide must include the exact command to run this module's tests in isolation (e.g., `go test ./internal/ticket-service/...`)

### Quality Checks

- [ ] Module overview ≤ 3 sentences
- [ ] Internal architecture includes directory tree with annotations
- [ ] Every sub-module has a row in the breakdown table
- [ ] At least one core execution flow is described with file:function references
- [ ] Code location index covers all source files
- [ ] Testing guide includes isolation run command
- [ ] Dependencies table distinguishes internal vs. external
