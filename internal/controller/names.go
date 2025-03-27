package controller

import "fmt"

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
