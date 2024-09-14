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
	"fmt"

	"github.com/pkg/errors"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	clusterutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/logger"
	poolutil "github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/pool"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/pkg/netbox"
)

const (
	PoolFinalizer   = "netbox.ipam.cluster.x-k8s.io/ippool"
	SecretFinalizer = "netbox.ipam.cluster.x-k8s.io/Secret"
)

// NetboxIPPoolReconciler reconciles a NetboxIPPool object
type NetboxIPPoolReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	netboxServiceFactory func(url, apiToken string) (netbox.Client, error)
}

func (r *NetboxIPPoolReconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ipamv1alpha1.NetboxIPPool{}).
		Watches(
			&ipamv1.IPAddress{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, clientObj client.Object) []reconcile.Request {
				ipAddress, ok := clientObj.(*ipamv1.IPAddress)
				if !ok {
					return nil
				}
				return ipAddressToNetboxIPPool(ipAddress)
			}),
		).
		Complete(r)
}

// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NetboxIPPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := logger.FromContext(ctx)
	log.Info("Reconciling NetboxIPPool")

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

	poolTypeRef := corev1.TypedLocalObjectReference{
		APIGroup: ptr.To(ipamv1alpha1.GroupVersion.Group),
		Kind:     pool.GetObjectKind().GroupVersionKind().Kind,
		Name:     pool.GetName(),
	}

	addressesInUse, err := poolutil.ListAddressesInUse(ctx, r.Client, pool.GetNamespace(), poolTypeRef)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list addresses")
	}

	// Handle deleted pools
	if !pool.GetDeletionTimestamp().IsZero() {
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

func (r *NetboxIPPoolReconciler) reconcileDelete(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) { //nolint:unparam
	log := logger.FromContext(ctx)

	if len(addressesInUse) > 0 {
		log.Info(
			fmt.Sprintf("%d addresses are still in use", len(addressesInUse)),
			"Pool", klog.KObj(pool),
		)
		return ctrl.Result{}, nil
	}

	if err := r.reconcileDeleteCredentialsSecret(ctx, pool); err != nil {
		return reconcile.Result{}, err
	}

	// Pool is deleted so remove the finalizer.
	ctrlutil.RemoveFinalizer(pool, PoolFinalizer)

	return reconcile.Result{}, nil
}

func (r *NetboxIPPoolReconciler) reconcileNormal(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) { //nolint:unparam
	log := logger.FromContext(ctx)

	secret, err := r.reconcileNormalCredentialsSecret(ctx, pool)
	if err != nil {
		log.Error(err, "could not retrieve credentialsRef")
		return reconcile.Result{}, err
	}

	netboxIPPool, err := r.getNetboxIPPool(ctx, secret, pool)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get Netbox IPPool")
	}

	poolCount := (int)(netboxIPPool.Range.GetCount().Int64())
	if pool.Spec.Gateway != "" {
		gatewayAddress, err := ipaddr.NewIPAddressString(pool.Spec.Gateway).ToAddress()
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to parse pool gateway")
		}

		if netboxIPPool.Range.Contains(gatewayAddress) {
			poolCount--
		}
	}

	inUseCount := len(addressesInUse)
	free := poolCount - inUseCount

	pool.Status.Addresses = &ipamv1alpha1.NetboxPoolStatusIPAddresses{
		Total: poolCount,
		Used:  inUseCount,
		Free:  free,
	}

	log.Info("Updating pool with usage info", "statusAddresses", pool.Status.Addresses)

	return ctrl.Result{}, nil
}

func (r *NetboxIPPoolReconciler) reconcileNormalCredentialsSecret(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool) (*corev1.Secret, error) {
	secret, err := r.getCredentialsRef(ctx, pool)
	if err != nil {
		return nil, err
	}

	helper, err := patch.NewHelper(secret, r.Client)
	if err != nil {
		return nil, err
	}

	// Ensure the pool is an owner and that the APIVersion is up-to-date.
	secret.SetOwnerReferences(clusterutil.EnsureOwnerRef(secret.GetOwnerReferences(),
		metav1.OwnerReference{
			APIVersion: ipamv1alpha1.GroupVersion.String(),
			Kind:       ipamv1alpha1.NetboxIPPoolKind,
			Name:       pool.GetName(),
			UID:        pool.GetUID(),
		},
	))

	// Ensure the finalizer is added.
	if !ctrlutil.ContainsFinalizer(secret, SecretFinalizer) {
		ctrlutil.AddFinalizer(secret, SecretFinalizer)
	}

	err = helper.Patch(ctx, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func (r *NetboxIPPoolReconciler) reconcileDeleteCredentialsSecret(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool) error {
	secret, err := r.getCredentialsRef(ctx, pool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	log := logger.FromContext(ctx)

	helper, err := patch.NewHelper(secret, r.Client)
	if err != nil {
		return err
	}

	// Remove the pool from the OwnerRef.
	secret.SetOwnerReferences(clusterutil.RemoveOwnerRef(secret.GetOwnerReferences(),
		metav1.OwnerReference{
			APIVersion: ipamv1alpha1.GroupVersion.String(),
			Kind:       ipamv1alpha1.NetboxIPPoolKind,
			Name:       pool.GetName(),
			UID:        pool.GetUID(),
		},
	))

	if len(secret.GetOwnerReferences()) <= 0 && ctrlutil.ContainsFinalizer(secret, SecretFinalizer) {
		log.Info(fmt.Sprintf("Removing finalizer %s", SecretFinalizer), "Secret", klog.KObj(secret))
		ctrlutil.RemoveFinalizer(secret, SecretFinalizer)
	}

	return helper.Patch(ctx, secret)
}

func (r *NetboxIPPoolReconciler) getCredentialsRef(ctx context.Context, pool *ipamv1alpha1.NetboxIPPool) (*corev1.Secret, error) {
	credRef := pool.Spec.CredentialsRef
	if credRef == nil {
		return nil, errors.New("pool does not has a CredentialsRef")
	}

	namespace := credRef.Namespace
	if len(namespace) == 0 {
		namespace = pool.GetNamespace()
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: namespace,
		Name:      credRef.Name,
	}
	err := r.Client.Get(ctx, secretKey, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (r *NetboxIPPoolReconciler) getNetboxIPPool(ctx context.Context, secret *corev1.Secret, pool *ipamv1alpha1.NetboxIPPool) (*netbox.NetboxIPPool, error) {
	nb, err := getNetboxClient(secret, r.netboxServiceFactory)
	if err != nil {
		return nil, errors.Wrap(err, "could not create Netbox client")
	}

	switch pool.Spec.Type {
	case ipamv1alpha1.PrefixType:
		return nb.GetPrefix(ctx, pool.Spec.CIDR, pool.Spec.Vrf)

	case ipamv1alpha1.IPRangeType:
		return nb.GetIPRange(ctx, pool.Spec.CIDR, pool.Spec.Vrf)
	}
	return nil, errors.New(fmt.Sprintf("unknown IPPoolType %s", pool.Spec.Type))
}

func ipAddressToNetboxIPPool(ipAddress *ipamv1.IPAddress) []reconcile.Request {
	if ipAddress.Spec.PoolRef.APIGroup != nil &&
		*ipAddress.Spec.PoolRef.APIGroup == ipamv1alpha1.GroupVersion.Group &&
		ipAddress.Spec.PoolRef.Kind == ipamv1alpha1.NetboxIPPoolKind {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: ipAddress.Namespace,
				Name:      ipAddress.Spec.PoolRef.Name,
			},
		}}
	}
	return nil
}
