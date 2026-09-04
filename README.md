# Terraform Provider for SecObserve

Manages the configuration of a [SecObserve](https://github.com/SecObserve/SecObserve)
instance with Terraform: products, product groups, branches, services,
memberships, authorization groups, rules, API import configurations and license
policies.

Runtime data — observations, license components, scan results, metrics, VEX
documents — is deliberately out of scope. That belongs in the CI/CD pipeline
that runs the scanners, not in Terraform state.

Built against SecObserve **1.58.0**.

## Status

Phase 0 (skeleton) is in place: the provider server, API transport layer,
containerized test environment and the `secobserve_parser` /
`secobserve_parsers` data sources. Resources land in subsequent phases:

| Phase | Contents |
|---|---|
| 1 | products, product groups, branches, services, memberships, product API tokens |
| 2 | users, authorization groups, instance settings |
| 3 | general and product rules, API import configurations |
| 4 | license groups, license policies, VEX counters |

## Requirements

- Terraform 1.11 or newer (write-only attributes are used for API keys).
- Go 1.25 to build from source.
- A SecObserve **superuser** user API token. SecObserve filters list responses
  for non-superusers and rejects writes to users, general rules, settings and
  periodic tasks; with a non-superuser token resources appear to drift on every
  plan. The provider warns at configure time if the token is not a superuser's.

## Usage

```hcl
terraform {
  required_providers {
    secobserve = {
      source = "jabbrwcky/secobserve"
    }
  }
}

provider "secobserve" {
  # or SECOBSERVE_BASE_URL / SECOBSERVE_API_TOKEN
  base_url  = "https://secobserve-backend.example.com"
  api_token = var.secobserve_api_token
}

data "secobserve_parser" "dependency_track" {
  name = "Dependency Track"
}
```

Note that `base_url` is the SecObserve **backend**, not the frontend.

### Obtaining an API token

The token endpoint takes a username and password in the request body and
ignores the `Authorization` header, so this is a one-off bootstrap step that
the provider cannot perform for itself:

```console
curl -X POST "$SECOBSERVE_BASE_URL/api/authentication/create_user_api_token/" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"...","name":"terraform"}'
```

## OIDC deployments

Both OIDC and local users are supported, but on an OIDC instance SecObserve
owns part of the user model and will overwrite Terraform:

- Membership of an authorization group that has `oidc_group` set is **deleted
  at the member's next login** by SecObserve's group synchronisation. Manage
  those memberships in the identity provider; the provider warns if you try.
- For OIDC users, `email`, `full_name`, `first_name` and `last_name` are
  overwritten from the token claims on every login. `is_active`,
  `is_superuser` and `is_external` remain safely manageable.
- Use the `secobserve_user` data source to *reference* OIDC-provisioned users
  instead of declaring them as resources.

`docs/design/api-quirks.md` documents these behaviours with references to the
upstream source.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md). In short:

```console
make up                        # containerized SecObserve + API token
eval "$(./test/bootstrap.sh)"
make test                      # unit tests
make testacc                   # acceptance tests
make down
```

## License

Mozilla Public License 2.0.
