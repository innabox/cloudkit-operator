package controller

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestWebhookClientCache(t *testing.T) {
	var (
		url1            = "http://webhook1.example.com"
		url2            = "http://webhook2.example.com"
		minInterval     = 2 * time.Second
		sleepBufferTime = 500 * time.Millisecond
		ctx             = context.TODO()
	)

	// Create a new webhook client for testing
	client := NewWebhookClient(10*time.Second, minInterval)

	t.Run("checkForExistingRequest returns 0 when no request exists", func(t *testing.T) {
		client.ResetCache()
		if got := client.checkForExistingRequest(ctx, url1); got != 0 {
			t.Errorf("Expected 0, got %v", got)
		}
	})

	t.Run("addInflightRequest stores the request", func(t *testing.T) {
		client.ResetCache()
		client.addInflightRequest(ctx, url1)
		if _, ok := client.inflightRequests.Load(url1); !ok {
			t.Errorf("Expected %s to be present in inflightRequests", url1)
		}
	})

	t.Run("checkForExistingRequest returns non-zero for recent request", func(t *testing.T) {
		client.ResetCache()
		client.addInflightRequest(ctx, url1)
		delta := client.checkForExistingRequest(ctx, url1)
		if delta <= 0 || delta > minInterval {
			t.Errorf("Expected delta in (0, %v], got %v", minInterval, delta)
		}
	})

	t.Run("purgeExpiredRequests only removes expired", func(t *testing.T) {
		client.ResetCache()
		client.addInflightRequest(ctx, url1)
		time.Sleep(minInterval + sleepBufferTime)
		client.addInflightRequest(ctx, url2)
		client.purgeExpiredRequests(ctx)

		_, exists1 := client.inflightRequests.Load(url1)
		_, exists2 := client.inflightRequests.Load(url2)

		if exists1 {
			t.Errorf("Expected %s to be purged", url1)
		}
		if !exists2 {
			t.Errorf("Expected %s to still be in inflightRequests", url2)
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		client.ResetCache()
		const workers = 10
		var wg sync.WaitGroup

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				u := url1
				if i%2 == 0 {
					u = url2
				}
				client.addInflightRequest(ctx, u)
				client.checkForExistingRequest(ctx, u)
				client.purgeExpiredRequests(ctx)
			}(i)
		}
		wg.Wait()
	})

	t.Run("verify sync.Map prevents data race with high concurrency", func(t *testing.T) {
		client.ResetCache()
		const goroutines = 100
		var wg sync.WaitGroup

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				url := url1
				if i%2 == 0 {
					url = url2
				}
				client.addInflightRequest(ctx, url)
				_ = client.checkForExistingRequest(ctx, url)
				client.purgeExpiredRequests(ctx)
			}(i)
		}
		wg.Wait()
	})
}
