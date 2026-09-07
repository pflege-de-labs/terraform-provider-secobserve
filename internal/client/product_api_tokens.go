package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ProductAPITokenRequest creates an API token scoped to one product.
//
// Name is limited to 32 characters, tighter than every other name in the API,
// and must be unique per product (core/services/product_api_token.py:16-23).
type ProductAPITokenRequest struct {
	Product int64  `json:"product"`
	Role    int64  `json:"role"`
	Name    string `json:"name"`
	// ExpirationDate is an ISO date, or nil for a token that never expires.
	ExpirationDate *string `json:"expiration_date"`
}

// ProductAPIToken is the read representation of a product API token. The
// secret is not part of it: it is returned exactly once, by the create call.
type ProductAPIToken struct {
	// ID is the underlying API_Token_Multiple primary key, which is also the
	// path segment used to delete the token.
	ID             int64   `json:"id"`
	Product        int64   `json:"product"`
	Role           int64   `json:"role"`
	Name           string  `json:"name"`
	ExpirationDate *string `json:"expiration_date"`
}

// GetID implements Named.
func (t ProductAPIToken) GetID() int64 { return t.ID }

// GetName implements Named.
func (t ProductAPIToken) GetName() string { return t.Name }

const productAPITokensPath = "api/product_api_tokens/"

// CreateProductAPIToken creates a token and returns the secret.
//
// The secret cannot be retrieved afterwards: the endpoint has no retrieve
// action and the value is never included in the list response
// (core/api/views_product.py:610-626). Losing it means recreating the token.
func (c *Client) CreateProductAPIToken(ctx context.Context, request ProductAPITokenRequest) (string, error) {
	var response struct {
		Token string `json:"token"`
	}
	if err := c.Post(ctx, productAPITokensPath, request, &response); err != nil {
		return "", err
	}
	if response.Token == "" {
		return "", fmt.Errorf("creating product API token %q: the API returned an empty token", request.Name)
	}
	return response.Token, nil
}

// ProductAPITokens lists the tokens of one product.
//
// The product filter is mandatory and the response is a bare results envelope
// with no pagination, because the endpoint is a plain ViewSet rather than a
// ModelViewSet (core/api/views_product.py:588-604).
func (c *Client) ProductAPITokens(ctx context.Context, productID int64) ([]ProductAPIToken, error) {
	query := url.Values{"product": {strconv.FormatInt(productID, 10)}}
	return ListUnpaginated[ProductAPIToken](ctx, c, productAPITokensPath, query)
}

// FindProductAPIToken locates a token by its natural key. Since the endpoint
// offers no retrieve action, reading a single token means listing the
// product's tokens and matching on the name.
func (c *Client) FindProductAPIToken(ctx context.Context, productID int64, name string) (ProductAPIToken, error) {
	tokens, err := c.ProductAPITokens(ctx, productID)
	if err != nil {
		return ProductAPIToken{}, err
	}
	for _, token := range tokens {
		if token.Name == name {
			return token, nil
		}
	}
	return ProductAPIToken{}, &ErrNotFound{
		Kind:  "product API token",
		Field: "product/name",
		Name:  fmt.Sprintf("%d/%s", productID, name),
	}
}

// DeleteProductAPIToken revokes a token. The id is the one returned by the
// list and create calls.
func (c *Client) DeleteProductAPIToken(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", productAPITokensPath, id), nil)
}
