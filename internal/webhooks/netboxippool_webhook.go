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

	poolutil "github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/pool"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api-ipam-provider-in-cluster/api/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
)

const (
	// SkipValidateDeleteWebhookAnnotation is an annotation that can be applied
	// to the NetboxIPPool to skip delete validation. Necessary for clusterctl move to work as expected.
	SkipValidateDeleteWebhookAnnotation = "ipam.cluster.x-k8s.io/skip-validate-delete-webhook"
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

// NetboxIPPoolWebhook implements a validating and defaulting webhook for NetboxIPPool.
type NetboxIPPoolWebhook struct {
	Client client.Reader
}

var (
	_ webhook.CustomDefaulter = &NetboxIPPoolWebhook{}
	_ webhook.CustomValidator = &NetboxIPPoolWebhook{}
)

// SetupWebhookWithManager will set up the manager to manage the webhooks
func (w *NetboxIPPoolWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&ipamv1alpha1.NetboxIPPool{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
	if err != nil {
		return err
	}
	return nil
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *NetboxIPPoolWebhook) Default(_ context.Context, obj runtime.Object) error {
	pool, ok := obj.(*ipamv1alpha1.NetboxIPPool)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", obj))
	}
	netboxiprangeglobalpoollog.Info("default", "name", pool.GetName())
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *NetboxIPPoolWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	pool, ok := obj.(*ipamv1alpha1.NetboxIPPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", obj))
	}
	netboxiprangeglobalpoollog.Info("validate create", "name", pool.GetName())
	return nil, w.validate(pool)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *NetboxIPPoolWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldPool, ok := oldObj.(*ipamv1alpha1.NetboxIPPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", oldObj))
	}
	newPool, ok := newObj.(*ipamv1alpha1.NetboxIPPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", newObj))
	}
	netboxiprangeglobalpoollog.Info("validate update", "oldPool", oldPool.GetName(), "newPool", newPool.GetName())
	err := w.validate(newPool)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *NetboxIPPoolWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	pool, ok := obj.(*ipamv1alpha1.NetboxIPPool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a NetboxPool but got a %T", obj))
	}
	netboxiprangeglobalpoollog.Info("validate delete", "name", pool.GetName())
	if _, ok := pool.GetAnnotations()[SkipValidateDeleteWebhookAnnotation]; ok {
		return nil, nil
	}
	poolTypeRef := corev1.TypedLocalObjectReference{
		APIGroup: ptr.To[string](pool.GetObjectKind().GroupVersionKind().Group),
		Kind:     pool.GetObjectKind().GroupVersionKind().Kind,
		Name:     pool.GetName(),
	}
	inUseAddresses, err := poolutil.ListAddressesInUse(ctx, w.Client, pool.GetNamespace(), poolTypeRef)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if len(inUseAddresses) > 0 {
		return nil, apierrors.NewBadRequest("Pool has IPAddresses allocated. Cannot delete Pool until all IPAddresses have been removed.")
	}
	return nil, nil
}

func (w *NetboxIPPoolWebhook) validate(newPool *ipamv1alpha1.NetboxIPPool) (reterr error) {
	var allErrs field.ErrorList
	defer func() {
		if len(allErrs) > 0 {
			reterr = apierrors.NewInvalid(v1alpha2.GroupVersion.WithKind(newPool.GetObjectKind().GroupVersionKind().Kind).GroupKind(), newPool.GetName(), allErrs)
		}
	}()

	if newPool.Spec.CIDR == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "CIDR"), newPool.Spec.CIDR, "CIDR is required"))
	}

	if newPool.Spec.CredentialsRef == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "CredentialsRef"),
			newPool.Spec.CredentialsRef, "CredentialsRef is required"))
	} else {
		if newPool.Spec.CredentialsRef.Name == "" {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "CredentialsRef.Name"),
				newPool.Spec.CredentialsRef.Name, "CredentialsRef.Name is required"))
		}
	}

	cidr, err := ipaddr.NewIPAddressString(newPool.Spec.CIDR).ToAddress()
	if err != nil || cidr.String() != newPool.Spec.CIDR {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "CIDR"),
			newPool.Spec.CIDR, "CIDR is not a valid CIDR"))
	}

	if newPool.Spec.Gateway != "" {
		gatewayIP, err := ipaddr.NewIPAddressString(newPool.Spec.Gateway).ToAddress()
		if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "Gateway"),
				newPool.Spec.Gateway, "Gateway is not a valid IP address"+" "+err.Error()))
		}

		if cidr != nil && !cidr.Contains(gatewayIP) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "Gateway"), newPool.Spec.Gateway, "CIDR must contain gateway"))
		}

		ipVersionsMatched := cidr != nil && ((cidr.IsIPv4() && gatewayIP.IsIPv4()) || (cidr.IsIPv6() && gatewayIP.IsIPv6()))

		if !ipVersionsMatched {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "CIDR"), newPool.Spec.CIDR, "CIDR and gateway are mixed IPv4 and IPv6 addresses"))
		}
	}

	return //nolint:nakedret
}
