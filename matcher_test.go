package pgock

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisteredMatchers(t *testing.T) {
	require.Equal(t, 7, len(MatchersHeader))
	require.Equal(t, 1, len(MatchersBody))
}

func TestNewMatcher(t *testing.T) {
	matcher := NewMatcher()
	// Funcs are not comparable, checking slice length as it's better than nothing
	// See https://golang.org/pkg/reflect/#DeepEqual
	require.Equal(t, len(Matchers), len(matcher.Matchers))
	require.Equal(t, len(Matchers), len(matcher.Get()))
}

func TestNewBasicMatcher(t *testing.T) {
	matcher := NewBasicMatcher()
	// Funcs are not comparable, checking slice length as it's better than nothing
	// See https://golang.org/pkg/reflect/#DeepEqual
	require.Equal(t, len(MatchersHeader), len(matcher.Matchers))
	require.Equal(t, len(MatchersHeader), len(matcher.Get()))
}

func TestNewEmptyMatcher(t *testing.T) {
	matcher := NewEmptyMatcher()
	require.Equal(t, 0, len(matcher.Matchers))
	require.Equal(t, 0, len(matcher.Get()))
}

func TestMatcherAdd(t *testing.T) {
	matcher := NewMatcher()
	require.Equal(t, len(Matchers), len(matcher.Matchers))
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return true, nil
	})
	require.Equal(t, len(Matchers)+1, len(matcher.Get()))
}

func TestMatcherSet(t *testing.T) {
	matcher := NewMatcher()
	matchers := []MatchFunc{}
	require.Equal(t, len(Matchers), len(matcher.Matchers))
	matcher.Set(matchers)
	require.Equal(t, matchers, matcher.Matchers)
	require.Equal(t, 0, len(matcher.Get()))
}

func TestMatcherGet(t *testing.T) {
	matcher := NewMatcher()
	matchers := []MatchFunc{}
	matcher.Set(matchers)
	require.Equal(t, matchers, matcher.Get())
}

func TestMatcherFlush(t *testing.T) {
	matcher := NewMatcher()
	require.Equal(t, len(Matchers), len(matcher.Matchers))
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return true, nil
	})
	require.Equal(t, len(Matchers)+1, len(matcher.Get()))
	matcher.Flush()
	require.Equal(t, 0, len(matcher.Get()))
}

func TestMatcherClone(t *testing.T) {
	matcher := DefaultMatcher.Clone()
	require.Equal(t, len(DefaultMatcher.Get()), len(matcher.Get()))
}

func TestMatcher(t *testing.T) {
	cases := []struct {
		method  string
		url     string
		matches bool
	}{
		{"GET", "http://foo.com/bar", true},
		{"GET", "http://foo.com/baz", true},
		{"GET", "http://foo.com/foo", false},
		{"POST", "http://foo.com/bar", false},
		{"POST", "http://bar.com/bar", false},
		{"GET", "http://foo.com", false},
	}

	matcher := NewMatcher()
	matcher.Flush()
	require.Equal(t, 0, len(matcher.Matchers))

	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.Method == "GET", nil
	})
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.URL.Host == "foo.com", nil
	})
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.URL.Path == "/baz" || req.URL.Path == "/bar", nil
	})

	for _, test := range cases {
		u, _ := url.Parse(test.url)
		req := &http.Request{Method: test.method, URL: u}
		matches, err := matcher.Match(req, nil)
		require.NoError(t, err)
		require.Equal(t, test.matches, matches)
	}
}

func TestMatchMock(t *testing.T) {
	cases := []struct {
		method  string
		url     string
		matches bool
	}{
		{"GET", "http://foo.com/bar", true},
		{"GET", "http://foo.com/baz", true},
		{"GET", "http://foo.com/foo", false},
		{"POST", "http://foo.com/bar", false},
		{"POST", "http://bar.com/bar", false},
		{"GET", "http://foo.com", false},
	}

	matcher := DefaultMatcher
	matcher.Flush()
	require.Equal(t, 0, len(matcher.Matchers))

	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.Method == "GET", nil
	})
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.URL.Host == "foo.com", nil
	})
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.URL.Path == "/baz" || req.URL.Path == "/bar", nil
	})

	g := NewTransport()
	defer g.Off()
	for _, test := range cases {
		g.Flush()
		mock := g.New(test.url).method(test.method, "").Mock

		u, _ := url.Parse(test.url)
		req := &http.Request{Method: test.method, URL: u}

		match, err := g.MatchMock(req)
		require.NoError(t, err)
		if test.matches {
			require.Equal(t, mock, match)
		} else {
			require.Nil(t, match)
		}
	}

	DefaultMatcher.Matchers = Matchers
}
