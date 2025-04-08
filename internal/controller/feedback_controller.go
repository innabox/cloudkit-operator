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

// Package controller implements the controller logic
package controller

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/innabox/cloudkit-operator/api/v1alpha1"
	fulfillmentv1 "github.com/innabox/cloudkit-operator/internal/api/fulfillment/v1"
	sharedv1 "github.com/innabox/cloudkit-operator/internal/api/shared/v1"
)

// FeedbackReconciler sends updates to the fulfillment service API.
type FeedbackReconciler struct {
	client.Client
	ClusterOrdersClient fulfillmentv1.ClusterOrdersClient
	ClustersClient      fulfillmentv1.ClustersClient
}

func NewFeedbackReconciler(client client.Client, grpcConn *grpc.ClientConn) *FeedbackReconciler {
	return &FeedbackReconciler{
		Client:              client,
		ClusterOrdersClient: fulfillmentv1.NewClusterOrdersClient(grpcConn),
		ClustersClient:      fulfillmentv1.NewClustersClient(grpcConn),
	}
}

func (r *FeedbackReconciler) Reconcile(ctx context.Context, request ctrl.Request) (result ctrl.Result, err error) {
	instance := &v1alpha1.ClusterOrder{}
	err = r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		err = client.IgnoreNotFound(err)
		return
	}
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		result, err = r.handleUpdate(ctx, instance)
	} else {
		result, err = r.handleDelete(ctx, instance)
	}
	return
}

func (r *FeedbackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterOrder{}).
		Complete(r)
}

func (r *FeedbackReconciler) handleUpdate(ctx context.Context, instance *v1alpha1.ClusterOrder) (result ctrl.Result,
	err error) {
	logger := ctrllog.FromContext(ctx)

	// Get the identifier of the order from the labels:
	clusterOrderId, ok := instance.Labels[cloudkitClusterOrderIdLabel]
	if !ok {
		logger.Info(
			"There is no label containing the cluster order identifier, will ignore it",
			"label", cloudkitClusterOrderIdLabel,
		)
		return
	}

	// Try to fetch the current representation of the cluster order:
	getRequest, err := r.ClusterOrdersClient.Get(ctx, &fulfillmentv1.ClusterOrdersGetRequest{
		Id: clusterOrderId,
	})
	if grpcstatus.Code(err) == grpccodes.NotFound {
		logger.Info(
			"Cluster order doesn't exist, will ignore it",
			"id", clusterOrderId,
			"error", err.Error(),
		)
		err = nil
		return
	}
	if err != nil {
		return
	}

	// Update the conditions to indicate that we the order has been accepted:
	clusterOrder := getRequest.Object
	if clusterOrder.Status == nil {
		clusterOrder.Status = &fulfillmentv1.ClusterOrderStatus{}
	}
	acceptedCondition := r.findOrCreateCondition(
		clusterOrder,
		fulfillmentv1.ClusterOrderConditionType_CLUSTER_ORDER_CONDITION_TYPE_ACCEPTED,
	)
	acceptedCondition.Status = sharedv1.ConditionStatus_CONDITION_STATUS_TRUE
	acceptedCondition.Reason = proto.String("KubernetesObjectCreated")
	acceptedCondition.Message = proto.String(fmt.Sprintf(
		"The kubernetes object '%s' has been created in namespace '%s'",
		instance.Name, instance.Namespace,
	))

	// Update the cluster order:
	_, err = r.ClusterOrdersClient.Update(ctx, &fulfillmentv1.ClusterOrdersUpdateRequest{
		Object: clusterOrder,
	})
	if err != nil {
		return
	}
	logger.Info(
		"Cluster order updated",
		"id", clusterOrderId,
	)

	return
}

func (r *FeedbackReconciler) handleDelete(ctx context.Context, instance *v1alpha1.ClusterOrder) (result ctrl.Result,
	err error) {
	// TODO.
	return
}

func (r *FeedbackReconciler) findOrCreateCondition(clusterOrder *fulfillmentv1.ClusterOrder,
	conditionType fulfillmentv1.ClusterOrderConditionType) *fulfillmentv1.ClusterOrderCondition {
	var condition *fulfillmentv1.ClusterOrderCondition
	for _, current := range clusterOrder.Status.Conditions {
		if current.Type == conditionType {
			condition = current
			break
		}
	}
	if condition == nil {
		condition = &fulfillmentv1.ClusterOrderCondition{
			Type:               conditionType,
			Status:             sharedv1.ConditionStatus_CONDITION_STATUS_FALSE,
			LastTransitionTime: timestamppb.Now(),
		}
		clusterOrder.Status.Conditions = append(clusterOrder.Status.Conditions, condition)
	}
	return condition
}
