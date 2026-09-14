package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// AuthorizationGroupMemberRequest grants a user membership of an
// authorization group.
//
// AuthorizationGroup and User are immutable after create
// (access_control/api/serializers.py: "Authorization group and user cannot
// be changed").
type AuthorizationGroupMemberRequest struct {
	AuthorizationGroup int64 `json:"authorization_group"`
	User               int64 `json:"user"`
	IsManager          bool  `json:"is_manager"`
}

// AuthorizationGroupMember is the read representation of an authorization
// group membership.
type AuthorizationGroupMember struct {
	ID                 int64 `json:"id"`
	AuthorizationGroup int64 `json:"authorization_group"`
	User               int64 `json:"user"`
	IsManager          bool  `json:"is_manager"`
}

const authorizationGroupMembersPath = "api/authorization_group_members/"

// CreateAuthorizationGroupMember grants a user membership of an
// authorization group.
func (c *Client) CreateAuthorizationGroupMember(
	ctx context.Context, request AuthorizationGroupMemberRequest,
) (AuthorizationGroupMember, error) {
	var created AuthorizationGroupMember
	err := c.Post(ctx, authorizationGroupMembersPath, request, &created)
	return created, err
}

// AuthorizationGroupMember reads a single membership.
func (c *Client) AuthorizationGroupMember(ctx context.Context, id int64) (AuthorizationGroupMember, error) {
	var member AuthorizationGroupMember
	err := c.Get(ctx, fmt.Sprintf("%s%d/", authorizationGroupMembersPath, id), nil, &member)
	return member, err
}

// UpdateAuthorizationGroupMemberIsManager changes only is_manager: the other
// two fields are immutable.
func (c *Client) UpdateAuthorizationGroupMemberIsManager(ctx context.Context, id int64, isManager bool) (AuthorizationGroupMember, error) {
	var updated AuthorizationGroupMember
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", authorizationGroupMembersPath, id),
		map[string]bool{"is_manager": isManager}, &updated)
	return updated, err
}

// DeleteAuthorizationGroupMember revokes a membership.
func (c *Client) DeleteAuthorizationGroupMember(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", authorizationGroupMembersPath, id), nil)
}

// FindAuthorizationGroupMember locates a membership by its natural key. Used
// for import, where only the group and user are known.
func (c *Client) FindAuthorizationGroupMember(ctx context.Context, authorizationGroupID, userID int64) (AuthorizationGroupMember, error) {
	query := url.Values{
		"authorization_group": {strconv.FormatInt(authorizationGroupID, 10)},
		"user":                {strconv.FormatInt(userID, 10)},
	}
	members, err := List[AuthorizationGroupMember](ctx, c, authorizationGroupMembersPath, query)
	if err != nil {
		return AuthorizationGroupMember{}, err
	}
	for _, member := range members {
		if member.AuthorizationGroup == authorizationGroupID && member.User == userID {
			return member, nil
		}
	}
	return AuthorizationGroupMember{}, &ErrNotFound{
		Kind:  "authorization group member",
		Field: "authorization_group/user",
		Name:  fmt.Sprintf("%d/%d", authorizationGroupID, userID),
	}
}
