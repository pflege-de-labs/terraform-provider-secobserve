# Blue/green rotation for a product API token.
#
# secobserve_product_api_token has no update endpoint -- any change replaces
# the token. The name is unique per product but SecObserve allows multiple
# tokens on the same product simultaneously, so a name that changes on
# rotation, combined with create_before_destroy, creates the new token before
# destroying the old one: no gap where no valid token exists.
#
# Automatic rotation: bump time_rotating's rotation_days (or swap it for
# random_id/random_uuid keyed off a different trigger) to change how often
# the name -- and therefore the token -- changes.
#
# Manual rotation: delete the time_rotating resource and hardcode a version
# suffix in the name instead, bumping it by hand when you want to rotate.

terraform {
  required_providers {
    secobserve = {
      source = "pflege-de-labs/secobserve"
    }
    time = {
      source = "hashicorp/time"
    }
  }
}

resource "time_rotating" "ci_token" {
  rotation_days = 30
}

resource "secobserve_product_api_token" "ci" {
  product = secobserve_product.checkout.id
  name    = "github-actions-${time_rotating.ci_token.id}"
  role    = "Upload"

  lifecycle {
    create_before_destroy = true
  }
}

# Capture the new secret before the old token is destroyed. In practice this
# output feeds a script or a separate resource (e.g. a secret manager entry)
# that writes the value somewhere the CI pipeline actually reads it from --
# the token is only ever returned once, at creation.
output "ci_token" {
  value     = secobserve_product_api_token.ci.token
  sensitive = true
}
