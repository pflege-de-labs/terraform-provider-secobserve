// Package common holds attribute blocks shared by the data sources.
//
// These are the read-only, runtime parts of a product that the resources
// deliberately leave out: carrying observation counts in resource state would
// churn on every scan and invite their use as configuration inputs.
package common

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// ObservationCounts is the per-severity count of active observations.
type ObservationCounts struct {
	ActiveCriticalObservationCount types.Int64 `tfsdk:"active_critical_observation_count"`
	ActiveHighObservationCount     types.Int64 `tfsdk:"active_high_observation_count"`
	ActiveMediumObservationCount   types.Int64 `tfsdk:"active_medium_observation_count"`
	ActiveLowObservationCount      types.Int64 `tfsdk:"active_low_observation_count"`
	ActiveNoneObservationCount     types.Int64 `tfsdk:"active_none_observation_count"`
	ActiveUnknownObservationCount  types.Int64 `tfsdk:"active_unknown_observation_count"`
}

// AddObservationCounts contributes the observation count attributes.
func AddObservationCounts(attributes map[string]schema.Attribute) {
	for attribute, severity := range map[string]string{
		"active_critical_observation_count": "critical severity",
		"active_high_observation_count":     "high severity",
		"active_medium_observation_count":   "medium severity",
		"active_low_observation_count":      "low severity",
		"active_none_observation_count":     "`None` severity",
		"active_unknown_observation_count":  "unknown severity",
	} {
		attributes[attribute] = schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Number of active observations of " + severity + ".",
		}
	}
}

// FromAPI fills the block from an API response.
func (o *ObservationCounts) FromAPI(product client.Product) {
	o.ActiveCriticalObservationCount = tfutil.Int64(product.ActiveCriticalObservationCount)
	o.ActiveHighObservationCount = tfutil.Int64(product.ActiveHighObservationCount)
	o.ActiveMediumObservationCount = tfutil.Int64(product.ActiveMediumObservationCount)
	o.ActiveLowObservationCount = tfutil.Int64(product.ActiveLowObservationCount)
	o.ActiveNoneObservationCount = tfutil.Int64(product.ActiveNoneObservationCount)
	o.ActiveUnknownObservationCount = tfutil.Int64(product.ActiveUnknownObservationCount)
}

// ContentFlags describes which kinds of findings a product has. SecObserve
// maintains them itself as observations are imported.
type ContentFlags struct {
	HasCloudResource       types.Bool `tfsdk:"has_cloud_resource"`
	HasComponent           types.Bool `tfsdk:"has_component"`
	HasDockerImage         types.Bool `tfsdk:"has_docker_image"`
	HasEndpoint            types.Bool `tfsdk:"has_endpoint"`
	HasKubernetesResource  types.Bool `tfsdk:"has_kubernetes_resource"`
	HasSource              types.Bool `tfsdk:"has_source"`
	HasPotentialDuplicates types.Bool `tfsdk:"has_potential_duplicates"`
}

// AddContentFlags contributes the content flag attributes.
func AddContentFlags(attributes map[string]schema.Attribute) {
	for attribute, subject := range map[string]string{
		"has_cloud_resource":       "cloud resources",
		"has_component":            "components",
		"has_docker_image":         "container images",
		"has_endpoint":             "endpoints",
		"has_kubernetes_resource":  "Kubernetes resources",
		"has_source":               "source files",
		"has_potential_duplicates": "potential duplicate observations",
	} {
		attributes[attribute] = schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the product has observations concerning " + subject + ".",
		}
	}
}

// FromAPI fills the block from an API response.
func (c *ContentFlags) FromAPI(product client.Product) {
	c.HasCloudResource = types.BoolValue(product.HasCloudResource)
	c.HasComponent = types.BoolValue(product.HasComponent)
	c.HasDockerImage = types.BoolValue(product.HasDockerImage)
	c.HasEndpoint = types.BoolValue(product.HasEndpoint)
	c.HasKubernetesResource = types.BoolValue(product.HasKubernetesResource)
	c.HasSource = types.BoolValue(product.HasSource)
	c.HasPotentialDuplicates = types.BoolValue(product.HasPotentialDuplicates)
}
