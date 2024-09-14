package netbox

type Vrf struct {
	Name string `json:"name,omitempty"`
}

type IPAddress struct {
	Address string `json:"address,omitempty"`
	Vrf     Vrf    `json:"vrf,omitempty"`
}

type IPAddressList struct {
	Count   int         `json:"count,omitempty"`
	Results []IPAddress `json:"results,omitempty"`
}

type IPRange struct {
	Id           int    `json:"id,omitempty"`
	Display      string `json:"display,omitempty"`
	StartAddress string `json:"start_address,omitempty"`
	EndAddress   string `json:"end_address,omitempty"`
	Vrf          Vrf    `json:"vrf,omitempty"`
}

type IPRangeList struct {
	Count   int       `json:"count,omitempty"`
	Results []IPRange `json:"results,omitempty"`
}

type Prefix struct {
	Id      int    `json:"id,omitempty"`
	Display string `json:"display,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	Vrf     Vrf    `json:"vrf,omitempty"`
}

type PrefixList struct {
	Count   int      `json:"count,omitempty"`
	Results []Prefix `json:"results,omitempty"`
}

type PrefixRequest struct {
	Id      int    `json:"id,omitempty"`
	Display string `json:"display,omitempty"`
	Address string `json:"address,omitempty"`
	Vrf     Vrf    `json:"vrf,omitempty"`
}
