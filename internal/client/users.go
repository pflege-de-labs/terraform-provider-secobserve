package client

import (
	"context"
	"fmt"
)

// UserRequest is the write representation of a user.
//
// This is UserUpdateSerializer's field set (access_control/api/serializers.py).
// Notably absent: a password. Users created through the API have no usable
// password until one is set out-of-band via PATCH
// /api/users/{id}/change_password/, which requires the *caller's own* current
// password and so cannot be driven by a token.
type UserRequest struct {
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	IsActive    bool   `json:"is_active"`
	IsSuperuser bool   `json:"is_superuser"`
	IsExternal  bool   `json:"is_external"`
}

// User is the read representation of a user.
//
// FullName is server-recomputed on every save whenever FirstName or LastName
// is non-empty (access_control/models.py User.save()) -- it only reflects
// what was submitted when both are empty. Always take it from the response.
type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	IsActive    bool   `json:"is_active"`
	IsSuperuser bool   `json:"is_superuser"`
	IsExternal  bool   `json:"is_external"`

	// Read-only, informational.
	IsOIDCUser  bool   `json:"is_oidc_user"`
	DateJoined  string `json:"date_joined"`
	HasPassword bool   `json:"has_password"`

	HasAuthorizationGroups bool `json:"has_authorization_groups"`
	HasProductGroupMembers bool `json:"has_product_group_members"`
	HasProductMembers      bool `json:"has_product_members"`
	HasAPITokens           bool `json:"has_api_tokens"`
}

// GetID implements Named.
func (u User) GetID() int64 { return u.ID }

// GetName implements Named. Usernames are unique (Django's AbstractUser), so
// they are the natural key for name lookups and import.
func (u User) GetName() string { return u.Username }

const usersPath = "api/users/"

// CreateUser creates a user. Superuser-only.
func (c *Client) CreateUser(ctx context.Context, request UserRequest) (User, error) {
	var created User
	err := c.Post(ctx, usersPath, request, &created)
	return created, err
}

// User reads a single user.
func (c *Client) User(ctx context.Context, id int64) (User, error) {
	var user User
	err := c.Get(ctx, fmt.Sprintf("%s%d/", usersPath, id), nil, &user)
	return user, err
}

// UpdateUser patches a user with the full managed field set.
func (c *Client) UpdateUser(ctx context.Context, id int64, request UserRequest) (User, error) {
	var updated User
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", usersPath, id), request, &updated)
	return updated, err
}

// DeleteUser deletes a user. The API refuses to let a user delete themselves;
// deleting the identity the provider authenticates as fails accordingly.
func (c *Client) DeleteUser(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", usersPath, id), nil)
}

// UserByUsername resolves a user by their exact, unique username.
func (c *Client) UserByUsername(ctx context.Context, username string) (User, error) {
	return FindByExactName[User](ctx, c, "user", usersPath, username, nil)
}
