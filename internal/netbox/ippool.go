package netbox

import (
	"fmt"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type NetboxIPPool struct {
	id       int
	isPrefix bool
	display  string
	vrf      string
	Range    *ipaddr.SequentialRange[*ipaddr.IPAddress]
	inuse    int32
}

func (p *NetboxIPPool) Total() int32 {
	return (int32)(p.Range.GetCount().Int64())
}

func (p *NetboxIPPool) InUse() int32 {
	return p.inuse
}

func (p *NetboxIPPool) Available() int32 {
	return p.Total() - p.InUse()
}

func (p *NetboxIPPool) String() string {
	name := "IPRange"
	if p.isPrefix {
		name = "Prefix"
	}
	return fmt.Sprintf("%s %s (%d): total %d, inuse: %d, available: %d ", name, p.display, p.id, p.Total(), p.InUse(), p.Available())
}
