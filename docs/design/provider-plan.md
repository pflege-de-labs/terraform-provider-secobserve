# Terraform Provider for SecObserve — implementation plan

Living design document: what this provider covers, the SecObserve API behaviours
that shape it, and the phased delivery plan. `api-quirks.md` is the reference for
the upstream behaviours themselves; this file is the plan built on top of them.

## Context

SecObserve (open-source vulnerability & license management, [SecObserve/SecObserve](https://github.com/SecObserve/SecObserve), pinned to the current release **v1.58.0**, published 2026-08-28) has no Terraform provider. Today, onboarding a product into SecObserve is manual clicking or ad-hoc scripting against its REST API: create product group, product, branches, services, members, authorization groups, API-import configurations, rules, license policies. That configuration is long-lived, environment-specific, and needs to be reproducible across instances (dev/prod) — exactly what Terraform is for.

Goal: a `terraform-provider-secobserve` that manages SecObserve **configuration** (declarative admin state) as Terraform resources. Explicitly out of scope: runtime data (observations, license components, scan results, metrics, VEX documents) and imperative operations (import observations, run scans, apply rules, exports). Those stay in CI/CD pipelines where they belong.

This repo is empty — greenfield, everything below is new.

### Decisions

1. **SDK: `terraform-plugin-framework`.** Non-negotiable in practice — SecObserve makes heavy use of tri-state booleans where `null` means "inherit from product group / global settings" (`security_gate_active`, `repository_branch_housekeeping_active`, `risk_acceptance_expiry_active`). SDKv2 cannot distinguish `null` from `false`, which would silently break inheritance.
2. **Client: hybrid.** Vendor the OpenAPI schema fetched from the pinned SecObserve release, generate types + low-level client with `oapi-codegen`, hand-write the transport layer (auth, pagination, error mapping, exact-name lookup helpers).
3. **Target version: the current SecObserve release**, currently v1.58.0. The vendored schema is refreshed per release (see *Version tracking* below), not per commit on `dev`.
4. **Auth: user API token of a superuser** (`Authorization: APIToken <token>`) — confirmed acceptable.
5. **Identity providers: both OIDC and local users supported.** OIDC changes what Terraform may safely own; see *OIDC caveats*.
6. **Dev & test: containerized SecObserve** plus a devcontainer for the provider itself.
7. **Publishing: deferred.** v1 wires up `goreleaser` + `tfplugindocs` so a release is possible, but the registry namespace decision is left open; local dev via `dev_overrides`.

### Version tracking

The vendored `api/openapi.json` is the contract. Bumping SecObserve:

1. Start the containerized instance at the new tag (`SECOBSERVE_VERSION` in `test/.env`).
2. `make schema` refetches `/api/oa3/schema/?format=json` and `make generate` regenerates the client.
3. `git diff api/openapi.json` is the upstream change report; CI fails on an uncommitted diff so a version bump is always a reviewable commit.
4. The provider records the schema's version string; `Configure` compares it with `GET /api/status/version/` and emits a warning on a minor/major mismatch (not an error — the provider should keep working against nearby versions).

## API facts that drive the design

Verified against the SecObserve source (`backend/application/**`). These are the non-obvious constraints — the design exists to absorb them.

### Auth & transport
- Single token scheme for both user and product tokens: `Authorization: APIToken <token>` (`access_control/services/api_token_authentication.py:11-30`). JWT (`JWT <token>`) expires and needs a password each renewal — unusable for a provider.
- **The provider identity must be a superuser.** Non-superusers get filtered list responses (users, authorization groups, notifications) and are blocked from writing `users`, `general_rules`, `settings`, `periodic_tasks`. A non-superuser token produces truncated reads → permanent spurious drift.
- Token minting (`POST /api/authentication/create_user_api_token/`) takes username+password in the body, **not** a token — it cannot be driven by the provider's own credentials. Tokens are created out-of-band.
- Pagination: page-number, `PAGE_SIZE=25`, `?page_size=` override (`config/settings/base.py:389`). Throttle: 100 req/s per user.
- Relations are `PrimaryKeyRelatedField` everywhere — **numeric IDs only**, no name-based writes.
- List filters on `name` are `icontains`, not exact (`core/api/filters.py:47-80`) — every name→ID lookup must filter client-side for exact match and error on ambiguity.
- Errors: `409 Conflict` on `ProtectedError`/`RestrictedError` (`notifications/api/exception_handler.py:28-33`).

### OIDC caveats

Both identity modes are supported, but OIDC (`OIDC_AUTHORITY` set on the backend) makes SecObserve itself an authority over parts of the user model. Verified in `access_control/services/oidc_authentication.py`:

- **`_synchronize_groups` (`:194-204`) removes the user from *every* `Authorization_Group` with a non-empty `oidc_group`, then re-adds only the groups matching the token's group claim.** This runs on first login and on every login where the groups hash changed (`:156-158`). Consequence: a `secobserve_authorization_group_member` pointing at a group with `oidc_group != ""` is **deleted at the member's next login**. Terraform re-adds it, the next login removes it — an unwinnable loop.
  → The provider emits a **warning diagnostic on create/update of `secobserve_authorization_group_member`** when the referenced group has a non-empty `oidc_group`, naming the group and telling the user to manage membership via the IdP instead. Groups with `oidc_group == ""` are untouched by sync and are safe for Terraform-managed membership.
- **OIDC users are created just-in-time on first login** (`_create_user`, `:104-138`). Terraform cannot usefully pre-create them: `_check_user_change` (`:141-168`) overwrites `email`, `full_name`, `first_name`, `last_name` from claims on every login, forces `is_oidc_user = true`, and calls `set_unusable_password()`.
  → For a user whose API state has `is_oidc_user == true`, the provider emits a warning if the config sets any of `email`, `full_name`, `first_name`, `last_name` — those are claim-owned and will drift back. `is_active`, `is_superuser` and `is_external` are *not* touched by `_check_user_change` and remain safely Terraform-managed.
- `is_external` is computed from the `Settings.internal_users` email regex list **only at create time** (`:118-126`), never re-evaluated. So it is legitimately manageable, and `secobserve_settings.internal_users` only affects users created after the change.
- `Authorization_Group.oidc_group` itself is ordinary configuration — it is the IdP-group→SecObserve-group mapping and is exactly the kind of thing Terraform should own.
- The mixed case works: local users can be created and fully managed by Terraform on an OIDC-enabled instance (they authenticate with a password or an API token; only `OIDC_ENABLE` on the *frontend* hides the login form, and `feature_disable_user_login` in Settings can disable it server-side).

Design consequence: `secobserve_user` ships alongside a **`secobserve_user` data source** so OIDC-provisioned users can be *referenced* by username (for `product_member`, `assessment_approvers`, …) without Terraform claiming ownership of them. The docs lead with the data source for OIDC deployments and with the resource for local-user deployments.

### Server-side mutation of submitted values (the main drift hazard)
| What | Behavior | Handling |
|---|---|---|
| `security_gate_threshold_*` (6 fields, product & product group) | Auto-filled from global Settings when `security_gate_active=true` and omitted; **forced to null** when `false` (`serializers_product.py:122-142`) | `Optional + Computed`, take server value from the create/update response |
| `security_gate_threshold_*` set to **`0`** | The fill-in test is `if not attrs.get(...)` — **falsy, not absent**. A submitted `0` counts as unset and is replaced by the instance default, which is `99999` for medium/low/none/unknown. A threshold of 0 is therefore unsettable for those four while the gate is active. | State follows the response so it never drifts; the provider emits a warning naming the requested and stored value (`tfutil.WarnInt64Override`) |
| `repository_branch_housekeeping_keep_inactive_days` / `_exempt_branches` | Same auto-fill / force-clear pattern (`:112-120`) | same |
| `propagate_branches` | An empty or filtered-empty list is normalized to `null`, not `[]` (`:660-663`) | `Optional + Computed`; normalize client-side before compare |
| `license_policy_item.license_expression` | Re-normalized via `spdx_licensing.validate(strict=True)` (`licenses/api/serializers.py:591-600`) — stored value can differ from input | `Optional + Computed` |
| `license_policy.ignore_component_types` vs `ignore_component_type_list` | Dual representation of the same column; the list is always injected into responses (`serializers.py:482-499`) | Expose **only** `ignore_component_type_list`; never map the comma-joined string |
| `product.observation_notification_status_list` | Serializer-only list alias for the comma-joined `observation_notification_statuses` column | Expose the list |
| Rule `approval_status`, `user`, `approval_*`, `rejection_remark` | All server-set; **any** PUT/PATCH resets approval and reassigns `user` (`rules/api/serializers.py:62-64,112-114`) | `Computed` only |

Rule for the whole provider: **always write the resource state from the API response body, never from the plan.**

### Never-readable / write-only values
- `product.issue_tracker_api_key` — accepted on write, stripped from responses without `Product_Edit` (`serializers_product.py:252-254`). Since the provider requires a superuser, it *does* round-trip in practice.
- `api_configuration.api_key` — same pattern (`import_observations/api/serializers.py:111-119`).
- `product_api_token` secret — returned **once** in the 201 body, never readable again.
- `api_configuration.test_connection` — write-only side-effecting flag, never returned.

**Revised from the original plan:** the two API keys are `Optional + Sensitive` (stored in state), *not* write-only. SecObserve enforces "either all or none of the issue tracker fields must be set" over `issue_tracker_{type,base_url,api_key,project_id}` (`serializers_product.py:565-574`). A write-only attribute is absent from state, so on a later update the provider could neither resend the key nor omit it: sending `""` fails the check, and omitting it also fails, because the check reads `attrs.get("issue_tracker_api_key")` and a missing key is falsy while the other three are set. Storing it sensitively is the only option that keeps updates working, and it buys real drift detection. `test_connection` stays genuinely write-only — it is ephemeral and participates in no such check.

### Immutable fields → `RequiresReplace`
`branch.product`, `service.product`, `product_member.{product,user}`, `product_authorization_group_member.{product,authorization_group}`, `authorization_group_member.{authorization_group,user}`, `product_rule.product`, `api_configuration.product`, `license_group_member.{license_group,user}`, `license_policy_member.{license_policy,user}`, and the two authorization-group-member equivalents. Several are enforced server-side with a 400 ("Product cannot be changed"), so without the plan modifier `apply` just fails.

### Nonstandard request requirements
- `DELETE /api/products/{id}/` and `/api/product_groups/{id}/` require `?name=<exact name>` as confirmation — case- and whitespace-sensitive, exactly one `name` param (`views_product.py:139-164`).
- `POST` to `branches`, `services`, `product_members`, `product_authorization_group_members` must carry a top-level `product` key or the **permission layer** rejects with a `ParseError` before validation (`authorization/api/permissions_base.py:11-24`).
- `GET /api/product_api_tokens/` requires `?product=<int>` and returns an unpaginated bare `{"results": [...]}`.
- `product_api_tokens` has no retrieve and no update: list / create / delete only, `name` max length **32**.
- Branch with `is_default_branch=true` **cannot be deleted** (400). `product.repository_default_branch` is read-only and set indirectly by flipping the flag on a branch.
- Creating a product fires a `post_save` signal adding the caller as `Product_Member` with role `Owner` (`core/signals.py:73-76`) — an unmanaged membership appears immediately after every product create.
- `products` and `product_groups` use a **different serializer for `list`** than for detail; `assessment_approvers` and `assessment_approver_authorization_groups` are absent from list responses. Reads must hit the detail endpoint.
- `Settings` is a singleton: `GET`/`PATCH /api/settings/1/` only (pk is parsed and ignored), superuser-only, no POST/PUT/DELETE (`commons/api/views.py:90-115`). Setting `feature_automatic_osv_scanning=true` force-enables `feature_license_management`.
- `license_groups`/`license_policies` expose `POST {id}/copy/` returning 201 + the new object — the intended way to derive an editable copy of the system-managed ScanCode/`Standard` objects, which are otherwise **overwritten nightly** by a periodic task.
- Rule approval: `PATCH {id}/approval/` is a separate call, only valid from state `Needs approval`, and **self-approval is rejected** (`rules/services/approval.py:16-17`). The provider can never approve its own rules — approval is inherently out-of-band.

### Enums to codify as validators
`Roles` 1–5 (`Reader=1, Upload=2, Writer=3, Maintainer=4, Owner=5`); `Severity` (`Unknown, None, Low, Medium, High, Critical`); `Status` (9 values, note embedded spaces: `False positive`, `Not affected`, …); `Issue_Tracker` (`GitHub, GitLab, Jira`); `OSVLinuxDistribution` (12 values); rule `type` (`Fields, Rego`); `evaluation_result` (`Allowed, Forbidden, Ignored, Review required, Unknown`); VEX justifications (14 values); PURL types (41 values). Sources: `core/types.py`, `issue_tracker/types.py`, `rules/types.py`, `licenses/types.py`, `authorization/services/roles_permissions.py`.

Cross-field server validation to mirror in `ValidateConfig` so errors surface at plan time:
- `issue_tracker_active` requires `issue_tracker_type`; `issue_tracker_{username,issue_type,status_closed}` required iff type is `Jira`, must be empty otherwise.
- `osv_linux_release` cannot be set without `osv_linux_distribution`.
- `rego_module` required iff rule `type == "Rego"`.
- `license_policy_item`: **exactly one** of `license_group` / `license` / `license_expression` / `non_spdx_license`.
- `license_policy.parent` depth limited to 1.
- `product.assessment_approver_authorization_groups`: each group must already be a `product_authorization_group_member` with role ≥ `Writer` (3) — a dependency ordering constraint users will hit.

## Repository layout

```
terraform-provider-secobserve/
  main.go                          # provider server entrypoint
  go.mod
  Makefile                         # build, install-local, schema, generate, test, testacc, up, down, lint
  .golangci.yml
  .goreleaser.yml                  # release wiring (registry target deferred)
  .github/workflows/{test,release}.yml
  .devcontainer/
    devcontainer.json              # Go + Terraform + gh, docker-outside-of-docker
    docker-compose.yml             # dev container + the test/ stack on one network
    post-create.sh                 # go mod download, install tfplugindocs/oapi-codegen/golangci-lint
  test/
    docker-compose.yml             # postgres + secobserve-backend (published image)
    docker-compose.oidc.yml        # overlay: keycloak + realm import + OIDC env
    keycloak/realm-secobserve.json # vendored from upstream keycloak/realm-secobserve.json
    .env                           # SECOBSERVE_VERSION, admin creds, encryption keys
    bootstrap.sh                   # wait for health, mint superuser API token, print exports
  api/
    openapi.json                   # vendored schema, fetched from the pinned SecObserve release
    generate.go                    # //go:generate oapi-codegen directives
  internal/
    client/
      generated.go                 # oapi-codegen output (types + low-level client)
      client.go                    # transport: base URL, APIToken auth, retry, throttle backoff
      errors.go                    # DRF error -> diagnostics (400 field errors, 409, 403)
      paginate.go                  # page-number iterator
      lookup.go                    # exact-name resolution over icontains filters
    provider/
      provider.go                  # schema, Configure, resource/datasource registration
    tfutil/
      configure.go                 # provider -> resource/datasource client hand-off
      convert.go                   # Terraform <-> JSON conversion, explicit about null
      override.go                  # warn when the server stored a different value
    schemacommon/                  # schema+model blocks shared by product and product group
      security_gate.go  branch_housekeeping.go  notifications.go
      risk_acceptance.go  branch_propagation.go  plan_modifiers.go
    resource/
      product/  product_group/  branch/  service/
      product_member/  product_authorization_group_member/  product_api_token/
      user/  authorization_group/  authorization_group_member/  settings/
      general_rule/  product_rule/  api_configuration/
      license_group/  license_group_member/  license_group_authorization_group_member/
      license_policy/  license_policy_item/  license_policy_member/
      license_policy_authorization_group_member/  vex_counter/
    datasource/
      product/  product_group/  branch/  service/
      user/  authorization_group/  parser/  license/
      license_group/  license_policy/  periodic_tasks/
    validators/                    # SecObserve-specific validators (regex, http URL)
  examples/                        # tfplugindocs input
  docs/                            # tfplugindocs output
  templates/                       # doc templates
  CONTRIBUTING.md
  docs/design/api-quirks.md        # long-form notes behind the one-line code comments
```

## Provider schema

```hcl
provider "secobserve" {
  base_url  = "https://secobserve.example.com"  # env SECOBSERVE_BASE_URL
  api_token = var.token                          # env SECOBSERVE_API_TOKEN, sensitive
  insecure  = false                              # skip TLS verify, dev only
  timeout   = "30s"
}
```

`Configure` verifies the token and the superuser requirement with `GET /api/users/me/`, and emits a **warning diagnostic if `is_superuser` is false**, naming the resources that will misbehave. It also reads `GET /api/status/version/` and warns on a mismatch with the vendored schema version.

## Development & test environment

Everything runs against a containerized SecObserve — no shared instance is ever needed for development, and CI uses the same stack.

### `test/docker-compose.yml`

Deliberately **not** the upstream `docker-compose-prod-*.yml`: those add Traefik and the frontend, which a provider test does not need. The stack is `postgres:15.19-alpine` plus `ghcr.io/secobserve/secobserve-backend:${SECOBSERVE_VERSION}` (published image, no source build) with the backend's gunicorn port **8000** published directly. Environment derived from the upstream prod compose (`docker-compose-prod-postgres.yml:58-105`):

- `ADMIN_USER=admin`, `ADMIN_PASSWORD` set explicitly (if unset, SecObserve generates a random one and only logs it).
- `ALLOWED_HOSTS=localhost,127.0.0.1,backend` — the upstream default is `secobserve-backend.localhost` and would reject direct-port requests.
- `DATABASE_*` pointing at the postgres service; fixed `DJANGO_SECRET_KEY` and `FIELD_ENCRYPTION_KEY` (test-only values, committed, clearly marked).
- A healthcheck on `GET /api/status/health/` (public, no auth) so `bootstrap.sh` can wait deterministically instead of sleeping.

`test/bootstrap.sh` waits for health, then `POST /api/authentication/create_user_api_token/` with the admin username/password to mint the superuser token, and prints `SECOBSERVE_BASE_URL` / `SECOBSERVE_API_TOKEN` exports. `make up` runs compose + bootstrap; `make down` tears down including volumes so every acceptance run starts from a fresh database (the SPDX/ScanCode initial load runs on first start, which the license tests depend on).

### `test/docker-compose.oidc.yml`

Overlay adding `keycloak/keycloak:26.7.2` with `start-dev --import-realm` and the realm JSON vendored from upstream `keycloak/realm-secobserve.json`, plus the backend OIDC variables taken from upstream `docker-compose-dev-keycloak.yml:36-44`: `OIDC_AUTHORITY=http://keycloak:8080/realms/secobserve`, `OIDC_CLIENT_ID=secobserve`, `OIDC_USERNAME=preferred_username`, `OIDC_FIRST_NAME=given_name`, `OIDC_LAST_NAME=family_name`, `OIDC_EMAIL=email`, `OIDC_GROUPS=groups`.

Used only by the OIDC-caveat acceptance tests (build tag `oidc`, skipped by default). Those tests are what prove the warning diagnostics fire and that the group-sync wipe is real: obtain a Keycloak token for a realm user, call any authenticated endpoint to trigger JIT provisioning and group sync, then assert the Terraform-managed membership on an `oidc_group`-mapped authorization group was removed.

### `.devcontainer/`

`devcontainer.json` with the Go feature (matching `go.mod`), the Terraform feature (CLI for the manual smoke test), `github-cli`, and **docker-outside-of-docker** so the `test/` stack is reachable from inside the container. `post-create.sh` runs `go mod download` and installs `oapi-codegen`, `tfplugindocs` and `golangci-lint` at pinned versions. A `.devcontainer/docker-compose.yml` joins the dev container to the `test/` stack's network so `SECOBSERVE_BASE_URL=http://backend:8000` works without published ports, and mounts a `~/.terraformrc` with the `dev_overrides` block pointing at the container's `$GOPATH/bin` — so `terraform plan` in `examples/` uses the locally built provider with no `terraform init`.

## Resources — phased delivery

Each phase is independently useful and mergeable.

### Phase 0 — Skeleton & environment
Devcontainer, `test/` compose stack + `bootstrap.sh`, vendored schema, generated client, provider server, `Configure`, transport layer, error mapping, pagination, exact-name lookup, `Makefile`, `dev_overrides` instructions, CI running unit tests + lint + one acceptance test against the containerized stack. One trivial data source (`secobserve_parser`) to prove the plumbing end to end.

### Phase 1 — Core products (in progress, branch `feat/phase-1-core-products`)

`secobserve_product`, `secobserve_product_group`, `secobserve_branch`, `secobserve_service`, `secobserve_product_member`, `secobserve_product_authorization_group_member`, `secobserve_product_api_token`.
Data sources: `secobserve_product`, `secobserve_product_group`, `secobserve_branch`, `secobserve_service`.

#### Attribute pattern (the core of this phase)

Every write sends the **full managed field set**, with Go `nil` mapping to JSON `null`. Omitting fields would make behaviour depend on which attributes happen to be set: SecObserve reads an absent key as "leave unchanged", while its fill-and-clear logic for the security gate and branch housekeeping only fires when the matching `*_active` flag is present in the payload. Sending everything makes the outcome deterministic.

Which schema shape a field gets follows from its backend declaration:

| Backend declaration | Schema shape | Why |
|---|---|---|
| `CharField(blank=True)`, no `null=True` (most strings) | `Optional + Computed` + `stringdefault.StaticString("")` | These reject a JSON `null`. A static `""` default makes the planned value always known, so removing the attribute from config *clears* it and nothing drifts. |
| `BooleanField(default=X)` | `Optional + Computed` + `booldefault.StaticBool(X)` | Same reasoning; the default mirrors the server's. |
| `BooleanField(null=True)` — `security_gate_active`, `repository_branch_housekeeping_active`, `risk_acceptance_expiry_active` | `Optional` only, no default | Genuine tri-state: `null` means *inherit from product group or instance settings* and must stay distinguishable from `false`. This is the reason for choosing the framework over SDKv2. |
| Server fills or clears it — the 6 thresholds, `repository_branch_housekeeping_keep_inactive_days`/`_exempt_branches`, `issue_tracker_base_url` | `Optional + Computed`, **no** default | The response is authoritative. On update the framework carries the prior state forward when config is null, which is what SecObserve expects. |
| Nullable ints with no server fill — `risk_acceptance_expiry_days`, `observation_notification_min_priority`, `license_policy`, `product_group` | `Optional` only | `null` is a real, storable value. |
| M2M and list attributes — `assessment_approvers`, `assessment_approver_authorization_groups`, `observation_notification_status_list` | `SetAttribute`, `Optional + Computed` + `setdefault.StaticValue(empty)` | The API returns `[]`, never null, and does not promise an order. A list would produce ordering diffs. |
| `propagate_branches` | `ListNestedAttribute`, `Optional` | Order is meaningful. An **empty** list is normalized to `null` server-side, so `ValidateConfig` rejects `[]` and tells the caller to omit the attribute — cheaper than explaining the resulting diff. |

Two rules hold everywhere: state is written from the **API response body**, never from the plan; and an immutable field carries `RequiresReplace` so a change becomes a replace instead of a 400 at apply time.

#### Shared schema blocks

The security gate (7 attributes), branch housekeeping (3), notifications (6), risk-acceptance expiry (2) and branch propagation (3) blocks are **identical** on products and product groups — 21 of the product group's 28 attributes. They live in a new `internal/schemacommon` package: one function per block that contributes its attributes to the schema map, paired with one struct per block that the resource models **embed by value**.

Verified against `terraform-plugin-framework@v1.19.0/internal/reflect/helpers.go:48-101`: value-embedded structs are flattened into the object and their promoted `tfsdk` fields addressed as if declared directly; the embedded field itself must carry no `tfsdk` tag, and duplicate promoted tags are an error. So the schema half and the model half of each block stay in one place and cannot drift apart.

Also in `internal/schemacommon`: `Int64UseStateForUnknown()`, `EmptyInt64Set()`, `EmptyStringSet()`.

#### Per-resource notes

- **`secobserve_product`** — the large one (~55 writable attributes). Schema grouped by concern: identity, repository and housekeeping, security gate, notifications, issue tracker, approvals, OSV and VulnerableCode, branch propagation, license policy. `ValidateConfig` mirrors the server's cross-field rules so they fail at plan time: the issue-tracker all-or-none rule, the Jira-only fields, `issue_tracker_active` requiring a type, and `osv_linux_release` requiring `osv_linux_distribution`.
- **Revised from the original plan:** the informational read-only fields — the six observation counters, the seven `has_*` flags, `products_count`, `security_gate_passed` — are **omitted from the resources entirely** and exposed on the **data sources** only. They are runtime data; carrying them in resource state adds churn and invites their use as inputs. The resources keep only the config-relevant computed attributes: `id`, `repository_default_branch` + `_name`, `product_group_name`.
- **`secobserve_product_group`** — same blocks, minus everything product-only (purl, cpe23, repository prefix, the whole issue tracker block, OSV, VulnerableCode, `apply_general_rules`). `is_product_group` is not exposed: the endpoint forces it to `true`.
- **`secobserve_branch`** — `is_default_branch` needs care. Setting it clears the flag on the product's other branches, which Terraform cannot see, so document that exactly one branch per product should set it. Delete of the default branch is refused (400); surface that as an actionable error telling the caller to move the flag first.
- **`secobserve_product_member` / `_authorization_group_member`** — `role` is exposed as a **name** (`Reader`…`Owner`) rather than the wire integer 1–5, validated against `client.RoleNames`. The API only range-checks 1–5, so a typo'd integer would otherwise be accepted silently. Update patches `role` alone, since the other two fields are immutable.
- **`secobserve_product_api_token`** — no retrieve and no update on the endpoint: Read lists the product's tokens and matches on name, and any change is a replace. `token` is `Computed + Sensitive + UseStateForUnknown`; it is returned once and is unrecoverable, so `ImportState` cannot populate it. Import therefore sets everything except `token` and the docs say so.

#### Import IDs

`product`/`product_group` by numeric id or exact name; `branch`/`service` by `<product_id>/<name>`; `product_member` by `<product_id>/<user_id>`; `product_authorization_group_member` by `<product_id>/<authorization_group_id>`; `product_api_token` by `<product_id>/<name>`, without the secret.

#### Delivered

All seven resources and four data sources are implemented, registered and verified to load in Terraform 1.15. Two findings changed the design during implementation, both recorded in `api-quirks.md`:

- **A security gate threshold of `0` is rejected at plan time.** Writing state from the response body is not enough: Terraform also requires the applied value to equal the planned one for an attribute the practitioner wrote. Accepting `0` produced `Provider produced inconsistent result after apply`. Caught by the stub-backed test, not by reading the source.
- **Server-recomputed attributes need a sibling-aware plan modifier.** For an unset `Optional + Computed` attribute the framework plans the prior state value, which the server contradicts whenever the controlling flag changes. Planning unknown unconditionally was worse — a permanent diff on every plan. `schemacommon.Int64UnknownWhenBoolSiblingChanges` narrows it to a changing sibling.

Testing added beyond the plan: a **stub SecObserve** (`internal/provider/stub_test.go`) reproducing the fill-and-clear logic, the truthiness test, the empty-propagation normalization, the delete confirmation and the GitHub base URL default. It runs the 13 drift-sensitive cases in CI with no container, and is what surfaced both findings above. The acceptance tests against a real instance (`core_acc_test.go`) then cover what a stub cannot vouch for: that the payloads are ones SecObserve actually accepts.

### Phase 2 — Access control & instance settings
`secobserve_user`, `secobserve_authorization_group`, `secobserve_authorization_group_member`, `secobserve_settings`.
Data sources: `secobserve_user`, `secobserve_authorization_group`, `secobserve_periodic_tasks`.

`secobserve_user` has **no password field** in `UserUpdateSerializer` — users created via API have no usable password. A local user created by Terraform therefore cannot log in until a password is set out-of-band (`PATCH /api/users/{id}/change_password/`, which requires the caller's *own* current password and so cannot be driven by a token). Document prominently; the resource emits a warning on create pointing this out. Self-delete returns 400.

OIDC handling per the caveats above:
- `secobserve_user`: warns when the API state says `is_oidc_user == true` and the config sets any claim-owned attribute (`email`, `full_name`, `first_name`, `last_name`). `is_active`/`is_superuser`/`is_external` stay managed.
- `secobserve_authorization_group`: `oidc_group` is a first-class managed attribute — this is the IdP mapping.
- `secobserve_authorization_group_member`: warns when the target group has a non-empty `oidc_group`, because login-time group sync will delete the membership.
- The `secobserve_user` data source is the documented path for OIDC deployments; the resource is for local users.

`secobserve_settings` is a singleton with ~55 keys. `Create` = `PATCH` (no POST exists), `Delete` = remove from state with a warning (there is nothing to delete; restoring upstream defaults would be surprising). Every key `Optional + Computed` so unmanaged keys are not clobbered. `vulnerablecode_api_key` is encrypted at rest but returned in plaintext on GET — mark sensitive.

### Phase 3 — Rules & API import
`secobserve_general_rule`, `secobserve_product_rule`, `secobserve_api_configuration`.
Data source: `secobserve_parser` (already added in Phase 0; extend with `type`/`source` filters).

`api_configuration.parser` and `rule.parser` take numeric IDs — the `secobserve_parser` data source is how users resolve `name` → `id`.

Rule approval is documented, not automated: `approval_status` is `Computed`, and the resource emits a warning after create/update when the value comes back `Needs approval`, telling the user a different principal must approve. **Known limitation to state in the docs:** because every update resets approval, a `terraform apply` that touches a rule un-approves it.

`general_rule` names are not reliably unique (`unique_together("product","name")` with `product=NULL`, and `GeneralRuleSerializer` excludes `product` so DRF's uniqueness validator never runs) — import by name must error on ambiguity rather than pick one.

### Phase 4 — License management & VEX counters
`secobserve_license_group` (+ `_member`, `_authorization_group_member`), `secobserve_license_policy` (+ `_item`, `_member`, `_authorization_group_member`), `secobserve_vex_counter`.
Data sources: `secobserve_license`, `secobserve_license_group`, `secobserve_license_policy`.

Add an optional `copy_from_id` on `secobserve_license_group` / `secobserve_license_policy` that routes create through `POST {id}/copy/` instead of a plain POST — the supported way to derive an editable copy of the nightly-regenerated ScanCode groups and the `Standard` policy. Mark it `RequiresReplace`. Document that policy copy drops `parent` and per-item `license_expression`/`comment`.

`license_policy_item` uniqueness is a serializer-only 5-tuple check with no DB constraint — surface the resulting 400 as a clear diagnostic.

## Testing

**Unit tests** (`go test ./...`, no SecObserve needed): transport layer against `httptest` — auth header format, pagination, DRF error → diagnostic mapping, 409 handling, exact-name lookup rejecting substring and ambiguous matches. Table-driven schema tests for every enum validator and every cross-field `ValidateConfig` rule.

**Acceptance tests** (`TF_ACC=1`, per-resource `resource.Test`): create → read → update → import → destroy, one per resource. `CheckDestroy` verifies the object is gone.

The critical assertion for this provider, applied to every resource: **apply twice, second plan must be empty.** All the server-side normalization listed above is a silent drift source; a plain create/read test will not catch it. Add explicit non-empty-plan checks (`plancheck.ExpectEmptyPlan()`) as a `ConfigPlanChecks` post-apply step.

Two unit tests specific to the Phase 1 design, both cheap and both guarding real failure modes:

- **Schema/model consistency.** Walk each resource's schema attribute names and compare them against the `tfsdk` tags reachable on its model, including promoted fields from the embedded shared blocks. A schema attribute with no model field, or the reverse, is a silent bug that otherwise only shows up as a confusing runtime error. Instantiate schemas via `resource.SchemaRequest`/`SchemaResponse` and assert `resp.Schema.ValidateImplementation` reports nothing.
- **Request completeness.** Assert that marshalling a `ProductRequest` built from an all-null model emits *every* managed JSON key, none of them dropped. This is what makes the fill-and-clear behaviour deterministic, and an accidental `omitempty` would quietly break it.

Targeted acceptance cases for the known hazards:
- Product with `security_gate_active = false` and no thresholds → thresholds come back null, plan stays empty.
- Product with `security_gate_active = true` and no thresholds → thresholds populated from Settings, plan stays empty.
- Product with `security_gate_active = true` and `security_gate_threshold_medium = 0` → server stores 99999, provider warns, plan stays empty.
- Product flipping `security_gate_active` from true to false → thresholds cleared server-side, plan stays empty (the case that proves state comes from the response, not the plan).
- Product with `propagate_branches = []` → rejected at plan time by `ValidateConfig` with a message telling the caller to omit the attribute.
- Product member `role` given an invalid name → plan-time error, not a 400 at apply.
- Product create → the auto-created Owner `Product_Member` does not break a subsequent `secobserve_product_member` for the same user (expect a 400 duplicate; document that the creating identity's membership is implicit).
- Branch `is_default_branch` flip between two branches, then destroy both (must reorder).
- Product destroy exercises the `?name=` confirmation path, including a name with leading/trailing whitespace.
- `license_policy_item` with a non-canonical `license_expression` → plan stays empty after normalization.
- `api_configuration` with `test_connection = true` (write-only) → no drift, and a failing connection surfaces as an apply error.

## Verification (end to end)

All steps run inside the devcontainer against the containerized instance; nothing touches your own SecObserve instance.

1. `make up` — starts `test/docker-compose.yml`, waits for `GET /api/status/health/`, mints the superuser API token, prints the exports.
2. `make schema && make generate && git diff --exit-code` — confirms the vendored schema and generated client match the running pinned version.
3. `make test` — unit tests + lint (no SecObserve needed).
4. `make testacc` — full acceptance suite with `TF_ACC=1`, including the **apply-twice / empty-plan** check on every resource.
5. `make testacc-oidc` — brings up the Keycloak overlay and runs the `oidc`-tagged tests that assert the group-sync wipe and the warning diagnostics.
6. Manual smoke test via `dev_overrides` (no `terraform init` needed): `examples/complete/` creates a product group, a product with an issue tracker and a security gate, two branches with one default, a service, an authorization group with a member and a product authorization-group membership, an API import configuration, a product rule, and a license policy copied from `Standard`. Then `terraform apply` → `terraform plan` (must be empty) → `terraform apply` again (no-op) → `terraform destroy`, and check the containerized instance's UI at the published port that nothing is left behind.
7. `make docs` (`tfplugindocs generate`) then `git diff --exit-code docs/`.
8. `make down` — tears down including volumes.
9. Final check against your own instance, read-only: point the provider at it with a superuser token and run `terraform plan` on an `import`-based config for a handful of existing products. An empty plan after import is the real proof the drift handling is correct against production-shaped data.

## Open items

- Registry namespace and release signing — blocks the first tagged release, not the code.
