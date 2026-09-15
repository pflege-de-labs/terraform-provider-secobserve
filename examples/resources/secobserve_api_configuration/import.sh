# <product id>/<configuration name>. test_connection is never populated on
# import: it is write-only, ephemeral state the API never returns. api_key
# does round-trip for a superuser token.
terraform import secobserve_api_configuration.trivy_import "12/Trivy scan import"
