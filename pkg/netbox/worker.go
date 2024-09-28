package netbox

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/logger"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

const (
	limit = 100
)

type poolKey struct {
	kind   PoolType
	cidr   string
	vrf    string
	tenant string
}

func (p *poolKey) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", p.kind, p.cidr, p.vrf, p.tenant)
}

type netboxPool struct {
	id      int
	display string
	kind    PoolType
	cidr    string
	vrf     string
	tenant  string
	rng     *ipaddr.SequentialRange[*ipaddr.IPAddress]
}

type PoolInfo struct {
	Id      int
	Display string
	Kind    PoolType
	Cidr    string
	Vrf     string
	Tenant  string
}

type fetchRequest[R any, T any] struct {
	ctx   context.Context
	req   R
	resch chan fetchResponse[T]
}

type fetchResponse[T any] struct {
	data T
	err  error
}

type poolFetcher struct {
	restyClient *resty.Client

	mu    sync.Mutex
	pools map[string]*netboxPool

	preqch chan fetchRequest[poolKey, *PoolInfo]
}

func newPoolFetcher() *poolFetcher {
	return &poolFetcher{
		pools:  make(map[string]*netboxPool),
		preqch: make(chan fetchRequest[poolKey, *PoolInfo]),
	}
}

func (f *poolFetcher) loop() {

	for {
		select {
		case req := <-f.preqch:
			if req.ctx.Err() != nil {
				req.resch <- fetchResponse[*PoolInfo]{err: req.ctx.Err()}
				continue
			}

			key := req.req

			// Look up the cache. If not present, fetch and put in cache
			f.mu.Lock()
			pool, ok := f.pools[key.String()]
			f.mu.Unlock()
			if !ok {
				pool, err := f.fetchPool(req.ctx, key)
				if err != nil {
					req.resch <- fetchResponse[*PoolInfo]{err: err}
					continue
				}

				f.mu.Lock()
				f.pools[key.String()] = pool
				f.mu.Unlock()
			}

			req.resch <- fetchResponse[*PoolInfo]{data: convertNetboxPoolToPoolInfo(pool)}

		}
	}
}

func (f *poolFetcher) FetchPoolInfo(ctx context.Context, kind PoolType, cidr, vrf, tenant string) (*PoolInfo, error) {
	key := poolKey{kind: kind, cidr: cidr, vrf: vrf, tenant: tenant}

	// If the pool is already in the cache, return it.
	f.mu.Lock()
	pool, ok := f.pools[key.String()]
	f.mu.Unlock()
	if ok {
		return convertNetboxPoolToPoolInfo(pool), nil
	}

	// If not, create request and synchronize on the loop.
	resch := make(chan fetchResponse[*PoolInfo], 1)
	select {
	case f.preqch <- fetchRequest[poolKey, *PoolInfo]{ctx: ctx, req: key, resch: resch}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case res := <-resch:
		return res.data, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *poolFetcher) fetchPool(ctx context.Context, key poolKey) (*netboxPool, error) {
	switch key.kind {
	case PrefixPoolType:
		return f.fetchPrefixPool(ctx, key.cidr, key.vrf)
	case IPRangePoolType:
		return f.fetchIPRangePool(ctx, key.cidr, key.vrf)
	}
	return nil, errors.New(fmt.Sprintf("unexpected pool type: %s", key.kind))
}

func (f *poolFetcher) fetchPrefixPool(ctx context.Context, prefix string, requestedVrf string) (*netboxPool, error) {
	prefixList := &PrefixList{}

	request := f.
		restyClient.
		R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetResult(prefixList)

	if prefix != "" {
		request.SetQueryParam("prefix", prefix)
	}

	response, err := request.Get("/prefixes")

	if err != nil {
		return nil, errors.Wrap(err, "failed to get prefix")
	}
	if isFailure(response) {
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
	return &netboxPool{
		id:      result.Id,
		kind:    PrefixPoolType,
		display: result.Display,
		vrf:     result.Vrf.Name,
		rng:     cidr.ToSequentialRange(),
	}, nil
}

func (f *poolFetcher) fetchIPRangePool(ctx context.Context, startAddress string, requestedVrf string) (*netboxPool, error) {
	ipRangeList := &IPRangeList{}

	request := f.
		restyClient.
		R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetResult(ipRangeList)

	if startAddress != "" {
		request.SetQueryParam("start_address", startAddress)
	}

	response, err := request.Get("/ip-ranges")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ip-range")
	}
	if isFailure(response) {
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
		return nil, fmt.Errorf("multiple ip-ranges matches start startAddress '%s',"+
			"there must be only one match", startAddress)
	}

	result := filteredResults[0]
	lower, err := ipaddr.NewIPAddressString(result.StartAddress).ToAddress()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not parse startAddress '%s'", result.StartAddress))
	}

	upper, err := ipaddr.NewIPAddressString(result.EndAddress).ToAddress()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not parse endAddress '%s'", result.EndAddress))
	}
	return &netboxPool{
		id:      result.Id,
		kind:    IPRangePoolType,
		display: result.Display,
		vrf:     result.Vrf.Name,
		rng:     lower.SpanWithRange(upper),
	}, nil
}

func (f *poolFetcher) gatherStatistics(ctx context.Context, pools []*NetboxIPPool) error {
	log := logger.FromContext(ctx)

	offset := 0
	for _, p := range pools {
		p.inuse = 0
	}
	for {
		addressList := &IPAddressList{}
		response, err := f.
			restyClient.
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

func isFailure(response *resty.Response) bool {
	return response.StatusCode() != 200
}

func convertNetboxPoolToPoolInfo(pool *netboxPool) *PoolInfo {
	return &PoolInfo{
		Id:      pool.id,
		Display: pool.display,
		Kind:    pool.kind,
		Cidr:    pool.cidr,
		Vrf:     pool.vrf,
		Tenant:  pool.tenant,
	}
}
