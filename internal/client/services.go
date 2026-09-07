package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ServiceRequest is the write representation of a service. A service is
// nothing but a named grouping of observations inside a product, hence the two
// fields.
type ServiceRequest struct {
	// Immutable after create, and read by the permission layer before
	// validation.
	Product int64  `json:"product"`
	Name    string `json:"name"`
}

// Service is the read representation of a service.
type Service struct {
	ID              int64  `json:"id"`
	Product         int64  `json:"product"`
	Name            string `json:"name"`
	NameWithProduct string `json:"name_with_product"`
}

// GetID implements Named.
func (s Service) GetID() int64 { return s.ID }

// GetName implements Named.
func (s Service) GetName() string { return s.Name }

const servicesPath = "api/services/"

// CreateService creates a service.
func (c *Client) CreateService(ctx context.Context, request ServiceRequest) (Service, error) {
	var created Service
	err := c.Post(ctx, servicesPath, request, &created)
	return created, err
}

// Service reads a single service.
func (c *Client) Service(ctx context.Context, id int64) (Service, error) {
	var service Service
	err := c.Get(ctx, fmt.Sprintf("%s%d/", servicesPath, id), nil, &service)
	return service, err
}

// UpdateService patches a service.
func (c *Client) UpdateService(ctx context.Context, id int64, request ServiceRequest) (Service, error) {
	var updated Service
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", servicesPath, id), request, &updated)
	return updated, err
}

// DeleteService deletes a service. This can fail with a 409 when observations
// still reference it as their origin service.
func (c *Client) DeleteService(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", servicesPath, id), nil)
}

// ServiceByName resolves a service by name within one product. Unlike the
// other name filters, the services filter matches exactly server-side; the
// client-side exact match is kept for consistency and costs nothing.
func (c *Client) ServiceByName(ctx context.Context, productID int64, name string) (Service, error) {
	return FindByExactName[Service](ctx, c, "service", servicesPath, name,
		url.Values{"product": {strconv.FormatInt(productID, 10)}})
}
