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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var netboxippoollog = logf.Log.WithName("netboxippool-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *NetboxIPPool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-ipam-cluster-x-k8s-io-v1alpha1-netboxippool,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.cluster.x-k8s.io,resources=netboxippools,verbs=create;update,versions=v1alpha1,name=mnetboxippool.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &NetboxIPPool{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NetboxIPPool) Default() {
	netboxippoollog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-ipam-cluster-x-k8s-io-v1alpha1-netboxippool,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.cluster.x-k8s.io,resources=netboxippools,verbs=create;update,versions=v1alpha1,name=vnetboxippool.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NetboxIPPool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NetboxIPPool) ValidateCreate() (admission.Warnings, error) {
	netboxippoollog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NetboxIPPool) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	netboxippoollog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NetboxIPPool) ValidateDelete() (admission.Warnings, error) {
	netboxippoollog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
