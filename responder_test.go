package pgock

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResponder(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	mres := g.New("http://foo.com").Reply(200).BodyString("foo")
	req := &http.Request{}

	res, err := Responder(req, mres, nil)
	require.NoError(t, err)
	require.Equal(t, "200 OK", res.Status)
	require.Equal(t, 200, res.StatusCode)

	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo", string(body))
}

func TestResponder_ReadTwice(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	mres := g.New("http://foo.com").Reply(200).BodyString("foo")
	req := &http.Request{}

	res, err := Responder(req, mres, nil)
	require.NoError(t, err)
	require.Equal(t, "200 OK", res.Status)
	require.Equal(t, 200, res.StatusCode)

	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo", string(body))

	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, []byte{}, body)
}

func TestResponderBodyGenerator(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	generator := func() io.ReadCloser {
		return io.NopCloser(strings.NewReader("foo"))
	}
	mres := g.New("http://foo.com").Reply(200).BodyGenerator(generator)
	req := &http.Request{}

	res, err := Responder(req, mres, nil)
	require.NoError(t, err)
	require.Equal(t, "200 OK", res.Status)
	require.Equal(t, 200, res.StatusCode)

	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo", string(body))
}

func TestResponderBodyGenerator_ReadTwice(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	generator := func() io.ReadCloser {
		return io.NopCloser(strings.NewReader("foo"))
	}
	mres := g.New("http://foo.com").Reply(200).BodyGenerator(generator)
	req := &http.Request{}

	res, err := Responder(req, mres, nil)
	require.NoError(t, err)
	require.Equal(t, "200 OK", res.Status)
	require.Equal(t, 200, res.StatusCode)

	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo", string(body))

	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, []byte{}, body)
}

func TestResponderBodyGenerator_Override(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	generator := func() io.ReadCloser {
		return io.NopCloser(strings.NewReader("foo"))
	}
	mres := g.New("http://foo.com").Reply(200).BodyGenerator(generator).BodyString("bar")
	req := &http.Request{}

	res, err := Responder(req, mres, nil)
	require.NoError(t, err)
	require.Equal(t, "200 OK", res.Status)
	require.Equal(t, 200, res.StatusCode)

	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "foo", string(body))

	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, []byte{}, body)
}

func TestResponderSupportsMultipleHeadersWithSameKey(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	mres := g.New("http://foo").
		Reply(200).
		AddHeader("Set-Cookie", "a=1").
		AddHeader("Set-Cookie", "b=2")
	req := &http.Request{}

	res, err := Responder(req, mres, nil)
	require.NoError(t, err)
	require.Equal(t, http.Header{"Set-Cookie": []string{"a=1", "b=2"}}, res.Header)
}

func TestResponderError(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	mres := g.New("http://foo.com").ReplyError(errors.New("error"))
	req := &http.Request{}

	res, err := Responder(req, mres, nil)
	require.Equal(t, "error", err.Error())
	require.True(t, res == nil)
}

func TestResponderCancelledContext(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	mres := g.New("http://foo.com").Get("").Reply(200).Delay(20 * time.Millisecond).BodyString("foo")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://foo.com", nil)

	res, err := Responder(req, mres, nil)

	require.Equal(t, context.Canceled, err)
	require.True(t, res == nil)
}

func TestResponderExpiredContext(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	mres := g.New("http://foo.com").Get("").Reply(200).Delay(20 * time.Millisecond).BodyString("foo")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://foo.com", nil)

	res, err := Responder(req, mres, nil)

	require.Equal(t, context.DeadlineExceeded, err)
	require.True(t, res == nil)
}

func TestResponderPreExpiredContext(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	mres := g.New("http://foo.com").Get("").Reply(200).BodyString("foo")

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Microsecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://foo.com", nil)

	res, err := Responder(req, mres, nil)

	require.Equal(t, context.DeadlineExceeded, err)
	require.True(t, res == nil)
}
