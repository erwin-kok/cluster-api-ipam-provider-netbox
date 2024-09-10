package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NetboxIPPoolKind       = "NetboxIPPoolKind"
	NetboxIPGlobalPoolKind = "NetboxIPGlobalPoolKind"

	SecretFinalizer = "netbox.ipam.cluster.x-k8s.io/Secret"
)

type NetboxPoolType string

var (
	PrefixType  = NetboxPoolType("Prefix")
	IPRangeType = NetboxPoolType("IPRange")
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
	GetKind() string
	PoolSpec() *NetboxIPPoolSpec
	PoolStatus() *NetboxIPPoolStatus
}
