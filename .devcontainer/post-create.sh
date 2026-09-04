#!/usr/bin/env bash
# One-time setup of the development container.
set -euo pipefail

cd /workspace

export GOWORK=off

echo "==> downloading Go modules"
go mod download

echo "==> installing pinned tools"
make tools

echo "==> configuring dev_overrides"
# GOBIN wins when set (mise, asdf and friends set it); GOPATH/bin otherwise.
bindir="$(go env GOBIN)"
[ -n "${bindir}" ] || bindir="$(go env GOPATH)/bin"

# With dev_overrides Terraform uses the locally built binary directly, so the
# examples work without `terraform init` and without a published provider.
cat > "${HOME}/.terraformrc" <<TERRAFORMRC
provider_installation {
  dev_overrides {
    "jabbrwcky/secobserve" = "${bindir}"
  }
  direct {}
}
TERRAFORMRC

echo "==> building and installing the provider"
make install

cat <<'MESSAGE'

Ready. Next steps:

  make up                        start the containerized SecObserve
  eval "$(./test/bootstrap.sh)"  export SECOBSERVE_BASE_URL / _API_TOKEN
  make test                      unit tests
  make testacc                   acceptance tests

MESSAGE
