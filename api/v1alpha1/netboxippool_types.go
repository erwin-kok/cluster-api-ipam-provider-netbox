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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetboxIPPoolSpec defines the desired state of NetboxIPPool.
type NetboxIPPoolSpec struct {
}

// NetboxIPPoolStatus defines the observed state of NetboxIPPool.
type NetboxIPPoolStatus struct {
	// Addresses reports the count of total, free, and used IPs in the pool.
	// +optional
	Addresses *NetboxIPPoolStatusIPAddresses `json:"ipAddresses,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=cluster-api
// +kubebuilder:printcolumn:name="Addresses",type="string",JSONPath=".spec.addresses",description="List of addresses, to allocate from"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.ipAddresses.total",description="Count of IPs configured for the pool"
// +kubebuilder:printcolumn:name="Free",type="integer",JSONPath=".status.ipAddresses.free",description="Count of unallocated IPs in the pool"
// +kubebuilder:printcolumn:name="Used",type="integer",JSONPath=".status.ipAddresses.used",description="Count of allocated IPs in the pool"

// NetboxIPPool is the Schema for the netboxippools API
type NetboxIPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetboxIPPoolSpec   `json:"spec,omitempty"`
	Status NetboxIPPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetboxIPPoolList contains a list of NetboxIPPool
type NetboxIPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetboxIPPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetboxIPPool{}, &NetboxIPPoolList{})
}

type NetboxIPPoolStatusIPAddresses struct {
}
