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
		resource1       = "test-resource-1"
		resource2       = "test-resource-2"
		minInterval     = 2 * time.Second
		sleepBufferTime = 500 * time.Millisecond
		ctx             = context.TODO()
	)

	// Create a new webhook client for testing
	client := NewWebhookClient(10*time.Second, minInterval)

	t.Run("checkForExistingRequest returns 0 when no request exists", func(t *testing.T) {
		client.ResetCache()
		if got := client.checkForExistingRequest(ctx, url1, resource1); got != 0 {
			t.Errorf("Expected 0, got %v", got)
		}
	})

	t.Run("addInflightRequest stores the request", func(t *testing.T) {
		client.ResetCache()
		client.addInflightRequest(ctx, url1, resource1)
		cacheKey := url1 + ":" + resource1
		if _, ok := client.inflightRequests.Load(cacheKey); !ok {
			t.Errorf("Expected %s to be present in inflightRequests", cacheKey)
		}
	})

	t.Run("checkForExistingRequest returns non-zero for recent request", func(t *testing.T) {
		client.ResetCache()
		client.addInflightRequest(ctx, url1, resource1)
		delta := client.checkForExistingRequest(ctx, url1, resource1)
		if delta <= 0 || delta > minInterval {
			t.Errorf("Expected delta in (0, %v], got %v", minInterval, delta)
		}
	})

	t.Run("purgeExpiredRequests only removes expired", func(t *testing.T) {
		client.ResetCache()
		client.addInflightRequest(ctx, url1, resource1)
		time.Sleep(minInterval + sleepBufferTime)
		client.addInflightRequest(ctx, url2, resource2)
		client.purgeExpiredRequests(ctx)

		cacheKey1 := url1 + ":" + resource1
		cacheKey2 := url2 + ":" + resource2
		_, exists1 := client.inflightRequests.Load(cacheKey1)
		_, exists2 := client.inflightRequests.Load(cacheKey2)

		if exists1 {
			t.Errorf("Expected %s to be purged", cacheKey1)
		}
		if !exists2 {
			t.Errorf("Expected %s to still be in inflightRequests", cacheKey2)
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
				r := resource1
				if i%2 == 0 {
					u = url2
					r = resource2
				}
				client.addInflightRequest(ctx, u, r)
				client.checkForExistingRequest(ctx, u, r)
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
				r := resource1
				if i%2 == 0 {
					url = url2
					r = resource2
				}
				client.addInflightRequest(ctx, url, r)
				_ = client.checkForExistingRequest(ctx, url, r)
				client.purgeExpiredRequests(ctx)
			}(i)
		}
		wg.Wait()
	})

	t.Run("same resource with different URLs are cached separately", func(t *testing.T) {
		client.ResetCache()
		// Add same resource to two different URLs
		client.addInflightRequest(ctx, url1, resource1)
		client.addInflightRequest(ctx, url2, resource1)

		// Both should exist as separate cache entries
		cacheKey1 := url1 + ":" + resource1
		cacheKey2 := url2 + ":" + resource1
		_, exists1 := client.inflightRequests.Load(cacheKey1)
		_, exists2 := client.inflightRequests.Load(cacheKey2)

		if !exists1 {
			t.Errorf("Expected %s to be in cache", cacheKey1)
		}
		if !exists2 {
			t.Errorf("Expected %s to be in cache", cacheKey2)
		}

		// Verify they are treated as different requests
		delta1 := client.checkForExistingRequest(ctx, url1, resource1)
		delta2 := client.checkForExistingRequest(ctx, url2, resource1)
		if delta1 == 0 || delta2 == 0 {
			t.Errorf("Expected both deltas to be non-zero, got delta1=%v, delta2=%v", delta1, delta2)
		}
	})

	t.Run("different resources with same URL are cached separately", func(t *testing.T) {
		client.ResetCache()
		// Add two different resources to the same URL
		client.addInflightRequest(ctx, url1, resource1)
		client.addInflightRequest(ctx, url1, resource2)

		// Both should exist as separate cache entries
		cacheKey1 := url1 + ":" + resource1
		cacheKey2 := url1 + ":" + resource2
		_, exists1 := client.inflightRequests.Load(cacheKey1)
		_, exists2 := client.inflightRequests.Load(cacheKey2)

		if !exists1 {
			t.Errorf("Expected %s to be in cache", cacheKey1)
		}
		if !exists2 {
			t.Errorf("Expected %s to be in cache", cacheKey2)
		}

		// Verify they are treated as different requests
		delta1 := client.checkForExistingRequest(ctx, url1, resource1)
		delta2 := client.checkForExistingRequest(ctx, url1, resource2)
		if delta1 == 0 || delta2 == 0 {
			t.Errorf("Expected both deltas to be non-zero, got delta1=%v, delta2=%v", delta1, delta2)
		}
	})
}
