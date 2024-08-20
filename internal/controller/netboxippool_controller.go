/*
Copyright 2024.

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
	"context"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
)

const (
	netBoxIPPoolKind = "NetboxIPPool"
	PoolFinalizer    = "netboxippool.ipam.cluster.x-k8s.io"
)

// NetboxIPPoolReconciler reconciles a NetboxIPPool object
type NetboxIPPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func AddNetboxIPPoolReconciler(mgr manager.Manager) error {
	reconciler := &NetboxIPPoolReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&ipamv1alpha1.NetboxIPPool{}).
		Watches(
			&ipamv1.IPAddress{},
			handler.EnqueueRequestsFromMapFunc(reconciler.ipAddressToNetboxIPPool)).
		Complete(reconciler)
}

func (r *NetboxIPPoolReconciler) ipAddressToNetboxIPPool(_ context.Context, clientObj client.Object) []reconcile.Request {
	ipAddress, ok := clientObj.(*ipamv1.IPAddress)
	if !ok {
		return nil
	}

	if ipAddress.Spec.PoolRef.APIGroup != nil &&
		*ipAddress.Spec.PoolRef.APIGroup == ipamv1alpha1.GroupVersion.Group &&
		ipAddress.Spec.PoolRef.Kind == netBoxIPPoolKind {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: ipAddress.Namespace,
				Name:      ipAddress.Spec.PoolRef.Name,
			},
		}}
	}
	return nil
}

// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NetboxIPPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling NetboxIPPool")

	pool := &ipamv1alpha1.NetboxIPPool{}
	if err := r.Client.Get(ctx, req.NamespacedName, pool); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, errors.Wrap(err, "could not fetch NetboxIPPool")
		}
		return ctrl.Result{}, nil
	}

	patchHelper, err := patch.NewHelper(pool, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		if err := patchHelper.Patch(ctx, pool); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	addressesInUse, err := r.getAddressesInUse(ctx, pool)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list addresses")
	}

	// Handle deleted pools
	if !pool.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, pool, addressesInUse)
	}

	// If the Pool doesn't have our finalizer, add it.
	// Requeue immediately after adding finalizer to avoid the race condition between init and delete
	if !ctrlutil.ContainsFinalizer(pool, PoolFinalizer) {
		ctrlutil.AddFinalizer(pool, PoolFinalizer)
		return reconcile.Result{}, nil
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, pool, addressesInUse)
}

func (r *NetboxIPPoolReconciler) reconcileDelete(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	if len(addressesInUse) > 0 {
		logger.Info("addresses are still in use", "Pool", klog.KObj(pool))
		return ctrl.Result{}, nil
	}

	// Pool is deleted so remove the finalizer.
	ctrlutil.RemoveFinalizer(pool, PoolFinalizer)

	return reconcile.Result{}, nil
}

func (r *NetboxIPPoolReconciler) reconcileNormal(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *NetboxIPPoolReconciler) getAddressesInUse(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool) ([]ipamv1.IPAddress, error) {
	return []ipamv1.IPAddress{}, nil
}
