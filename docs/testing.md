# Testing

## Running Tests

```bash
go test ./...                                        # all packages
go test ./internal/ui/feed                           # single package
go test ./internal/ui/feed -run TestRenderPost       # single test
go test ./internal/api/bluesky/... -v                # verbose
```

---

## Core Patterns

### All tests use t.Parallel()

Every test function calls `t.Parallel()` immediately. This is consistent across the whole codebase.

### BlueskyClient interface mocking

`internal/api/bluesky/client.go` defines `BlueskyClient` as an interface. UI packages take this interface rather than the concrete type, enabling mock-based testing without any network.

**Canonical example**: `cmd/noms/integration_test.go` has a full `newMockClient()` that implements all `BlueskyClient` methods. Use it as a reference when writing new screen tests.

```go
type mockClient struct{}

func (m *mockClient) GetTimeline(ctx context.Context, limit int, cursor string) ([]*bsky.FeedDef_FeedViewPost, string, error) {
    // return test data
}
// ... all other interface methods
```

### Voresky client: httptest.Server

`VoreskyClient` is a concrete struct (no interface), so its tests spin up a real `httptest.NewServer` and point the client at it:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // handle requests
}))
defer srv.Close()
client := voresky.NewVoreskyClient(srv.URL, auth)
```

### No database mocking

There's no database in noms, so this doesn't apply. All state is in-memory or via the API interfaces.

---

## Test Helpers

### Feed / Thread tests

- `createTestPost()` — builds a `*bsky.FeedDef_FeedViewPost` with sensible defaults
- `stripAnsi()` — strips ANSI escape sequences from rendered output for string comparisons
- `strPtr(s string) *string` — convenience for string pointer fields
- `stubImageRenderer` — implements the image renderer interface with no-ops, preventing any Kitty/Sixel output during tests

### API tests

- `newTestServer()` / `newTestClient()` — pattern in `internal/api/bluesky/*_test.go` for standing up an `httptest.Server` and a client pointing at it
- Auth tests use `httptest.Server` to simulate the OAuth endpoints (PAR, token exchange, refresh)

---

## Test Organization

Package layout follows the standard Go convention:

- `package foo_test` (black-box) for most tests
- `package foo` only when unexported symbols are needed

Table-driven tests with `t.Run` subtests are used for anything with multiple input cases:

```go
tests := []struct {
    name  string
    input string
    want  string
}{
    {"empty", "", ""},
    {"basic", "hello", "hello"},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        t.Parallel()
        got := myFunc(tc.input)
        if got != tc.want {
            t.Errorf("got %q, want %q", got, tc.want)
        }
    })
}
```

---

## What's Not Tested

- **GDScript**: no headless runner available
- **Visual rendering**: ANSI/lipgloss output is tested by stripping ANSI and doing string comparisons, not pixel-level screenshots
- **OAuth browser flow**: the loopback server is tested in isolation; the browser-open step is not
