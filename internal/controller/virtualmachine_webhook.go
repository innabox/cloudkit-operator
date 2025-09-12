/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

type InflightVMRequest struct {
	createTime time.Time
}

var (
	inflightVMRequests sync.Map // map[string]InflightVMRequest
)

func checkForExistingVMRequest(ctx context.Context, virtualMachineName string, minimumRequestInterval time.Duration) time.Duration {
	var delta time.Duration

	log := ctrllog.FromContext(ctx)
	if value, ok := inflightVMRequests.Load(virtualMachineName); ok {
		request := value.(InflightVMRequest)
		delta = time.Since(request.createTime)
		if delta >= minimumRequestInterval {
			delta = 0
		}
		log.Info("skip webhook (virtual machine found in cache)", "virtualMachine", virtualMachineName, "delta", delta, "minimumRequestInterval", minimumRequestInterval)
	}
	purgeExpiredVMRequests(ctx, minimumRequestInterval)
	return delta
}

func addInflightVMRequest(ctx context.Context, virtualMachineName string, minimumRequestInterval time.Duration) {
	log := ctrllog.FromContext(ctx)
	inflightVMRequests.Store(virtualMachineName, InflightVMRequest{
		createTime: time.Now(),
	})
	log.Info("add webhook to cache", "virtualMachine", virtualMachineName)
	purgeExpiredVMRequests(ctx, minimumRequestInterval)
}

func purgeExpiredVMRequests(ctx context.Context, minimumRequestInterval time.Duration) {
	log := ctrllog.FromContext(ctx)
	inflightVMRequests.Range(func(key, value any) bool {
		virtualMachineName := key.(string)
		request := value.(InflightVMRequest)
		if delta := time.Since(request.createTime); delta > minimumRequestInterval {
			log.Info("expire cache entry for webhook", "virtualMachine", virtualMachineName, "minimumRequestInterval", minimumRequestInterval)
			inflightVMRequests.Delete(virtualMachineName)
		}
		return true
	})
}

func triggerCreateVMWebHook(ctx context.Context, url string, instance *cloudkitv1alpha1.VirtualMachine, minimumRequestInterval time.Duration) (time.Duration, error) {
	log := ctrllog.FromContext(ctx)

	if delta := checkForExistingVMRequest(ctx, instance.Name, minimumRequestInterval); delta != 0 {
		return delta, nil
	}

	log.Info("trigger create-vm webhook", "url", url)

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

	addInflightVMRequest(ctx, instance.Name, minimumRequestInterval)
	return 0, nil
}

func triggerDeleteVMWebHook(ctx context.Context, url string, instance *cloudkitv1alpha1.VirtualMachine, minimumRequestInterval time.Duration) (time.Duration, error) {
	log := ctrllog.FromContext(ctx)

	if delta := checkForExistingVMRequest(ctx, instance.Name, minimumRequestInterval); delta != 0 {
		return delta, nil
	}

	log.Info("trigger delete-vm webhook", "url", url)

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

	addInflightVMRequest(ctx, instance.Name, minimumRequestInterval)
	return 0, nil
}
