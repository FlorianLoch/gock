package pgock

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewResponse(t *testing.T) {
	res := NewResponse()

	res.Status(200)
	require.Equal(t, 200, res.StatusCode)

	res.SetHeader("foo", "bar")
	require.Equal(t, "bar", res.Header.Get("foo"))

	res.Delay(1000 * time.Millisecond)
	require.Equal(t, 1000*time.Millisecond, res.ResponseDelay)

	res.EnableNetworking()
	require.True(t, res.UseNetwork)
}

func TestResponseStatus(t *testing.T) {
	res := NewResponse()
	require.Equal(t, 0, res.StatusCode)
	res.Status(200)
	require.Equal(t, 200, res.StatusCode)
}

func TestResponseType(t *testing.T) {
	res := NewResponse()
	res.Type("json")
	require.Equal(t, "application/json", res.Header.Get("Content-Type"))

	res = NewResponse()
	res.Type("xml")
	require.Equal(t, "application/xml", res.Header.Get("Content-Type"))

	res = NewResponse()
	res.Type("foo/bar")
	require.Equal(t, "foo/bar", res.Header.Get("Content-Type"))
}

func TestResponseSetHeader(t *testing.T) {
	res := NewResponse()
	res.SetHeader("foo", "bar")
	res.SetHeader("bar", "baz")
	require.Equal(t, "bar", res.Header.Get("foo"))
	require.Equal(t, "baz", res.Header.Get("bar"))
}

func TestResponseAddHeader(t *testing.T) {
	res := NewResponse()
	res.AddHeader("foo", "bar")
	res.AddHeader("foo", "baz")
	require.Equal(t, "bar", res.Header.Get("foo"))
	require.Equal(t, "baz", res.Header["Foo"][1])
}

func TestResponseSetHeaders(t *testing.T) {
	res := NewResponse()
	res.SetHeaders(map[string]string{"foo": "bar", "bar": "baz"})
	require.Equal(t, "bar", res.Header.Get("foo"))
	require.Equal(t, "baz", res.Header.Get("bar"))
}

func TestResponseBody(t *testing.T) {
	res := NewResponse()
	res.Body(bytes.NewBuffer([]byte("foo bar")))
	require.Equal(t, "foo bar", string(res.BodyBuffer))
}

func TestResponseBodyGenerator(t *testing.T) {
	res := NewResponse()
	generator := func() io.ReadCloser {
		return io.NopCloser(bytes.NewBuffer([]byte("foo bar")))
	}
	res.BodyGenerator(generator)
	bytes, err := io.ReadAll(res.BodyGen())
	require.NoError(t, err)
	require.Equal(t, "foo bar", string(bytes))
}

func TestResponseBodyString(t *testing.T) {
	res := NewResponse()
	res.BodyString("foo bar")
	require.Equal(t, "foo bar", string(res.BodyBuffer))
}

func TestResponseFile(t *testing.T) {
	res := NewResponse()
	res.File("pgock.go")
	require.Equal(t, "package pgock", string(res.BodyBuffer)[:13])
}

func TestResponseJSON(t *testing.T) {
	res := NewResponse()
	res.JSON(map[string]string{"foo": "bar"})
	require.Equal(t, `{"foo":"bar"}`, string(res.BodyBuffer)[:13])
	require.Equal(t, "application/json", res.Header.Get("Content-Type"))
}

func TestResponseXML(t *testing.T) {
	res := NewResponse()
	type xml struct {
		Data string `xml:"data"`
	}
	res.XML(xml{Data: "foo"})
	require.Equal(t, `<xml><data>foo</data></xml>`, string(res.BodyBuffer))
	require.Equal(t, "application/xml", res.Header.Get("Content-Type"))
}

func TestResponseMap(t *testing.T) {
	res := NewResponse()
	require.Equal(t, 0, len(res.Mappers))
	res.Map(func(res *http.Response) *http.Response {
		return res
	})
	require.Equal(t, 1, len(res.Mappers))
}

func TestResponseFilter(t *testing.T) {
	res := NewResponse()
	require.Equal(t, 0, len(res.Filters))
	res.Filter(func(res *http.Response) bool {
		return true
	})
	require.Equal(t, 1, len(res.Filters))
}

func TestResponseSetError(t *testing.T) {
	res := NewResponse()
	require.NoError(t, res.Error)
	res.SetError(errors.New("foo error"))
	require.Equal(t, "foo error", res.Error.Error())
}

func TestResponseDelay(t *testing.T) {
	res := NewResponse()
	require.Equal(t, 0*time.Microsecond, res.ResponseDelay)
	res.Delay(100 * time.Millisecond)
	require.Equal(t, 100*time.Millisecond, res.ResponseDelay)
}

func TestResponseEnableNetworking(t *testing.T) {
	res := NewResponse()
	require.False(t, res.UseNetwork)
	res.EnableNetworking()
	require.True(t, res.UseNetwork)
}

func TestResponseDone(t *testing.T) {
	res := NewResponse()
	res.Mock = &Mocker{request: &Request{Counter: 1}, disabler: new(disabler)}
	require.False(t, res.Done())
	res.Mock.Disable()
	require.True(t, res.Done())
}
