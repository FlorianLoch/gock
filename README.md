# pgock

Versatile HTTP mocking made easy in [Go](https://golang.org), built for parallel tests. Works with anything that uses `net/http`.

`pgock` is a fork of [gock](https://github.com/h2non/gock) reworked around a self-contained `*Transport`: there is no package-level state and nothing mutates `http.DefaultTransport`, so every test owns its mocks and can run under `t.Parallel()` without interfering with others.

Heavily inspired by [nock](https://github.com/node-nock/nock); there is also a Python port, [pook](https://github.com/h2non/pook).

## Features

- Fluent, declarative DSL for HTTP mock definitions.
- **Instance-based, no global state — safe for `t.Parallel()`.**
- Match on method, URL, query params, headers, and body, with full regex support.
- Built-in helpers for JSON/XML matching and replies.
- Persistent mocks and counted (TTL) mocks.
- Switch between mock-only and partial real-networking modes.
- Map/filter intercepted requests for fine-grained matching.
- Standard `http.RoundTripper` integration — drop into any `net/http`-compatible client.
- Simulate response delay and context cancellation.
- Lightweight: a single small runtime dependency (`github.com/h2non/parth`).

## Installation

```bash
go get github.com/exaring/pgock
```

## Getting started

A representative test:

```go
package mypkg_test

import (
    "encoding/json"
    "net/http"
    "testing"

    "github.com/exaring/pgock"
)

func TestFetchUser(t *testing.T) {
    // 1. Each test owns its own mock transport. No package-level state is touched,
    //    so t.Parallel() and concurrent tests are safe.
    g := pgock.NewTransport()

    // 2. Off() flushes registered mocks and switches the transport off: any
    //    late request through it fails with pgock.ErrTransportDisabled rather
    //    than silently reaching the real network. Always pair NewTransport()
    //    with a deferred Off().
    defer g.Off()

    // 3. Describe what the code under test should be allowed to send, and
    //    what response should come back. The DSL is fluent:
    //      g.New(host)         -- start a mock
    //          .Method(path)   -- pin method + path
    //          .MatchXxx(...)  -- additional match conditions
    //          .Reply(status)  -- transition into the response builder
    //          .JSON(...)      -- response body
    g.New("https://api.example.com").
        Get("/users/42").
        MatchHeader("Authorization", "^Bearer .+$").
        Reply(200).
        JSON(map[string]interface{}{"id": 42, "name": "Ada"})

    // 4. Hand the transport to the *http.Client used by the code under test.
    //    g.Client() is a convenience for &http.Client{Transport: g}; use the
    //    explicit form when you also need to set timeouts, redirect policy, etc.
    client := g.Client()

    req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/users/42", nil)
    req.Header.Set("Authorization", "Bearer s3cret")
    res, err := client.Do(req)
    if err != nil {
        t.Fatalf("request: %v", err)
    }
    defer res.Body.Close()

    // 5. The response was built from the mock; no real network call happened.
    var user struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }
    if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if user.Name != "Ada" {
        t.Fatalf("got %q, want %q", user.Name, "Ada")
    }

    // 6. Optional: assert every declared mock was actually consumed. A leftover
    //    mock typically means the code under test never made the call you expected.
    if !g.IsDone() {
        t.Errorf("pending mocks remain: %d", len(g.Pending()))
    }
}
```

A few things worth knowing once you start writing real tests with this:

- **Matching is FIFO.** When several mocks would match the same request, the first one declared wins. Declare specific mocks before generic catch-alls.
- **Unmatched requests fail by default.** They produce `pgock.ErrCannotMatch`. Call `g.EnableNetworking()` to let unmatched requests fall through to the real network (and use `g.NetworkingFilter(fn)` for finer control over which ones). Pass an `*http.Client` — `g.EnableNetworking(myClient)` — to route that real traffic through the client's own transport, preserving its custom TLS, proxy or dialer config; otherwise it goes through `http.DefaultTransport`.
- **`g.GetUnmatchedRequests()`** returns the slice of requests that didn't match any mock — handy when diagnosing a failing test.
- **Counters:** by default each mock is single-use. Use `.Times(n)` for a counted mock or `.Persist()` for one that never expires.

## How it works

`*pgock.Transport` implements `http.RoundTripper`. When a request comes in:

1. It is matched against the transport's registered mocks in declaration order.
2. If a mock matches, a synthetic `*http.Response` is built from the mock's `Reply(...)` chain and returned. No syscall happens.
3. If no mock matches and real networking is disabled, the request fails with `ErrCannotMatch` and the request is appended to the transport's unmatched-request log.
4. If no mock matches but networking is enabled (and any registered `NetworkingFilter`s allow it), the request is forwarded to the delegate transport (`http.DefaultTransport` by default).

A transport that has been switched off (`Off()` / `Disable()`) refuses every request with `ErrTransportDisabled`. The real network is only ever reachable through the explicit opt-ins — `EnableNetworking` on the transport or `EnableNetworking()` on an individual mock — never as a silent fallback.

There is **no package-level state**. Each `*pgock.Transport` owns its mock registry, observer, networking config, and unmatched-request log. Tests holding distinct transports cannot interfere with each other.

## Intercepting code that uses `http.DefaultClient`

The recommended pattern is to inject a `*http.Client` (or an `http.RoundTripper`) into the code under test. When that is impossible — for example, a third-party library that calls `http.Get` directly — the escape hatch `g.InstrumentDefaultClient()` swaps `http.DefaultClient.Transport` for the duration of the test:

```go
g := pgock.NewTransport()
defer g.Off() // also restores http.DefaultClient.Transport

g.InstrumentDefaultClient()
g.New("http://foo.com").Reply(200)

// Third-party code that calls http.Get internally is now routed through g.
thirdparty.Do("http://foo.com")
```

This is an **anti-pattern**: it mutates process-wide state and is not safe to call from parallel tests. Use it only when the code under test genuinely cannot accept an injected client.

## Custom matchers

A matcher is just `func(*http.Request, *pgock.Request) (bool, error)`. Add one with `(*Request).AddMatcher(fn)` to extend the built-in chain, or replace it entirely with `(*Request).SetMatcher(matcher)`.

## API reference

See the [godoc reference](https://pkg.go.dev/github.com/exaring/pgock) for the full API.

## License

MIT
