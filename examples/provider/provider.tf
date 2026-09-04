terraform {
  required_providers {
    secobserve = {
      source = "jabbrwcky/secobserve"
    }
  }
}

# base_url and api_token are usually supplied via SECOBSERVE_BASE_URL and
# SECOBSERVE_API_TOKEN so the token never ends up in a state file or a diff.
provider "secobserve" {
  base_url = "https://secobserve-backend.example.com"
}
