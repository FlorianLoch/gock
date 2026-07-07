package pgock

// Register adds the given mock to this Transport's set of registered mocks.
// Mocks that are already registered (same pointer identity) are ignored.
func (t *Transport) Register(mock Mock) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, existing := range t.mocks {
		if existing == mock {
			return
		}
	}

	mock.Request().Mock = mock
	mock.Response().Mock = mock
	t.mocks = append(t.mocks, mock)
}

// GetAll returns a snapshot of the currently registered mocks.
func (t *Transport) GetAll() []Mock {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]Mock, len(t.mocks))
	copy(out, t.mocks)
	return out
}

// Exists reports whether the given mock is currently registered.
func (t *Transport) Exists(m Mock) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, mock := range t.mocks {
		if mock == m {
			return true
		}
	}
	return false
}

// Remove unregisters the given mock by pointer identity. No-op if the mock
// is not registered.
func (t *Transport) Remove(m Mock) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, mock := range t.mocks {
		if mock == m {
			copy(t.mocks[i:], t.mocks[i+1:])
			// Clear the now-duplicated tail slot so the removed mock can be
			// garbage-collected instead of being pinned by the backing array.
			t.mocks[len(t.mocks)-1] = nil
			t.mocks = t.mocks[:len(t.mocks)-1]
			return
		}
	}
}

// Flush removes every registered mock.
func (t *Transport) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.mocks = nil
}

// Pending returns the mocks that have not yet been fully consumed. It also
// prunes done mocks as a side effect.
func (t *Transport) Pending() []Mock {
	t.Clean()
	return t.GetAll()
}

// IsDone reports whether every registered mock has been consumed.
func (t *Transport) IsDone() bool {
	return !t.IsPending()
}

// IsPending reports whether at least one mock is still waiting to be consumed.
func (t *Transport) IsPending() bool {
	return len(t.Pending()) > 0
}

// Clean removes mocks whose request counters have been exhausted.
func (t *Transport) Clean() {
	t.mu.Lock()
	defer t.mu.Unlock()

	kept := make([]Mock, 0, len(t.mocks))
	for _, mock := range t.mocks {
		if mock.Done() {
			continue
		}
		kept = append(kept, mock)
	}
	t.mocks = kept
}
