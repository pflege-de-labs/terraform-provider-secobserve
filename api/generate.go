// Package api holds the vendored SecObserve OpenAPI schema and the generated
// client models.
//
// The schema is not checked in by hand: run `make up` to start the pinned
// SecObserve release, then `make schema` to refetch it from
// /api/oa3/schema/?format=json. CI fails when the committed schema and the
// generated code disagree, so bumping the target SecObserve version is always
// a reviewable diff.
//
// Only models are generated. The transport layer is hand-written in
// internal/client, because the generated client cannot express SecObserve's
// APIToken scheme, page-number pagination or the delete-confirmation query
// parameters.
package api

//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.json
