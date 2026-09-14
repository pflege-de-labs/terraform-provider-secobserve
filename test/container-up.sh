#!/usr/bin/env bash
# Starts the SecObserve backend with Apple's `container` CLI, against a
# PostgreSQL running on the macOS host (typically a Homebrew service).
#
# The Docker Compose stack in this directory is self-contained; this script is
# the alternative for people who run `container` and keep Postgres on the host.
# See apple-container.md for the setup it expects and why.
#
#   ./test/container-up.sh                     # auto-detect how to reach the host
#   ./test/container-up.sh 192.168.178.72      # or name the host address
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
set -a && source "${script_dir}/.env" && set +a

container_name="${CONTAINER_NAME:-secobserve-backend}"
host_port="${BACKEND_PORT:-8000}"
database_db="${DATABASE_DB:-secobserve}"
database_user="${DATABASE_USER:-secobserve}"
database_password="${DATABASE_PASSWORD:-secobserve}"
dns_domain="host.container.internal"

log() { printf '%s\n' "$*" >&2; }

# How the container reaches the host. Apple's vmnet gateway (192.168.64.1) is
# internal to the NAT and cannot be bound on the host, so there are only two
# options: the DNS localhost domain, which redirects to the host loopback, or
# the host's own LAN address.
detect_host_address() {
  if [[ $# -ge 1 && -n "${1}" ]]; then
    printf '%s' "${1}"
    return
  fi
  if container system dns list 2>/dev/null | grep -qw "${dns_domain}"; then
    printf '%s' "${dns_domain}"
    return
  fi
  for interface in en0 en10 en1; do
    local address
    address="$(ipconfig getifaddr "${interface}" 2>/dev/null || true)"
    if [[ -n "${address}" ]]; then
      printf '%s' "${address}"
      return
    fi
  done
  return 1
}

if ! database_host="$(detect_host_address "${1:-}")"; then
  log "ERROR: could not work out how the container should reach the host."
  log ""
  log "Either create the DNS domain once (needs sudo, and disables Private Relay):"
  log "  sudo container system dns create ${dns_domain} --localhost 203.0.113.113"
  log "or pass the host's LAN address explicitly:"
  log "  ./test/container-up.sh 192.168.178.72"
  exit 1
fi

if [[ "${database_host}" == "${dns_domain}" ]]; then
  log "reaching the host via ${dns_domain} (Postgres can stay on loopback)"
else
  log "reaching the host at ${database_host}"
  log "note: Postgres must listen on that address and pg_hba must allow the"
  log "      container subnet 192.168.64.0/24 -- see test/apple-container.md"
fi

container system start >/dev/null 2>&1 || true

if container list --all --quiet 2>/dev/null | grep -qx "${container_name}"; then
  log "removing the existing ${container_name} container"
  container stop "${container_name}" >/dev/null 2>&1 || true
  container rm "${container_name}" >/dev/null 2>&1 || true
fi

# Written outside the repository and removed immediately: it holds the
# throwaway keys from .env and is only needed while the container starts.
env_dir="$(mktemp -d)"
trap 'rm -rf "${env_dir}"' EXIT
env_file="${env_dir}/secobserve"

cat > "${env_file}" <<ENVFILE
ADMIN_USER=${ADMIN_USER}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
ADMIN_EMAIL=${ADMIN_EMAIL}

GUNICORN_WORKERS=2
GUNICORN_THREADS=4

DATABASE_ENGINE=django.db.backends.postgresql
DATABASE_HOST=${database_host}
DATABASE_PORT=${DATABASE_PORT:-5432}
DATABASE_DB=${database_db}
DATABASE_USER=${database_user}
DATABASE_PASSWORD=${database_password}

ALLOWED_HOSTS=localhost,127.0.0.1
CORS_ALLOWED_ORIGINS=http://localhost:3000

DJANGO_SECRET_KEY=${DJANGO_SECRET_KEY}
FIELD_ENCRYPTION_KEY=${FIELD_ENCRYPTION_KEY}

OIDC_AUTHORITY=
OIDC_CLIENT_ID=
OIDC_USERNAME=
OIDC_FIRST_NAME=
OIDC_LAST_NAME=
OIDC_FULL_NAME=
OIDC_EMAIL=
OIDC_GROUPS=
ENVFILE

log "starting ${container_name} (SecObserve ${SECOBSERVE_VERSION})"
# gunicorn's bind address is hardcoded to 0.0.0.0:5000 in the image's
# entrypoint (docker/backend/prod/django/entrypoint:run_server), not 8000 --
# host_port only picks the host side of the publish mapping.
container run --detach \
  --name "${container_name}" \
  --env-file "${env_file}" \
  --publish "${host_port}:5000" \
  "ghcr.io/secobserve/secobserve-backend:${SECOBSERVE_VERSION}" >/dev/null

base_url="http://localhost:${host_port}"
deadline=$((SECONDS + 300))

log "waiting for ${base_url}/api/status/health/"
until curl -fsS -o /dev/null "${base_url}/api/status/health/" 2>/dev/null; do
  if ! container list --quiet 2>/dev/null | grep -qx "${container_name}"; then
    log "ERROR: the container exited. Logs:"
    container logs "${container_name}" 2>&1 | tail -30 >&2
    exit 1
  fi
  if ((SECONDS > deadline)); then
    log "ERROR: SecObserve did not become healthy within 300s"
    log "check the logs with: container logs ${container_name}"
    exit 1
  fi
  sleep 2
done

log "backend is healthy"
log ""
log "next: eval \"\$(SECOBSERVE_BASE_URL=${base_url} ./test/bootstrap.sh)\""
