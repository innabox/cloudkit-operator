package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cache "github.com/muesli/cache2go"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
)

var inflightRequests *cache.CacheTable = cache.Cache("inflightRequests")

func triggerWebHook(ctx context.Context, url string, instance *cloudkitv1alpha1.ClusterOrder, minimumRequestInterval string) (time.Duration, error) {

	log := ctrllog.FromContext(ctx)
	minDelta, _ := time.ParseDuration(minimumRequestInterval)

	// Rate limiting by url.
	if item, err := (*inflightRequests).Value(url); err != nil {
		inflightTime := item.Data().(time.Time)
		delta := time.Since(inflightTime)
		return delta, nil
	}

	log.Info("Triggering webhook " + url)

	jsonData, err := json.Marshal(instance)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	inflightRequests.Add(url, minDelta, time.Now())
	return 0, nil
}
