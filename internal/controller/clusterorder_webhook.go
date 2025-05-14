package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
)

var inflightRequests map[string]time.Time

const DefaultRequestInterval string = "5m"

type ErrMinInterval struct {
	RemainingTime string
}

func (e *ErrMinInterval) Error() string {
	return fmt.Sprintf("request inside the minimum request interval of %v", string(e.RemainingTime))
}

func triggerWebHook(ctx context.Context, url string, instance *cloudkitv1alpha1.ClusterOrder, minimumRequestInterval string) error {
	log := ctrllog.FromContext(ctx)

	if inflightTime, found := inflightRequests[url]; found {
		minDelta, err := time.ParseDuration(minimumRequestInterval)
		if err != nil {
			log.Info("Invalid minimum request interval.  Setting it to the default.", "MinimumRequestInterval", DefaultRequestInterval, "")
			minDelta, _ = time.ParseDuration(DefaultRequestInterval)
		}

		delta := time.Since(inflightTime)
		if delta < minDelta {
			log.Info("Tring to call webhook too soon.", "url",
				url, "minInterval", minimumRequestInterval, "delta",
				delta, "inflightTime", inflightTime)
			return &ErrMinInterval{
				RemainingTime: string(delta),
			}
		}

		delete(inflightRequests, url)
	}

	log.Info("Triggering webhook " + url)

	jsonData, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	inflightRequests[url] = time.Now()
	return nil
}
