package controller

import (
	"fmt"

	v1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	defaultServiceAccountName string = "cloudkit"
	defaultHostedClusterName  string = "cluster"
	defaultRoleBindingName    string = "cloudkit"
	cloudkitAppName           string = "cloudkit-operator"

	cloudkitNamePrefix string = "cloudkit.openshift.io"
)

var (
	cloudkitClusterOrderNameLabel      string = fmt.Sprintf("%s/clusterorder", cloudkitNamePrefix)
	cloudkitClusterOrderNamespaceLabel string = fmt.Sprintf("%s/clusterordernamespace", cloudkitNamePrefix)
	cloudkitFinalizer                  string = fmt.Sprintf("%s/finalizer", cloudkitNamePrefix)
)

func generateNamespaceName(instance *v1alpha1.ClusterOrder) string {
	return fmt.Sprintf("cluster-%s-%s", instance.GetName(), rand.String(6))
}
