package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// BranchRequest is the write representation of a branch or version.
type BranchRequest struct {
	// Product is immutable after create and must be present in the body even
	// before validation runs: the permission layer reads it directly
	// (authorization/api/permissions_base.py:11-24).
	Product int64  `json:"product"`
	Name    string `json:"name"`

	// Setting this to true clears the flag on the product's other branches and
	// updates the product's read-only repository_default_branch
	// (core/services/branch.py:6-19). A branch with the flag set cannot be
	// deleted.
	IsDefaultBranch bool `json:"is_default_branch"`

	HousekeepingProtect  bool   `json:"housekeeping_protect"`
	Purl                 string `json:"purl"`
	CPE23                string `json:"cpe23"`
	OSVLinuxDistribution string `json:"osv_linux_distribution"`
	OSVLinuxRelease      string `json:"osv_linux_release"`
}

// Branch is the read representation of a branch.
type Branch struct {
	ID                   int64   `json:"id"`
	Product              int64   `json:"product"`
	Name                 string  `json:"name"`
	NameWithProduct      string  `json:"name_with_product"`
	IsDefaultBranch      bool    `json:"is_default_branch"`
	LastImport           *string `json:"last_import"`
	HousekeepingProtect  bool    `json:"housekeeping_protect"`
	Purl                 string  `json:"purl"`
	CPE23                string  `json:"cpe23"`
	OSVLinuxDistribution string  `json:"osv_linux_distribution"`
	OSVLinuxRelease      string  `json:"osv_linux_release"`
}

// GetID implements Named.
func (b Branch) GetID() int64 { return b.ID }

// GetName implements Named.
func (b Branch) GetName() string { return b.Name }

const branchesPath = "api/branches/"

// CreateBranch creates a branch.
func (c *Client) CreateBranch(ctx context.Context, request BranchRequest) (Branch, error) {
	var created Branch
	err := c.Post(ctx, branchesPath, request, &created)
	return created, err
}

// Branch reads a single branch.
func (c *Client) Branch(ctx context.Context, id int64) (Branch, error) {
	var branch Branch
	err := c.Get(ctx, fmt.Sprintf("%s%d/", branchesPath, id), nil, &branch)
	return branch, err
}

// UpdateBranch patches a branch.
func (c *Client) UpdateBranch(ctx context.Context, id int64, request BranchRequest) (Branch, error) {
	var updated Branch
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", branchesPath, id), request, &updated)
	return updated, err
}

// DeleteBranch deletes a branch. SecObserve refuses to delete the product's
// default branch (core/api/views_product.py:541-546).
func (c *Client) DeleteBranch(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", branchesPath, id), nil)
}

// BranchByName resolves a branch by name within one product. Branch names are
// unique per product, not globally.
func (c *Client) BranchByName(ctx context.Context, productID int64, name string) (Branch, error) {
	return FindByExactName[Branch](ctx, c, "branch", branchesPath, name,
		url.Values{"product": {strconv.FormatInt(productID, 10)}})
}
