# Prefer the numeric id: general rule names are not reliably unique, and
# import by name fails with an ambiguity error if more than one rule shares
# the name.
import {
  to = secobserve_general_rule.ignore_dev_dependencies
  id = "5"
}
