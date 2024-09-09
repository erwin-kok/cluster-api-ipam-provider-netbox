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
	"fmt"
	"github.com/pkg/errors"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetboxPrefixGlobalPoolSpec defines the desired state of NetboxPrefixGlobalPool
type NetboxPrefixGlobalPoolSpec struct {
	// Prefix is the prefix in CIDR notation, e.g. 20.0.0.0/14.
	Prefix string `json:"prefix"`

	// Gateway
	// +optional
	Gateway string `json:"gateway,omitempty"`
}

// NetboxPrefixGlobalPoolStatus defines the observed state of NetboxPrefixGlobalPool
type NetboxPrefixGlobalPoolStatus struct {
	// Addresses reports the count of total, free, and used IPs in the pool.
	// +optional
	Addresses *NetboxPoolStatusIPAddresses `json:"ipAddresses,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=cluster-api
// +kubebuilder:printcolumn:name="Addresses",type="string",JSONPath=".spec.addresses",description="List of addresses, to allocate from"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.ipAddresses.total",description="Count of IPs configured for the pool"
// +kubebuilder:printcolumn:name="Free",type="integer",JSONPath=".status.ipAddresses.free",description="Count of unallocated IPs in the pool"
// +kubebuilder:printcolumn:name="Used",type="integer",JSONPath=".status.ipAddresses.used",description="Count of allocated IPs in the pool"

// NetboxPrefixGlobalPool is the Schema for the netboxprefixglobalpools API
type NetboxPrefixGlobalPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetboxPrefixGlobalPoolSpec   `json:"spec,omitempty"`
	Status NetboxPrefixGlobalPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetboxPrefixGlobalPoolList contains a list of NetboxPrefixGlobalPool
type NetboxPrefixGlobalPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetboxPrefixGlobalPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetboxPrefixGlobalPool{}, &NetboxPrefixGlobalPoolList{})
}

// PoolSpec implements the genericInClusterPool interface.
func (p *NetboxPrefixGlobalPool) PoolSpec() GenericPoolSpec {
	return &p.Spec
}

// PoolStatus implements the genericInClusterPool interface.
func (p *NetboxPrefixGlobalPool) PoolStatus() GenericPoolStatus {
	return &p.Status
}

func (p *NetboxPrefixGlobalPoolSpec) ToSequentialRange() (*ipaddr.SequentialRange[*ipaddr.IPAddress], error) {
	cidr, err := ipaddr.NewIPAddressString(p.Prefix).ToAddress()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not parse prefix '%s'", p.Prefix))
	}
	return cidr.ToSequentialRange(), nil
}

func (p *NetboxPrefixGlobalPoolSpec) GetGateway() string {
	return p.Gateway
}

func (p *NetboxPrefixGlobalPoolStatus) SetAddresses(addresses *NetboxPoolStatusIPAddresses) {
	p.Addresses = addresses
}

func (p *NetboxPrefixGlobalPoolStatus) GetAddresses() *NetboxPoolStatusIPAddresses {
	return p.Addresses
}
