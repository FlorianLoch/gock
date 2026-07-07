package pgock

import (
	"errors"
	"net/http"
	"sync"
)

// ErrCannotMatch is returned when no mock matches the intercepted request.
var ErrCannotMatch = errors.New("pgock: cannot match any request")

// ErrTransportDisabled is returned for every request issued through a disabled
// Transport (after Off or Disable). A disabled Transport never forwards
// requests to the real network: a test client that outlives its mocks should
// fail loudly, not silently talk to real services.
var ErrTransportDisabled = errors.New("pgock: transport is disabled (was it used after Off?)")

// Transport implements http.RoundTripper and owns all state needed to intercept
// and match HTTP requests against a set of registered mocks.
//
// A Transport is fully self-contained: the registered mocks, observer,
// networking config, and the slice of unmatched requests all live on the
// instance. This makes it safe to use a distinct *Transport per test and to
// call t.Parallel() across tests, in contrast to libraries that mutate
// package-level state or http.DefaultTransport.
//
// The zero value is a disabled Transport (every request fails with
// ErrTransportDisabled); use NewTransport.
type Transport struct {
	// mu guards the registry (mocks) and configuration fields below. It is
	// never held while user-supplied matchers/filters/mappers run, so those
	// callbacks may freely re-enter the Transport's store/config methods.
	mu sync.Mutex

	// matchMu serializes the match-and-consume phase of RoundTrip across
	// concurrent requests so two requests can never both consume the same
	// finite-counter mock. It is deliberately distinct from mu: matching runs
	// user-supplied callbacks, and those must be able to call back into the
	// (mu-guarded) store/config without deadlocking. One rule remains for
	// those callbacks: they must not issue a request through this same
	// Transport, since that would re-enter matchMu and self-deadlock.
	matchMu sync.Mutex

	// delegate is the underlying http.RoundTripper used for real-network
	// fallback: unmatched requests when networking is enabled, and mocks that
	// opt into real networking. It defaults to http.DefaultTransport and can
	// be redirected through a caller-supplied client via EnableNetworking.
	delegate http.RoundTripper

	// enabled controls whether RoundTrip attempts to match against mocks.
	// When false every request fails with ErrTransportDisabled.
	enabled bool

	// mocks is the set of registered mocks for this Transport.
	mocks []Mock

	// networking, when true, allows real network traffic for requests that
	// don't match any mock. Default: false (unmatched requests fail).
	networking bool

	// networkingFilters gates real networking on a per-request basis.
	networkingFilters []FilterRequestFunc

	// observer, if non-nil, receives every intercepted request and the
	// matched mock (or nil if no match).
	observer ObserverFunc

	// unmatched records requests that didn't match any mock; queryable via
	// GetUnmatchedRequests / HasUnmatchedRequests.
	unmatched []*http.Request

	// prevDefaultClientTransport stores the previous value of
	// http.DefaultClient.Transport, captured by InstrumentDefaultClient so
	// RestoreDefaultClient (or Off) can undo the mutation.
	prevDefaultClientTransport http.RoundTripper
	defaultClientInstrumented  bool
}

// NewTransport constructs an enabled Transport that delegates to
// http.DefaultTransport when forwarding non-mocked traffic.
func NewTransport() *Transport {
	return &Transport{
		delegate: http.DefaultTransport,
		enabled:  true,
	}
}

// Client returns a new *http.Client wired to this Transport. The returned
// client is the recommended way to issue requests in tests; it gives every
// test full isolation without touching any process-wide state.
func (t *Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}

// Intercept enables mock matching on this Transport. Idempotent; new
// transports are already enabled.
func (t *Transport) Intercept() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = true
}

// Disable turns off this Transport. While disabled, every request fails with
// ErrTransportDisabled; nothing is ever forwarded to the real network.
func (t *Transport) Disable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = false
}

// Intercepting reports whether this Transport is currently matching against
// mocks (true) or refusing all requests with ErrTransportDisabled (false).
func (t *Transport) Intercepting() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enabled
}

// Off disables the Transport, drops all registered mocks, and undoes any
// InstrumentDefaultClient mutation. Requests issued through the Transport
// after Off fail with ErrTransportDisabled.
func (t *Transport) Off() {
	t.RestoreDefaultClient()
	t.Flush()
	t.Disable()
}

// OffAll is like Off but additionally clears the slice of unmatched requests.
func (t *Transport) OffAll() {
	t.Off()
	t.CleanUnmatchedRequests()
}

// Observe registers an observer that is invoked for every intercepted
// request together with the matched mock (or nil if no match was found).
// Passing nil clears the observer. When concurrent requests race through
// the same Transport, the observer is invoked concurrently too — it must
// be safe for parallel use.
func (t *Transport) Observe(fn ObserverFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.observer = fn
}

// EnableNetworking allows unmatched requests to reach the real network via the
// delegate transport. By default unmatched requests fail with ErrCannotMatch.
//
// Pass an *http.Client to route real traffic through that client's Transport,
// preserving any custom TLS, proxy or dialer configuration it carries (a nil
// client Transport means http.DefaultTransport). Called with no argument it
// only flips networking on and leaves the current delegate untouched
// (http.DefaultTransport on a freshly constructed Transport).
func (t *Transport) EnableNetworking(client ...*http.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.networking = true

	if len(client) == 0 || client[0] == nil {
		return
	}

	rt := client[0].Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	// Never delegate back into ourselves: that would recurse forever on the
	// networking fallback. Fall back to the default transport instead.
	if rt == http.RoundTripper(t) {
		rt = http.DefaultTransport
	}
	t.delegate = rt
}

// DisableNetworking blocks real-network fallback for unmatched requests
// (the default).
func (t *Transport) DisableNetworking() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.networking = false
}

// NetworkingFilter adds a predicate consulted when networking is enabled.
// A request only escapes to the real network when every filter returns true.
func (t *Transport) NetworkingFilter(fn FilterRequestFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.networkingFilters = append(t.networkingFilters, fn)
}

// DisableNetworkingFilters clears the registered networking filters.
func (t *Transport) DisableNetworkingFilters() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.networkingFilters = nil
}

// GetUnmatchedRequests returns the requests this Transport has received that
// did not match any registered mock.
func (t *Transport) GetUnmatchedRequests() []*http.Request {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*http.Request, len(t.unmatched))
	copy(out, t.unmatched)
	return out
}

// HasUnmatchedRequests reports whether any intercepted request has failed to
// match a mock since the Transport was created (or last cleaned).
func (t *Transport) HasUnmatchedRequests() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.unmatched) > 0
}

// CleanUnmatchedRequests drops the slice of recorded unmatched requests.
func (t *Transport) CleanUnmatchedRequests() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.unmatched = nil
}

// InstrumentDefaultClient routes calls that use http.DefaultClient (e.g.
// http.Get, http.Post) through this Transport by replacing
// http.DefaultClient.Transport.
//
// This is an ANTI-PATTERN. It mutates process-wide state, breaks test
// isolation, and is not safe to call from parallel tests. It exists only as
// an escape hatch for code paths that won't accept a caller-supplied
// *http.Client — for example, third-party libraries that issue requests via
// http.Get internally. Prefer t.Client() and pass it explicitly wherever you
// can.
//
// Pair every call with RestoreDefaultClient (typically via defer) so a
// subsequent test does not inherit the mutation. Calling Off() also restores
// the previous transport, provided this Transport is still the installed one
// (i.e. nothing has overlaid it in the meantime).
func (t *Transport) InstrumentDefaultClient() {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Idempotency keys off the live default-client transport rather than a
	// sticky boolean. This guarantees two things: (1) we never capture
	// ourselves as prevDefaultClientTransport (which would make restore a
	// no-op / recurse), and (2) a no-op RestoreDefaultClient that left the
	// flag set — e.g. because another Transport had overlaid us — does not
	// wedge this method into silently doing nothing on a later call.
	if http.DefaultClient.Transport == http.RoundTripper(t) {
		return
	}
	t.prevDefaultClientTransport = http.DefaultClient.Transport
	http.DefaultClient.Transport = t
	t.defaultClientInstrumented = true
}

// RestoreDefaultClient reverses an InstrumentDefaultClient call. No-op if
// InstrumentDefaultClient was never called on this Transport. To avoid
// clobbering an unrelated transport that has since been installed,
// RestoreDefaultClient only restores when http.DefaultClient.Transport is
// still this Transport. If a later Instrument call has overlaid this one,
// the call is a no-op and the saved previous transport is preserved so a
// subsequent invocation (e.g. once the overlaying transport has restored
// itself) can complete the unwind.
func (t *Transport) RestoreDefaultClient() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.defaultClientInstrumented {
		return
	}
	if http.DefaultClient.Transport != t {
		return
	}
	http.DefaultClient.Transport = t.prevDefaultClientTransport
	t.prevDefaultClientTransport = nil
	t.defaultClientInstrumented = false
}

// RoundTrip implements http.RoundTripper. When the Transport is enabled, the
// request is matched against the registered mocks; on a disabled Transport
// every request fails with ErrTransportDisabled. The real network is only
// reachable through the explicit opt-ins (EnableNetworking or a mock that
// enables networking).
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	if !t.enabled {
		t.mu.Unlock()
		return nil, ErrTransportDisabled
	}
	delegate := t.delegate
	if delegate == nil {
		// Defensive: a Transport constructed without NewTransport has no
		// delegate. Fall back rather than panic on the networking path.
		delegate = http.DefaultTransport
	}

	// Snapshot the registry and config under mu, then release it before
	// matching. Matching runs user-supplied matchers/filters/mappers, which
	// may call back into this Transport; holding mu across them would
	// self-deadlock since mu also guards the store/config methods.
	mocks := make([]Mock, len(t.mocks))
	copy(mocks, t.mocks)
	observer := t.observer
	networking := t.networking
	filters := t.networkingFilters
	t.mu.Unlock()

	// matchMu (not mu) serializes the match-and-consume phase so concurrent
	// RoundTrips cannot both consume the same finite-counter mock, while still
	// allowing matcher callbacks to re-enter the mu-guarded store/config.
	t.matchMu.Lock()
	mock, err := matchMocks(mocks, req)
	t.matchMu.Unlock()

	if err != nil {
		return nil, err
	}

	if observer != nil {
		observer(req, mock)
	}

	useNetwork := shouldUseNetwork(req, mock, networking, filters)
	if !useNetwork && mock == nil {
		t.mu.Lock()
		t.unmatched = append(t.unmatched, req)
		t.mu.Unlock()
		return nil, ErrCannotMatch
	}

	defer t.Clean()

	var res *http.Response
	if useNetwork {
		res, err = delegate.RoundTrip(req)
		if err != nil || mock == nil {
			return res, err
		}
	}

	return Responder(req, mock.Response(), res)
}

// shouldUseNetwork decides whether the current request should escape to the
// real network.
func shouldUseNetwork(req *http.Request, mock Mock, networking bool, filters []FilterRequestFunc) bool {
	if mock != nil && mock.Response().UseNetwork {
		return true
	}
	if !networking {
		return false
	}
	for _, filter := range filters {
		if !filter(req) {
			return false
		}
	}
	return true
}
