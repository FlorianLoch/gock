package pgock

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreRegister(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	require.Equal(t, 0, len(g.mocks))
	mock := g.New("foo").Mock
	g.Register(mock)
	require.Equal(t, 1, len(g.mocks))
	require.Equal(t, mock, mock.Request().Mock)
	require.Equal(t, mock, mock.Response().Mock)
}

func TestStoreGetAll(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	require.Equal(t, 0, len(g.mocks))
	mock := g.New("foo").Mock
	store := g.GetAll()
	require.Equal(t, 1, len(g.mocks))
	require.Equal(t, 1, len(store))
	require.Equal(t, mock, store[0])
}

func TestStoreExists(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	require.Equal(t, 0, len(g.mocks))
	mock := g.New("foo").Mock
	require.Equal(t, 1, len(g.mocks))
	require.True(t, g.Exists(mock))
}

func TestStorePending(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("foo")
	require.Equal(t, g.Pending(), g.mocks)
}

func TestStoreIsPending(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("foo")
	require.True(t, g.IsPending())
	g.Flush()
	require.False(t, g.IsPending())
}

func TestStoreIsDone(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	g.New("foo")
	require.False(t, g.IsDone())
	g.Flush()
	require.True(t, g.IsDone())
}

func TestStoreRemove(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	require.Equal(t, 0, len(g.mocks))
	mock := g.New("foo").Mock
	require.Equal(t, 1, len(g.mocks))
	require.True(t, g.Exists(mock))

	g.Remove(mock)
	require.False(t, g.Exists(mock))

	g.Remove(mock)
	require.False(t, g.Exists(mock))
}

func TestStoreFlush(t *testing.T) {
	g := NewTransport()
	defer g.Off()
	require.Equal(t, 0, len(g.mocks))

	mock1 := g.New("foo").Mock
	mock2 := g.New("foo").Mock
	require.Equal(t, 2, len(g.mocks))
	require.True(t, g.Exists(mock1))
	require.True(t, g.Exists(mock2))

	g.Flush()
	require.Equal(t, 0, len(g.mocks))
	require.False(t, g.Exists(mock1))
	require.False(t, g.Exists(mock2))
}
