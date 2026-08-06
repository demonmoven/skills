# Per-Module Documentation Template (Java/Spring Boot)

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

ticket-application is the Spring Boot app managing ticket processing in the CSP platform. It exposes REST APIs for ticket lifecycle operations and delegates business logic to the service layer. It serves as the primary backend service for the ticket domain.
```

#### Section 2: Internal Architecture

- Directory tree with annotations for every directory
- Architectural pattern description (e.g., Spring Boot layered, hexagonal)
- DI/wiring mechanism if applicable

```markdown
## Internal Architecture

ticket-application/
├── src/main/java/com/csp/ticket/
│   ├── controller/       # REST API controllers
│   ├── service/          # Business logic
│   ├── repository/       # JPA repositories
│   ├── entity/           # JPA entities
│   ├── config/           # Spring configuration
│   └── TicketApplication.java
├── src/main/resources/
│   ├── application.yml            # Default configuration
│   └── application-*.yml          # Profile-specific configuration
└── src/test/java/com/csp/ticket/

**Pattern**: Spring Boot layered (controller → service → repository).
**Wiring**: Spring Boot auto-configuration with `@ComponentScan`; dependencies injected via constructor injection (`@Autowired` or `@RequiredArgsConstructor`).
```

#### Section 3: Sub-Module Breakdown

- Table: Sub-Module / Path / Responsibility / Key Exports

```markdown
## Sub-Module Breakdown

| Sub-Module | Path | Responsibility | Key Exports |
|------------|------|----------------|-------------|
| Controller | src/main/java/.../controller/ | REST API endpoints | TicketController, TicketQueryController |
| Service | src/main/java/.../service/ | Business logic and orchestration | TicketService, NotificationService |
| Repository | src/main/java/.../repository/ | Data access via JPA | TicketRepository, TicketHistoryRepository |
| Entity | src/main/java/.../entity/ | JPA entity definitions | Ticket, TicketHistory |
| Config | src/main/java/.../config/ | Spring configuration beans | SecurityConfig, SwaggerConfig |
```

#### Section 4: Core Execution Flows

- For each flow: name, entry point (file:function), step-by-step call chain, error handling path
- Must reference actual file paths and function names

```markdown
## Core Execution Flows

### Flow: Ticket Creation

- **Entry Point**: `controller/TicketController.java → createTicket()`
- **Call Chain**:
  1. `controller/TicketController.java → createTicket()` receives HTTP request
  2. `service/TicketService.java → validateAndCreate()` validates input and creates entity
  3. `repository/TicketRepository.java → save()` persists the ticket
  4. `service/NotificationService.java → sendConfirmation()` dispatches confirmation
- **Error Handling**: Returns 400 for validation failure, 500 for persistence failure

### Flow: Ticket Assignment

- **Entry Point**: `controller/TicketController.java → assignTicket()`
- **Call Chain**:
  1. `controller/TicketController.java → assignTicket()` receives HTTP request
  2. `service/TicketService.java → assignTicket()` resolves assignee and updates entity
  3. `repository/TicketRepository.java → save()` persists the updated ticket
  4. `service/NotificationService.java → notifyAssignee()` sends assignment notification
- **Error Handling**: Returns 404 if ticket not found, 400 if assignee invalid, 500 for persistence failure
```

#### Section 5: Code Location Index

- Table: File Path / Responsibility / Key Exports
- Must cover ALL source files in the module — no file may be omitted

```markdown
## Code Location Index

| File Path | Responsibility | Key Exports |
|-----------|---------------|-------------|
| TicketApplication.java | Application entry point | main |
| controller/TicketController.java | REST endpoints for ticket CRUD | createTicket, getTicket, assignTicket |
| controller/TicketQueryController.java | Read-only query endpoints | searchTickets, getTicketHistory |
| service/TicketService.java | Ticket business logic | validateAndCreate, assignTicket |
| service/NotificationService.java | Notification dispatch | sendConfirmation, notifyAssignee |
| repository/TicketRepository.java | Ticket data access | save, findById, findByStatus |
| repository/TicketHistoryRepository.java | Ticket history data access | save, findByTicketId |
| entity/Ticket.java | Ticket data model | Ticket |
| entity/TicketHistory.java | Ticket history data model | TicketHistory |
| config/SecurityConfig.java | Security filter chain setup | securityFilterChain |
| config/SwaggerConfig.java | OpenAPI/Swagger setup | openAPI |
```

#### Section 6: Configuration Reference

- Config files (application.yml, application-*.yml), env vars, feature flags

```markdown
## Configuration Reference

| Config | Location | Description | Default |
|--------|----------|-------------|---------|
| application.yml | src/main/resources/ | Default Spring Boot configuration | — |
| application-dev.yml | src/main/resources/ | Development profile overrides | — |
| application-prod.yml | src/main/resources/ | Production profile overrides | — |
| CSP_TICKET_DB_URL | Environment | Database connection URL | jdbc:postgresql://localhost:5432/csp_ticket |
| CSP_TICKET_DB_USER | Environment | Database username | csp_ticket |
| CSP_TICKET_DB_PASSWORD | Environment | Database password | (required) |
| CSP_NOTIFICATION_ENABLED | Environment | Enable notification dispatch | true |
| csp.ticket.max-retries | application.yml | Max retry count for persistence | 3 |
| csp.ticket.assignment.strategy | application.yml | Ticket assignment strategy | round-robin |
```

#### Section 7: Testing Guide

- Test directory structure
- Exact command to run this module's tests in isolation
- Test conventions

```markdown
## Testing Guide

**Test Directory**: `src/test/java/com/csp/ticket/`

**Run in Isolation**:
```bash
mvn test -pl ticket-application
```

**Run Single Test**:
```bash
mvn test -pl ticket-application -Dtest=TicketServiceTest
```

**Coverage Report**:
```bash
mvn jacoco:report -pl ticket-application
```

**Conventions**:
- Test files named `*Test.java` (unit) or `*IT.java` (integration)
- One test class per source class
- Use `@SpringBootTest` for integration tests, plain JUnit 5 for unit tests
- Mock external dependencies with `@MockBean` (Spring context) or `Mockito.mock()` (standalone)
- Integration tests use H2 in-memory database via `application-test.yml`
```

#### Section 8: Dependencies

- Table: Dependency / Version / Purpose / Internal or External

```markdown
## Dependencies

| Dependency | Version | Purpose | Source |
|------------|---------|---------|--------|
| spring-boot-starter-web | 3.x | REST framework | External |
| spring-boot-starter-data-jpa | 3.x | Data access | External |
| spring-boot-starter-validation | 3.x | Bean validation | External |
| spring-boot-starter-security | 3.x | Authentication/authorization | External |
| com.bytedance:csp-common | 1.x | CSP shared utilities | Internal |
| com.bytedance:csp-notification-client | 1.x | Notification service client | Internal |
| org.postgresql:postgresql | 42.x | PostgreSQL JDBC driver | External |
| com.h2database:h2 | 2.x | In-memory database for tests | External |
```

---

### Content Rules

1. Core execution flows must reference actual file paths and function names (e.g., `controller/TicketController.java → createTicket()`)
2. Code location index must cover ALL source files in the module — no file may be omitted
3. Testing guide must include the exact command to run this module's tests in isolation (e.g., `mvn test -pl ticket-application`)

### Quality Checks

- [ ] Module overview ≤ 3 sentences
- [ ] Internal architecture includes directory tree with annotations
- [ ] Every sub-module has a row in the breakdown table
- [ ] At least one core execution flow is described with file:line references
- [ ] Code location index covers all source files
- [ ] Testing guide includes isolation run command
- [ ] Dependencies table distinguishes internal vs. external
