package pgock

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
)

// ObserverFunc receives every intercepted request together with the mock it
// matched against (nil when no match was found). Register one via
// (*Transport).Observe.
type ObserverFunc func(*http.Request, Mock)

// DumpRequest is a ready-made ObserverFunc that prints the wire form of the
// intercepted request and whether a mock matched.
var DumpRequest ObserverFunc = func(request *http.Request, mock Mock) {
	bytes, _ := httputil.DumpRequestOut(request, true)
	fmt.Println(string(bytes))
	fmt.Printf("\nMatches: %v\n---\n", mock != nil)
}

// New creates and registers a new HTTP mock against this Transport and
// returns the Request DSL for further configuration.
func (t *Transport) New(uri string) *Request {
	res := NewResponse()
	req := NewRequest()
	req.URLStruct, res.Error = url.Parse(normalizeURI(uri))

	exp := NewMock(req, res)
	t.Register(exp)

	return req
}

func normalizeURI(uri string) string {
	if ok, _ := regexp.MatchString("^http[s]?", uri); !ok {
		return "http://" + uri
	}
	return uri
}
