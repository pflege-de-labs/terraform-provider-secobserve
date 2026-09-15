# SecObserve API quirks

Behaviours of the SecObserve REST API that shape this provider. Each entry
names the upstream source location it was verified against, in the
[SecObserve](https://github.com/SecObserve/SecObserve) repository at tag
`v1.58.0`. Code comments point here rather than repeating the detail.

## Authentication

- Both user API tokens and product API tokens use the same wire format:
  `Authorization: APIToken <token>`
  (`backend/application/access_control/services/api_token_authentication.py:11-30`).
  There is no `Bearer` or `Token` prefix. `Bearer` is OIDC ID tokens, `JWT` is
  the frontend's session token.
- **A superuser token is required.** Non-superusers receive filtered list
  responses for users, authorization groups and notifications
  (`access_control/queries/*.py`) and cannot write users, general rules,
  settings or periodic tasks. Partial reads make Terraform see permanent
  phantom drift.
- Tokens are minted at
  `POST /api/authentication/create_user_api_token/`, which takes username and
  password in the request body and ignores the `Authorization` header
  (`access_control/api/views.py:266-292`). The provider therefore cannot
  create or rotate its own credentials — that is a bootstrap step.
- One token per `(user, name)`; a duplicate name is a 400
  (`access_control/services/user_api_token.py:13-15`). The request serializer
  allows `name` up to 255 characters while the column is 32
  (`serializers.py:327` vs `models.py:156`), so long names fail at the
  database layer.

## Version reporting

`GET /api/status/version/` returns the literal string `version_unknown` unless
the backend runs from a released container image: the real version is
substituted into the view's source by the Dockerfile at build time
(`docker/backend/prod/django/Dockerfile:90`,
`backend/application/commons/api/views.py:24`). An instance started from a
source checkout therefore cannot report its version, and the provider must
treat `version_unknown` as "unknown", never as a mismatch.

## Transport

- Pagination is page-number based with `PAGE_SIZE = 25` and a `page_size`
  override (`backend/config/settings/base.py:389-390`). `/api/product_api_tokens/`
  is a plain `ViewSet` and returns a bare `{"results": [...]}` with no
  pagination envelope (`core/api/views_product.py:603-604`).
- Authenticated callers are throttled at 100 requests per second
  (`config/settings/base.py:392-396`).
- Every relation is a DRF `PrimaryKeyRelatedField`: writes accept **numeric
  ids only**. The only name-based inputs in the whole API are the imperative
  `*_by_name` import endpoints (`config/urls.py:106-131`).
- List filters on `name` use `icontains`, not equality
  (`core/api/filters.py:47-80`). A name lookup must therefore filter for an
  exact match client-side. `/api/services/` is the one exception where `name`
  is exact (`filters.py:194-202`).
- Names are not always unique. General rules are constrained by
  `unique_together("product", "name")` with `product = NULL` for general
  rules, so the database does not enforce uniqueness, and
  `GeneralRuleSerializer` excludes `product` so DRF's uniqueness validator
  never runs either (`rules/models.py:66-70`, `rules/api/serializers.py:33`).
  Ambiguous name lookups must be reported, not resolved arbitrarily.

## Error shapes

The backend produces four distinct error bodies:

| Shape | When |
|---|---|
| `{"field": ["message", ...]}` | serializer field validation, 400 |
| `{"field": "message"}` | `ValidationError` raised inside a view, 400 |
| `{"detail": "message"}` | DRF permission and 404 handling |
| `{"message": "message"}` | `ProtectedError`/`RestrictedError` mapped to 409 by `notifications/api/exception_handler.py:28-33` |

A single 400 body can mix the first two shapes across different fields.

## OIDC

Verified in `access_control/services/oidc_authentication.py`:

- `_synchronize_groups` (`:194-204`) removes the user from **every**
  `Authorization_Group` with a non-empty `oidc_group` and then re-adds only
  the groups present in the token's group claim. It runs on first login and on
  every login where the group set changed (`:156-158`). A Terraform-managed
  membership in an `oidc_group`-mapped group is therefore deleted at the
  member's next login, and re-adding it only restarts the cycle.
- OIDC users are provisioned just-in-time on first login (`_create_user`,
  `:104-138`). `_check_user_change` (`:141-168`) overwrites `email`,
  `full_name`, `first_name` and `last_name` from the claims on every login,
  forces `is_oidc_user = true` and calls `set_unusable_password()`. Those
  fields cannot be owned by Terraform.
- `is_active`, `is_superuser` and `is_external` are *not* touched by
  `_check_user_change`, so they remain safely managed. `is_external` is
  derived from the `internal_users` email regexes only at create time
  (`:118-126`).
- `Authorization_Group.oidc_group` is ordinary configuration: it is the
  IdP-group to SecObserve-group mapping.

## Server-side mutation of submitted values

These are silent, so a resource that writes state from the plan rather than
from the response body will show a permanent diff.

| Field | Behaviour | Source |
|---|---|---|
| `security_gate.threshold_*` | filled from global settings when the gate is active and the value was omitted; forced to null when it is inactive | `core/api/serializers_product.py:122-142` |
| `security_gate.threshold_*` set to `0` | the fill-in test is `if not attrs.get(...)` — truthiness, not presence — so a submitted `0` counts as unset and is replaced by the instance default (`99999` for medium, low, none and unknown) | `core/api/serializers_product.py:122-134` |
| `repository_branch_housekeeping.keep_inactive_days`, `.exempt_branches` | same fill-or-clear pattern driven by housekeeping being active | `serializers_product.py:112-120` |
| `propagate_branches` | an empty or effectively-empty list is stored as `null`, not `[]` | `serializers_product.py:660-663` |
| `license_policy_item.license_expression` | re-normalised through `spdx_licensing.validate(strict=True)`; the stored value can differ from the submitted one | `licenses/api/serializers.py:591-600` |
| `license_policy.ignore_component_types` | dual representation with `ignore_component_type_list`; the list form is always injected into responses | `licenses/api/serializers.py:482-499` |
| `product.observation_notification_statuses` | comma-joined column exposed as the list `observation_notification_status_list` | `serializers_product.py:256-270` |
| rule `approval_status`, `user`, `approval_*`, `rejection_remark` | server-owned; **any** update resets approval and reassigns `user` | `rules/api/serializers.py:62-64,112-114`, `rules/models.py:75-101` |

### What this means for the provider

Writing state from the response body is necessary but not sufficient.
Terraform additionally requires that the value applied to an attribute equals
the value that was planned for it, unless the plan said unknown. Two
consequences fall out of that, both learned the hard way and both covered by
tests in `internal/provider/product_stub_test.go`:

1. **A threshold of `0` is not representable and is rejected at plan time.**
   Accepting it produces `Provider produced inconsistent result after apply:
   .security_gate.threshold_medium: was cty.NumberIntVal(0), but now
   cty.NumberIntVal(99999)`. The rejection is unconditional rather than only
   when the gate is active, because the gate can also be switched on by the
   product group, which is not visible at plan time. While the gate is off
   the thresholds are not evaluated anyway, so nothing is lost.

2. **An explicit threshold/housekeeping value combined with `active = false`
   inside the same block is likewise unrepresentable**, for the same reason
   as the `0` case: SecObserve force-clears the dependent values
   unconditionally, and Terraform forbids an applied value that differs from
   an explicitly-configured one. `schemacommon.ValidateSecurityGate` /
   `ValidateBranchHousekeeping` reject the combination at plan time.

3. **`security_gate` and `repository_branch_housekeeping` are real Terraform
   blocks (`schema.SingleNestedBlock`), not nested attributes, and are the
   one deliberate exception to "always write state from the response body"
   (see the top of this file).** Every field inside, including `active`, is
   plain `Optional` with no `Computed`, so state is echoed from the
   plan/prior state by the resource's `Create`/`Read`/`Update`, never
   populated from `FromAPI`. Block *presence* is what encodes the tri-state:
   omitting the block means inherit; writing it (even empty) activates the
   feature; `active = false` inside is the explicit way to switch it off
   without inheriting. This is what makes toggling the block in and out of
   existence freely a guaranteed empty-plan no-op — there is nothing
   computed to carry forward, so there is nothing for the server's
   fill-or-clear to contradict. The price is that a server-filled default
   value is never reflected back into the resource (matching this
   provider's existing policy of reserving informational/computed data for
   data sources, not resources) — `issue_tracker_base_url` still needs the
   sibling-aware `schemacommon.StringUnknownWhenStringSiblingChanges` plan
   modifier because it *is* `Optional + Computed` (its GitHub default is
   useful to see in state).

   Two real costs of this trade-off, found by the acceptance suite rather
   than reasoned out in advance: **`terraform import` cannot recover a
   configured block** (there is no plan or prior state to echo during
   import, so the block always comes back empty even when SecObserve has
   real values configured — `TestAccProductLifecycle`'s
   `ImportStateVerifyIgnore` documents this), and **a value changed directly
   in SecObserve rather than through Terraform is not detected as drift** on
   the next plan, unlike every other attribute this provider manages. Both
   are documented on the blocks themselves. Accepted deliberately: the
   alternative (reading from the response) is exactly what produced the
   "provider produced inconsistent result" crash this design fixes, and both
   gaps only matter for out-of-band changes, which are the less common path
   for a Terraform-managed resource.

### Inheritance chain for the security gate and branch housekeeping

Not previously documented anywhere in this repo, and surprising enough to be
worth calling out explicitly (`core/services/security_gate.py:20-24`,
`core/services/housekeeping.py:29-38`, `commons/models.py:21,139`):

- The **instance-wide `Settings` default is `true`** for both
  (`security_gate_active`/`branch_housekeeping_active` columns, non-nullable
  `BooleanField(default=True)`). Omitting `security_gate`/
  `repository_branch_housekeeping` at every level means the gate and
  housekeeping are **on** by default, not off — this is why omitting the
  block means inherit, never disabled; see the schema attribute
  descriptions.
- If a product belongs to a product group, **the group's explicit
  `active` always overrides the product's own value outright** — this is
  not "most specific wins". The product's own `security_gate`/
  `repository_branch_housekeeping` block (including its thresholds) is only
  consulted when the product group's is absent. Only when *both* are absent
  does the instance-wide default apply.
- Consequence: a product's `security_gate`/`repository_branch_housekeeping`
  block is silently never consulted whenever its product group has its own
  block present — the provider cannot detect this at plan time (it would
  require reading a second resource), so it is documented rather than
  validated.

## Values that never round-trip

- `product.issue_tracker_api_key` is accepted on write but removed from
  responses unless the caller holds `Product_Edit`
  (`serializers_product.py:252-254`). Since the provider requires a superuser
  it does round-trip in practice, which matters because it **cannot** be a
  write-only attribute: SecObserve requires `issue_tracker_type`,
  `issue_tracker_base_url`, `issue_tracker_api_key` and
  `issue_tracker_project_id` to be either all set or all empty
  (`serializers_product.py:565-574`). A write-only value is absent from state,
  so a later update could neither resend the key nor omit it — sending `""`
  fails the check, and so does omitting it, because the check reads
  `attrs.get("issue_tracker_api_key")` and a missing key is falsy while the
  other three are set. It is therefore a plain sensitive attribute, stored in
  state.
- `api_configuration.api_key` behaves the same way
  (`import_observations/api/serializers.py:111-119`).
- `api_configuration.test_connection` is a write-only side-effecting flag that
  performs a live connection test and is never returned
  (`import_observations/api/serializers.py:105,121-164`).
- A product API token's secret is returned exactly once, in the 201 body
  (`core/api/views_product.py:626`), and the endpoint has no retrieve action at
  all. Import can never restore it.

## Non-standard request requirements

- `DELETE /api/products/{id}/` and `DELETE /api/product_groups/{id}/` require
  the exact resource name in a `name` query parameter as confirmation. The
  comparison is case- and whitespace-sensitive and expects exactly one `name`
  parameter (`core/api/views_product.py:139-164`).
- `POST` to `branches`, `services`, `product_members` and
  `product_authorization_group_members` must carry a top-level `product` key.
  It is read by the permission layer before validation, so a missing key is a
  `ParseError`, not a field error
  (`authorization/api/permissions_base.py:11-24`).
- `GET /api/product_api_tokens/` requires a `product` query parameter and
  rejects a non-numeric one (`core/api/views_product.py:588-604`).
- A branch with `is_default_branch = true` cannot be deleted
  (`views_product.py:541-546`), and `product.repository_default_branch` is
  read-only: it is set as a side effect of flipping the flag on a branch
  (`core/services/branch.py:6-19`).
- Creating a product triggers a `post_save` signal that adds the caller as a
  `Product_Member` with role `Owner` (`core/signals.py:73-76`). An unmanaged
  membership therefore exists immediately after every product create.
- `products` and `product_groups` serve a different serializer for `list` than
  for `retrieve`; `assessment_approvers` and
  `assessment_approver_authorization_groups` are absent from list responses
  (`serializers_product.py:403-429`). Reads must use the detail endpoint.
- `Settings` is a singleton reachable only as `GET`/`PATCH
  /api/settings/{pk}/`, where `pk` is parsed and ignored. There is no POST,
  PUT or DELETE (`commons/api/views.py:90-115`). Setting
  `feature_automatic_osv_scanning` to true also force-enables
  `feature_license_management` (`views.py:108-109`).
- Rule approval is a separate `PATCH {id}/approval/` call, valid only from the
  `Needs approval` state, and self-approval is rejected outright
  (`rules/services/approval.py:11-23`). The identity that creates a rule can
  never approve it.
- `license_groups` and `license_policies` expose `POST {id}/copy/` returning
  201 plus the new object (`licenses/api/views.py:386-411`, `:527-552`). This
  is the supported way to derive an editable object from the system-managed
  ScanCode groups and the `Standard` policy, which a periodic task otherwise
  overwrites every 24 hours
  (`background_tasks/periodic_tasks/license_tasks.py:25`,
  `licenses/services/license_group.py:48-59`).

## Immutable fields

Enforced server-side with a 400, so they need `RequiresReplace` plan
modifiers or apply simply fails: `branch.product`, `service.product`,
`product_member.{product,user}`,
`product_authorization_group_member.{product,authorization_group}`,
`authorization_group_member.{authorization_group,user}`,
`product_rule.product`, `api_configuration.product`,
`license_group_member.{license_group,user}`,
`license_policy_member.{license_policy,user}` and the two
authorization-group-member equivalents.

## System-managed reference data

Not creatable through the API; reference it, do not manage it.

| Object | Populated by |
|---|---|
| `Parser` | `register_parsers` management command at startup, from the on-disk parser packages |
| `License` (SPDX) | `initial_license_load` plus a nightly SPDX License List import |
| License groups named `"<category> (ScanCode LicenseDB)"` | `import_scancode_licensedb()`, re-run every 24 hours |
| License policy `"Standard"` | `create_scancode_standard_policy()` at initial load |
