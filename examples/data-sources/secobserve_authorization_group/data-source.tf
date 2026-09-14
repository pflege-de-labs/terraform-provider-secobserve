data "secobserve_authorization_group" "platform_team" {
  name = "Platform Team"
}

output "platform_team_manages_itself" {
  value = data.secobserve_authorization_group.platform_team.is_manager
}
