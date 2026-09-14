package client

import (
	"context"
	"net/url"
	"strconv"
)

// VEXCounter is the per-(document_id_prefix, year) sequence used to mint VEX
// document ids ("<year>_<counter>"). It is not exposed as a resource: the
// counter is incremented server-side on every CSAF/OpenVEX export
// (vex/services/vex_base.py:13-18, vex/services/csaf_generator.py:79,
// vex/services/openvex_generator.py:69), so a Terraform-managed value would
// show drift on every export and a write could rewind the sequence, causing
// duplicate document ids. It is read-only here, via the
// secobserve_vex_counter data source.
type VEXCounter struct {
	ID               int64  `json:"id"`
	DocumentIDPrefix string `json:"document_id_prefix"`
	Year             int64  `json:"year"`
	Counter          int64  `json:"counter"`
}

const vexCountersPath = "api/vex/vex_counters/"

// VEXCounterByPrefixAndYear resolves a counter by its natural key. The
// document_id_prefix filter is icontains (vex/api/filters.py), year is exact,
// so the prefix match is re-applied client-side; the (prefix, year) pair is
// DB-unique so an exact match is unambiguous.
func (c *Client) VEXCounterByPrefixAndYear(ctx context.Context, prefix string, year int64) (VEXCounter, error) {
	var zero VEXCounter

	query := url.Values{
		"document_id_prefix": {prefix},
		"year":               {strconv.FormatInt(year, 10)},
	}
	candidates, err := List[VEXCounter](ctx, c, vexCountersPath, query)
	if err != nil {
		return zero, err
	}
	for _, candidate := range candidates {
		if candidate.DocumentIDPrefix == prefix && candidate.Year == year {
			return candidate, nil
		}
	}
	return zero, &ErrNotFound{Kind: "VEX counter", Name: prefix, Field: "document_id_prefix/year"}
}
