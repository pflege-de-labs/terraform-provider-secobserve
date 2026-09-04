# Resolve a parser name to the numeric id that api_configuration and the rule
# resources expect.
data "secobserve_parser" "dependency_track" {
  name = "Dependency Track"
}

output "parser_id" {
  value = data.secobserve_parser.dependency_track.id
}
