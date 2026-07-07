package pgock

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/nbio/st"
)

func TestTransportMatch(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	const uri = "http://foo.com"
	g.New(uri).Reply(204)
	u, _ := url.Parse(uri)
	req := &http.Request{URL: u}
	res, err := g.RoundTrip(req)
	st.Expect(t, err, nil)
	st.Expect(t, res.StatusCode, 204)
	st.Expect(t, res.Request, req)
}

func TestTransportCannotMatch(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("http://foo.com").Reply(204)
	u, _ := url.Parse("http://127.0.0.1:1234")
	req := &http.Request{URL: u}
	_, err := g.RoundTrip(req)
	st.Expect(t, err, ErrCannotMatch)
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
	st.Expect(t, g.Intercepting(), false)
	st.Expect(t, err, ErrTransportDisabled)
	st.Expect(t, res == nil, true)
	st.Expect(t, int(atomic.LoadInt32(&hits)), 0)
}
