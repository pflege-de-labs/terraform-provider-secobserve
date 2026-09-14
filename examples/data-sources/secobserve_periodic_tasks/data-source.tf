# Check whether the license import has failed recently.
data "secobserve_periodic_tasks" "license_import" {
  task   = "Import SPDX licenses"
  status = "Failure"
}

output "recent_license_import_failures" {
  value = length(data.secobserve_periodic_tasks.license_import.tasks)
}
