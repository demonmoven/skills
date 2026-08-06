# Go Testing Framework Configuration

## 1. Test Organization

### 1.1 Directory Structure

```
.
├── cmd/
│   └── server/
│       └── main_test.go
├── internal/
│   ├── handler/
│   │   └── handler_test.go
│   ├── service/
│   │   └── service_test.go
│   └── repository/
│       └── repository_test.go
└── pkg/
    └── utils/
        └── utils_test.go
```

- Test files are in the same directory as source files, named `*_test.go`
- Test package uses `xxx_test` naming (black-box testing) or same package naming (white-box testing)

### 1.2 Standard Library testing

Go projects use the standard library `testing` by default, third-party testing frameworks are not required.

```go
package handler_test

import "testing"

func TestExample(t *testing.T) {
    got := 1 + 1
    if got != 2 {
        t.Errorf("expected 2, got %d", got)
    }
}
```

### 1.3 Helper Libraries (Optional)

| Library | Purpose | When to Use |
|---------|---------|-------------|
| `github.com/stretchr/testify` | Simplified assertions and mocks | Recommended for complex projects |
| `github.com/golang/mock` | Interface mock generation | When mocking RPC calls |
| `go.uber.org/mock` | Active fork of golang/mock | Preferred for new projects |

## 2. Common Go Test Patterns

### 2.1 HTTP Handler Test (Hertz)

```go
package handler_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGetTicketHandler(t *testing.T) {
    // Setup
    req := httptest.NewRequest(http.MethodGet, "/api/tickets/123", nil)
    w := httptest.NewRecorder()

    // Execute
    GetTicketHandler(w, req)

    // Assert
    assert.Equal(t, http.StatusOK, w.Code)

    var resp TicketResponse
    err := json.NewDecoder(w.Body).Decode(&resp)
    require.NoError(t, err)
    assert.Equal(t, "123", resp.ID)
}

func TestCreateTicketHandler_InvalidInput(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/api/tickets", nil)
    w := httptest.NewRecorder()

    CreateTicketHandler(w, req)

    assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

### 2.2 Service Test with Mock

```go
package service_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock repository using testify
type MockTicketRepo struct {
    mock.Mock
}

func (m *MockTicketRepo) FindByID(id string) (*Ticket, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Ticket), args.Error(1)
}

func TestRouteTicket(t *testing.T) {
    repo := new(MockTicketRepo)
    repo.On("FindByID", "123").Return(&Ticket{ID: "123", Status: "open"}, nil)

    svc := NewTicketService(repo)
    result, err := svc.RouteTicket("123")

    assert.NoError(t, err)
    assert.Equal(t, "routed", result.Status)
    repo.AssertExpectations(t)
}

func TestRouteTicket_NotFound(t *testing.T) {
    repo := new(MockTicketRepo)
    repo.On("FindByID", "999").Return(nil, ErrNotFound)

    svc := NewTicketService(repo)
    _, err := svc.RouteTicket("999")

    assert.Error(t, err)
    assert.Equal(t, ErrNotFound, err)
}
```

### 2.3 Integration Test

```go
package integration_test

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/suite"
)

type TicketIntegrationSuite struct {
    suite.Suite
    server *httptest.Server
}

func (s *TicketIntegrationSuite) SetupSuite() {
    // Start test server with real dependencies
    s.server = httptest.NewServer(setupRouter())
}

func (s *TicketIntegrationSuite) TearDownSuite() {
    s.server.Close()
}

func (s *TicketIntegrationSuite) TestCreateAndGetTicket() {
    // Create ticket
    body := bytes.NewBufferString(`{"title":"test ticket"}`)
    resp, err := http.Post(s.server.URL+"/api/tickets", "application/json", body)
    s.Require().NoError(err)
    s.Equal(http.StatusCreated, resp.StatusCode)
    resp.Body.Close()

    // Get ticket
    resp, err = http.Get(s.server.URL + "/api/tickets/1")
    s.Require().NoError(err)
    s.Equal(http.StatusOK, resp.StatusCode)
    resp.Body.Close()
}

func TestTicketIntegrationSuite(t *testing.T) {
    suite.Run(t, new(TicketIntegrationSuite))
}
```

### 2.4 Table-Driven Test Pattern

```go
func TestRouteTicket_TableDriven(t *testing.T) {
    tests := []struct {
        name       string
        ticketID   string
        wantStatus string
        wantErr    error
    }{
        {
            name:       "open ticket routes successfully",
            ticketID:   "123",
            wantStatus: "routed",
            wantErr:    nil,
        },
        {
            name:       "closed ticket cannot be routed",
            ticketID:   "456",
            wantStatus: "",
            wantErr:    ErrTicketClosed,
        },
        {
            name:       "nonexistent ticket returns not found",
            ticketID:   "999",
            wantStatus: "",
            wantErr:    ErrNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := new(MockTicketRepo)
            repo.On("FindByID", tt.ticketID).Return(&Ticket{ID: tt.ticketID}, tt.wantErr)

            svc := NewTicketService(repo)
            result, err := svc.RouteTicket(tt.ticketID)

            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.wantStatus, result.Status)
            }
        })
    }
}
```

## 3. Coverage Requirements

- **Minimum Coverage**: Core logic ≥ 60%, utility functions ≥ 80%
- Coverage output: `go test ./... -coverprofile=coverage.out`
- Coverage view: `go tool cover -html=coverage.out`

## 4. Verification Checklist

- [ ] `go test ./...` executes successfully
- [ ] Test files follow `*_test.go` naming convention
- [ ] Coverage report can be generated
- [ ] CI includes test stage

## 5. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Test Runner | go testing (stdlib) | — | None (stdlib is standard) |
| Assertions | testify | — | Never (stdlib assertions too verbose for complex checks) |
| Mocking | go.uber.org/mock | gomock (original) | gomock only if already in project |
| HTTP Testing | net/http/httptest | — | None (stdlib is standard) |
| Integration | testify/suite | — | Complex multi-step test scenarios |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 6. Auto-Completion Procedure

### Detecting Test Framework Status

1. Check for `*_test.go` files → tests exist
2. Check for `testing` import in test files → stdlib used
3. Check for `testify` in `go.mod` → assertion library installed
4. If no test files → full auto-completion
5. If tests exist but no assertion library → suggest adding testify

### Go Testing Auto-Completion Steps

1. Add testify dependency: `go get github.com/stretchr/testify`
2. Create test directory structure matching source code
3. Generate smoke test file:
   ```go
   package main_test

   import (
       "testing"

       "github.com/stretchr/testify/assert"
   )

   func TestSmoke(t *testing.T) {
       assert.True(t, true, "smoke test should pass")
   }
   ```
4. Run: `go test ./...` → must exit 0
5. Add coverage to CI: include `-coverprofile=coverage.out` in test command

### Acceptance Criteria

- [ ] `*_test.go` files exist with at least one test
- [ ] `testify` in go.mod (direct or indirect)
- [ ] `go test ./...` exits 0
- [ ] Coverage report can be generated (`go test ./... -coverprofile=coverage.out`)
