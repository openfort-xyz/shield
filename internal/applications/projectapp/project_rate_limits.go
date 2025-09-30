package projectapp

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func NewRealClock() RealClock {
	return RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}

type RequestTracker struct {
	mu       sync.RWMutex
	requests map[string]*ProjectRequestData
	cleanup  chan struct{}
	done     chan struct{}
	clock    Clock
}

type ProjectRequestData struct {
	count  int64
	window time.Time
}

func NewRequestTracker(clock Clock) *RequestTracker {
	rt := &RequestTracker{
		requests: make(map[string]*ProjectRequestData),
		cleanup:  make(chan struct{}),
		done:     make(chan struct{}),
		clock:    clock,
	}

	go rt.cleanupWorker()
	return rt
}

func (rt *RequestTracker) TrackRequest(projectID string, rateLimit int64) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := rt.clock.Now()
	currentWindow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())

	data, exists := rt.requests[projectID]
	if !exists || data.window.Before(currentWindow) {
		rt.requests[projectID] = &ProjectRequestData{
			count:  1,
			window: currentWindow,
		}
		return true
	}

	if data.count >= rateLimit {
		return false
	}

	data.count++
	return true
}

func (rt *RequestTracker) GetRequestCount(projectID string) int64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	data, exists := rt.requests[projectID]
	if !exists {
		return 0
	}

	now := rt.clock.Now()
	currentWindow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())

	if data.window.Before(currentWindow) {
		return 0
	}

	return data.count
}

func (rt *RequestTracker) cleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rt.cleanupOldEntries()
		case <-rt.cleanup:
			rt.cleanupOldEntries()
		case <-rt.done:
			return
		}
	}
}

func (rt *RequestTracker) cleanupOldEntries() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := rt.clock.Now()
	currentWindow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())

	for projectID, data := range rt.requests {
		if data.window.Before(currentWindow) {
			delete(rt.requests, projectID)
		}
	}
}

func (rt *RequestTracker) Close() {
	close(rt.done)
}
