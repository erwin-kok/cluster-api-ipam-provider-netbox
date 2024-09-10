package controller

import (
	"context"
	"fmt"

	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/netbox"
	"github.com/pkg/errors"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	clusterutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	poolutil "github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/pool"
)

const (
	PoolFinalizer = "netboxippool.ipam.cluster.x-k8s.io"
)

func genericReconcile(ctx context.Context, c client.Client, pool ipamv1alpha1.GenericNetboxPool) (_ ctrl.Result, reterr error) {
	patchHelper, err := patch.NewHelper(pool, c)
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

	addressesInUse, err := poolutil.ListAddressesInUse(ctx, c, pool.GetNamespace(), poolTypeRef)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list addresses")
	}

	// Handle deleted pools
	if !pool.GetDeletionTimestamp().IsZero() {
		return reconcileDelete(ctx, c, pool, addressesInUse)
	}

	// If the Pool doesn't have our finalizer, add it.
	// Requeue immediately after adding finalizer to avoid the race condition between init and delete
	if !ctrlutil.ContainsFinalizer(pool, PoolFinalizer) {
		ctrlutil.AddFinalizer(pool, PoolFinalizer)
		return reconcile.Result{}, nil
	}

	// Handle non-deleted clusters
	return reconcileNormal(ctx, c, pool, addressesInUse)
}

func reconcileDelete(ctx context.Context, c client.Client, pool ipamv1alpha1.GenericNetboxPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	if len(addressesInUse) > 0 {
		logger.Info(
			fmt.Sprintf("%d addresses are still in use", len(addressesInUse)),
			"Pool", klog.KObj(pool),
		)
		return ctrl.Result{}, nil
	}

	if err := reconcileDeleteCredentialsSecret(ctx, c, pool); err != nil {
		return reconcile.Result{}, err
	}

	// Pool is deleted so remove the finalizer.
	ctrlutil.RemoveFinalizer(pool, PoolFinalizer)

	return reconcile.Result{}, nil
}

func reconcileNormal(ctx context.Context, c client.Client, pool ipamv1alpha1.GenericNetboxPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	secret, err := reconcileNormalCredentialsSecret(ctx, c, pool)
	if err != nil {
		logger.Error(err, "could not retrieve credentialsRef")
		return reconcile.Result{}, err
	}

	netboxIPPool, err := getNetboxIPPool(ctx, secret, pool)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get Netbox IPPool")
	}

	poolCount := (int)(netboxIPPool.Range.GetCount().Int64())
	if pool.PoolSpec().Gateway != "" {
		gatewayAddress, err := ipaddr.NewIPAddressString(pool.PoolSpec().Gateway).ToAddress()
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to parse pool gateway")
		}

		if netboxIPPool.Range.Contains(gatewayAddress) {
			poolCount--
		}
	}

	inUseCount := len(addressesInUse)
	free := poolCount - inUseCount

	pool.PoolStatus().Addresses = &ipamv1alpha1.NetboxPoolStatusIPAddresses{
		Total: poolCount,
		Used:  inUseCount,
		Free:  free,
	}

	logger.Info("Updating pool with usage info", "statusAddresses", pool.PoolStatus().Addresses)

	return ctrl.Result{}, nil
}

func reconcileNormalCredentialsSecret(ctx context.Context, c client.Client, pool ipamv1alpha1.GenericNetboxPool) (*corev1.Secret, error) {
	secret, err := getCredentialsRef(ctx, c, pool)
	if err != nil {
		return nil, err
	}

	helper, err := patch.NewHelper(secret, c)
	if err != nil {
		return nil, err
	}

	// Ensure the pool is an owner and that the APIVersion is up-to-date.
	secret.SetOwnerReferences(clusterutil.EnsureOwnerRef(secret.GetOwnerReferences(),
		metav1.OwnerReference{
			APIVersion: ipamv1alpha1.GroupVersion.String(),
			Kind:       pool.GetKind(),
			Name:       pool.GetName(),
			UID:        pool.GetUID(),
		},
	))

	// Ensure the finalizer is added.
	if !ctrlutil.ContainsFinalizer(secret, ipamv1alpha1.SecretFinalizer) {
		ctrlutil.AddFinalizer(secret, ipamv1alpha1.SecretFinalizer)
	}

	err = helper.Patch(ctx, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func reconcileDeleteCredentialsSecret(ctx context.Context, c client.Client, pool ipamv1alpha1.GenericNetboxPool) error {
	secret, err := getCredentialsRef(ctx, c, pool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	logger := ctrl.LoggerFrom(ctx)

	helper, err := patch.NewHelper(secret, c)
	if err != nil {
		return err
	}

	// Remove the pool from the OwnerRef.
	secret.SetOwnerReferences(clusterutil.RemoveOwnerRef(secret.GetOwnerReferences(),
		metav1.OwnerReference{
			APIVersion: ipamv1alpha1.GroupVersion.String(),
			Kind:       pool.GetKind(),
			Name:       pool.GetName(),
			UID:        pool.GetUID(),
		},
	))

	if len(secret.GetOwnerReferences()) <= 0 && ctrlutil.ContainsFinalizer(secret, ipamv1alpha1.SecretFinalizer) {
		logger.Info(fmt.Sprintf("Removing finalizer %s", ipamv1alpha1.SecretFinalizer), "Secret", klog.KObj(secret))
		ctrlutil.RemoveFinalizer(secret, ipamv1alpha1.SecretFinalizer)
	}

	return helper.Patch(ctx, secret)
}

func getCredentialsRef(ctx context.Context, c client.Client, pool ipamv1alpha1.GenericNetboxPool) (*corev1.Secret, error) {
	credRef := pool.PoolSpec().CredentialsRef
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
	err := c.Get(ctx, secretKey, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func getNetboxIPPool(ctx context.Context, secret *corev1.Secret, pool ipamv1alpha1.GenericNetboxPool) (*netbox.NetboxIPPool, error) {
	url := getData(secret, netbox.UrlKey)
	if url != "" {
		return nil, errors.New("can not connect to Netbox, secret must contain url")
	}
	apiToken := getData(secret, netbox.ApiTokenKey)
	if url != "" {
		return nil, errors.New("can not connect to Netbox, secret must contain apiToken")
	}

	nb := netbox.NewNetBoxClient(url, apiToken)

	switch pool.PoolSpec().Type {
	case ipamv1alpha1.PrefixType:
		return nb.GetPrefix(ctx, pool.PoolSpec().Address, pool.PoolSpec().Vrf)

	case ipamv1alpha1.IPRangeType:
		return nb.GetIPRange(ctx, pool.PoolSpec().Address, pool.PoolSpec().Vrf)
	}
	return nil, errors.New(fmt.Sprintf("unknown IPPoolType %s", pool.PoolSpec().Type))
}

func getData(secret *corev1.Secret, key string) string {
	if secret.Data == nil {
		return ""
	}
	if val, ok := secret.Data[key]; ok {
		return string(val)
	}
	return ""
}
