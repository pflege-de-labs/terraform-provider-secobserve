# Terraform Query (List Resources) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users run `terraform query` / `-generate-config-out` to bring existing SecObserve objects under Terraform.

**Architecture:** Each listed resource gains a resource identity `{id}` (set in Create/Read/Update, accepted by ImportState) and implements `list.ListResource` on its existing struct. Two shared helpers in `internal/tfutil` carry the identity plumbing and the result streaming; per-resource code is a client list call plus the existing `fromAPI`.

**Tech Stack:** Go, terraform-plugin-framework v1.19.0 (`list`, `resource/identityschema`), terraform-plugin-testing v1.16.0 (`querycheck`, `ImportBlockWithResourceIdentity`), tfplugindocs v0.25.0.

**Spec:** [docs/design/list-resources.md](../../design/list-resources.md)

## Global Constraints

- `export GOWORK=off` for every `go` command.
- Identity schema is exactly one attribute: `id`, Int64, `RequiredForImport: true`.
- Query/identity acceptance steps use `tfversion.SkipBelow(tfversion.Version1_14_0)`; the full existing suite must still pass on Terraform 1.11 (CI matrix `1.11.*`, `1.15.*`).
- State is always written from the API response (`fromAPI`), never from config — CLAUDE.md rule.
- Product and product_group keep `security_gate` / `repository_branch_housekeeping` out of list results (spec: "Accepted gap").
- Every resource touched keeps its "apply twice → empty plan" acceptance assertion.
- Conventional Commits; work in a worktree: `git worktree add ../secobserve-terraform-feat-list-resources -b feat/list-resources`.

## Review Focus

1. **Existing state without identity.** A state written by the previous provider version has no identity; the first `plan` after upgrade must refresh without "Missing Resource Identity After Read". → Task 2 acceptance step using `ExternalProviders` with the last released version, then the dev build, expecting an empty plan.
2. **`/api/products/` returning product groups.** `secobserve_product` must not list groups (generated config would then fight `secobserve_product_group`). → Task 2: `Products()` drops `IsProductGroup` items client-side, pinned by httptest unit test `TestProductsExcludesGroups` in `internal/client` (holds whatever the endpoint returns).
3. **Import by identity on an int64 attribute.** `ImportStatePassthroughID` would write a string into `id`; only the identity path may use passthrough. → Task 1 unit test `TestImportByIdentity_SkipsWhenIDSet`.
4. **`limit` in the list block.** Results past `req.Limit` must not be emitted (Terraform stops reading, but per-item `fromAPI` calls like license_group's extra GET still cost requests). → Task 1 `TestStreamList_HonoursLimit`.
5. **Child list without a valid `product`.** Missing/unknown `product` must be a config error, not a list of every branch on the instance. → Task 6 `Required: true` on the filter plus a query step with a nonexistent product id expecting zero results, not an error.

---

### Task 1: Shared identity and list helpers

**Files:**
- Create: `internal/tfutil/identity.go`, `internal/tfutil/list.go`
- Test: `internal/tfutil/identity_test.go`, `internal/tfutil/list_test.go`

**Interfaces:**
- Produces:
  - `func IDIdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse)` — sets `identityschema.Schema{Attributes: {"id": identityschema.Int64Attribute{RequiredForImport: true}}}`. Resources call it from their `IdentitySchema` method.
  - `func SetIdentityID(ctx context.Context, identity *tfsdk.ResourceIdentity, id types.Int64) diag.Diagnostics` — no-op on nil identity.
  - `func ImportByIdentity(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) bool` — returns false when `req.ID != ""`; otherwise calls `resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)` and returns true.
  - `func StreamList[T client.Named](ctx context.Context, req list.ListRequest, stream *list.ListResultsStream, items []T, toResource func(context.Context, T, *diag.Diagnostics) any)` — sets `stream.Results` to an iterator that, per item up to `req.Limit` (0 = unlimited), builds `req.NewListResult(ctx)`, sets identity `id = item.GetID()`, `DisplayName = item.GetName()`, and only when `req.IncludeResource` calls `toResource` and `result.Resource.Set`. Diagnostics from `toResource` go on that result.
  - `func ListError(summary string, err error) iter.Seq[list.ListResult]` — wraps `list.ListResultsStreamDiagnostics` with one error diagnostic.

- [ ] **Step 1: Write failing tests**

`list_test.go` uses a fake `type item struct{id int64; name string}` implementing `client.Named`, a resource schema `{id Int64 Computed, name String Computed}` and `IDIdentitySchema`'s schema, and collects results by ranging `stream.Results`:

```go
func TestStreamList_HonoursLimit(t *testing.T)            // 3 items, Limit 2 → 2 results
func TestStreamList_IdentityAndDisplayName(t *testing.T)  // identity id == 7, DisplayName == "seven"
func TestStreamList_SkipsResourceWhenNotIncluded(t *testing.T) // IncludeResource=false → toResource never called
func TestStreamList_ResourceDiagnosticsOnResult(t *testing.T)  // toResource adds error → that result HasError()
```

`identity_test.go`:

```go
func TestImportByIdentity_SkipsWhenIDSet(t *testing.T) // req.ID "42" → returns false, resp.State untouched
func TestImportByIdentity_CopiesID(t *testing.T)       // req.Identity id=42 → returns true, state id == 42
func TestSetIdentityID_NilIdentity(t *testing.T)       // nil identity → no diagnostics, no panic
```

- [ ] **Step 2:** `go test ./internal/tfutil/ -run 'StreamList|ImportByIdentity|SetIdentityID'` → FAIL (undefined).
- [ ] **Step 3:** Implement the five functions above.
- [ ] **Step 4:** Same command → PASS; `make lint` clean.
- [ ] **Step 5:** Commit `feat: add shared resource identity and list helpers`.

---

### Task 2: `secobserve_product` end to end (proves the shape)

**Files:**
- Modify: `internal/client/products.go` (add list func), `internal/resource/product/resource.go`, `internal/provider/provider.go`
- Create: `internal/resource/product/list.go`, `internal/provider/list_acc_test.go`, `examples/list-resources/secobserve_product/list-resource.tfquery.hcl`
- Test: `internal/provider/list_acc_test.go`, `internal/provider/core_acc_test.go` (existing lifecycle)

**Interfaces:**
- Consumes: Task 1 helpers.
- Produces:
  - `func (c *Client) Products(ctx context.Context) ([]Product, error)` — `List[Product](ctx, c, productsPath, nil)`, then drop items with `IsProductGroup` (Review Focus 2).
  - `func NewList() list.ListResource` in package `product`, returning `&productResource{}`.
  - Provider: `func (p *secObserveProvider) ListResources(context.Context) []func() list.ListResource` and `resp.ListResourceData = apiClient` in `Configure`. Later tasks append to `ListResources`.
  - Pattern every later task copies: `IdentitySchema` method → `tfutil.IDIdentitySchema`; `resp.Diagnostics.Append(tfutil.SetIdentityID(ctx, resp.Identity, <state>.ID)...)` after each successful `resp.State.Set` in Create/Read/Update; `if tfutil.ImportByIdentity(ctx, req, resp) { return }` as the first line of `ImportState`; `ListResourceConfigSchema` returns an empty `listschema.Schema{}` (top-level) and `MarkdownDescription` for the list resource.

- [ ] **Step 1a: Write the failing unit test** `TestProductsExcludesGroups` in `internal/client/products_test.go`: httptest server answers `GET /api/products/` with one page holding `{id:1,name:"p",is_product_group:false}` and `{id:2,name:"g",is_product_group:true}` → `Products` returns only id 1.
- [ ] **Step 1b: Write the failing acceptance test** `TestAccProductList` in `list_acc_test.go`, `TerraformVersionChecks: SkipBelow(Version1_14_0)`:
  1. Config: one `secobserve_product` with a random name; `ConfigStateChecks: statecheck.ExpectIdentity("secobserve_product.test", map[string]knownvalue.Check{"id": knownvalue.NotNull()})`.
  2. `Query: true`, config `provider "secobserve" {}` + `list "secobserve_product" "all" { provider = secobserve }` (written to a separate `.tfquery.hcl`, so it cannot reference step-1 resources); checks: `querycheck.ExpectLengthAtLeast("secobserve_product.all", 1)`, `querycheck.ExpectResourceDisplayName("secobserve_product.all", queryfilter.ByDisplayName(knownvalue.StringExact(productName)), knownvalue.StringExact(productName))`.
  3. `ResourceName: "secobserve_product.test"`, `ImportState: true`, `ImportStateKind: resource.ImportBlockWithResourceIdentity`, `ImportStateVerify: true`, reusing the `ImportStateVerifyIgnore` list from `TestAccProductLifecycle`.

  Add `TestAccProductIdentityUpgrade` (Review Focus 1): step 1 `ExternalProviders: {"secobserve": {Source: "pflege-de-labs/secobserve", VersionConstraint: "0.2.0"}}` creates a product with a minimal config (`name` only, so v0.2.0 understands it); step 2 uses `ProtoV6ProviderFactories` with the same config and `ConfigPlanChecks.PreApply: plancheck.ExpectEmptyPlan()`. SkipBelow 1.12.
- [ ] **Step 2:** `GOWORK=off go test ./internal/client/ -run TestProductsExcludesGroups` → FAIL; `make up && eval "$(./test/bootstrap.sh)" && GOWORK=off TF_ACC=1 go test ./internal/provider/ -run 'TestAccProduct(List|IdentityUpgrade)' -v` → FAIL (`list` resource type not supported / no identity).
- [ ] **Step 3:** Implement: client `Products`; identity on `productResource` per the pattern; `list.go` with `ListResourceConfigSchema` and `List` (`items, err := r.client.Products(ctx)`; on error `stream.Results = tfutil.ListError("Could not list SecObserve products", err)`; else `tfutil.StreamList` with a closure calling `m.fromAPI` and returning `m`); provider wiring. List resource `MarkdownDescription` states the gate/housekeeping gap and sensitive-attribute note from the spec in two sentences.
- [ ] **Step 4:** Step 2 command → PASS. Then `make test` and the full `make testacc` → PASS (existing `TestAccProductLifecycle` still produces empty second plans). Run `make testacc` once with Terraform 1.11 (`mise exec terraform@1.11 -- make testacc` or equivalent) → PASS, query tests SKIP.
- [ ] **Step 5:** Write `examples/list-resources/secobserve_product/list-resource.tfquery.hcl` (one `list` block, no `config`). `make docs`; confirm `docs/list-resources/product.md` is generated (if tfplugindocs places it elsewhere, follow its output and adjust the example path for later tasks).
- [ ] **Step 6:** Commit `feat: support terraform query for secobserve_product`.

---

### Task 3: `secobserve_product_group`

**Files:** Modify `internal/client/products.go`, `internal/resource/product_group/resource.go`, `internal/provider/provider.go`, `internal/provider/list_acc_test.go`. Create `internal/resource/product_group/list.go`, `examples/list-resources/secobserve_product_group/list-resource.tfquery.hcl`.

**Interfaces:** Consumes Task 2 pattern. Produces `func (c *Client) ProductGroups(ctx context.Context) ([]Product, error)` over `productGroupsPath`; `product_group.NewList`.

- [ ] **Step 1:** `TestAccProductGroupList`: same three steps as Task 2 (create group, query + display-name check, identity import with `TestAccProductGroupLifecycle`'s ignore list).
- [ ] **Step 2:** Run `-run TestAccProductGroupList` → FAIL.
- [ ] **Step 3:** Implement per the Task 2 pattern; same description note on gate/housekeeping.
- [ ] **Step 4:** Run → PASS; `make testacc` → PASS.
- [ ] **Step 5:** `make docs`; commit `feat: support terraform query for secobserve_product_group`.

---

### Task 4: Unfiltered top-level resources — `user`, `authorization_group`, `general_rule`, `license_policy`

**Files:** Modify `internal/client/{users,authorization_groups,rules,license_policies}.go`, each `internal/resource/<name>/resource.go`, `internal/provider/provider.go`, `internal/provider/list_acc_test.go`. Create `internal/resource/<name>/list.go` and `examples/list-resources/secobserve_<name>/list-resource.tfquery.hcl` for each.

**Interfaces:**
- Produces: `Users(ctx) ([]User, error)`, `AuthorizationGroups(ctx) ([]AuthorizationGroup, error)`, `GeneralRules(ctx) ([]Rule, error)`, `LicensePolicies(ctx) ([]LicensePolicy, error)` — each one `List[T]` call on the existing path constant; `NewList` in each package. `user`'s display name is `Username` (its `GetName`).

- [ ] **Step 1:** One test per resource, `TestAcc{User,AuthorizationGroup,GeneralRule,LicensePolicy}List`, same three-step shape; reuse each resource's existing lifecycle test config and `ImportStateVerifyIgnore`.
- [ ] **Step 2:** Run `-run 'TestAcc(User|AuthorizationGroup|GeneralRule|LicensePolicy)List'` → FAIL.
- [ ] **Step 3:** Implement per pattern. Check each `Read` for values carried from prior state rather than `fromAPI` (planning found none for these four); if one appears, the list closure leaves that attribute null and the resource description says so.
- [ ] **Step 4:** Run → PASS; `make testacc` → PASS.
- [ ] **Step 5:** `make docs`; commit `feat: support terraform query for users, authorization groups, general rules and license policies`.

---

### Task 5: `secobserve_license_group`

Separate because its `fromAPI(ctx, group, licenses, diags)` needs `ListLicensesInGroup` per group.

**Files:** Modify `internal/client/licenses.go`, `internal/resource/license_group/resource.go`, provider, `list_acc_test.go`. Create `internal/resource/license_group/list.go`, example file.

**Interfaces:** Produces `LicenseGroups(ctx) ([]LicenseGroup, error)`; `license_group.NewList`.

- [ ] **Step 1:** `TestAccLicenseGroupList`: create a group with two licenses; query with `include_resource = true` in the list block and `querycheck.ExpectResourceKnownValues` asserting the group's `licenses` attribute has 2 elements; identity import step.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement; the `toResource` closure calls `ListLicensesInGroup` and appends its error to the result's diagnostics (StreamList only invokes it when `IncludeResource`, so plain listing stays one paginated call).
- [ ] **Step 4:** Run → PASS; `make testacc` → PASS.
- [ ] **Step 5:** `make docs`; commit `feat: support terraform query for secobserve_license_group`.

---

### Task 6: Product children — `branch`, `service`, `api_configuration`, `product_rule`

**Files:** Modify `internal/client/{branches,services,api_configurations,rules}.go`, each resource's `resource.go`, provider, `list_acc_test.go`. Extend `internal/tfutil/list.go` (below). Create each `list.go` and example file.

**Interfaces:**
- Produces:
  - `func ProductFilterSchema(resp *list.ListResourceSchemaResponse)` in `tfutil/list.go` — one attribute `product`, `listschema.Int64Attribute{Required: true}`, description "Id of the product whose <kind> to list."
  - `type ProductFilter struct { Product types.Int64 `tfsdk:"product"` }` in the same file; `List` reads it with `req.Config.Get`.
  - Client: `Branches(ctx, productID int64) ([]Branch, error)`, `Services(ctx, productID int64) ([]Service, error)`, `APIConfigurations(ctx, productID int64) ([]APIConfiguration, error)`, `ProductRules(ctx, productID int64) ([]Rule, error)` — `List[T]` with `url.Values{"product": {strconv.FormatInt(productID, 10)}}`, same filter the `*ByName` lookups already use.

- [ ] **Step 1:** Per resource `TestAcc{Branch,Service,APIConfiguration,ProductRule}List`: the query file cannot reference step-1 state, so create the parent product out of band in the test function (`client.New` with `SECOBSERVE_BASE_URL`/`SECOBSERVE_API_TOKEN`, `t.Cleanup` → `DeleteProduct`) and `fmt.Sprintf` its id into both the child's resource config and `config { product = %d }` → `ExpectLengthAtLeast 1` + display name; identity import step. Put that setup in one helper `testAccOutOfBandProduct(t *testing.T) client.Product` in `list_acc_test.go`. Add `TestAccBranchListUnknownProduct` (Review Focus 5): `config { product = 999999999 }` → `querycheck.ExpectLength("secobserve_branch.none", 0)`, no error.
- [ ] **Step 2:** Run `-run 'TestAcc(Branch|Service|APIConfiguration|ProductRule)List'` → FAIL.
- [ ] **Step 3:** Implement per pattern with `ProductFilterSchema`. A nonexistent id on DRF's `ModelChoiceFilter` most likely returns 400 (`product: Select a valid choice`), not an empty page; let `TestAccBranchListUnknownProduct` decide, and map only that validation error to zero results — surface every other error.
- [ ] **Step 4:** Run → PASS; `make testacc` → PASS.
- [ ] **Step 5:** `make docs`; commit `feat: support terraform query for product-scoped resources`.

---

### Task 7: Docs and final verification

**Files:** Modify `docs/design/index.md` (link `list-resources.md`), `docs/design/api-quirks.md` (one line under the security-gate trade-off: the import gap also applies to `terraform query` generated config), `README.md` if it lists features.

- [ ] **Step 1:** Edit the three files.
- [ ] **Step 2:** Run the CLAUDE.md "Before committing" block in full: `go build ./... && go vet ./... && make lint && make test && make testacc && make docs` → all pass, `git status` shows no doc diff after `make docs`.
- [ ] **Step 3:** Manual smoke on the dev instance: a directory with the 11 example `list` blocks, `terraform query -generate-config-out=generated.tf`, then fill the `null # sensitive` values and `terraform plan` → only `import` actions, no updates or replacements.
- [ ] **Step 4:** Commit `docs: document terraform query support`; open PR.
