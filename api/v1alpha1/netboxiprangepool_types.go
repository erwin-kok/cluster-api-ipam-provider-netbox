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

// NetboxIPRangePoolSpec defines the desired state of NetboxIPRangePool
type NetboxIPRangePoolSpec struct {
	// StartAddress is the StartAddress of the IP-Range
	StartAddress string `json:"startAddress"`

	// EndAddress is the EndAddress of the IP-Range
	EndAddress string `json:"endAddress"`

	// Gateway
	// +optional
	Gateway string `json:"gateway,omitempty"`
}

// NetboxIPRangePoolStatus defines the observed state of NetboxIPRangePool
type NetboxIPRangePoolStatus struct {
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

// NetboxIPRangePool is the Schema for the netboxiprangepools API
type NetboxIPRangePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetboxIPRangePoolSpec   `json:"spec,omitempty"`
	Status NetboxIPRangePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetboxIPRangePoolList contains a list of NetboxIPRangePool
type NetboxIPRangePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetboxIPRangePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetboxIPRangePool{}, &NetboxIPRangePoolList{})
}

// PoolSpec implements the genericInClusterPool interface.
func (p *NetboxIPRangePool) PoolSpec() GenericPoolSpec {
	return &p.Spec
}

// PoolStatus implements the genericInClusterPool interface.
func (p *NetboxIPRangePool) PoolStatus() GenericPoolStatus {
	return &p.Status
}

func (p *NetboxIPRangePoolSpec) ToSequentialRange() (*ipaddr.SequentialRange[*ipaddr.IPAddress], error) {
	startAddress, err := ipaddr.NewIPAddressString(p.StartAddress).ToAddress()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not parse start address '%s'", p.StartAddress))
	}
	endAddress, err := ipaddr.NewIPAddressString(p.EndAddress).ToAddress()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not parse end address '%s'", p.EndAddress))
	}
	return startAddress.SpanWithRange(endAddress), nil
}

func (p *NetboxIPRangePoolSpec) GetGateway() string {
	return p.Gateway
}

func (p *NetboxIPRangePoolStatus) SetAddresses(addresses *NetboxPoolStatusIPAddresses) {
	p.Addresses = addresses
}

func (p *NetboxIPRangePoolStatus) GetAddresses() *NetboxPoolStatusIPAddresses {
	return p.Addresses
}
