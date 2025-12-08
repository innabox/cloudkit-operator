package controller

import (
	"fmt"

	v1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
)

const (
	defaultServiceAccountName    string = "cloudkit"
	defaultHostedClusterName     string = "cluster"
	defaultRoleBindingName       string = "cloudkit"
	defaultClusterOrderNamespace string = "cloudkit-orders"
	cloudkitAppName              string = "cloudkit-operator"

	cloudkitNamePrefix string = "cloudkit.openshift.io"
)

var (
	cloudkitClusterOrderNameLabel     string = fmt.Sprintf("%s/clusterorder", cloudkitNamePrefix)
	cloudkitClusterOrderIDLabel       string = fmt.Sprintf("%s/clusterorder-uuid", cloudkitNamePrefix)
	cloudkitFinalizer                 string = fmt.Sprintf("%s/finalizer", cloudkitNamePrefix)
	cloudkitManagementStateAnnotation string = fmt.Sprintf("%s/management-state", cloudkitNamePrefix)
)

func generateNamespaceName(instance *v1alpha1.ClusterOrder) string {
	return fmt.Sprintf("%s-%s", instance.GetNamespace(), instance.GetName())
}
