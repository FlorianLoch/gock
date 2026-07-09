package pgock

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMock(t *testing.T) {

	req := NewRequest()
	res := NewResponse()
	mock := NewMock(req, res)
	require.False(t, mock.disabler.isDisabled())
	require.Equal(t, len(DefaultMatcher.Get()), len(mock.matcher.Get()))

	require.Equal(t, req, mock.Request())
	require.Equal(t, mock, mock.Request().Mock)
	require.Equal(t, res, mock.Response())
	require.Equal(t, mock, mock.Response().Mock)
}

func TestMockDisable(t *testing.T) {

	req := NewRequest()
	res := NewResponse()
	mock := NewMock(req, res)

	require.False(t, mock.disabler.isDisabled())
	mock.Disable()
	require.True(t, mock.disabler.isDisabled())

	matches, err := mock.Match(&http.Request{})
	require.NoError(t, err)
	require.False(t, matches)
}

func TestMockDone(t *testing.T) {

	req := NewRequest()
	res := NewResponse()

	mock := NewMock(req, res)
	require.False(t, mock.disabler.isDisabled())
	require.False(t, mock.Done())

	mock = NewMock(req, res)
	require.False(t, mock.disabler.isDisabled())
	mock.Disable()
	require.True(t, mock.Done())

	mock = NewMock(req, res)
	require.False(t, mock.disabler.isDisabled())
	mock.request.Counter = 0
	require.True(t, mock.Done())

	mock = NewMock(req, res)
	require.False(t, mock.disabler.isDisabled())
	mock.request.Persisted = true
	require.False(t, mock.Done())
}

func TestMockSetMatcher(t *testing.T) {

	req := NewRequest()
	res := NewResponse()
	mock := NewMock(req, res)

	require.Equal(t, len(DefaultMatcher.Get()), len(mock.matcher.Get()))
	matcher := NewMatcher()
	matcher.Flush()
	matcher.Add(func(req *http.Request, ereq *Request) (bool, error) {
		return true, nil
	})
	mock.SetMatcher(matcher)
	require.Equal(t, 1, len(mock.matcher.Get()))
	require.False(t, mock.disabler.isDisabled())

	matches, err := mock.Match(&http.Request{})
	require.NoError(t, err)
	require.True(t, matches)
}

func TestMockAddMatcher(t *testing.T) {

	req := NewRequest()
	res := NewResponse()
	mock := NewMock(req, res)

	require.Equal(t, len(DefaultMatcher.Get()), len(mock.matcher.Get()))
	matcher := NewMatcher()
	matcher.Flush()
	mock.SetMatcher(matcher)
	mock.AddMatcher(func(req *http.Request, ereq *Request) (bool, error) {
		return true, nil
	})
	require.False(t, mock.disabler.isDisabled())
	require.Equal(t, matcher, mock.matcher)

	matches, err := mock.Match(&http.Request{})
	require.NoError(t, err)
	require.True(t, matches)
}

func TestMockMatch(t *testing.T) {

	req := NewRequest()
	res := NewResponse()
	mock := NewMock(req, res)

	matcher := NewMatcher()
	matcher.Flush()
	mock.SetMatcher(matcher)
	calls := 0
	mock.AddMatcher(func(req *http.Request, ereq *Request) (bool, error) {
		calls++
		return true, nil
	})
	mock.AddMatcher(func(req *http.Request, ereq *Request) (bool, error) {
		calls++
		return true, nil
	})
	require.False(t, mock.disabler.isDisabled())
	require.Equal(t, matcher, mock.matcher)

	matches, err := mock.Match(&http.Request{})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.True(t, matches)
}
