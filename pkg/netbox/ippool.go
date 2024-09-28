package netbox

import (
	"fmt"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type PoolType string

var (
	PrefixPoolType  = PoolType("Prefix")
	IPRangePoolType = PoolType("IPRange")
)

type NetboxIPPool struct {
	Id      int
	Type    PoolType
	Display string
	Vrf     string
	Range   *ipaddr.SequentialRange[*ipaddr.IPAddress]
	inuse   int
}

func (p *NetboxIPPool) Contains(address *ipaddr.IPAddress) bool {
	return p.Range.Contains(address)
}

func (p *NetboxIPPool) Total() int {
	return (int)(p.Range.GetCount().Int64())
}

func (p *NetboxIPPool) InUse() int {
	return p.inuse
}

func (p *NetboxIPPool) Available() int {
	return p.Total() - p.InUse()
}

func (p *NetboxIPPool) String() string {
	return fmt.Sprintf("%s %s (%d): total %d, inuse: %d, available: %d ",
		p.Type, p.Display, p.Id, p.Total(), p.InUse(), p.Available())
}
