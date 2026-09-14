package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// License is a read-only SPDX license reference. SecObserve seeds these from
// the SPDX license list; there is no write endpoint.
type License struct {
	ID            int64  `json:"id"`
	SPDXID        string `json:"spdx_id"`
	Name          string `json:"name"`
	Reference     string `json:"reference"`
	IsOSIApproved *bool  `json:"is_osi_approved"`
	IsDeprecated  *bool  `json:"is_deprecated"`
}

const licensesPath = "api/licenses/"

// LicenseBySPDXID resolves a license by its unique spdx_id. The list filter
// is icontains (licenses/api/filters.py), so the exact match is applied
// client-side; spdx_id is DB-unique so a match, once found, is unambiguous.
func (c *Client) LicenseBySPDXID(ctx context.Context, spdxID string) (License, error) {
	var zero License

	candidates, err := List[License](ctx, c, licensesPath, url.Values{"spdx_id": {spdxID}})
	if err != nil {
		return zero, err
	}
	for _, candidate := range candidates {
		if candidate.SPDXID == spdxID {
			return candidate, nil
		}
	}
	return zero, &ErrNotFound{Kind: "license", Name: spdxID, Field: "spdx_id"}
}

// LicenseGroupRequest is the writable field set of a license group. Licenses
// are not part of this request: the API excludes the licenses M2M from the
// license_groups serializer entirely and only mutates it through the
// add_license/remove_license actions.
type LicenseGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// LicenseGroup is the read representation of a license group.
//
// IsManager, IsInLicensePolicy and the Has* flags are all computed per the
// requesting identity or live database state; they are informational, never
// something to plan a diff against.
type LicenseGroup struct {
	ID                     int64  `json:"id"`
	Name                   string `json:"name"`
	Description            string `json:"description"`
	IsPublic               bool   `json:"is_public"`
	IsManager              bool   `json:"is_manager"`
	IsInLicensePolicy      bool   `json:"is_in_license_policy"`
	HasLicenses            bool   `json:"has_licenses"`
	HasUsers               bool   `json:"has_users"`
	HasAuthorizationGroups bool   `json:"has_authorization_groups"`
}

// GetID implements Named.
func (g LicenseGroup) GetID() int64 { return g.ID }

// GetName implements Named.
func (g LicenseGroup) GetName() string { return g.Name }

const licenseGroupsPath = "api/license_groups/"

// CreateLicenseGroup creates a license group. Licenses are added separately
// via AddLicenseToGroup.
//
// If the caller is not a superuser, SecObserve's post_save signal
// (licenses/signals.py:18-37) implicitly makes the creator a manager member
// -- if a secobserve_license_group_member for that same user is also
// declared, its create fails as a duplicate. Run the provider as a
// superuser to avoid this.
func (c *Client) CreateLicenseGroup(ctx context.Context, request LicenseGroupRequest) (LicenseGroup, error) {
	var created LicenseGroup
	err := c.Post(ctx, licenseGroupsPath, request, &created)
	return created, err
}

// LicenseGroup reads a single license group by id.
func (c *Client) LicenseGroup(ctx context.Context, id int64) (LicenseGroup, error) {
	var group LicenseGroup
	err := c.Get(ctx, fmt.Sprintf("%s%d/", licenseGroupsPath, id), nil, &group)
	return group, err
}

// UpdateLicenseGroup updates a license group's own fields.
func (c *Client) UpdateLicenseGroup(ctx context.Context, id int64, request LicenseGroupRequest) (LicenseGroup, error) {
	var updated LicenseGroup
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", licenseGroupsPath, id), request, &updated)
	return updated, err
}

// DeleteLicenseGroup deletes a license group. SecObserve returns 409 if any
// license_policy_item still references it (License_Policy_Item.license_group
// is on_delete=PROTECT).
func (c *Client) DeleteLicenseGroup(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", licenseGroupsPath, id), nil)
}

// LicenseGroupByName resolves a license group by its exact, unique name.
func (c *Client) LicenseGroupByName(ctx context.Context, name string) (LicenseGroup, error) {
	return FindByExactName[LicenseGroup](ctx, c, "license group", licenseGroupsPath, name, nil)
}

// licenseAddRemoveRequest is the body for add_license/remove_license.
type licenseAddRemoveRequest struct {
	License int64 `json:"license"`
}

// AddLicenseToGroup adds a license to a group.
//
// Not idempotent: SecObserve rejects re-adding a license already in the
// group with a 400 ("License ... is already in this license group"), so
// callers must diff against ListLicensesInGroup first.
func (c *Client) AddLicenseToGroup(ctx context.Context, groupID, licenseID int64) error {
	return c.Post(ctx, fmt.Sprintf("%s%d/add_license/", licenseGroupsPath, groupID),
		licenseAddRemoveRequest{License: licenseID}, nil)
}

// RemoveLicenseFromGroup removes a license from a group. Idempotent: removing
// a license that is not a member is a no-op on the server.
func (c *Client) RemoveLicenseFromGroup(ctx context.Context, groupID, licenseID int64) error {
	return c.Post(ctx, fmt.Sprintf("%s%d/remove_license/", licenseGroupsPath, groupID),
		licenseAddRemoveRequest{License: licenseID}, nil)
}

// ListLicensesInGroup lists the licenses currently in a group.
//
// The license_groups serializer never exposes its licenses M2M, so this is
// the only way to read current membership back: License's own list filter
// supports an exact license_groups=<id> match (licenses/api/filters.py:187).
func (c *Client) ListLicensesInGroup(ctx context.Context, groupID int64) ([]License, error) {
	return List[License](ctx, c, licensesPath, url.Values{"license_groups": {strconv.FormatInt(groupID, 10)}})
}

// LicenseGroupMemberRequest grants a user manage rights on a license group.
type LicenseGroupMemberRequest struct {
	LicenseGroup int64 `json:"license_group"`
	User         int64 `json:"user"`
	IsManager    bool  `json:"is_manager"`
}

// LicenseGroupMember is the read representation of a license group
// membership.
type LicenseGroupMember struct {
	ID           int64 `json:"id"`
	LicenseGroup int64 `json:"license_group"`
	User         int64 `json:"user"`
	IsManager    bool  `json:"is_manager"`
}

const licenseGroupMembersPath = "api/license_group_members/"

// CreateLicenseGroupMember grants a user manage rights on a license group.
func (c *Client) CreateLicenseGroupMember(ctx context.Context, request LicenseGroupMemberRequest) (LicenseGroupMember, error) {
	var created LicenseGroupMember
	err := c.Post(ctx, licenseGroupMembersPath, request, &created)
	return created, err
}

// LicenseGroupMember reads a single membership.
func (c *Client) LicenseGroupMember(ctx context.Context, id int64) (LicenseGroupMember, error) {
	var member LicenseGroupMember
	err := c.Get(ctx, fmt.Sprintf("%s%d/", licenseGroupMembersPath, id), nil, &member)
	return member, err
}

// UpdateLicenseGroupMemberIsManager updates a membership's is_manager flag.
// LicenseGroup and User are immutable (licenses/api/serializers.py:341-345).
func (c *Client) UpdateLicenseGroupMemberIsManager(ctx context.Context, id int64, isManager bool) (LicenseGroupMember, error) {
	var updated LicenseGroupMember
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", licenseGroupMembersPath, id),
		map[string]bool{"is_manager": isManager}, &updated)
	return updated, err
}

// DeleteLicenseGroupMember revokes a membership.
func (c *Client) DeleteLicenseGroupMember(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", licenseGroupMembersPath, id), nil)
}

// FindLicenseGroupMember locates a membership by its natural key, for import.
func (c *Client) FindLicenseGroupMember(ctx context.Context, licenseGroupID, userID int64) (LicenseGroupMember, error) {
	query := url.Values{
		"license_group": {strconv.FormatInt(licenseGroupID, 10)},
		"user":          {strconv.FormatInt(userID, 10)},
	}
	members, err := List[LicenseGroupMember](ctx, c, licenseGroupMembersPath, query)
	if err != nil {
		return LicenseGroupMember{}, err
	}
	if len(members) == 0 {
		return LicenseGroupMember{}, &ErrNotFound{
			Kind: "license group member",
			Name: fmt.Sprintf("%d/%d", licenseGroupID, userID),
		}
	}
	return members[0], nil
}

// LicenseGroupAuthorizationGroupMemberRequest grants an authorization group
// manage rights on a license group.
type LicenseGroupAuthorizationGroupMemberRequest struct {
	LicenseGroup       int64 `json:"license_group"`
	AuthorizationGroup int64 `json:"authorization_group"`
	IsManager          bool  `json:"is_manager"`
}

// LicenseGroupAuthorizationGroupMember is the read representation.
type LicenseGroupAuthorizationGroupMember struct {
	ID                 int64 `json:"id"`
	LicenseGroup       int64 `json:"license_group"`
	AuthorizationGroup int64 `json:"authorization_group"`
	IsManager          bool  `json:"is_manager"`
}

const licenseGroupAuthorizationGroupMembersPath = "api/license_group_authorization_group_members/"

// CreateLicenseGroupAuthorizationGroupMember grants an authorization group
// manage rights on a license group.
func (c *Client) CreateLicenseGroupAuthorizationGroupMember(
	ctx context.Context, request LicenseGroupAuthorizationGroupMemberRequest,
) (LicenseGroupAuthorizationGroupMember, error) {
	var created LicenseGroupAuthorizationGroupMember
	err := c.Post(ctx, licenseGroupAuthorizationGroupMembersPath, request, &created)
	return created, err
}

// LicenseGroupAuthorizationGroupMember reads a single membership.
func (c *Client) LicenseGroupAuthorizationGroupMember(ctx context.Context, id int64) (LicenseGroupAuthorizationGroupMember, error) {
	var member LicenseGroupAuthorizationGroupMember
	err := c.Get(ctx, fmt.Sprintf("%s%d/", licenseGroupAuthorizationGroupMembersPath, id), nil, &member)
	return member, err
}

// UpdateLicenseGroupAuthorizationGroupMemberIsManager updates is_manager.
func (c *Client) UpdateLicenseGroupAuthorizationGroupMemberIsManager(
	ctx context.Context, id int64, isManager bool,
) (LicenseGroupAuthorizationGroupMember, error) {
	var updated LicenseGroupAuthorizationGroupMember
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", licenseGroupAuthorizationGroupMembersPath, id),
		map[string]bool{"is_manager": isManager}, &updated)
	return updated, err
}

// DeleteLicenseGroupAuthorizationGroupMember revokes a membership.
func (c *Client) DeleteLicenseGroupAuthorizationGroupMember(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", licenseGroupAuthorizationGroupMembersPath, id), nil)
}

// FindLicenseGroupAuthorizationGroupMember locates a membership by its
// natural key, for import.
func (c *Client) FindLicenseGroupAuthorizationGroupMember(
	ctx context.Context, licenseGroupID, authorizationGroupID int64,
) (LicenseGroupAuthorizationGroupMember, error) {
	query := url.Values{
		"license_group":       {strconv.FormatInt(licenseGroupID, 10)},
		"authorization_group": {strconv.FormatInt(authorizationGroupID, 10)},
	}
	members, err := List[LicenseGroupAuthorizationGroupMember](ctx, c, licenseGroupAuthorizationGroupMembersPath, query)
	if err != nil {
		return LicenseGroupAuthorizationGroupMember{}, err
	}
	if len(members) == 0 {
		return LicenseGroupAuthorizationGroupMember{}, &ErrNotFound{
			Kind: "license group authorization group member",
			Name: fmt.Sprintf("%d/%d", licenseGroupID, authorizationGroupID),
		}
	}
	return members[0], nil
}
