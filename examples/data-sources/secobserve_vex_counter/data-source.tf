# Read-only: the counter is incremented by SecObserve itself on every
# CSAF/OpenVEX export, so there is no corresponding resource. Useful for
# auditing the next document id a given prefix/year will get.
data "secobserve_vex_counter" "advisories_2026" {
  document_id_prefix = "ACME-ADV"
  year               = 2026
}

output "next_vex_document_id" {
  value = "${data.secobserve_vex_counter.advisories_2026.year}_${data.secobserve_vex_counter.advisories_2026.counter + 1}"
}
