package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// pageSize is the maximum SecObserve honours for the page_size override.
// The default is 25, which would triple the request count on large instances.
const pageSize = 100

// Page is the DRF page-number pagination envelope.
type Page[T any] struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []T     `json:"results"`
}

// List fetches every page of a collection endpoint and returns the flattened
// results. Callers pass filters via query; page and page_size are managed here.
func List[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	params := url.Values{}
	for key, values := range query {
		params[key] = values
	}
	params.Set("page_size", strconv.Itoa(pageSize))

	var all []T
	for page := 1; ; page++ {
		params.Set("page", strconv.Itoa(page))

		var result Page[T]
		if err := c.Get(ctx, path, params, &result); err != nil {
			return nil, err
		}
		all = append(all, result.Results...)

		// Trust "next" rather than comparing len(all) to count: the count can
		// shift underneath us while paging through a live instance.
		if result.Next == nil || *result.Next == "" || len(result.Results) == 0 {
			return all, nil
		}
		if page > 10_000 {
			return nil, fmt.Errorf("listing %s: pagination did not terminate", path)
		}
	}
}

// unpaginated is the bare envelope returned by /api/product_api_tokens/, which
// is a plain ViewSet and therefore not paginated.
type unpaginated[T any] struct {
	Results []T `json:"results"`
}

// ListUnpaginated fetches a collection endpoint that returns a bare
// {"results": [...]} body without pagination.
func ListUnpaginated[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	var result unpaginated[T]
	if err := c.Get(ctx, path, query, &result); err != nil {
		return nil, err
	}
	return result.Results, nil
}
