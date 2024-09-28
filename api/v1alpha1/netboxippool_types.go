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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetboxPoolType string

const (
	NetboxIPPoolKind = "NetboxIPPool"
)

var (
	PrefixType  = NetboxPoolType("Prefix")
	IPRangeType = NetboxPoolType("IPRange")
)

// NetboxIPPoolSpec defines the desired state of NetboxIPPool
type NetboxIPPoolSpec struct {
	// Type of the pool. Can either be Prefix or IPRange
	// +kubebuilder:validation:Enum=Prefix;IPRange
	Type NetboxPoolType `json:"type"`

	// Depending on the type, an CIDR is either the prefix or the start address of an ip-range, in CIDR notation.
	CIDR string `json:"cidr"`

	// Vrf where the CIDR is part of. If not provided, the "Global" Vrf is used.
	// +optional
	Vrf string `json:"vrf,omitempty"`

	// Gateway
	// +optional
	Gateway string `json:"gateway,omitempty"`

	// CredentialsRef is a reference to a Secret that contains the credentials to use for accessing th Netbox instance.
	// if no namespace is provided, the namespace of the NetboxIPPool will be used.
	CredentialsRef *corev1.SecretReference `json:"credentialsRef,omitempty"`
}

// NetboxIPPoolStatus defines the observed state of NetboxIPPool
type NetboxIPPoolStatus struct {
	// Addresses reports the count of total, free, and used IPs in the pool.
	// +optional
	Addresses *NetboxPoolStatusIPAddresses `json:"ipAddresses,omitempty"`

	// NetboxId is the Id in Netbox.
	// +optional
	NetboxId int `json:"netboxId,omitempty"`

	// NetboxType is the Type in Netbox.
	// +optional
	NetboxType string `json:"netboxType,omitempty"`
}

// NetboxPoolStatusIPAddresses contains the count of total, free, and used IPs in a pool.
type NetboxPoolStatusIPAddresses struct {
	// Total is the total number of IPs configured for the pool.
	// Counts greater than int can contain will report as math.MaxInt.
	Total int `json:"total"`

	// Free is the count of unallocated IPs in the pool.
	// Counts greater than int can contain will report as math.MaxInt.
	Free int `json:"free"`

	// Used is the count of allocated IPs in the pool.
	// Counts greater than int can contain will report as math.MaxInt.
	Used int `json:"used"`

	// Extra is the count of allocated IPs in the pool.
	// Counts greater than int can contain will report as math.MaxInt.
	Extra int `json:"extra"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=cluster-api
// +kubebuilder:printcolumn:name="Addresses",type="string",JSONPath=".spec.addresses",description="List of addresses, to allocate from"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.ipAddresses.total",description="Count of IPs configured for the pool"
// +kubebuilder:printcolumn:name="Free",type="integer",JSONPath=".status.ipAddresses.free",description="Count of unallocated IPs in the pool"
// +kubebuilder:printcolumn:name="Used",type="integer",JSONPath=".status.ipAddresses.used",description="Count of allocated IPs in the pool"
// +k8s:defaulter-gen=true

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
