package pgock

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRequest(t *testing.T) {
	req := NewRequest()
	req.URL("http://foo.com")
	require.Equal(t, "foo.com", req.URLStruct.Host)
	require.Equal(t, "http", req.URLStruct.Scheme)
	req.MatchHeader("foo", "bar")
	require.Equal(t, "bar", req.Header.Get("foo"))
}

func TestRequestSetURL(t *testing.T) {
	req := NewRequest()
	req.URL("http://foo.com")
	req.SetURL(&url.URL{Host: "bar.com", Path: "/foo"})
	require.Equal(t, "bar.com", req.URLStruct.Host)
	require.Equal(t, "/foo", req.URLStruct.Path)
}

func TestRequestPath(t *testing.T) {
	req := NewRequest()
	req.URL("http://foo.com")
	req.Path("/foo")
	require.Equal(t, "http", req.URLStruct.Scheme)
	require.Equal(t, "foo.com", req.URLStruct.Host)
	require.Equal(t, "/foo", req.URLStruct.Path)
}

func TestRequestBody(t *testing.T) {
	req := NewRequest()
	req.Body(bytes.NewBuffer([]byte("foo bar")))
	require.Equal(t, "foo bar", string(req.BodyBuffer))
}

func TestRequestBodyString(t *testing.T) {
	req := NewRequest()
	req.BodyString("foo bar")
	require.Equal(t, "foo bar", string(req.BodyBuffer))
}

func TestRequestFile(t *testing.T) {
	req := NewRequest()
	req.File("pgock.go")
	require.Equal(t, "package pgock", string(req.BodyBuffer)[:13])
}

func TestRequestJSON(t *testing.T) {
	req := NewRequest()
	req.JSON(map[string]string{"foo": "bar"})
	require.Equal(t, `{"foo":"bar"}`, string(req.BodyBuffer)[:13])
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

func TestRequestXML(t *testing.T) {
	req := NewRequest()
	type xml struct {
		Data string `xml:"data"`
	}
	req.XML(xml{Data: "foo"})
	require.Equal(t, `<xml><data>foo</data></xml>`, string(req.BodyBuffer))
	require.Equal(t, "application/xml", req.Header.Get("Content-Type"))
}

func TestRequestMatchType(t *testing.T) {
	req := NewRequest()
	req.MatchType("json")
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))

	req = NewRequest()
	req.MatchType("html")
	require.Equal(t, "text/html", req.Header.Get("Content-Type"))

	req = NewRequest()
	req.MatchType("foo/bar")
	require.Equal(t, "foo/bar", req.Header.Get("Content-Type"))
}

func TestRequestBasicAuth(t *testing.T) {
	req := NewRequest()
	req.BasicAuth("bob", "qwerty")
	require.Equal(t, "Basic Ym9iOnF3ZXJ0eQ==", req.Header.Get("Authorization"))
}

func TestRequestMatchHeader(t *testing.T) {
	req := NewRequest()
	req.MatchHeader("foo", "bar")
	req.MatchHeader("bar", "baz")
	req.MatchHeader("UPPERCASE", "bat")
	req.MatchHeader("Mixed-CASE", "foo")

	require.Equal(t, "bar", req.Header.Get("foo"))
	require.Equal(t, "baz", req.Header.Get("bar"))
	require.Equal(t, "bat", req.Header.Get("UPPERCASE"))
	require.Equal(t, "foo", req.Header.Get("Mixed-CASE"))
}

func TestRequestHeaderPresent(t *testing.T) {
	req := NewRequest()
	req.HeaderPresent("foo")
	req.HeaderPresent("bar")
	req.HeaderPresent("UPPERCASE")
	req.HeaderPresent("Mixed-CASE")
	require.Equal(t, ".*", req.Header.Get("foo"))
	require.Equal(t, ".*", req.Header.Get("bar"))
	require.Equal(t, ".*", req.Header.Get("UPPERCASE"))
	require.Equal(t, ".*", req.Header.Get("Mixed-CASE"))
}

func TestRequestMatchParam(t *testing.T) {
	req := NewRequest()
	req.MatchParam("foo", "bar")
	req.MatchParam("bar", "baz")
	require.Equal(t, "bar", req.URLStruct.Query().Get("foo"))
	require.Equal(t, "baz", req.URLStruct.Query().Get("bar"))
}

func TestRequestMatchParams(t *testing.T) {
	req := NewRequest()
	req.MatchParams(map[string]string{"foo": "bar", "bar": "baz"})
	require.Equal(t, "bar", req.URLStruct.Query().Get("foo"))
	require.Equal(t, "baz", req.URLStruct.Query().Get("bar"))
}

func TestRequestPresentParam(t *testing.T) {
	req := NewRequest()
	req.ParamPresent("key")
	require.Equal(t, ".*", req.URLStruct.Query().Get("key"))
}

func TestRequestPathParam(t *testing.T) {
	req := NewRequest()
	req.PathParam("key", "value")
	require.Equal(t, "value", req.PathParams["key"])
}

func TestRequestPersist(t *testing.T) {
	req := NewRequest()
	require.False(t, req.Persisted)
	req.Persist()
	require.True(t, req.Persisted)
}

func TestRequestTimes(t *testing.T) {
	req := NewRequest()
	require.Equal(t, 1, req.Counter)
	req.Times(3)
	require.Equal(t, 3, req.Counter)
}

func TestRequestMap(t *testing.T) {
	req := NewRequest()
	require.Equal(t, 0, len(req.Mappers))
	req.Map(func(req *http.Request) *http.Request {
		return req
	})
	require.Equal(t, 1, len(req.Mappers))
}

func TestRequestFilter(t *testing.T) {
	req := NewRequest()
	require.Equal(t, 0, len(req.Filters))
	req.Filter(func(req *http.Request) bool {
		return true
	})
	require.Equal(t, 1, len(req.Filters))
}

func TestRequestEnableNetworking(t *testing.T) {
	req := NewRequest()
	req.Response = &Response{}
	require.False(t, req.Response.UseNetwork)
	req.EnableNetworking()
	require.True(t, req.Response.UseNetwork)
}

func TestRequestResponse(t *testing.T) {
	req := NewRequest()
	res := NewResponse()
	req.Response = res
	chain := req.Reply(200)
	require.Equal(t, res, chain)
	require.Equal(t, 200, chain.StatusCode)
}

func TestRequestReplyFunc(t *testing.T) {
	req := NewRequest()
	res := NewResponse()
	req.Response = res
	chain := req.ReplyFunc(func(r *Response) {
		r.Status(204)
	})
	require.Equal(t, res, chain)
	require.Equal(t, 204, chain.StatusCode)
}

func TestRequestMethods(t *testing.T) {
	req := NewRequest()
	req.Get("/foo")
	require.Equal(t, "GET", req.Method)
	require.Equal(t, "/foo", req.URLStruct.Path)

	req = NewRequest()
	req.Post("/foo")
	require.Equal(t, "POST", req.Method)
	require.Equal(t, "/foo", req.URLStruct.Path)

	req = NewRequest()
	req.Put("/foo")
	require.Equal(t, "PUT", req.Method)
	require.Equal(t, "/foo", req.URLStruct.Path)

	req = NewRequest()
	req.Delete("/foo")
	require.Equal(t, "DELETE", req.Method)
	require.Equal(t, "/foo", req.URLStruct.Path)

	req = NewRequest()
	req.Patch("/foo")
	require.Equal(t, "PATCH", req.Method)
	require.Equal(t, "/foo", req.URLStruct.Path)

	req = NewRequest()
	req.Head("/foo")
	require.Equal(t, "HEAD", req.Method)
	require.Equal(t, "/foo", req.URLStruct.Path)
}

func TestRequestSetMatcher(t *testing.T) {

	matcher := NewEmptyMatcher()
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.URL.Host == "foo.com", nil
	})
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return req.Header.Get("foo") == "bar", nil
	})
	ereq := NewRequest()
	mock := NewMock(ereq, &Response{})
	mock.SetMatcher(matcher)
	ereq.Mock = mock

	headers := make(http.Header)
	headers.Set("foo", "bar")
	req := &http.Request{
		URL:    &url.URL{Host: "foo.com", Path: "/bar"},
		Header: headers,
	}

	match, err := ereq.Mock.Match(req)
	require.NoError(t, err)
	require.True(t, match)
}

func TestRequestAddMatcher(t *testing.T) {

	ereq := NewRequest()
	mock := NewMock(ereq, &Response{})
	mock.matcher = NewMatcher()
	ereq.Mock = mock

	ereq.AddMatcher(func(req *http.Request, ereq *Request) (bool, error) {
		return req.URL.Host == "foo.com", nil
	})
	ereq.AddMatcher(func(req *http.Request, ereq *Request) (bool, error) {
		return req.Header.Get("foo") == "bar", nil
	})

	headers := make(http.Header)
	headers.Set("foo", "bar")
	req := &http.Request{
		URL:    &url.URL{Host: "foo.com", Path: "/bar"},
		Header: headers,
	}

	match, err := ereq.Mock.Match(req)
	require.NoError(t, err)
	require.True(t, match)
}
