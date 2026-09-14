# Agent guidance

Orientation for coding agents working in this repo. Human contributors: see
[CONTRIBUTING.md](CONTRIBUTING.md). Design background and the SecObserve API
quirks this provider exists to absorb: [docs/design/](docs/design/).

## Workflow rules

- **Conventional Commits.** Every commit message starts with a type —
  `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:` — followed by a
  concise summary. Use the body to explain *why*, especially for anything
  found by running the code rather than by reading it; see the existing log
  (`git log --oneline`) for the level of detail expected.
- **Branch + PR, never commit to `main` directly.** Work happens on a
  feature branch and merges via pull request. `main` is the trunk;
  `git log --oneline main` shows what has actually shipped.
- **Prefer a git worktree over switching the working directory's branch.**
  When starting work on a feature, use `git worktree add ../secobserve-terraform-<branch> -b <branch>`
  rather than `git checkout -b <branch>` in place. This keeps the current
  checkout's branch and build artifacts (`GOBIN` binary, `.terraform/` in any
  manual smoke test) undisturbed if something else needs the main checkout
  mid-task, and avoids losing uncommitted work to an accidental branch switch.
  Remove the worktree (`git worktree remove`) once its branch is merged.

## Repo-specific facts worth knowing before editing

- **`GOWORK=off` is load-bearing.** The Makefile and CI export it because a
  parent directory may contain an unrelated `go.work`; this module must never
  be absorbed into it. Set it in your own shell too when running `go` commands
  directly: `export GOWORK=off`.
- **Everything in `internal/client/*.go` should be read alongside
  [docs/design/api-quirks.md](docs/design/api-quirks.md).** SecObserve rewrites,
  clears, or normalizes a surprising number of submitted values server-side —
  that file is the catalogue, with source references into the upstream
  SecObserve repo. Don't reintroduce a "just read the state back from the
  plan" shortcut; state is always written from the API response body.
- **`internal/schemacommon` holds attribute blocks shared by `product` and
  `product_group`**, paired schema+model, embedded by value into each
  resource's model. If you touch one of those attributes, check whether the
  sibling resource needs the same change.
- **A security gate threshold of `0` is rejected at plan time**, not silently
  overridden — see `schemacommon.SecurityGate.ValidateSecurityGate`. Don't
  relax that validator without understanding why it's there (the server
  substitutes its own default for a submitted `0` while the gate is active,
  and Terraform forbids a provider from applying a different value than what
  was planned).
- **Two dev environments for the containerized SecObserve instance**: Docker
  Compose (`make up` / `make down`, the default, what CI uses) and Apple's
  `container` CLI (`make container-up` / `make container-down`, see
  [test/apple-container.md](test/apple-container.md)). The backend's real
  listen port is `5000`, hardcoded in the upstream image and not
  configurable — only host-published access legitimately uses `8000`.

## Before committing

```sh
export GOWORK=off
go build ./... && go vet ./...
make lint
make test                          # unit tests, no SecObserve needed
make up && eval "$(./test/bootstrap.sh)" && make testacc   # or make container-up
make docs                          # if any schema/description changed
```

`make testacc` is the real test for anything touching a resource: the
critical assertion throughout this provider is that applying twice produces
an empty plan. A plain create/read pass will not catch the drift hazards
documented in `docs/design/api-quirks.md`.
