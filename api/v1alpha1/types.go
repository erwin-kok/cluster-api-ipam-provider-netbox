package v1alpha1

import (
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NetboxPrefixPoolKind        = "NetboxPrefixPool"
	NetboxPrefixGlobalPoolKind  = "NetboxPrefixGlobalPool"
	NetboxIPRangePoolKind       = "NetboxIPRangePool"
	NetboxIPRangeGlobalPoolKind = "NetboxIPRangeGlobalPool"
)

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
}

// GenericNetboxPool is a common interface for Netbox pools.
// +kubebuilder:object:generate=false
type GenericNetboxPool interface {
	client.Object
	PoolSpec() GenericPoolSpec
	PoolStatus() GenericPoolStatus
}

// GenericPoolSpec is a common interface for Netbox PoolSpec
// +kubebuilder:object:generate=false
type GenericPoolSpec interface {
	ToSequentialRange() (*ipaddr.SequentialRange[*ipaddr.IPAddress], error)
	GetGateway() string
}

// GenericPoolStatus is a common interface for Netbox PoolStatus
// +kubebuilder:object:generate=false
type GenericPoolStatus interface {
	SetAddresses(addresses *NetboxPoolStatusIPAddresses)
	GetAddresses() *NetboxPoolStatusIPAddresses
}
