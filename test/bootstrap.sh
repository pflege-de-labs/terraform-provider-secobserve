#!/usr/bin/env bash
# Waits for the containerized SecObserve to become healthy and mints a
# superuser API token for the provider.
#
# Prints shell exports on stdout so it can be eval'd:
#
#   eval "$(test/bootstrap.sh)"
#
# Pass --format=env to emit bare KEY=value lines instead, for GITHUB_ENV and
# .env files.
#
# Progress and errors go to stderr to keep stdout eval-safe.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
set -a && source "${script_dir}/.env" && set +a

format=shell
for arg in "$@"; do
  case "${arg}" in
    --format=env) format=env ;;
    --format=shell) format=shell ;;
    *) echo "unknown argument: ${arg}" >&2; exit 2 ;;
  esac
done

base_url="${SECOBSERVE_BASE_URL:-http://localhost:${BACKEND_PORT:-8000}}"
token_name="${TOKEN_NAME:-terraform}"
deadline=$((SECONDS + 300))

log() { printf '%s\n' "$*" >&2; }

log "waiting for ${base_url}/api/status/health/"
until curl -fsS -o /dev/null "${base_url}/api/status/health/" 2>/dev/null; do
  if ((SECONDS > deadline)); then
    log "ERROR: SecObserve did not become healthy within 300s"
    log "check the logs with: docker compose -f ${script_dir}/docker-compose.yml logs backend"
    exit 1
  fi
  sleep 2
done
log "backend is healthy"

# The token endpoint takes username and password in the body -- it cannot be
# driven by a token, which is why this is a bootstrap step and not something
# the provider can do for itself.
#
# Revoke first so re-running is idempotent: a second token with the same name
# is rejected with a 400.
curl -fsS -o /dev/null -X POST "${base_url}/api/authentication/revoke_user_api_token/" \
  -H 'Content-Type: application/json' \
  -d "$(printf '{"username":"%s","password":"%s","name":"%s"}' "${ADMIN_USER}" "${ADMIN_PASSWORD}" "${token_name}")" \
  || log "note: no pre-existing token to revoke"

response="$(curl -fsS -X POST "${base_url}/api/authentication/create_user_api_token/" \
  -H 'Content-Type: application/json' \
  -d "$(printf '{"username":"%s","password":"%s","name":"%s"}' "${ADMIN_USER}" "${ADMIN_PASSWORD}" "${token_name}")")"

token="$(printf '%s' "${response}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
if [[ -z "${token}" ]]; then
  log "ERROR: could not parse token out of: ${response}"
  exit 1
fi

log "minted API token '${token_name}' for superuser '${ADMIN_USER}'"
if [[ "${format}" == "env" ]]; then
  printf 'SECOBSERVE_BASE_URL=%s\n' "${base_url}"
  printf 'SECOBSERVE_API_TOKEN=%s\n' "${token}"
else
  printf 'export SECOBSERVE_BASE_URL=%q\n' "${base_url}"
  printf 'export SECOBSERVE_API_TOKEN=%q\n' "${token}"
fi
