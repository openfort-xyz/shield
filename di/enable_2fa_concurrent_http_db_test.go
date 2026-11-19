package di

import (
	"net/http"
	"sync"
	"testing"
)

func TestEnable2FA_ConcurrentRequestsAllSucceed_HTTP(t *testing.T) {
	const requests = 8
	c, apiKey, _ := newProject(t)

	statuses := make([]int, requests)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			statuses[i] = c.do(http.MethodPost, "/project/enable-2fa", apiKey, nil).Status
		}()
	}
	close(start)
	wg.Wait()

	for i, status := range statuses {
		if status != http.StatusOK {
			t.Errorf("request %d: status %d, want 200", i, status)
		}
	}
}
