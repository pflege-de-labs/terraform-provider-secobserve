# Either the numeric id or the exact product name works.
terraform import secobserve_product.checkout 12
terraform import secobserve_product.checkout "checkout-service"

# Import cannot recover security_gate/repository_branch_housekeeping values
# (their state is echoed from configuration, never read back from
# SecObserve) -- add them to your configuration to match what's actually
# configured, if anything.
