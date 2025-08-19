package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
)

type InflightRequest struct {
	createTime time.Time
}

var (
	inflightRequestsLock *sync.RWMutex              = &sync.RWMutex{}
	inflightRequests     map[string]InflightRequest = make(map[string]InflightRequest)
)

func checkForExistingRequest(ctx context.Context, clusterOrderName string, minimumRequestInterval time.Duration) time.Duration {
	var delta time.Duration

	log := ctrllog.FromContext(ctx)
	inflightRequestsLock.RLock()
	request, ok := inflightRequests[clusterOrderName]
	if ok {
		delta = time.Since(request.createTime)
		if delta >= minimumRequestInterval {
			delta = 0
		}
		log.Info("skip webhook (cluster order found in cache)", "clusterOrder", clusterOrderName, "delta", delta, "minimumRequestInterval", minimumRequestInterval)
	}
	inflightRequestsLock.RUnlock()
	purgeExpiredRequests(ctx, minimumRequestInterval)
	return delta
}

func addInflightRequest(ctx context.Context, clusterOrderName string, minimumRequestInterval time.Duration) {
	log := ctrllog.FromContext(ctx)
	inflightRequestsLock.Lock()
	inflightRequests[clusterOrderName] = InflightRequest{
		createTime: time.Now(),
	}
	log.Info("add webhook to cache", "clusterOrder", clusterOrderName)
	inflightRequestsLock.Unlock()
	purgeExpiredRequests(ctx, minimumRequestInterval)
}

func purgeExpiredRequests(ctx context.Context, minimumRequestInterval time.Duration) {
	var expiredRequests = []string{}

	log := ctrllog.FromContext(ctx)
	inflightRequestsLock.RLock()
	for clusterOrderName, request := range inflightRequests {
		if delta := time.Since(request.createTime); delta > minimumRequestInterval {
			expiredRequests = append(expiredRequests, clusterOrderName)
		}
	}
	inflightRequestsLock.RUnlock()

	if len(expiredRequests) > 0 {
		inflightRequestsLock.Lock()
		for _, clusterOrderName := range expiredRequests {
			log.Info("expire cache entry for webhook", "clusterOrder", clusterOrderName, "minimumRequestInterval", minimumRequestInterval)
			delete(inflightRequests, clusterOrderName)
		}
		inflightRequestsLock.Unlock()
	}
}

func triggerWebHook(ctx context.Context, url string, instance *cloudkitv1alpha1.ClusterOrder, minimumRequestInterval time.Duration) (time.Duration, error) {

	log := ctrllog.FromContext(ctx)

	if delta := checkForExistingRequest(ctx, instance.Name, minimumRequestInterval); delta != 0 {
		return delta, nil
	}

	log.Info("trigger webhook", "url", url)

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

	addInflightRequest(ctx, instance.Name, minimumRequestInterval)
	return 0, nil
}
