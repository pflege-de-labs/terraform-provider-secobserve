# List resources (`terraform query`)

Status: accepted, 2026-11-02.

## Context

Terraform 1.14 added `terraform query`: `list` blocks in `*.tfquery.hcl`
files ask a provider to enumerate existing objects, and
`terraform query -generate-config-out=generated.tf` turns the results into
`import` blocks plus resource configuration. This is the bulk path for
bringing an existing SecObserve instance under Terraform.

In terraform-plugin-framework (v1.19) a list result must carry a **resource
identity**. Identity is a separate schema on the managed resource; once a
resource declares one, the framework errors unless Create, Read and Update
all return it ("Missing Resource Identity After …"). Generated `import`
blocks use `identity = { … }`, so `ImportState` receives an empty `req.ID`.

## Decisions

- **Identity is `{ id = <int64> }` for every listed resource.** Every
  resource here is keyed by SecObserve's numeric primary key already; names
  are not reliably unique (general rules, see `client.GeneralRuleByName`).
- **`ImportState` tries identity first, then the existing string import.**
  Name/composite string imports keep working unchanged.
- **The list resource lives on the managed resource's struct** (one `list.go`
  per package, sharing `Metadata` and `Configure`), registered through a
  second constructor `NewList`. No parallel package tree.
- **List results are built with the resource's existing `fromAPI`.** No
  list-only conversion path, so generated config and post-import state agree.
- **Listed:** `product_group`, `product`, `user`, `authorization_group`,
  `general_rule`, `license_group`, `license_policy` (no filter);
  `branch`, `service`, `api_configuration`, `product_rule` (required
  `product` filter, the same id the resource's `product` attribute takes).
- **Not listed:** `settings` (singleton, import with any id),
  `product_api_token` (secret unrecoverable — a generated config could only
  ever plan a replacement), the seven `*_member` resources and
  `license_policy_item` (import by lookup already works; revisit on demand).

## Accepted gap: product security gate and branch housekeeping

`security_gate` and `repository_branch_housekeeping` are never read from the
API (see the plan-modifier note in [api-quirks.md](api-quirks.md)). Reusing
`fromAPI` means generated config **omits these blocks even when the instance
has them configured**. An omitted block means "inherit"; the next update of
that product sends `security_gate_active: null` and resets a live gate.

This is today's `terraform import` behaviour, now reachable in bulk. Filling
the blocks from the API was rejected: instance defaults include thresholds of
`0`, which `ValidateSecurityGate` rejects at plan time, so most generated
configs would not validate. Mitigation: the list resources' descriptions and
docs tell users to add these blocks by hand before the first apply.

Related: sensitive attributes (`api_key`, `basic_auth_password`,
`issue_tracker_api_key`) are written by Terraform as `null # sensitive` in
generated config and must be filled in before apply.

## Compatibility

Identity requires Terraform ≥ 1.12 on the wire and `terraform query`
≥ 1.14. The provider's floor stays 1.11: identity schemas are ignored by
older clients, and the 1.11 CI leg running the existing suite is the
regression check.
