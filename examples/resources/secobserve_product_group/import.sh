# Either the numeric id or the exact name works. Products and product groups
# share one name space, so a name identifies at most one of either.
terraform import secobserve_product_group.payments 5
terraform import secobserve_product_group.payments "Payments"

# Import cannot recover security_gate/repository_branch_housekeeping values
# (their state is echoed from configuration, never read back from
# SecObserve) -- add them to your configuration to match what's actually
# configured, if anything.
