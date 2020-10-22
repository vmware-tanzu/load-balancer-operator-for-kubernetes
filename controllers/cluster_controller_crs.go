package controllers

import (
	"context"

	"github.com/go-logr/logr"
	clusterv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// reconcileCRS creates the CRS for AKO deployment in workload clusters
func (r *ClusterReconciler) reconcileCRS(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1alpha3.Cluster,
) error {
	return nil
}
