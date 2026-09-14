package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// LicensePolicyRequest is the writable field set of a license policy.
//
// IgnoreComponentTypeList is sent instead of the raw comma-joined string:
// the serializer accepts either, but the list form avoids the string form's
// no-trimming quirk (licenses/api/serializers.py:471-488), where a
// space-separated string round-trips with a stray leading space per element.
type LicensePolicyRequest struct {
	Parent                  *int64   `json:"parent"`
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	IsPublic                bool     `json:"is_public"`
	IgnoreComponentTypeList []string `json:"ignore_component_type_list"`
}

// LicensePolicy is the read representation of a license policy.
//
// ParentName, IsParent, IsManager and the Has* flags are computed per the
// requesting identity or live database state; treat them as informational.
type LicensePolicy struct {
	ID                      int64    `json:"id"`
	Parent                  *int64   `json:"parent"`
	ParentName              string   `json:"parent_name"`
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	IsPublic                bool     `json:"is_public"`
	IgnoreComponentTypeList []string `json:"ignore_component_type_list"`
	IsParent                bool     `json:"is_parent"`
	IsManager               bool     `json:"is_manager"`
	HasProducts             bool     `json:"has_products"`
	HasProductGroups        bool     `json:"has_product_groups"`
	HasItems                bool     `json:"has_items"`
	HasUsers                bool     `json:"has_users"`
	HasAuthorizationGroups  bool     `json:"has_authorization_groups"`
}

// GetID implements Named.
func (p LicensePolicy) GetID() int64 { return p.ID }

// GetName implements Named.
func (p LicensePolicy) GetName() string { return p.Name }

const licensePoliciesPath = "api/license_policies/"

// CreateLicensePolicy creates a license policy.
//
// If the caller is not a superuser, the post_save signal
// (licenses/signals.py:18-37) implicitly makes the creator a manager member,
// exactly like CreateLicenseGroup.
func (c *Client) CreateLicensePolicy(ctx context.Context, request LicensePolicyRequest) (LicensePolicy, error) {
	var created LicensePolicy
	err := c.Post(ctx, licensePoliciesPath, request, &created)
	return created, err
}

// LicensePolicy reads a single license policy by id.
func (c *Client) LicensePolicy(ctx context.Context, id int64) (LicensePolicy, error) {
	var policy LicensePolicy
	err := c.Get(ctx, fmt.Sprintf("%s%d/", licensePoliciesPath, id), nil, &policy)
	return policy, err
}

// UpdateLicensePolicy updates a license policy.
//
// Setting parent on an instance that already has children, or to itself,
// fails with a 400 raised from the serializer's update() rather than
// validate() (licenses/api/serializers.py:490-499); the provider does not
// duplicate this check client-side, matching how the product_rule/
// api_configuration server-side checks were left server-side in Phase 3.
func (c *Client) UpdateLicensePolicy(ctx context.Context, id int64, request LicensePolicyRequest) (LicensePolicy, error) {
	var updated LicensePolicy
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", licensePoliciesPath, id), request, &updated)
	return updated, err
}

// DeleteLicensePolicy deletes a license policy. SecObserve returns 409 if a
// child policy or a product/product group still references it
// (License_Policy.parent and Product.license_policy are both
// on_delete=PROTECT).
func (c *Client) DeleteLicensePolicy(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", licensePoliciesPath, id), nil)
}

// LicensePolicyByName resolves a license policy by its exact, unique name.
func (c *Client) LicensePolicyByName(ctx context.Context, name string) (LicensePolicy, error) {
	return FindByExactName[LicensePolicy](ctx, c, "license policy", licensePoliciesPath, name, nil)
}

// LicensePolicyItemRequest is the writable field set of a license policy
// item.
//
// All four discriminator fields (LicenseGroup, License, LicenseExpression,
// NonSPDXLicense) are always sent, even when empty: the serializer's
// validate() overwrites all four on the server-side instance from whatever
// the request carries before checking "exactly one set"
// (licenses/api/serializers.py:530-534), so a partial PATCH that omits the
// other three silently clears them and then fails validation. This mirrors
// the project's existing rule of always writing the full managed field set.
type LicensePolicyItemRequest struct {
	LicensePolicy     int64  `json:"license_policy"`
	LicenseGroup      *int64 `json:"license_group"`
	License           *int64 `json:"license"`
	LicenseExpression string `json:"license_expression"`
	NonSPDXLicense    string `json:"non_spdx_license"`
	EvaluationResult  string `json:"evaluation_result"`
	Comment           string `json:"comment"`
}

// LicensePolicyItem is the read representation of a license policy item.
type LicensePolicyItem struct {
	ID                int64  `json:"id"`
	LicensePolicy     int64  `json:"license_policy"`
	LicenseGroup      *int64 `json:"license_group"`
	LicenseGroupName  string `json:"license_group_name"`
	License           *int64 `json:"license"`
	LicenseSPDXID     string `json:"license_spdx_id"`
	LicenseExpression string `json:"license_expression"`
	NonSPDXLicense    string `json:"non_spdx_license"`
	EvaluationResult  string `json:"evaluation_result"`
	Comment           string `json:"comment"`
}

const licensePolicyItemsPath = "api/license_policy_items/"

// CreateLicensePolicyItem creates a license policy item.
func (c *Client) CreateLicensePolicyItem(ctx context.Context, request LicensePolicyItemRequest) (LicensePolicyItem, error) {
	var created LicensePolicyItem
	err := c.Post(ctx, licensePolicyItemsPath, request, &created)
	return created, err
}

// LicensePolicyItem reads a single item.
func (c *Client) LicensePolicyItem(ctx context.Context, id int64) (LicensePolicyItem, error) {
	var item LicensePolicyItem
	err := c.Get(ctx, fmt.Sprintf("%s%d/", licensePolicyItemsPath, id), nil, &item)
	return item, err
}

// UpdateLicensePolicyItem updates a license policy item. See
// LicensePolicyItemRequest's doc comment for why the full body is required.
func (c *Client) UpdateLicensePolicyItem(ctx context.Context, id int64, request LicensePolicyItemRequest) (LicensePolicyItem, error) {
	var updated LicensePolicyItem
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", licensePolicyItemsPath, id), request, &updated)
	return updated, err
}

// DeleteLicensePolicyItem deletes a license policy item.
func (c *Client) DeleteLicensePolicyItem(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", licensePolicyItemsPath, id), nil)
}

// LicensePolicyMemberRequest grants a user manage rights on a license
// policy.
type LicensePolicyMemberRequest struct {
	LicensePolicy int64 `json:"license_policy"`
	User          int64 `json:"user"`
	IsManager     bool  `json:"is_manager"`
}

// LicensePolicyMember is the read representation of a license policy
// membership.
type LicensePolicyMember struct {
	ID            int64 `json:"id"`
	LicensePolicy int64 `json:"license_policy"`
	User          int64 `json:"user"`
	IsManager     bool  `json:"is_manager"`
}

const licensePolicyMembersPath = "api/license_policy_members/"

// CreateLicensePolicyMember grants a user manage rights on a license policy.
func (c *Client) CreateLicensePolicyMember(ctx context.Context, request LicensePolicyMemberRequest) (LicensePolicyMember, error) {
	var created LicensePolicyMember
	err := c.Post(ctx, licensePolicyMembersPath, request, &created)
	return created, err
}

// LicensePolicyMember reads a single membership.
func (c *Client) LicensePolicyMember(ctx context.Context, id int64) (LicensePolicyMember, error) {
	var member LicensePolicyMember
	err := c.Get(ctx, fmt.Sprintf("%s%d/", licensePolicyMembersPath, id), nil, &member)
	return member, err
}

// UpdateLicensePolicyMemberIsManager updates a membership's is_manager flag.
func (c *Client) UpdateLicensePolicyMemberIsManager(ctx context.Context, id int64, isManager bool) (LicensePolicyMember, error) {
	var updated LicensePolicyMember
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", licensePolicyMembersPath, id),
		map[string]bool{"is_manager": isManager}, &updated)
	return updated, err
}

// DeleteLicensePolicyMember revokes a membership.
func (c *Client) DeleteLicensePolicyMember(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", licensePolicyMembersPath, id), nil)
}

// FindLicensePolicyMember locates a membership by its natural key, for
// import.
func (c *Client) FindLicensePolicyMember(ctx context.Context, licensePolicyID, userID int64) (LicensePolicyMember, error) {
	query := url.Values{
		"license_policy": {strconv.FormatInt(licensePolicyID, 10)},
		"user":           {strconv.FormatInt(userID, 10)},
	}
	members, err := List[LicensePolicyMember](ctx, c, licensePolicyMembersPath, query)
	if err != nil {
		return LicensePolicyMember{}, err
	}
	if len(members) == 0 {
		return LicensePolicyMember{}, &ErrNotFound{
			Kind: "license policy member",
			Name: fmt.Sprintf("%d/%d", licensePolicyID, userID),
		}
	}
	return members[0], nil
}

// LicensePolicyAuthorizationGroupMemberRequest grants an authorization group
// manage rights on a license policy.
type LicensePolicyAuthorizationGroupMemberRequest struct {
	LicensePolicy      int64 `json:"license_policy"`
	AuthorizationGroup int64 `json:"authorization_group"`
	IsManager          bool  `json:"is_manager"`
}

// LicensePolicyAuthorizationGroupMember is the read representation.
type LicensePolicyAuthorizationGroupMember struct {
	ID                 int64 `json:"id"`
	LicensePolicy      int64 `json:"license_policy"`
	AuthorizationGroup int64 `json:"authorization_group"`
	IsManager          bool  `json:"is_manager"`
}

const licensePolicyAuthorizationGroupMembersPath = "api/license_policy_authorization_group_members/"

// CreateLicensePolicyAuthorizationGroupMember grants an authorization group
// manage rights on a license policy.
func (c *Client) CreateLicensePolicyAuthorizationGroupMember(
	ctx context.Context, request LicensePolicyAuthorizationGroupMemberRequest,
) (LicensePolicyAuthorizationGroupMember, error) {
	var created LicensePolicyAuthorizationGroupMember
	err := c.Post(ctx, licensePolicyAuthorizationGroupMembersPath, request, &created)
	return created, err
}

// LicensePolicyAuthorizationGroupMember reads a single membership.
func (c *Client) LicensePolicyAuthorizationGroupMember(ctx context.Context, id int64) (LicensePolicyAuthorizationGroupMember, error) {
	var member LicensePolicyAuthorizationGroupMember
	err := c.Get(ctx, fmt.Sprintf("%s%d/", licensePolicyAuthorizationGroupMembersPath, id), nil, &member)
	return member, err
}

// UpdateLicensePolicyAuthorizationGroupMemberIsManager updates is_manager.
func (c *Client) UpdateLicensePolicyAuthorizationGroupMemberIsManager(
	ctx context.Context, id int64, isManager bool,
) (LicensePolicyAuthorizationGroupMember, error) {
	var updated LicensePolicyAuthorizationGroupMember
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", licensePolicyAuthorizationGroupMembersPath, id),
		map[string]bool{"is_manager": isManager}, &updated)
	return updated, err
}

// DeleteLicensePolicyAuthorizationGroupMember revokes a membership.
func (c *Client) DeleteLicensePolicyAuthorizationGroupMember(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", licensePolicyAuthorizationGroupMembersPath, id), nil)
}

// FindLicensePolicyAuthorizationGroupMember locates a membership by its
// natural key, for import.
func (c *Client) FindLicensePolicyAuthorizationGroupMember(
	ctx context.Context, licensePolicyID, authorizationGroupID int64,
) (LicensePolicyAuthorizationGroupMember, error) {
	query := url.Values{
		"license_policy":      {strconv.FormatInt(licensePolicyID, 10)},
		"authorization_group": {strconv.FormatInt(authorizationGroupID, 10)},
	}
	members, err := List[LicensePolicyAuthorizationGroupMember](ctx, c, licensePolicyAuthorizationGroupMembersPath, query)
	if err != nil {
		return LicensePolicyAuthorizationGroupMember{}, err
	}
	if len(members) == 0 {
		return LicensePolicyAuthorizationGroupMember{}, &ErrNotFound{
			Kind: "license policy authorization group member",
			Name: fmt.Sprintf("%d/%d", licensePolicyID, authorizationGroupID),
		}
	}
	return members[0], nil
}
