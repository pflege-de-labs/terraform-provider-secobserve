# Either the numeric id or the exact name works. Products and product groups
# share one name space, so a name identifies at most one of either.
terraform import secobserve_product_group.payments 5
terraform import secobserve_product_group.payments "Payments"
