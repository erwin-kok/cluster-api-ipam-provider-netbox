package netbox

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/netbox-community/go-netbox/v3"
	"github.com/pkg/errors"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type AddressPool struct {
	id       int
	isPrefix bool
	display  string
	vrf      string
	Range    *ipaddr.SequentialRange[*ipaddr.IPAddress]
	inuse    int32
}

func (p *AddressPool) Total() int32 {
	return (int32)(p.Range.GetCount().Int64())
}

func (p *AddressPool) InUse() int32 {
	return p.inuse
}

func (p *AddressPool) Available() int32 {
	return p.Total() - p.InUse()
}

func (p *AddressPool) String() string {
	name := "IPRange"
	if p.isPrefix {
		name = "Prefix"
	}
	return fmt.Sprintf("%s %s (%d): total %d, inuse: %d, available: %d ", name, p.display, p.id, p.Total(), p.InUse(), p.Available())
}

type Client struct {
	api    *netbox.APIClient
	client *resty.Client
}

func NewNetBoxClient(host, apiToken string) (*Client, error) {
	api := netbox.NewAPIClientFor(host, apiToken)
	client := resty.New().
		SetBaseURL("http://localhost:8000/api/ipam").
		SetAuthScheme("Token").
		SetAuthToken("b1f2db68f235158beea51b0554fc067814221c3a")
	return &Client{
		api:    api,
		client: client,
	}, nil
}

func (c *Client) Bla(ctx context.Context, id int32) {
	// x, _, _ := c.api.IpamAPI.IpamPrefixesAvailableIpsCreate(ctx, 3).
	// 	IPAddressRequest([]netbox.IPAddressRequest{
	// 		{
	// 			Address: "30.10.0.2",
	// 		},
	// 	}).
	// 	Execute()
	// fmt.Sprintf("%s", x)
}

func (c *Client) GetPrefix(ctx context.Context, prefix string, requestedVrf string) (*AddressPool, error) {
	prefixList := &PrefixList{}
	request :=
		c.client.
			R().
			SetHeader("Accept", "application/json").
			SetResult(prefixList).
			SetContext(ctx)

	request.SetQueryParam("prefix", prefix)

	response, err := request.Get("/prefixes")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get prefix")
	}
	if response.StatusCode() != 200 {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve prefixes successfully. (%d)", response.StatusCode()))
	}

	var filteredResults []Prefix
	for _, p := range prefixList.Results {
		if requestedVrf == "" || (requestedVrf != "" && p.Vrf.Name == requestedVrf) {
			filteredResults = append(filteredResults, p)
		}
	}
	if len(filteredResults) == 0 {
		return nil, fmt.Errorf("no prefix matches '%s''", prefix)
	}
	if len(filteredResults) != 1 {
		return nil, fmt.Errorf("multiple prefixes matches '%s', there must be only one match", prefix)
	}

	result := filteredResults[0]
	cidr, err := ipaddr.NewIPAddressString(result.Prefix).ToAddress()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not parse prefix '%s'", result.Prefix))
	}

	return &AddressPool{
		id:       result.Id,
		isPrefix: true,
		display:  result.Display,
		vrf:      result.Vrf.Name,
		Range:    cidr.ToSequentialRange(),
	}, nil
}

func (c *Client) GetIPRange(ctx context.Context, startAddress string, requestedVrf string) (*AddressPool, error) {
	ipRangeList := &IPRangeList{}
	request :=
		c.client.
			R().
			SetHeader("Accept", "application/json").
			SetResult(ipRangeList).
			SetContext(ctx)

	request.SetQueryParam("start_address", startAddress)

	response, err := request.Get("/ip-ranges")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ip-range")
	}
	if response.StatusCode() != 200 {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve ip-ranges successfully. (%d)", response.StatusCode()))
	}

	var filteredResults []IPRange
	for _, p := range ipRangeList.Results {
		if requestedVrf == "" || (requestedVrf != "" && p.Vrf.Name == requestedVrf) {
			filteredResults = append(filteredResults, p)
		}
	}
	if len(filteredResults) == 0 {
		return nil, fmt.Errorf("no ip-range matches start startAddress '%s'", startAddress)
	}
	if len(filteredResults) != 1 {
		return nil, fmt.Errorf("multiple ip-ranges matches start startAddress '%s', there must be only one match", startAddress)
	}

	result := filteredResults[0]
	lower := ipaddr.NewIPAddressString(result.StartAddress).GetAddress()
	upper := ipaddr.NewIPAddressString(result.EndAddress).GetAddress()

	return &AddressPool{
		id:       result.Id,
		isPrefix: false,
		display:  result.Display,
		vrf:      result.Vrf.Name,
		Range:    lower.SpanWithRange(upper),
	}, nil
}

func (c *Client) NextAvailableAddress(ctx context.Context) (*ipaddr.IPAddress, error) {
	prefix := &PrefixRequest{}
	request :=
		c.client.
			R().
			SetHeader("Accept", "application/json").
			SetResult(prefix).
			SetContext(ctx)
	response, err := request.Post("/prefixes/5/available-ips/")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get prefix")
	}
	if response.StatusCode() != 201 {
		return nil, errors.Wrap(err, fmt.Sprintf("could not create next available address. (%d)", response.StatusCode()))
	}
	ipAddress, err := ipaddr.NewIPAddressString(prefix.Address).ToAddress()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid IpAddress %s", prefix.Address))
	}
	return ipAddress, nil
}
