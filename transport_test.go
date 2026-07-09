package pgock

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransportMatch(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	const uri = "http://foo.com"
	g.New(uri).Reply(204)
	u, _ := url.Parse(uri)
	req := &http.Request{URL: u}
	res, err := g.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)
	require.Equal(t, req, res.Request)
}

func TestTransportCannotMatch(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Reply(204)
	u, _ := url.Parse("http://127.0.0.1:1234")
	req := &http.Request{URL: u}
	_, err := g.RoundTrip(req)
	require.Equal(t, ErrCannotMatch, err)
}

func TestTransportDisabledRefusesRequests(t *testing.T) {
	g := NewTransport()
	defer g.Off()

	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = fmt.Fprintln(w, "Hello, world")
	}))
	defer ts.Close()

	g.New(ts.URL).Reply(200)
	g.Disable()

	u, _ := url.Parse(ts.URL)
	req := &http.Request{URL: u, Header: make(http.Header)}

	res, err := g.RoundTrip(req)
	require.False(t, g.Intercepting())
	require.Equal(t, ErrTransportDisabled, err)
	require.True(t, res == nil)
	require.Equal(t, 0, int(atomic.LoadInt32(&hits)))
}
