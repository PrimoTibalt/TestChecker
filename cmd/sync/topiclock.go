package main

import "sync"

// topicLockSet hands out one lock per topic file. Several partners sync at the
// same time now, so two of them can easily land on the same topic at once — one
// writing it while another reads it to pass it on. Locking per name instead of
// globally keeps unrelated topics syncing in parallel.
type topicLockSet struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

var topicLocks = newTopicLockSet()

func newTopicLockSet() *topicLockSet {
	return &topicLockSet{locks: map[string]*sync.Mutex{}}
}

// Lock takes the lock for one topic and returns the function releasing it.
// Locks are never dropped from the map: a topic name is small, there are only
// ever a handful of them, and reusing the same lock is what makes it a lock.
func (s *topicLockSet) Lock(name string) func() {
	s.mu.Lock()
	lock, ok := s.locks[name]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[name] = lock
	}
	s.mu.Unlock()

	lock.Lock()
	return lock.Unlock
}
