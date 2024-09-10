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

package webhooks

import (
	"context"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
)

// log is for logging in this package.
var netboxiprangeglobalpoollog = logf.Log.WithName("netboxippool-resource")

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-ipam-cluster-x-k8s-io-v1alpha1-netboxprefixpool,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxprefixpools,versions=v1alpha2,name=validation.netboxprefixpool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-ipam-cluster-x-k8s-io-v1alpha1-netboxprefixpool,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxprefixpools,versions=v1alpha2,name=default.netboxprefixpool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-ipam-cluster-x-k8s-io-v1alpha1-netboxprefixglobalpool,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxprefixglobalpools,versions=v1alpha2,name=validation.netboxprefixglobalpool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-ipam-cluster-x-k8s-io-v1alpha1-netboxprefixglobalpool,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxprefixglobalpools,versions=v1alpha2,name=default.netboxprefixglobalpool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-ipam-cluster-x-k8s-io-v1alpha1-netboxiprangepool,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxiprangepools,versions=v1alpha2,name=validation.netboxiprangepool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-ipam-cluster-x-k8s-io-v1alpha1-netboxiprangepool,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxiprangepools,versions=v1alpha2,name=default.netboxiprangepool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-ipam-cluster-x-k8s-io-v1alpha1-netboxiprangeglobalpool,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxiprangeglobalpools,versions=v1alpha2,name=validation.netboxiprangeglobalpool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-ipam-cluster-x-k8s-io-v1alpha1-netboxiprangeglobalpool,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=ipam.cluster.x-k8s.io,resources=netboxiprangeglobalpools,versions=v1alpha2,name=default.netboxiprangeglobalpool.ipam.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

// NetboxIPPool implements a validating and defaulting webhook for NetboxPrefixPool, NetboxPrefixGlobalPool, GlobalInClusterIPPool, NetboxIPRangePool and NetboxIPRangeGlobalPool.
type NetboxIPPool struct {
	Client client.Reader
}

var (
	_ webhook.CustomDefaulter = &NetboxIPPool{}
	_ webhook.CustomValidator = &NetboxIPPool{}
)

// SetupWebhookWithManager will set up the manager to manage the webhooks
func (w *NetboxIPPool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&ipamv1alpha1.NetboxIPPool{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
	if err != nil {
		return err
	}
	err = ctrl.NewWebhookManagedBy(mgr).
		For(&ipamv1alpha1.NetboxGlobalIPPool{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
	if err != nil {
		return err
	}
	return nil
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *NetboxIPPool) Default(_ context.Context, obj runtime.Object) error {
	pool, ok := obj.(ipamv1alpha1.GenericNetboxPool)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", obj))
	}
	netboxiprangeglobalpoollog.Info("default", "name", pool.GetName())
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *NetboxIPPool) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	pool, ok := obj.(ipamv1alpha1.GenericNetboxPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", obj))
	}

	netboxiprangeglobalpoollog.Info("validate create", "name", pool.GetName())

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *NetboxIPPool) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldPool, ok := oldObj.(ipamv1alpha1.GenericNetboxPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", oldObj))
	}

	newPool, ok := newObj.(ipamv1alpha1.GenericNetboxPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", newObj))
	}

	netboxiprangeglobalpoollog.Info("validate update", "oldPool", oldPool.GetName(), "newPool", newPool.GetName())

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *NetboxIPPool) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	pool, ok := obj.(ipamv1alpha1.GenericNetboxPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", obj))
	}

	netboxiprangeglobalpoollog.Info("validate delete", "name", pool.GetName())

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, apierrors.NewBadRequest(fmt.Sprintf("XXX REMOVE a %T", obj))
}
