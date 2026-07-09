package pgock

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMockSimple(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Reply(201).JSON(map[string]string{"foo": "bar"})
	res, err := g.Client().Get("http://foo.com")
	require.NoError(t, err)
	require.Equal(t, 201, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, `{"foo":"bar"}`, string(body)[:13])
}

func TestMockOff(t *testing.T) {
	g := NewTransport()
	g.New("http://foo.com").Reply(201).JSON(map[string]string{"foo": "bar"})
	g.Off()
	_, err := g.Client().Get("http://127.0.0.1:3123")
	require.True(t, errors.Is(err, ErrTransportDisabled))
}

func TestMockBodyStringResponse(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Reply(200).BodyString("foo bar")
	res, err := g.Client().Get("http://foo.com")
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo bar", string(body))
}

func TestMockBodyMatch(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").BodyString("foo bar").Reply(201).BodyString("foo foo")
	res, err := g.Client().Post("http://foo.com", "text/plain", bytes.NewBuffer([]byte("foo bar")))
	require.NoError(t, err)
	require.Equal(t, 201, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo foo", string(body))
}

func TestMockBodyCannotMatch(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").BodyString("foo foo").Reply(201).BodyString("foo foo")
	_, err := g.Client().Post("http://foo.com", "text/plain", bytes.NewBuffer([]byte("foo bar")))
	require.Error(t, err)
}

func TestMockBodyMatchCompressed(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Compression("gzip").BodyString("foo bar").Reply(201).BodyString("foo foo")

	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	_, _ = w.Write([]byte("foo bar"))
	_ = w.Close()
	req, err := http.NewRequest("POST", "http://foo.com", &compressed)
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "text/plain")
	res, err := g.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, 201, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo foo", string(body))
}

func TestMockBodyCannotMatchCompressed(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Compression("gzip").BodyString("foo bar").Reply(201).BodyString("foo foo")
	_, err := g.Client().Post("http://foo.com", "text/plain", bytes.NewBuffer([]byte("foo bar")))
	require.Error(t, err)
}

func TestMockBodyMatchJSON(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").
		Post("/bar").
		JSON(map[string]string{"foo": "bar"}).
		Reply(201).
		JSON(map[string]string{"bar": "foo"})

	res, err := g.Client().Post("http://foo.com/bar", "application/json", bytes.NewBuffer([]byte(`{"foo":"bar"}`)))
	require.NoError(t, err)
	require.Equal(t, 201, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, `{"bar":"foo"}`, string(body)[:13])
}

func TestMockBodyCannotMatchJSON(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").
		Post("/bar").
		JSON(map[string]string{"bar": "bar"}).
		Reply(201).
		JSON(map[string]string{"bar": "foo"})

	_, err := g.Client().Post("http://foo.com/bar", "application/json", bytes.NewBuffer([]byte(`{"foo":"bar"}`)))
	require.Error(t, err)
}

func TestMockBodyMatchCompressedJSON(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").
		Post("/bar").
		Compression("gzip").
		JSON(map[string]string{"foo": "bar"}).
		Reply(201).
		JSON(map[string]string{"bar": "foo"})

	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	_, _ = w.Write([]byte(`{"foo":"bar"}`))
	_ = w.Close()
	req, err := http.NewRequest("POST", "http://foo.com/bar", &compressed)
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	res, err := g.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, 201, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, `{"bar":"foo"}`, string(body)[:13])
}

func TestMockBodyCannotMatchCompressedJSON(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").
		Post("/bar").
		JSON(map[string]string{"bar": "bar"}).
		Reply(201).
		JSON(map[string]string{"bar": "foo"})

	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	_, _ = w.Write([]byte(`{"foo":"bar"}`))
	_ = w.Close()
	req, err := http.NewRequest("POST", "http://foo.com/bar", &compressed)
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	_, err = g.Client().Do(req)
	require.Error(t, err)
}

func TestMockMatchHeaders(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").
		MatchHeader("Content-Type", "(.*)/plain").
		Reply(200).
		BodyString("foo foo")

	res, err := g.Client().Post("http://foo.com", "text/plain", bytes.NewBuffer([]byte("foo bar")))
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo foo", string(body))
}

func TestMockMap(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	mock := g.New("http://bar.com")
	mock.Map(func(req *http.Request) *http.Request {
		req.URL.Host = "bar.com"
		return req
	})
	mock.Reply(201).JSON(map[string]string{"foo": "bar"})

	res, err := g.Client().Get("http://foo.com")
	require.NoError(t, err)
	require.Equal(t, 201, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, `{"foo":"bar"}`, string(body)[:13])
}

func TestMockFilter(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	mock := g.New("http://foo.com")
	mock.Filter(func(req *http.Request) bool {
		return req.URL.Host == "foo.com"
	})
	mock.Reply(201).JSON(map[string]string{"foo": "bar"})

	res, err := g.Client().Get("http://foo.com")
	require.NoError(t, err)
	require.Equal(t, 201, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, `{"foo":"bar"}`, string(body)[:13])
}

func TestMockCounterDisabled(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Reply(204)
	require.Equal(t, 1, len(g.GetAll()))
	res, err := g.Client().Get("http://foo.com")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)
	require.Equal(t, 0, len(g.GetAll()))
}

func TestMockEnableNetwork(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "Hello, world")
	}))
	defer ts.Close()

	g.EnableNetworking()
	defer g.DisableNetworking()

	g.New(ts.URL).Reply(204)
	require.Equal(t, 1, len(g.GetAll()))

	res, err := g.Client().Get(ts.URL)
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)
	require.Equal(t, 0, len(g.GetAll()))

	res, err = g.Client().Get(ts.URL)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
}

func TestMockEnableNetworkFilter(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "Hello, world")
	}))
	defer ts.Close()

	g.EnableNetworking()
	defer g.DisableNetworking()

	g.NetworkingFilter(func(req *http.Request) bool {
		return strings.Contains(req.URL.Host, "127.0.0.1")
	})
	defer g.DisableNetworkingFilters()

	g.New(ts.URL).Reply(0).SetHeader("foo", "bar")
	require.Equal(t, 1, len(g.GetAll()))

	res, err := g.Client().Get(ts.URL)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	require.Equal(t, "bar", res.Header.Get("foo"))
	require.Equal(t, 0, len(g.GetAll()))
}

func TestMockPersistent(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").
		Get("/bar").
		Persist().
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	for i := 0; i < 5; i++ {
		res, err := g.Client().Get("http://foo.com/bar")
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)
		body, _ := io.ReadAll(res.Body)
		require.Equal(t, `{"foo":"bar"}`, string(body)[:13])
	}
}

func TestMockPersistTimes(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://127.0.0.1:1234").
		Get("/bar").
		Times(4).
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	for i := 0; i < 5; i++ {
		res, err := g.Client().Get("http://127.0.0.1:1234/bar")
		if i == 4 {
			require.Error(t, err)
			break
		}

		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)
		body, _ := io.ReadAll(res.Body)
		require.Equal(t, `{"foo":"bar"}`, string(body)[:13])
	}
}

func TestUnmatched(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	_, err := g.Client().Get("http://server.com/unmatched")
	require.Error(t, err)

	unmatched := g.GetUnmatchedRequests()
	require.Equal(t, 1, len(unmatched))
	require.Equal(t, "server.com", unmatched[0].URL.Host)
	require.Equal(t, "/unmatched", unmatched[0].URL.Path)
	require.True(t, g.HasUnmatchedRequests())
}

func TestMultipleMocks(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	g.New("http://server.com").
		Get("/foo").
		Reply(200).
		JSON(map[string]string{"value": "foo"})

	g.New("http://server.com").
		Get("/bar").
		Reply(200).
		JSON(map[string]string{"value": "bar"})

	g.New("http://server.com").
		Get("/baz").
		Reply(200).
		JSON(map[string]string{"value": "baz"})

	tests := []struct {
		path string
	}{
		{"/foo"},
		{"/bar"},
		{"/baz"},
	}

	client := g.Client()
	for _, test := range tests {
		res, err := client.Get("http://server.com" + test.path)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)
		body, _ := io.ReadAll(res.Body)
		require.Equal(t, `{"value":"`+test.path[1:]+`"}`, string(body)[:15])
	}

	_, err := client.Get("http://server.com/foo")
	require.Error(t, err)
}

func TestCustomClient(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	g.New("http://foo.com").Reply(204)
	require.Equal(t, 1, len(g.GetAll()))

	req, err := http.NewRequest("GET", "http://foo.com", nil)
	require.NoError(t, err)
	client := &http.Client{Transport: g}

	res, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)
}

func TestInstrumentDefaultClient(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	prev := http.DefaultClient.Transport
	g.InstrumentDefaultClient()
	require.Equal(t, http.RoundTripper(g), http.DefaultClient.Transport)

	g.New("http://foo.com").Reply(204)
	res, err := http.Get("http://foo.com")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	g.RestoreDefaultClient()
	require.Equal(t, prev, http.DefaultClient.Transport)
}

// TestRestoreDefaultClientDoesNotClobberOthers covers a misuse of the
// escape hatch: if a second InstrumentDefaultClient call lands on top of
// the first, the first transport's RestoreDefaultClient must not clobber
// the second one. With LIFO restore order (the natural ordering for
// stacked `defer`s) we get back to the original transport.
func TestRestoreDefaultClientDoesNotClobberOthers(t *testing.T) {
	g1 := NewTransport()
	g2 := NewTransport()
	defer g1.RestoreDefaultClient()
	defer g2.RestoreDefaultClient()

	prev := http.DefaultClient.Transport

	g1.InstrumentDefaultClient()
	g2.InstrumentDefaultClient() // overlay: saves g1 as its prev
	require.Equal(t, http.RoundTripper(g2), http.DefaultClient.Transport)

	// Out-of-order restore: g1 is not current, must not clobber g2.
	g1.RestoreDefaultClient()
	require.Equal(t, http.RoundTripper(g2), http.DefaultClient.Transport)

	// LIFO finishes the unwind.
	g2.RestoreDefaultClient()
	require.Equal(t, http.RoundTripper(g1), http.DefaultClient.Transport)

	g1.RestoreDefaultClient()
	require.Equal(t, prev, http.DefaultClient.Transport)
}

func TestOffRestoresDefaultClient(t *testing.T) {
	g := NewTransport()
	defer g.RestoreDefaultClient() // safety net if assertions fail mid-test
	prev := http.DefaultClient.Transport
	g.InstrumentDefaultClient()
	require.Equal(t, http.RoundTripper(g), http.DefaultClient.Transport)
	g.Off()
	require.Equal(t, prev, http.DefaultClient.Transport)
}

func TestMockRegExpMatching(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").
		Post("/bar").
		MatchHeader("Authorization", "Bearer (.*)").
		BodyString(`{"foo":".*"}`).
		Reply(200).
		SetHeader("Server", "pgock").
		JSON(map[string]string{"foo": "bar"})

	req, _ := http.NewRequest("POST", "http://foo.com/bar", bytes.NewBuffer([]byte(`{"foo":"baz"}`)))
	req.Header.Set("Authorization", "Bearer s3cr3t")

	res, err := g.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	require.Equal(t, "pgock", res.Header.Get("Server"))

	body, _ := io.ReadAll(res.Body)
	require.Equal(t, `{"foo":"bar"}`, string(body)[:13])
}

func TestObserve(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	var observedRequest *http.Request
	var observedMock Mock
	g.Observe(func(request *http.Request, mock Mock) {
		observedRequest = request
		observedMock = mock
	})
	g.New("http://observe-foo.com").Reply(200)
	req, _ := http.NewRequest("POST", "http://observe-foo.com", nil)

	_, _ = g.Client().Do(req)

	require.Equal(t, "observe-foo.com", observedRequest.Host)
	require.Equal(t, "observe-foo.com", observedMock.Request().URLStruct.Host)
}

func TestTryCreatingRacesInNew(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.New("http://example.com")
		}()
	}
	wg.Wait()
}

// TestConcurrentRoundTripsRespectCounter verifies that when N goroutines
// race for the same Counter=1 mock on one Transport, exactly one wins.
// Without atomic matching under the transport mutex, two goroutines could
// both observe Counter=1 and both decrement, double-consuming the mock.
func TestConcurrentRoundTripsRespectCounter(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://race.example").Reply(200)

	const n = 32
	var wg sync.WaitGroup
	var hits, misses int32
	client := g.Client()
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := client.Get("http://race.example")
			if err == nil && res.StatusCode == 200 {
				atomic.AddInt32(&hits, 1)
			} else {
				atomic.AddInt32(&misses, 1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, int(hits))
	require.Equal(t, n-1, int(misses))
}

// TestParallelTransports demonstrates the headline property of the redesign:
// independent *Transport instances do not share state, so tests using them
// can run with t.Parallel() without interfering with each other.
func TestParallelTransports(t *testing.T) {
	for i := 0; i < 8; i++ {
		i := i
		t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
			t.Parallel()
			g := NewTransport()
			defer g.Off()

			host := fmt.Sprintf("http://worker-%d.example", i)
			g.New(host).Reply(200).BodyString(fmt.Sprintf("hello %d", i))

			res, err := g.Client().Get(host)
			require.NoError(t, err)
			require.Equal(t, 200, res.StatusCode)
			body, _ := io.ReadAll(res.Body)
			require.Equal(t, fmt.Sprintf("hello %d", i), string(body))

			require.True(t, g.IsDone())
		})
	}
}

// roundTripFunc adapts a function to http.RoundTripper for use as a custom
// networking delegate in tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestMatcherCanReenterTransport guards against the self-deadlock that would
// occur if RoundTrip held the registry/config lock while running a
// user-supplied matcher: matching now runs without that lock held, so a
// matcher that calls back into the Transport must not hang.
func TestMatcherCanReenterTransport(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://reenter.example").
		Filter(func(req *http.Request) bool {
			// Re-enter the Transport during matching. This acquires the
			// registry lock; if RoundTrip still held it, this would deadlock.
			_ = g.GetAll()
			return true
		}).
		Reply(200)

	done := make(chan struct{})
	var res *http.Response
	var err error
	go func() {
		res, err = g.Client().Get("http://reenter.example")
		close(done)
	}()

	select {
	case <-done:
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)
	case <-time.After(5 * time.Second):
		t.Fatal("RoundTrip deadlocked: a matcher re-entered the Transport")
	}
}

// TestMatcherErrorNotTrackedAsUnmatched verifies that a request whose matcher
// returns an error is not recorded in the unmatched-requests registry: an
// errored match is distinct from "no mock matched".
func TestMatcherErrorNotTrackedAsUnmatched(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://err.example").
		AddMatcher(func(req *http.Request, ereq *Request) (bool, error) {
			return false, errors.New("matcher boom")
		}).
		Reply(200)

	_, err := g.Client().Get("http://err.example")
	require.Error(t, err)
	require.False(t, g.HasUnmatchedRequests())
	require.Equal(t, 0, len(g.GetUnmatchedRequests()))
}

// TestEnableNetworkingWithCustomClient verifies that real-network fallback is
// routed through the Transport of a caller-supplied client, rather than always
// using http.DefaultTransport.
func TestEnableNetworkingWithCustomClient(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	var used int32
	custom := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&used, 1)
		return &http.Response{
			StatusCode: 299,
			Body:       io.NopCloser(strings.NewReader("via custom")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	g.EnableNetworking(custom)
	defer g.DisableNetworking()

	// No mock is registered, so the request falls through to the network,
	// which must be the custom client's transport (status 299 is its tell).
	res, err := g.Client().Get("http://custom.example")
	require.NoError(t, err)
	require.Equal(t, 1, int(atomic.LoadInt32(&used)))
	require.Equal(t, 299, res.StatusCode)
}

// TestNewAfterOffFailsLoudly guards the disabled-transport semantics: a mock
// registered after Off() must not silently route requests to the real network;
// the request has to fail with ErrTransportDisabled instead.
func TestNewAfterOffFailsLoudly(t *testing.T) {
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
	}))
	defer ts.Close()

	g := NewTransport()
	g.Off()
	g.New(ts.URL).Reply(204)

	_, err := g.Client().Get(ts.URL)
	require.True(t, errors.Is(err, ErrTransportDisabled))
	require.Equal(t, 0, int(atomic.LoadInt32(&hits)))
}

// TestMatchParamInvalidRegexErrors verifies that an invalid regular expression
// in a query-param expectation surfaces as an error instead of being silently
// swallowed as a non-match.
func TestMatchParamInvalidRegexErrors(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").MatchParam("q", "(unclosed").Reply(200)

	_, err := g.Client().Get("http://foo.com?q=value")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "error parsing regexp"))
	// An errored match is distinct from "no mock matched".
	require.False(t, g.HasUnmatchedRequests())
}

// TestMockTimesZeroNeverMatches: a Times(0) mock is already exhausted, so it
// must never match. Previously its counter went negative on the first match
// and the mock stayed active forever.
func TestMockTimesZeroNeverMatches(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Times(0).Reply(200)

	for i := 0; i < 2; i++ {
		_, err := g.Client().Get("http://foo.com")
		require.True(t, errors.Is(err, ErrCannotMatch))
	}
	require.True(t, g.IsDone())
}

// TestMockBodyExpectationWithNoRequestBody: a bodyless request against a mock
// that expects a body must be a plain non-match, not a nil-dereference panic.
func TestMockBodyExpectationWithNoRequestBody(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").BodyString("expected").Reply(200)

	_, err := g.Client().Get("http://foo.com")
	require.True(t, errors.Is(err, ErrCannotMatch))
}

// TestNetworkFallbackPreservesCompressedBody verifies that matching a mock
// with a gzip body expectation does not corrupt the request: when the mock
// does not match and the request falls through to the real network, the
// delegate must receive the original compressed bytes, not the decompressed
// form left over from matching.
func TestNetworkFallbackPreservesCompressedBody(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	var received []byte
	custom := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		received, _ = io.ReadAll(req.Body)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	g.EnableNetworking(custom)

	g.New("http://fallback.example").Compression("gzip").BodyString("expected").Reply(204)

	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	_, _ = w.Write([]byte("actual payload"))
	_ = w.Close()
	wire := compressed.Bytes()

	req, _ := http.NewRequest("POST", "http://fallback.example", bytes.NewReader(wire))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "text/plain")

	res, err := g.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode) // served by the fallback, not the mock
	require.True(t, bytes.Equal(received, wire))
}

// closeTrackingBody records whether Close was called on it.
type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error { b.closed = true; return nil }

// TestNetworkedMockClosesReplacedBody verifies that when a networked mock
// overrides the body of a real response, the original network body is closed
// rather than leaking its connection.
func TestNetworkedMockClosesReplacedBody(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	body := &closeTrackingBody{Reader: strings.NewReader("real body")}
	custom := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       body,
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	g.EnableNetworking(custom)
	g.DisableNetworking() // keep the delegate; only the mock opts into networking

	g.New("http://networked.example").Reply(200).BodyString("mock body").EnableNetworking()

	res, err := g.Client().Get("http://networked.example")
	require.NoError(t, err)
	out, _ := io.ReadAll(res.Body)
	require.Equal(t, "mock body", string(out))
	require.True(t, body.closed)
}

// TestInstrumentDefaultClientReinstrumentsAfterOverlay verifies that a no-op
// RestoreDefaultClient (because another Transport overlaid this one) does not
// wedge InstrumentDefaultClient into silently doing nothing on a later call.
func TestInstrumentDefaultClientReinstrumentsAfterOverlay(t *testing.T) {
	g1 := NewTransport()
	g2 := NewTransport()
	prev := http.DefaultClient.Transport
	defer func() { http.DefaultClient.Transport = prev }()

	g1.InstrumentDefaultClient()
	g2.InstrumentDefaultClient() // overlay g1
	require.Equal(t, http.RoundTripper(g2), http.DefaultClient.Transport)

	// Off() calls RestoreDefaultClient, which no-ops because g2 is current.
	g1.Off()
	require.Equal(t, http.RoundTripper(g2), http.DefaultClient.Transport)

	// Re-instrumenting g1 must actually re-install it. Before the fix the
	// sticky instrumented flag made this a silent no-op.
	g1.InstrumentDefaultClient()
	require.Equal(t, http.RoundTripper(g1), http.DefaultClient.Transport)
}
