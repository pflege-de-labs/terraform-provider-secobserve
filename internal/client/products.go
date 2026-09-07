package client

import (
	"context"
	"fmt"
	"net/url"
)

// ProductRequest is the write representation of a product.
//
// Every field is serialized on every request, with nil mapping to JSON null.
// Omitting fields would make the outcome depend on which attributes happen to
// be set: SecObserve reads an absent key as "leave unchanged", while its
// fill-and-clear logic for the security gate and branch housekeeping only
// fires when the matching *_active flag is present in the payload.
type ProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	ProductGroup     *int64 `json:"product_group"`
	Purl             string `json:"purl"`
	CPE23            string `json:"cpe23"`
	RepositoryPrefix string `json:"repository_prefix"`

	ApplyGeneralRules bool `json:"apply_general_rules"`

	SecurityGateFields
	BranchHousekeepingFields
	NotificationFields
	ApprovalFields
	ApproverFields
	RiskAcceptanceFields
	LicensePolicyFields
	BranchPropagationFields
	IssueTrackerFields
	ScannerFields
}

// ProductGroupRequest is the write representation of a product group.
//
// ProductGroupSerializer uses an explicit field allowlist rather than an
// exclude list, so the product-only attributes -- purl, cpe23, repository
// prefix, the issue tracker block, the scanners -- do not exist here.
type ProductGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	SecurityGateFields
	BranchHousekeepingFields
	NotificationFields
	ApprovalFields
	ApproverFields
	RiskAcceptanceFields
	LicensePolicyFields
	BranchPropagationFields
}

// Product is the read representation of a product or product group. Both
// endpoints answer with the same underlying model.
type Product struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	IsProductGroup bool   `json:"is_product_group"`

	ProductGroup     *int64  `json:"product_group"`
	ProductGroupName *string `json:"product_group_name"`
	Purl             string  `json:"purl"`
	CPE23            string  `json:"cpe23"`
	RepositoryPrefix string  `json:"repository_prefix"`

	// Read-only: set as a side effect of flipping is_default_branch on a
	// branch (core/services/branch.py:6-19).
	RepositoryDefaultBranch     *int64  `json:"repository_default_branch"`
	RepositoryDefaultBranchName *string `json:"repository_default_branch_name"`

	ApplyGeneralRules  bool  `json:"apply_general_rules"`
	SecurityGatePassed *bool `json:"security_gate_passed"`

	SecurityGateFields
	BranchHousekeepingFields
	NotificationFields
	ApprovalFields
	ApproverFields
	RiskAcceptanceFields
	LicensePolicyFields
	BranchPropagationFields
	IssueTrackerFields
	ScannerFields

	// Informational, exposed by the data sources only.
	ActiveCriticalObservationCount *int64 `json:"active_critical_observation_count"`
	ActiveHighObservationCount     *int64 `json:"active_high_observation_count"`
	ActiveMediumObservationCount   *int64 `json:"active_medium_observation_count"`
	ActiveLowObservationCount      *int64 `json:"active_low_observation_count"`
	ActiveNoneObservationCount     *int64 `json:"active_none_observation_count"`
	ActiveUnknownObservationCount  *int64 `json:"active_unknown_observation_count"`
	ProductsCount                  *int64 `json:"products_count"`

	HasCloudResource       bool `json:"has_cloud_resource"`
	HasComponent           bool `json:"has_component"`
	HasDockerImage         bool `json:"has_docker_image"`
	HasEndpoint            bool `json:"has_endpoint"`
	HasKubernetesResource  bool `json:"has_kubernetes_resource"`
	HasSource              bool `json:"has_source"`
	HasPotentialDuplicates bool `json:"has_potential_duplicates"`
}

// GetID implements Named.
func (p Product) GetID() int64 { return p.ID }

// GetName implements Named.
func (p Product) GetName() string { return p.Name }

const (
	productsPath      = "api/products/"
	productGroupsPath = "api/product_groups/"
)

// CreateProduct creates a product and returns the server's representation,
// which may differ from the request where SecObserve rewrites values.
func (c *Client) CreateProduct(ctx context.Context, request ProductRequest) (Product, error) {
	var created Product
	err := c.Post(ctx, productsPath, request, &created)
	return created, err
}

// Product reads a single product. The detail endpoint is used deliberately:
// the list serializer omits the two assessment-approver attributes.
func (c *Client) Product(ctx context.Context, id int64) (Product, error) {
	var product Product
	err := c.Get(ctx, fmt.Sprintf("%s%d/", productsPath, id), nil, &product)
	return product, err
}

// UpdateProduct patches a product with the full managed field set.
func (c *Client) UpdateProduct(ctx context.Context, id int64, request ProductRequest) (Product, error) {
	var updated Product
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", productsPath, id), request, &updated)
	return updated, err
}

// DeleteProduct deletes a product.
//
// The name is a mandatory confirmation parameter, compared case- and
// whitespace-sensitively (core/api/views_product.py:139-164). Deleting
// cascades to branches, services, observations and the product's API tokens.
func (c *Client) DeleteProduct(ctx context.Context, id int64, name string) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", productsPath, id), url.Values{"name": {name}})
}

// ProductByName resolves a product by its globally unique name.
func (c *Client) ProductByName(ctx context.Context, name string) (Product, error) {
	return FindByExactName[Product](ctx, c, "product", productsPath, name, nil)
}

// CreateProductGroup creates a product group. The endpoint forces
// is_product_group to true, so it is not part of the request.
func (c *Client) CreateProductGroup(ctx context.Context, request ProductGroupRequest) (Product, error) {
	var created Product
	err := c.Post(ctx, productGroupsPath, request, &created)
	return created, err
}

// ProductGroup reads a single product group.
func (c *Client) ProductGroup(ctx context.Context, id int64) (Product, error) {
	var group Product
	err := c.Get(ctx, fmt.Sprintf("%s%d/", productGroupsPath, id), nil, &group)
	return group, err
}

// UpdateProductGroup patches a product group.
func (c *Client) UpdateProductGroup(ctx context.Context, id int64, request ProductGroupRequest) (Product, error) {
	var updated Product
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", productGroupsPath, id), request, &updated)
	return updated, err
}

// DeleteProductGroup deletes a product group. Like products this needs the
// name as confirmation, and it cascades to every product in the group.
func (c *Client) DeleteProductGroup(ctx context.Context, id int64, name string) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", productGroupsPath, id), url.Values{"name": {name}})
}

// ProductGroupByName resolves a product group by name. Products and product
// groups share one unique constraint on the name.
func (c *Client) ProductGroupByName(ctx context.Context, name string) (Product, error) {
	return FindByExactName[Product](ctx, c, "product group", productGroupsPath, name, nil)
}
