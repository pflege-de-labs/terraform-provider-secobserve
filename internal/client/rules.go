package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// VEXRemediation is one entry of a rule's new_vex_remediations.
type VEXRemediation struct {
	Category string `json:"category"`
	Text     string `json:"text"`
}

// RuleFields is the field set shared by general rules and product rules.
// Product rules are exactly this plus Product; general rules are exactly
// this alone -- GeneralRuleSerializer excludes "product" entirely
// (rules/api/serializers.py:33), so it is never part of this block.
type RuleFields struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Type controls whether the rule matches on the Field* attributes below
	// or on RegoModule.
	Type   string `json:"type"`
	Parser *int64 `json:"parser"`

	ScannerPrefix                     string `json:"scanner_prefix"`
	Title                             string `json:"title"`
	DescriptionObservation            string `json:"description_observation"`
	OriginComponentNameVersion        string `json:"origin_component_name_version"`
	OriginComponentPurl               string `json:"origin_component_purl"`
	OriginDockerImageNameTag          string `json:"origin_docker_image_name_tag"`
	OriginEndpointURL                 string `json:"origin_endpoint_url"`
	OriginServiceName                 string `json:"origin_service_name"`
	OriginSourceFile                  string `json:"origin_source_file"`
	OriginCloudQualifiedResource      string `json:"origin_cloud_qualified_resource"`
	OriginKubernetesQualifiedResource string `json:"origin_kubernetes_qualified_resource"`

	NewSeverity         string `json:"new_severity"`
	NewStatus           string `json:"new_status"`
	NewVEXJustification string `json:"new_vex_justification"`
	// NewVEXRemediations: an empty (or effectively-empty, e.g. every item
	// missing both category and text) list is normalized to null server-side
	// (commons/services/functions.py validate_vex_remediations:52-86).
	NewVEXRemediations []VEXRemediation `json:"new_vex_remediations"`

	// RegoModule is required when Type is RuleTypeRego, and otherwise unused.
	RegoModule string `json:"rego_module"`

	Enabled bool `json:"enabled"`
}

// Rule is the read representation of a general rule or a product rule.
//
// The approval fields are entirely server-owned: any create or update resets
// ApprovalStatus and reassigns User (rules/models.py Rule.save():75-101), and
// approving a rule is a separate call the identity that created it can never
// make (rules/services/approval.py:16-17 rejects self-approval). None of
// these are ever sent in a request.
type Rule struct {
	ID int64 `json:"id"`
	// Product is absent from the JSON entirely for a general rule, not null.
	Product *int64 `json:"product,omitempty"`

	RuleFields

	User                 string  `json:"user"`
	UserFullName         *string `json:"user_full_name"`
	ApprovalStatus       string  `json:"approval_status"`
	RejectionRemark      string  `json:"rejection_remark"`
	ApprovalDate         *string `json:"approval_date"`
	ApprovalUser         *string `json:"approval_user"`
	ApprovalUserFullName *string `json:"approval_user_full_name"`
}

// GetID implements Named.
func (r Rule) GetID() int64 { return r.ID }

// GetName implements Named.
func (r Rule) GetName() string { return r.Name }

const (
	generalRulesPath = "api/general_rules/"
	productRulesPath = "api/product_rules/"
)

// CreateGeneralRule creates a general rule. Superuser-only.
func (c *Client) CreateGeneralRule(ctx context.Context, request RuleFields) (Rule, error) {
	var created Rule
	err := c.Post(ctx, generalRulesPath, request, &created)
	return created, err
}

// GeneralRule reads a single general rule.
func (c *Client) GeneralRule(ctx context.Context, id int64) (Rule, error) {
	var rule Rule
	err := c.Get(ctx, fmt.Sprintf("%s%d/", generalRulesPath, id), nil, &rule)
	return rule, err
}

// UpdateGeneralRule patches a general rule. This resets its approval status
// server-side, regardless of what actually changed.
func (c *Client) UpdateGeneralRule(ctx context.Context, id int64, request RuleFields) (Rule, error) {
	var updated Rule
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", generalRulesPath, id), request, &updated)
	return updated, err
}

// DeleteGeneralRule deletes a general rule.
func (c *Client) DeleteGeneralRule(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", generalRulesPath, id), nil)
}

// GeneralRuleByName resolves a general rule by name.
//
// General rule names are not reliably unique: unique_together("product",
// "name") never fires because product is always NULL for a general rule
// (NULLs compare distinct in PostgreSQL), and GeneralRuleSerializer excludes
// "product" so DRF's own uniqueness validator is never even attached
// (rules/models.py:66-70, rules/api/serializers.py:33). FindByExactName
// reports ambiguity rather than guessing when this happens.
func (c *Client) GeneralRuleByName(ctx context.Context, name string) (Rule, error) {
	return FindByExactName[Rule](ctx, c, "general rule", generalRulesPath, name, nil)
}

// CreateProductRule creates a product rule.
func (c *Client) CreateProductRule(ctx context.Context, product int64, request RuleFields) (Rule, error) {
	var created Rule
	err := c.Post(ctx, productRulesPath, productRuleRequest{Product: product, RuleFields: request}, &created)
	return created, err
}

// ProductRule reads a single product rule.
func (c *Client) ProductRule(ctx context.Context, id int64) (Rule, error) {
	var rule Rule
	err := c.Get(ctx, fmt.Sprintf("%s%d/", productRulesPath, id), nil, &rule)
	return rule, err
}

// UpdateProductRule patches a product rule with the full managed field set,
// product included: SecObserve rejects a changed product with a 400, but the
// permission layer reads "product" out of the body before validation even
// runs (authorization/api/permissions_base.py:11-24), so it must always be
// present.
func (c *Client) UpdateProductRule(ctx context.Context, id, product int64, request RuleFields) (Rule, error) {
	var updated Rule
	err := c.Patch(ctx, fmt.Sprintf("%s%d/", productRulesPath, id),
		productRuleRequest{Product: product, RuleFields: request}, &updated)
	return updated, err
}

// DeleteProductRule deletes a product rule.
func (c *Client) DeleteProductRule(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("%s%d/", productRulesPath, id), nil)
}

// ProductRuleByName resolves a product rule by name within one product.
// Unlike general rules, product rule names are genuinely unique per product:
// unique_together("product", "name") is enforced, and ProductRuleSerializer
// uses fields="__all__" so DRF's uniqueness validator is attached.
func (c *Client) ProductRuleByName(ctx context.Context, productID int64, name string) (Rule, error) {
	return FindByExactName[Rule](ctx, c, "product rule", productRulesPath, name,
		url.Values{"product": {strconv.FormatInt(productID, 10)}})
}

// productRuleRequest adds the product field on top of RuleFields for the
// product_rules endpoint, which -- unlike general_rules -- includes it.
type productRuleRequest struct {
	Product int64 `json:"product"`
	RuleFields
}
