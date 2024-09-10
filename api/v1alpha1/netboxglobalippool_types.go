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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=cluster-api
// +kubebuilder:printcolumn:name="Addresses",type="string",JSONPath=".spec.addresses",description="List of addresses, to allocate from"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.ipAddresses.total",description="Count of IPs configured for the pool"
// +kubebuilder:printcolumn:name="Free",type="integer",JSONPath=".status.ipAddresses.free",description="Count of unallocated IPs in the pool"
// +kubebuilder:printcolumn:name="Used",type="integer",JSONPath=".status.ipAddresses.used",description="Count of allocated IPs in the pool"

// NetboxGlobalIPPool is the Schema for the netboxglobalippools API
type NetboxGlobalIPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetboxIPPoolSpec   `json:"spec,omitempty"`
	Status NetboxIPPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetboxGlobalIPPoolList contains a list of NetboxGlobalIPPool
type NetboxGlobalIPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetboxGlobalIPPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetboxGlobalIPPool{}, &NetboxGlobalIPPoolList{})
}

func (p *NetboxGlobalIPPool) GetKind() string {
	return NetboxIPGlobalPoolKind
}

// PoolSpec implements the genericInClusterPool interface.
func (p *NetboxGlobalIPPool) PoolSpec() *NetboxIPPoolSpec {
	return &p.Spec
}

// PoolStatus implements the genericInClusterPool interface.
func (p *NetboxGlobalIPPool) PoolStatus() *NetboxIPPoolStatus {
	return &p.Status
}
