# Either the numeric id or the exact name works, but general rule names are
# not reliably unique (product is always null, so PostgreSQL doesn't enforce
# uniqueness across them) -- import by name fails with an ambiguity error if
# more than one rule shares the name. Prefer the id.
terraform import secobserve_general_rule.ignore_dev_dependencies 5
