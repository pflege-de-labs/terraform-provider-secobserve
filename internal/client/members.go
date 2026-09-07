package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Roles are SecObserve's product roles. The API validates only the numeric
// range 1..5, so an out-of-enum value inside that range is accepted; the
// provider validates against these names instead.
const (
	RoleReader     int64 = 1
	RoleUpload     int64 = 2
	RoleWriter     int64 = 3
	RoleMaintainer int64 = 4
	RoleOwner      int64 = 5
)

// RoleNames maps the role name used in Terraform configuration to the numeric
// value the API expects.
var RoleNames = map[string]int64{
	"Reader":     RoleReader,
	"Upload":     RoleUpload,
	"Writer":     RoleWriter,
	"Maintainer": RoleMaintainer,
	"Owner":      RoleOwner,
}

// RoleName returns the name of a numeric role, or the number as a string when
// it is outside the known enum.
func RoleName(role int64) string {
	for name, value := range RoleNames {
		if value == role {
			return name
		}
	}
	return strconv.FormatInt(role, 10)
}

// ProductMemberRequest grants a user a role on a product or product group.
//
// Product and user are immutable after create
// (core/api/serializers_product.py:751-754). Assigning or changing the Owner
// role requires the caller to be an Owner or a superuser.
type ProductMemberRequest struct {
	Product int64 `json:"product"`
	User    int64 `json:"user"`
	Role    int64 `json:"role"`
}

// ProductMember is the read representation of a product membership.
type ProductMember struct {
	ID      int64 `json:"id"`
	Product int64 `json:"product"`
	User    int64 `json:"user"`
	Role    int64 `json:"role"`
}

// ProductAuthorizationGroupMemberRequest grants an authorization group a role
// on a product or product group.
type ProductAuthorizationGroupMemberRequest struct {
	Product            int64 `json:"product"`
	AuthorizationGroup int64 `json:"authorization_group"`
	Role               int64 `json:"role"`
}

// ProductAuthorizationGroupMember is the read representation of an
// authorization group's membership in a product.
type ProductAuthorizationGroupMember struct {
	ID                 int64 `json:"id"`
	Product            int64 `json:"product"`
	AuthorizationGroup int64 `json:"authorization_group"`
	Role               int64 `json:"role"`
}

const (
	productMembersPath                   = "api/product_members/"
	productAuthorizationGroupMembersPath = "api/product_authorization_group_members/"
)

// CreateProductMember grants a role to a user.
//
// Note that creating a product implicitly makes its creator an Owner member
// (core/signals.py:73-76), so granting a role to the identity that created the
// product fails as a duplicate.
func (c *Client) CreateProductMember(ctx context.Context, request ProductMemberRequest) (ProductMember, error) {
	var created ProductMember
	err := c.Post(ctx, productMembersPath, request, &created)
	return created, err
}

// ProductMember reads a single membership.
func (c *Client) ProductMember(ctx context.Context, id int64) (ProductMember, error) {
	var member ProductMember
	err := c.Get(ctx, fmt.Sprintf("%s%d/", productMembersPath, id), nil, &member)
	return member, err
}

// UpdateProductMemberRole changes a membership's role. Product and user cannot
// be changed, so only the role is sent.
func (c *Client) UpdateProductMemberRole(ctx context.Context, id, role int64) (ProductMember, error) {
	var updated ProductMember
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", productMembersPath, id), map[string]int64{"role": role}, &updated)
	return updated, err
}

// DeleteProductMember revokes a membership.
func (c *Client) DeleteProductMember(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", productMembersPath, id), nil)
}

// FindProductMember locates a membership by its natural key. Used for import,
// where only the product and user are known.
func (c *Client) FindProductMember(ctx context.Context, productID, userID int64) (ProductMember, error) {
	query := url.Values{
		"product": {strconv.FormatInt(productID, 10)},
		"user":    {strconv.FormatInt(userID, 10)},
	}
	members, err := List[ProductMember](ctx, c, productMembersPath, query)
	if err != nil {
		return ProductMember{}, err
	}
	for _, member := range members {
		if member.Product == productID && member.User == userID {
			return member, nil
		}
	}
	return ProductMember{}, &ErrNotFound{
		Kind:  "product member",
		Field: "product/user",
		Name:  fmt.Sprintf("%d/%d", productID, userID),
	}
}

// CreateProductAuthorizationGroupMember grants a role to an authorization
// group.
func (c *Client) CreateProductAuthorizationGroupMember(
	ctx context.Context, request ProductAuthorizationGroupMemberRequest,
) (ProductAuthorizationGroupMember, error) {
	var created ProductAuthorizationGroupMember
	err := c.Post(ctx, productAuthorizationGroupMembersPath, request, &created)
	return created, err
}

// ProductAuthorizationGroupMember reads a single authorization group
// membership.
func (c *Client) ProductAuthorizationGroupMember(
	ctx context.Context, id int64,
) (ProductAuthorizationGroupMember, error) {
	var member ProductAuthorizationGroupMember
	err := c.Get(ctx, fmt.Sprintf("%s%d/", productAuthorizationGroupMembersPath, id), nil, &member)
	return member, err
}

// UpdateProductAuthorizationGroupMemberRole changes the role.
//
// SecObserve refuses to lower the role below Writer while the group is a
// designated assessment approver for the product or its product group
// (core/api/serializers_product.py:813-826).
func (c *Client) UpdateProductAuthorizationGroupMemberRole(
	ctx context.Context, id, role int64,
) (ProductAuthorizationGroupMember, error) {
	var updated ProductAuthorizationGroupMember
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", productAuthorizationGroupMembersPath, id),
		map[string]int64{"role": role}, &updated)
	return updated, err
}

// DeleteProductAuthorizationGroupMember revokes an authorization group's
// membership.
func (c *Client) DeleteProductAuthorizationGroupMember(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", productAuthorizationGroupMembersPath, id), nil)
}

// FindProductAuthorizationGroupMember locates a membership by its natural key.
func (c *Client) FindProductAuthorizationGroupMember(
	ctx context.Context, productID, authorizationGroupID int64,
) (ProductAuthorizationGroupMember, error) {
	query := url.Values{
		"product":             {strconv.FormatInt(productID, 10)},
		"authorization_group": {strconv.FormatInt(authorizationGroupID, 10)},
	}
	members, err := List[ProductAuthorizationGroupMember](ctx, c, productAuthorizationGroupMembersPath, query)
	if err != nil {
		return ProductAuthorizationGroupMember{}, err
	}
	for _, member := range members {
		if member.Product == productID && member.AuthorizationGroup == authorizationGroupID {
			return member, nil
		}
	}
	return ProductAuthorizationGroupMember{}, &ErrNotFound{
		Kind:  "product authorization group member",
		Field: "product/authorization_group",
		Name:  fmt.Sprintf("%d/%d", productID, authorizationGroupID),
	}
}
