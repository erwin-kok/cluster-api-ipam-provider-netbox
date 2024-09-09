package netbox

import (
	"context"
	"fmt"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"strconv"

	"github.com/pkg/errors"
)

const (
	limit = 100
)

func (c *Client) GatherStatistics(ctx context.Context, pools []*AddressPool) error {
	offset := 0
	for _, p := range pools {
		p.inuse = 0
	}
	for {
		addressList := &IPAddressList{}
		response, err := c.client.
			R().
			SetHeader("Accept", "application/json").
			SetQueryParams(map[string]string{
				"limit":  strconv.Itoa(limit),
				"offset": strconv.Itoa(offset),
			}).
			SetResult(addressList).
			SetContext(ctx).
			Get("/ip-addresses")
		if err != nil {
			return errors.Wrap(err, "failed to get ip-addresses")
		}
		if response.StatusCode() != 200 {
			return errors.Wrap(err, fmt.Sprintf("could not retrieve ip-addresses successfully. (%d)", response.StatusCode()))
		}
		if len(addressList.Results) == 0 {
			break
		}
		for _, a := range addressList.Results {
			address := ipaddr.NewIPAddressString(a.Address).GetAddress()
			vrf := a.Vrf.Name
			for _, p := range pools {
				if vrf == p.vrf && p.Range.Contains(address) {
					p.inuse++
				}
			}

			// addr := ipaddr.NewIPAddressString(a.Address).GetAddress().ToPrefixBlock()
			//
			// lower := ipaddr.NewIPAddressString("10.0.0.1/24").GetAddress()
			// upper := ipaddr.NewIPAddressString("10.0.50.10/24").GetAddress()
			// rng := lower.SpanWithRange(upper)
			// y := rng.SpanWithPrefixBlocks()
			// x := rng.GetCount()
			//
			// cidr := ipaddr.NewIPAddressString("30.10.0.0/16").GetAddress().ToSequentialRange()
			//
			// fmt.Printf("%v", rng.Contains(ipaddr.NewIPAddressString("10.0.0.5").GetAddress()))
			//
			// fmt.Printf("%v", y)
			//
			// fmt.Printf("%v %v %d %v %v %d", addr, rng, x, y, cidr, len(y))
			// vrf := ""
			// if a.Vrf != nil && a.Vrf.Name != nil {
			//
			// }
		}
		offset += limit
	}
	return nil
}
