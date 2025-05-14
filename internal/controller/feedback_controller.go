/*
Copyright (c) 2025 Red Hat Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the
License. You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific
language governing permissions and limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	clnt "sigs.k8s.io/controller-runtime/pkg/client"

	ckv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
	ffv1 "github.com/innabox/cloudkit-operator/internal/api/fulfillment/v1"
	privatev1 "github.com/innabox/cloudkit-operator/internal/api/private/v1"
	sharedv1 "github.com/innabox/cloudkit-operator/internal/api/shared/v1"
)

// FeedbackReconciler sends updates to the fulfillment service.
type FeedbackReconciler struct {
	logger                logr.Logger
	hubClient             clnt.Client
	publicOrdersClient    ffv1.ClusterOrdersClient
	privateOrdersClient   privatev1.ClusterOrdersClient
	publicClustersClient  ffv1.ClustersClient
	privateClustersClient privatev1.ClustersClient
}

// feedbackReconcilerTask contains data that is used for the reconciliation of a specific cluster order, so there is less
// need to pass around as function parameters that and other related objects.
type feedbackReconcilerTask struct {
	r              *FeedbackReconciler
	object         *ckv1alpha1.ClusterOrder
	publicOrder    *ffv1.ClusterOrder
	privateOrder   *privatev1.ClusterOrder
	publicCluster  *ffv1.Cluster
	privateCluster *privatev1.Cluster
	hostedCluster  *hypershiftv1beta1.HostedCluster
}

// NewFeedbackReconciler creates a reconciler that sends to the fulfillment service updates about cluster orders.
func NewFeedbackReconciler(logger logr.Logger, hubClient clnt.Client, grpcConn *grpc.ClientConn) *FeedbackReconciler {
	return &FeedbackReconciler{
		logger:                logger,
		hubClient:             hubClient,
		publicOrdersClient:    ffv1.NewClusterOrdersClient(grpcConn),
		privateOrdersClient:   privatev1.NewClusterOrdersClient(grpcConn),
		publicClustersClient:  ffv1.NewClustersClient(grpcConn),
		privateClustersClient: privatev1.NewClustersClient(grpcConn),
	}
}

// SetupWithManager adds the reconciler to the controller manager.
func (r *FeedbackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("clusterorder-feedback").
		For(&ckv1alpha1.ClusterOrder{}).
		Complete(r)
}

// Reconcile is the implementation of the reconciler interface.
func (r *FeedbackReconciler) Reconcile(ctx context.Context, request ctrl.Request) (result ctrl.Result, err error) {
	// Fetch the object to reconcile, and do nothing if it no longer exists:
	object := &ckv1alpha1.ClusterOrder{}
	err = r.hubClient.Get(ctx, request.NamespacedName, object)
	if err != nil {
		err = clnt.IgnoreNotFound(err)
		return //nolint:nakedret
	}

	// Get the identifier of the order from the labels. If this isn't present it means that the object wasn't
	// created by the fulfillment service, so we ignore it.
	orderID, ok := object.Labels[cloudkitClusterOrderIDLabel]
	if !ok {
		r.logger.Info(
			"There is no label containing the order identifier, will ignore it",
			"label", cloudkitClusterOrderIDLabel,
		)
		return
	}

	// Fetch all the relevant objects:
	publicOrder, privateOrder, err := r.fetchOrder(ctx, orderID)
	if err != nil {
		return
	}
	var publicCluster *ffv1.Cluster
	var privateCluster *privatev1.Cluster
	clusterID := publicOrder.GetStatus().GetClusterId()
	if clusterID != "" {
		publicCluster, privateCluster, err = r.fetchCluster(ctx, clusterID)
		if err != nil {
			return
		}
	}

	// Create a task to do the rest of the job, but using copies of the objects, so that we can later compare the
	// before and after values and save only the objects that have changed.
	t := &feedbackReconcilerTask{
		r:              r,
		object:         object,
		publicOrder:    clone(publicOrder),
		privateOrder:   clone(privateOrder),
		publicCluster:  clone(publicCluster),
		privateCluster: clone(privateCluster),
	}
	if object.ObjectMeta.DeletionTimestamp.IsZero() {
		result, err = t.handleUpdate(ctx)
	} else {
		result, err = t.handleDelete(ctx)
	}
	if err != nil {
		return
	}

	// Save the objects that have changed:
	err = r.saveCluster(ctx, publicCluster, t.publicCluster, privateCluster, t.privateCluster)
	if err != nil {
		return
	}
	err = r.saveOrder(ctx, publicOrder, t.publicOrder, privateOrder, t.privateOrder)
	if err != nil {
		return
	}
	return
}

func (r *FeedbackReconciler) fetchOrder(ctx context.Context, id string) (public *ffv1.ClusterOrder,
	private *privatev1.ClusterOrder, err error) {
	publicResponse, err := r.publicOrdersClient.Get(ctx, ffv1.ClusterOrdersGetRequest_builder{
		Id: id,
	}.Build())
	if err != nil {
		return
	}
	privateResponse, err := r.privateOrdersClient.Get(ctx, privatev1.ClusterOrdersGetRequest_builder{
		Id: id,
	}.Build())
	if err != nil {
		return
	}
	public = publicResponse.GetObject()
	if !public.HasSpec() {
		public.SetSpec(&ffv1.ClusterOrderSpec{})
	}
	if !public.HasStatus() {
		public.SetStatus(&ffv1.ClusterOrderStatus{})
	}
	private = privateResponse.GetObject()
	return
}

func (r *FeedbackReconciler) fetchCluster(ctx context.Context, id string) (public *ffv1.Cluster,
	private *privatev1.Cluster, err error) {
	publicResponse, err := r.publicClustersClient.Get(ctx, ffv1.ClustersGetRequest_builder{
		Id: id,
	}.Build())
	if err != nil {
		return
	}
	privateResponse, err := r.privateClustersClient.Get(ctx, privatev1.ClustersGetRequest_builder{
		Id: id,
	}.Build())
	if err != nil {
		return
	}
	public = publicResponse.GetObject()
	if !public.HasSpec() {
		public.SetSpec(&ffv1.ClusterSpec{})
	}
	if !public.HasStatus() {
		public.SetStatus(&ffv1.ClusterStatus{})
	}
	private = privateResponse.GetObject()
	return
}

func (r *FeedbackReconciler) saveOrder(ctx context.Context, publicBefore, publicAfter *ffv1.ClusterOrder,
	privateBefore, privateAfter *privatev1.ClusterOrder) error {
	if !equal(publicAfter, publicBefore) {
		_, err := r.publicOrdersClient.Update(ctx, ffv1.ClusterOrdersUpdateRequest_builder{
			Object: publicAfter,
		}.Build())
		if err != nil {
			return err
		}
	}
	if !equal(privateAfter, privateBefore) {
		_, err := r.privateOrdersClient.Update(ctx, privatev1.ClusterOrdersUpdateRequest_builder{
			Object: privateAfter,
		}.Build())
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *FeedbackReconciler) saveCluster(ctx context.Context, publicBefore, publicAfter *ffv1.Cluster,
	privateBefore, privateAfter *privatev1.Cluster) error {
	if !equal(publicAfter, publicBefore) {
		_, err := r.publicClustersClient.Update(ctx, ffv1.ClustersUpdateRequest_builder{
			Object: publicAfter,
		}.Build())
		if err != nil {
			return err
		}
	}
	if !equal(privateAfter, privateBefore) {
		_, err := r.privateClustersClient.Update(ctx, privatev1.ClustersUpdateRequest_builder{
			Object: privateAfter,
		}.Build())
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *feedbackReconcilerTask) handleUpdate(ctx context.Context) (result ctrl.Result, err error) {
	err = t.syncConditions()
	if err != nil {
		return
	}
	err = t.syncPhase(ctx)
	if err != nil {
		return
	}
	return
}

func (t *feedbackReconcilerTask) syncConditions() error {
	for _, condition := range t.object.Status.Conditions {
		err := t.syncCondition(condition)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *feedbackReconcilerTask) syncCondition(condition metav1.Condition) error {
	switch ckv1alpha1.ClusterOrderConditionType(condition.Type) {
	case ckv1alpha1.ClusterOrderConditionAccepted:
		return t.syncConditionAccepted(condition)
	case ckv1alpha1.ClusterOrderConditionProgressing:
		// TODO: There is no equivalent condition.
	case ckv1alpha1.ClusterOrderConditionControlPlaneAvailable:
		return t.syncConditionFulfilled(condition)
	case ckv1alpha1.ClusterOrderConditionAvailable:
		// TODO: There is no equivalent condition.
	default:
		t.r.logger.Info(
			"Unknown condition, will ignore it",
			"condition", condition.Type,
		)
	}
	return nil
}

func (t *feedbackReconcilerTask) syncConditionFulfilled(condition metav1.Condition) error {
	orderCondition := t.findOrderCondition(ffv1.ClusterOrderConditionType_CLUSTER_ORDER_CONDITION_TYPE_FULFILLED)
	oldStatus := orderCondition.GetStatus()
	newStatus := t.mapConditionStatus(condition.Status)
	orderCondition.SetStatus(newStatus)
	orderCondition.SetMessage(condition.Message)
	if newStatus != oldStatus {
		orderCondition.SetLastTransitionTime(timestamppb.Now())
	}
	return nil
}

func (t *feedbackReconcilerTask) syncConditionAccepted(condition metav1.Condition) error {
	orderCondition := t.findOrderCondition(ffv1.ClusterOrderConditionType_CLUSTER_ORDER_CONDITION_TYPE_ACCEPTED)
	oldStatus := orderCondition.GetStatus()
	newStatus := t.mapConditionStatus(condition.Status)
	orderCondition.SetStatus(newStatus)
	orderCondition.SetMessage(condition.Message)
	if newStatus != oldStatus {
		orderCondition.SetLastTransitionTime(timestamppb.Now())
	}
	return nil
}

func (t *feedbackReconcilerTask) mapConditionStatus(status metav1.ConditionStatus) sharedv1.ConditionStatus {
	switch status {
	case metav1.ConditionFalse:
		return sharedv1.ConditionStatus_CONDITION_STATUS_FALSE
	case metav1.ConditionTrue:
		return sharedv1.ConditionStatus_CONDITION_STATUS_TRUE
	default:
		return sharedv1.ConditionStatus_CONDITION_STATUS_UNSPECIFIED
	}
}

func (t *feedbackReconcilerTask) syncPhase(ctx context.Context) error {
	switch t.object.Status.Phase {
	case ckv1alpha1.ClusterOrderPhaseAccepted:
		return t.syncPhaseAccepted()
	case ckv1alpha1.ClusterOrderPhaseProgressing:
		return t.syncPhaseProgressing()
	case ckv1alpha1.ClusterOrderPhaseFailed:
		return t.syncPhaseFailed()
	case ckv1alpha1.ClusterOrderPhaseReady:
		return t.syncPhaseReady(ctx)
	default:
		t.r.logger.Info(
			"Unknown phase, will ignore it",
			"phase", t.object.Status.Phase,
		)
		return nil
	}
}

func (t *feedbackReconcilerTask) syncPhaseAccepted() error {
	t.publicOrder.GetStatus().SetState(ffv1.ClusterOrderState_CLUSTER_ORDER_STATE_PROGRESSING)
	return nil
}

func (t *feedbackReconcilerTask) syncPhaseProgressing() error {
	t.publicOrder.GetStatus().SetState(ffv1.ClusterOrderState_CLUSTER_ORDER_STATE_PROGRESSING)
	return nil
}

func (t *feedbackReconcilerTask) syncPhaseFailed() error {
	t.publicOrder.GetStatus().SetState(ffv1.ClusterOrderState_CLUSTER_ORDER_STATE_FAILED)
	return nil
}

func (t *feedbackReconcilerTask) syncPhaseReady(ctx context.Context) error {
	// If the order doesn't a have a reference to the cluster yet, then we need to create the new cluster, otherwise
	// the details will have been already fetched.
	publicOrderStatus := t.publicOrder.GetStatus()
	if publicOrderStatus.GetClusterId() == "" {
		err := t.createCluster(ctx)
		if err != nil {
			return err
		}
	}

	// Set the status of the order:
	publicOrderStatus.SetState(ffv1.ClusterOrderState_CLUSTER_ORDER_STATE_FULFILLED)
	publicOrderStatus.SetClusterId(t.publicCluster.GetId())

	// Set the status of the cluster:
	publicClusterStatus := t.publicCluster.GetStatus()
	publicClusterStatus.SetState(ffv1.ClusterState_CLUSTER_STATE_READY)

	// In order to get the API and console URL we need to fetch the hosted cluster:
	err := t.fetchHostedCluster(ctx)
	if err != nil {
		return err
	}
	apiURL := t.calculateAPIURL()
	if apiURL != "" {
		publicClusterStatus.SetApiUrl(apiURL)
	}
	consoleURL := t.calculateConsoleURL()
	if consoleURL != "" {
		publicClusterStatus.SetConsoleUrl(consoleURL)
	}

	// Save the order and hub identifiers in the private cluster:
	t.privateCluster.SetOrderId(t.publicOrder.GetId())
	t.privateCluster.SetHubId(t.privateOrder.GetHubId())

	return nil
}

func (t *feedbackReconcilerTask) createCluster(ctx context.Context) error {
	// Create the empty cluster:
	publicResponse, err := t.r.publicClustersClient.Create(ctx, ffv1.ClustersCreateRequest_builder{
		Object: ffv1.Cluster_builder{
			Spec:   &ffv1.ClusterSpec{},
			Status: &ffv1.ClusterStatus{},
		}.Build(),
	}.Build())
	if err != nil {
		return err
	}
	public := publicResponse.GetObject()

	// The private cluster will be created automatically by the server, so we only need to fetch it:
	privateResponse, err := t.r.privateClustersClient.Get(ctx, privatev1.ClustersGetRequest_builder{
		Id: public.GetId(),
	}.Build())
	if err != nil {
		return err
	}
	private := privateResponse.GetObject()

	// Save the results:
	t.publicCluster = public
	t.privateCluster = private

	return nil
}

func (t *feedbackReconcilerTask) fetchHostedCluster(ctx context.Context) error {
	hostedClusterRef := t.object.Status.ClusterReference
	if hostedClusterRef == nil || hostedClusterRef.Namespace == "" || hostedClusterRef.HostedClusterName == "" {
		return nil
	}
	hostedClusterKey := clnt.ObjectKey{
		Namespace: hostedClusterRef.Namespace,
		Name:      hostedClusterRef.HostedClusterName,
	}
	hostedCluster := &hypershiftv1beta1.HostedCluster{}
	err := t.r.hubClient.Get(ctx, hostedClusterKey, hostedCluster)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	t.hostedCluster = hostedCluster
	return nil
}

func (t *feedbackReconcilerTask) calculateAPIURL() string {
	if t.hostedCluster == nil {
		return ""
	}
	apiEndpoint := t.hostedCluster.Status.ControlPlaneEndpoint
	if apiEndpoint.Host == "" || apiEndpoint.Port == 0 {
		return ""
	}
	return fmt.Sprintf("https://%s:%d", apiEndpoint.Host, apiEndpoint.Port)
}

func (t *feedbackReconcilerTask) calculateConsoleURL() string {
	if t.hostedCluster == nil {
		return ""
	}
	return fmt.Sprintf(
		"https://console-openshift-console.apps.%s.%s",
		t.hostedCluster.Name, t.hostedCluster.Spec.DNS.BaseDomain,
	)
}

func (t *feedbackReconcilerTask) handleDelete(ctx context.Context) (result ctrl.Result, err error) {
	// TODO.
	return
}

func (t *feedbackReconcilerTask) findOrderCondition(kind ffv1.ClusterOrderConditionType) *ffv1.ClusterOrderCondition {
	var condition *ffv1.ClusterOrderCondition
	for _, current := range t.publicOrder.Status.Conditions {
		if current.Type == kind {
			condition = current
			break
		}
	}
	if condition == nil {
		condition = &ffv1.ClusterOrderCondition{
			Type:   kind,
			Status: sharedv1.ConditionStatus_CONDITION_STATUS_FALSE,
		}
		t.publicOrder.Status.Conditions = append(t.publicOrder.Status.Conditions, condition)
	}
	return condition
}

func clone[M proto.Message](message M) M {
	return proto.Clone(message).(M)
}

func equal[M proto.Message](x, y M) bool {
	return proto.Equal(x, y)
}
