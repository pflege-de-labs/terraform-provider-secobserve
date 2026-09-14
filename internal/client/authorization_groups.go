package client

import (
	"context"
	"fmt"
)

// AuthorizationGroupRequest is the write representation of an authorization
// group.
type AuthorizationGroupRequest struct {
	Name string `json:"name"`
	// OIDCGroup maps an IdP group claim to this authorization group. When
	// non-empty, SecObserve's login-time group sync takes over membership of
	// this group entirely -- see AuthorizationGroupMember below.
	OIDCGroup string `json:"oidc_group"`
}

// AuthorizationGroup is the read representation of an authorization group.
type AuthorizationGroup struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	OIDCGroup string `json:"oidc_group"`

	// Read-only, informational.
	HasProductGroupMembers bool `json:"has_product_group_members"`
	HasProductMembers      bool `json:"has_product_members"`
	HasUsers               bool `json:"has_users"`
	IsManager              bool `json:"is_manager"`
}

// GetID implements Named.
func (g AuthorizationGroup) GetID() int64 { return g.ID }

// GetName implements Named.
func (g AuthorizationGroup) GetName() string { return g.Name }

const authorizationGroupsPath = "api/authorization_groups/"

// CreateAuthorizationGroup creates an authorization group.
func (c *Client) CreateAuthorizationGroup(ctx context.Context, request AuthorizationGroupRequest) (AuthorizationGroup, error) {
	var created AuthorizationGroup
	err := c.Post(ctx, authorizationGroupsPath, request, &created)
	return created, err
}

// AuthorizationGroup reads a single authorization group.
func (c *Client) AuthorizationGroup(ctx context.Context, id int64) (AuthorizationGroup, error) {
	var group AuthorizationGroup
	err := c.Get(ctx, fmt.Sprintf("%s%d/", authorizationGroupsPath, id), nil, &group)
	return group, err
}

// UpdateAuthorizationGroup patches an authorization group.
func (c *Client) UpdateAuthorizationGroup(ctx context.Context, id int64, request AuthorizationGroupRequest) (AuthorizationGroup, error) {
	var updated AuthorizationGroup
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", authorizationGroupsPath, id), request, &updated)
	return updated, err
}

// DeleteAuthorizationGroup deletes an authorization group.
func (c *Client) DeleteAuthorizationGroup(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", authorizationGroupsPath, id), nil)
}

// AuthorizationGroupByName resolves an authorization group by its exact,
// unique name.
func (c *Client) AuthorizationGroupByName(ctx context.Context, name string) (AuthorizationGroup, error) {
	return FindByExactName[AuthorizationGroup](ctx, c, "authorization group", authorizationGroupsPath, name, nil)
}
