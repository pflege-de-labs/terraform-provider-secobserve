package client

import (
	"context"
	"strings"
)

// UnknownVersion is what /api/status/version/ reports when SecObserve runs
// from source instead of a released image: the version string is substituted
// into views.py by the Dockerfile at build time
// (docker/backend/prod/django/Dockerfile), so a source checkout returns the
// untouched placeholder. Treated as "cannot tell", never as a mismatch.
const UnknownVersion = "version_unknown"

// Version is the response of GET /api/status/version/.
type Version struct {
	Version string `json:"version"`
}

// Version reports the SecObserve release of the target instance.
func (c *Client) Version(ctx context.Context) (Version, error) {
	var version Version
	err := c.Get(ctx, "api/status/version/", nil, &version)
	return version, err
}

// CurrentUser is the subset of GET /api/users/me/ the provider needs.
type CurrentUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	FullName    string `json:"full_name"`
	IsSuperuser bool   `json:"is_superuser"`
	IsExternal  bool   `json:"is_external"`
	IsOIDCUser  bool   `json:"is_oidc_user"`
}

// Me identifies the token's owner. Used at configure time to verify the token
// and to check the superuser requirement.
func (c *Client) Me(ctx context.Context) (CurrentUser, error) {
	var user CurrentUser
	err := c.Get(ctx, "api/users/me/", nil, &user)
	return user, err
}

// MajorMinor reduces a version string to "major.minor" for compatibility
// comparison; patch releases never change the API surface.
func MajorMinor(version string) string {
	parts := strings.SplitN(strings.TrimPrefix(version, "v"), ".", 3)
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1]
}
