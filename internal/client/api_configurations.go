package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// APIConfigurationRequest is the write representation of an API import
// configuration.
type APIConfigurationRequest struct {
	// Product is immutable after create; SecObserve rejects a change with a
	// 400 ("Product cannot be changed"), and the permission layer reads it
	// out of the body before validation even runs, so it must always be
	// present.
	Product int64  `json:"product"`
	Name    string `json:"name"`
	Parser  int64  `json:"parser"`
	BaseURL string `json:"base_url"`

	ProjectKey string `json:"project_key"`
	// APIKey is stripped from responses unless the caller holds
	// Api_Configuration_Edit; since the provider requires a superuser it
	// round-trips in practice.
	APIKey string `json:"api_key"`
	Query  string `json:"query"`

	BasicAuthEnabled  bool   `json:"basic_auth_enabled"`
	BasicAuthUsername string `json:"basic_auth_username"`
	// BasicAuthPassword is encrypted at rest but, unlike APIKey, is never
	// stripped from responses.
	BasicAuthPassword string `json:"basic_auth_password"`
	VerifySSL         bool   `json:"verify_ssl"`

	AutomaticImportEnabled bool `json:"automatic_import_enabled"`
	// AutomaticImportBranch must belong to Product; SecObserve validates this
	// server-side and answers with a 400 if it does not.
	AutomaticImportBranch             *int64 `json:"automatic_import_branch"`
	AutomaticImportService            *int64 `json:"automatic_import_service"`
	AutomaticImportDockerImageNameTag string `json:"automatic_import_docker_image_name_tag"`
	AutomaticImportEndpointURL        string `json:"automatic_import_endpoint_url"`
	AutomaticImportKubernetesCluster  string `json:"automatic_import_kubernetes_cluster"`

	// TestConnection is a write-only, side-effecting flag: when true,
	// SecObserve performs a live connection test using the rest of this
	// payload and fails the whole request if it does not succeed. It is
	// never present in a response.
	TestConnection bool `json:"test_connection,omitempty"`
}

// APIConfiguration is the read representation of an API import configuration.
type APIConfiguration struct {
	ID      int64  `json:"id"`
	Product int64  `json:"product"`
	Name    string `json:"name"`
	Parser  int64  `json:"parser"`
	BaseURL string `json:"base_url"`

	ProjectKey string `json:"project_key"`
	// APIKey is absent from the JSON entirely (not null, not "") for a
	// caller without Api_Configuration_Edit -- which never applies to this
	// provider's required superuser token.
	APIKey string `json:"api_key"`
	Query  string `json:"query"`

	BasicAuthEnabled  bool   `json:"basic_auth_enabled"`
	BasicAuthUsername string `json:"basic_auth_username"`
	BasicAuthPassword string `json:"basic_auth_password"`
	VerifySSL         bool   `json:"verify_ssl"`

	AutomaticImportEnabled            bool   `json:"automatic_import_enabled"`
	AutomaticImportBranch             *int64 `json:"automatic_import_branch"`
	AutomaticImportService            *int64 `json:"automatic_import_service"`
	AutomaticImportDockerImageNameTag string `json:"automatic_import_docker_image_name_tag"`
	AutomaticImportEndpointURL        string `json:"automatic_import_endpoint_url"`
	AutomaticImportKubernetesCluster  string `json:"automatic_import_kubernetes_cluster"`
}

// GetID implements Named.
func (a APIConfiguration) GetID() int64 { return a.ID }

// GetName implements Named.
func (a APIConfiguration) GetName() string { return a.Name }

const apiConfigurationsPath = "api/api_configurations/"

// CreateAPIConfiguration creates an API import configuration. If
// request.TestConnection is true, SecObserve performs a live connection
// check as part of this call and fails the request if it does not succeed.
func (c *Client) CreateAPIConfiguration(ctx context.Context, request APIConfigurationRequest) (APIConfiguration, error) {
	var created APIConfiguration
	err := c.Post(ctx, apiConfigurationsPath, request, &created)
	return created, err
}

// APIConfiguration reads a single API import configuration. The response
// never carries api_key for a caller without Api_Configuration_Edit; for
// this provider's required superuser token it always does.
func (c *Client) APIConfiguration(ctx context.Context, id int64) (APIConfiguration, error) {
	var config APIConfiguration
	err := c.Get(ctx, fmt.Sprintf("%s%d/", apiConfigurationsPath, id), nil, &config)
	return config, err
}

// UpdateAPIConfiguration patches an API import configuration.
func (c *Client) UpdateAPIConfiguration(ctx context.Context, id int64, request APIConfigurationRequest) (APIConfiguration, error) {
	var updated APIConfiguration
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", apiConfigurationsPath, id), request, &updated)
	return updated, err
}

// DeleteAPIConfiguration deletes an API import configuration.
func (c *Client) DeleteAPIConfiguration(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", apiConfigurationsPath, id), nil)
}

// APIConfigurationByName resolves an API import configuration by name within
// one product. The server's own name filter matches exactly here (there is
// no icontains override in ApiConfigurationFilter), but the exact match is
// still applied client-side for consistency with every other name lookup in
// this provider.
func (c *Client) APIConfigurationByName(ctx context.Context, productID int64, name string) (APIConfiguration, error) {
	return FindByExactName[APIConfiguration](ctx, c, "API configuration", apiConfigurationsPath, name,
		url.Values{"product": {strconv.FormatInt(productID, 10)}})
}
