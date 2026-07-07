## v2.0.0 (unreleased)

  * **Forked from [gock](https://github.com/h2non/gock) and renamed to `pgock`**
    ("the gock for parallel tests"). The module path is now
    `github.com/exaring/pgock` and the Go package is `pgock`; update imports and
    qualified references accordingly (`gock.X` → `pgock.X`).
  * **BREAKING**: removed all package-level state and the package-level API
    (the old `gock.New`, `gock.Off`, `gock.Intercept`, `gock.InterceptClient`,
    etc.). All state now lives on `*pgock.Transport`. Tests construct their own
    transport with `pgock.NewTransport()` and either hand it to an
    `*http.Client` directly or use the new `g.Client()` helper. This makes
    `pgock` safe to use with `t.Parallel()` and removes the implicit
    mutation of `http.DefaultTransport`.
  * Added `(*Transport).Client()` returning a pre-wired `*http.Client`.
  * Added `(*Transport).InstrumentDefaultClient()` /
    `(*Transport).RestoreDefaultClient()` as an opt-in compatibility
    escape hatch for code paths that use `http.DefaultClient` and cannot
    accept an injected client. Documented as an anti-pattern.
  * `(*Transport).EnableNetworking` now optionally accepts an `*http.Client`,
    routing real-network fallback through that client's transport (preserving
    custom TLS/proxy/dialer config) instead of always using
    `http.DefaultTransport`.
  * **BREAKING**: a disabled `Transport` (after `Off()` / `Disable()`) no
    longer forwards requests to the real network; every request fails with the
    new `ErrTransportDisabled` sentinel. In gock, `Disable()` had to restore
    real networking because the library hijacked the process-wide
    `http.DefaultTransport`; in the instance-based model a test client that
    outlives its mocks should fail loudly instead of silently talking to real
    services. Real traffic only flows through the explicit opt-ins
    (`(*Transport).EnableNetworking` and per-mock `EnableNetworking()`).
  * fix(matchers): `MatchQueryParams` propagates regular-expression compile
    errors again instead of silently treating them as a non-match (a
    regression against gock introduced while silencing a linter).
  * fix(matchers): `MatchBody` no longer panics on a request without a body —
    a bodyless request against a mock with a body expectation is a plain
    non-match. (Inherited from gock.)
  * fix(matchers): `MatchBody` restores the request body in its original
    (possibly compressed) form after matching, so a request that falls through
    to the real network is forwarded intact instead of with a decompressed
    body and a stale `Content-Encoding` header. (Inherited from gock.)
  * fix(mock): a mock registered with `Times(0)` (or a negative counter) can
    never match. Previously the counter went negative on the first match and
    the mock stayed active forever. (Inherited from gock.)
  * fix(responder): when a networked mock overrides the body of a real
    response, the original network body is closed instead of leaking its
    connection. (Inherited from gock.)
  * Removed the `_examples` directory; the README's Getting started section
    replaces it.
  * fix(transport): match mocks without holding the registry/config lock so a
    user-supplied matcher/filter/mapper can call back into the transport
    without deadlocking; a dedicated lock still serializes match-and-consume so
    concurrent requests never double-consume a finite-counter mock.
  * fix(transport): requests whose matcher returns an error are no longer
    recorded in the unmatched-request log.
  * fix(transport): `InstrumentDefaultClient` keys idempotency off the live
    `http.DefaultClient.Transport`, so a no-op restore (when another transport
    has overlaid this one) no longer wedges a later re-instrument into silently
    doing nothing, and the saved "previous" transport can never be itself.
  * fix(transport): a zero-value `Transport` (constructed without
    `NewTransport`) no longer panics: it starts out disabled, so requests fail
    with `ErrTransportDisabled`, and its real-network delegate defaults to
    `http.DefaultTransport`.
  * fix(store): `Remove` clears the freed tail slot so a removed mock is no
    longer pinned by the backing array.

## v1.2.0 / 2022-10-19

  * refactor(package): import path changed to github.com/h2non/gock

## v1.1.2 / 2021-08-03

  * fix(mock): fix race condition in mock.go file (#92)

## v1.1.1 / 2021-07-14

  * feat(matchers): Support custom MIME types (#88)

## v1.1.0 / 2021-06-02
  
  * Add context expiration cancellation support (#86)

## v1.0.16 / 2020-11-23
  
  * Fix regexp matching issues in headers (#59)

## v1.0.15 / 2019-07-03
  
  * NewMatcher() will now return objects that completely separate one another. (#55)
  * add request Options (#49)
  * fix typo: function -> func (#52)
  * feat(docs): change note
  * feat(docs): add net/http support
  * Add Basic Auth (#47)
  * Update LICENSE (#46)

## v1.0.14 / 2019-01-31

  * feat(version): bump to v1.0.14
  * feat: add go.mod

## v1.0.13 / 2019-01-30

  * Add PathParam matcher (#42)

## v1.0.12 / 2018-11-13

  * Fix possible data race. (#41)

## v1.0.11 / 2018-10-29

  * Do not reset response body (#40)
  * refactor(travis): remove unsupported versions for golint based on Go release policy support
  * feat(gock): add gock.Observe to support inspection of the outgoing intercepted HTTP traffic (#38)

## v1.0.10 / 2018-09-09

  * Support multiple response headers with same name #35 (#36)

## v1.0.9 / 2018-06-14

  * fix(url-encoding) add exact match test in MatchPath (#34)
  * fix(travis): use string notation for Go versions

## v1.0.8 / 2018-02-28

  * chore(LICENSE): update year ;)
  * feat(docs): add additional tips and examples
  * feat(gock): ignore already intercepted http.Client

## v1.0.7 / 2017-12-21

  * Make MatchHost case insensitive. (#31)
  * refactor(docs): remove codesponsor :(
  * add example when request reply with error (#28)
  * feat(docs): add sponsor ad
  * Add example networking partially enabled (#23)

## v1.0.6 / 2017-07-27

  * fix(#23): mock transport deadlock

## v1.0.5 / 2017-07-26

  * feat(#25, #24): use content type only if missing while matching JSON/XML
  * feat(#24): add CleanUnmatchedRequests() and OffAll() public functions
  * feat(version): bump to v1.0.5
  * fix(store): use proper indent style
  * fix(mutex): use different mutex for store
  * feat(travis): add Go 1.8 CI support

## v1.0.4 / 2017-02-14

  * Update README to include most up to date version (#17)
  * Update MatchBody() to compare if key + value pairs of JSON match regardless of order they are in. (#16)
  * feat(examples): add new example for unmatch case
  * refactor(docs): add pook reference

## 1.0.3 / 14-11-2016

- feat(#13): adds `GetUnmatchedRequests()` and `HasUnmatchedRequests()` API functions.

## 1.0.2 / 10-11-2016

- fix(#11): adds `Compression()` method for output HTTP traffic body compression processing and matching.

## 1.0.1 / 07-09-2016

- fix(#9): missing URL query param matcher.

## 1.0.0 / 19-04-2016

- feat(version): first major version release.

## 0.1.6 / 19-04-2016

- fix(#7): if error configured, RoundTripper should reply with `nil` response.

## 0.1.5 / 09-04-2016

- feat(#5): support `ReplyFunc` for convenience.

## 0.1.4 / 16-03-2016

- feat(api): add `IsDone()` method.
- fix(responder): return mock error if present.
- feat(#4): support define request/response body from file disk.

## 0.1.3 / 09-03-2016

- feat(matcher): add content type matcher helper method supporting aliases.
- feat(interceptor): add function to restore HTTP client transport.
- feat(matcher): add URL scheme matcher function.
- fix(request): ignore base slash path.
- feat(api): add Off() method for easier restore and clean up.
- feat(store): add public API for pending mocks.

## 0.1.2 / 04-03-2016

- fix(matcher): body matchers no used by default.
- feat(matcher): add matcher factories for multiple cases.

## 0.1.1 / 04-03-2016

- fix(params): persist query params accordingly.

## 0.1.0 / 02-03-2016

- First release.
