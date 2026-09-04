# Discover which parsers can be driven by an API import configuration.
data "secobserve_parsers" "api" {
  source = "API"
}

output "api_parser_names" {
  value = [for parser in data.secobserve_parsers.api.parsers : parser.name]
}
