package netbox

import (
	"context"
	"fmt"
	"strconv"

	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/logger"
	"github.com/seancfoley/ipaddress-go/ipaddr"

	"github.com/pkg/errors"
)

const (
	limit = 100
)

func (c *client) GatherStatistics(ctx context.Context, pools []*NetboxIPPool) error {
	log := logger.FromContext(ctx)

	offset := 0
	for _, p := range pools {
		p.inuse = 0
	}
	for {
		addressList := &IPAddressList{}
		response, err := c.restyClient.
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
			address, rerr := ipaddr.NewIPAddressString(a.Address).ToAddress()
			if rerr != nil {
				log.Error(err, fmt.Sprintf("could not parse ipAddress %s", a.Address))
				continue
			}
			vrf := a.Vrf.Name
			for _, p := range pools {
				if vrf == p.Vrf && p.Contains(address) {
					p.inuse++
				}
			}
		}
		offset += limit
	}
	return nil
}
