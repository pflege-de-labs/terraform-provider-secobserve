package client

import (
	"context"
	"net/url"
)

// Parser is a vulnerability scanner parser. Parsers are system-managed: they
// are registered from the on-disk parser packages by the register_parsers
// management command at startup and cannot be created through the API, so the
// provider only ever reads them.
type Parser struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Source     string `json:"source"`
	SBOM       bool   `json:"sbom"`
	ModuleName string `json:"module_name"`
	ClassName  string `json:"class_name"`
}

// GetID implements Named.
func (p Parser) GetID() int64 { return p.ID }

// GetName implements Named.
func (p Parser) GetName() string { return p.Name }

// Parser resolves a parser by its unique name.
func (c *Client) Parser(ctx context.Context, name string) (Parser, error) {
	return FindByExactName[Parser](ctx, c, "parser", "api/parsers/", name, nil)
}

// Parsers lists parsers, optionally filtered by type and source.
func (c *Client) Parsers(ctx context.Context, parserType, source string) ([]Parser, error) {
	query := url.Values{}
	if parserType != "" {
		query.Set("type", parserType)
	}
	if source != "" {
		query.Set("source", source)
	}
	return List[Parser](ctx, c, "api/parsers/", query)
}
