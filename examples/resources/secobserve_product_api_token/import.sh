# <product id>/<token name>. The secret is unrecoverable: SecObserve returns
# it only once, in the create response, and never again. Import populates
# every other attribute and leaves `token` null; the provider warns about
# this on import.
terraform import secobserve_product_api_token.ci "12/ci"
