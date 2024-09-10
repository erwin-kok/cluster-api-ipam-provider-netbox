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

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NetboxGlobalIPPoolReconciler reconciles a NetboxGlobalIPPool object
type NetboxGlobalIPPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func AddNetboxGlobalIPPoolReconciler(mgr manager.Manager) error {
	reconciler := &NetboxGlobalIPPoolReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&ipamv1alpha1.NetboxGlobalIPPool{}).
		Watches(
			&ipamv1.IPAddress{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, clientObj client.Object) []reconcile.Request {
				ipAddress, ok := clientObj.(*ipamv1.IPAddress)
				if !ok {
					return nil
				}
				return ipAddressToNetboxIPPool(ipAddress, ipamv1alpha1.NetboxIPGlobalPoolKind)
			}),
		).
		Complete(reconciler)
}

// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxglobalippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxglobalippools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxglobalippools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NetboxGlobalIPPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling NetboxGlobalIPPool")

	pool := &ipamv1alpha1.NetboxGlobalIPPool{}
	if err := r.Client.Get(ctx, req.NamespacedName, pool); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, errors.Wrap(err, "could not fetch NetboxGlobalIPPool")
		}
		return ctrl.Result{}, nil
	}
	return genericReconcile(ctx, r.Client, pool)
}