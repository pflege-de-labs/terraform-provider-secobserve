# Running SecObserve with Apple's `container` CLI

An alternative to the Docker Compose stack in this directory, for machines
without Docker: only the SecObserve backend runs in a container, and PostgreSQL
stays on the host as a Homebrew service.

Compose remains the default and is what CI uses. This route exists because it
needs no Docker, at the cost of one piece of host configuration.

`make container-up` does everything in "Start it" below. The rest of this
document explains the one-time setup it depends on and why.

## What you need

- Apple's `container` CLI (verified with 1.3.1) and `container system start`.
- A Homebrew PostgreSQL service. Verified against `postgresql@18`; SecObserve
  1.58 runs Django 6.1 with psycopg 3, so 16 or 18 are both fine. The upstream
  Compose file pins 15.
- Apple silicon needs no emulation: `ghcr.io/secobserve/secobserve-backend`
  publishes a `linux/arm64` image.

## Database

The Homebrew PostgreSQL formulae are keg-only, so their binaries are not on
`PATH` by default.

```bash
export PATH="/opt/homebrew/opt/postgresql@18/bin:$PATH"
createuser --createdb secobserve
psql -d postgres -c "ALTER ROLE secobserve WITH PASSWORD 'secobserve';"
createdb -O secobserve secobserve
```

## Reaching the host from the container

This is the only awkward part, and the reason this document exists.

Apple gives each container an address on `192.168.64.0/24` with the gateway at
`192.168.64.1`. **That gateway is not bound to any host interface** — it lives
inside Apple's vmnet NAT — so a service on your Mac cannot listen there. Nor do
`.local` mDNS names resolve inside a container. That leaves two options.

Either way, **PostgreSQL sees the container's own address as the client**, for
example `192.168.64.8`, so `pg_hba.conf` needs a rule for the container subnet:

```conf
# /opt/homebrew/var/postgresql@18/pg_hba.conf — append
host    all   all   192.168.64.0/24   scram-sha-256
```

```bash
brew services reload postgresql@18    # or: pg_ctl reload
```

Use `scram-sha-256`, not `trust`: `trust` would let anything on the vmnet
connect as any role without a password.

### Option A — Apple's DNS localhost domain (recommended)

```bash
sudo container system dns create host.container.internal --localhost 203.0.113.113
```

Then `DATABASE_HOST=host.container.internal`. Apple redirects that address to
the host's loopback, so **`listen_addresses` can stay at its default of
`localhost`** and PostgreSQL is never exposed to your network. The `pg_hba` rule
above is still required.

Two caveats from Apple's documentation: creating a localhost domain **disables
Private Relay**, and the packet-filter rule is **removed on restart**, so the
command has to be re-run after a reboot.

`container-up.sh` picks this up automatically when `container system dns list`
shows the domain.

### Option B — the host's LAN address

No `sudo` and nothing to re-run after a reboot, but it means opening PostgreSQL
up:

```conf
# /opt/homebrew/var/postgresql@18/postgresql.conf
listen_addresses = 'localhost,192.168.178.72'   # your host's LAN address
```

> **Security:** unlike option A, this makes PostgreSQL reachable from your whole
> local network. The `pg_hba` rule is then the only thing restricting access, so
> do not weaken it. Binding to a specific address also means PostgreSQL fails to
> start once that address changes — a different network or a new DHCP lease.
> `listen_addresses = '*'` avoids the startup failure and widens exposure
> further.

Pass the address explicitly, since auto-detection prefers option A:

```bash
./test/container-up.sh 192.168.178.72
```

## Start it

```bash
make container-up                       # or ./test/container-up.sh [host-address]
eval "$(./test/bootstrap.sh)"           # exports SECOBSERVE_BASE_URL and _API_TOKEN
make testacc
```

`container-up.sh` reads the admin credentials, version and throwaway keys from
`test/.env`, works out how to reach the host, publishes gunicorn's real listen
port (5000, hardcoded in the image and not configurable) as 8000 on the
loopback, and waits for `GET /api/status/health/`. If the container exits
while starting, it prints the container's logs instead of waiting for the
timeout.

Migrations run and the admin user is created on first start, which takes
roughly half a minute.

## Stopping

```bash
make container-down
```

This stops and removes the container. Unlike `make down` for the Compose stack,
it does **not** touch the database: that lives on the host and survives. To
start from a clean database:

```bash
export PATH="/opt/homebrew/opt/postgresql@18/bin:$PATH"
dropdb secobserve && createdb -O secobserve secobserve
```

Worth doing before a full acceptance run: SecObserve performs its SPDX and
ScanCode license import on first start, and the license tests depend on that
state being pristine.

## Troubleshooting

**`no pg_hba.conf entry for host "192.168.64.x"`** — the `pg_hba` rule above is
missing or PostgreSQL has not reloaded. This is the first thing that goes wrong
with either option; the redirect in option A does not make the connection look
like loopback.

**`connection to server at "..." failed: Connection refused`** — PostgreSQL is
not listening on the address the container used. With option A, check the DNS
domain still exists (`container system dns list`); it is dropped on restart.
With option B, check `listen_addresses` includes your current LAN address.

**Container exits immediately** — `container logs secobserve-backend`. A
database problem shows up as a `peewee.OperationalError` from huey's storage
setup during startup.

**Health check times out at 300s but `container logs` shows gunicorn started
cleanly** — check `container logs secobserve-backend | grep Listening`. If it
says `Listening at: http://0.0.0.0:5000`, the container's publish mapping is
wrong: gunicorn's bind port is hardcoded in the image and is not 8000. Confirm
`container inspect secobserve-backend` shows `"containerPort": 5000` under
`publishedPorts`, not `8000`.

**Health check resets (`Recv failure: Connection reset by peer`) against a
container that has been running for a long time** — the app itself may still
be healthy; check `container logs` for recent successful background-task
entries before assuming otherwise. This looks like Apple's port-forward
tunnel going stale, plausibly across host sleep/wake, independent of the app
or the database. `container stop`/`container rm` and a fresh
`./test/container-up.sh` is the fix, not debugging the app.

## Not covered

There is no `container` equivalent of `docker-compose.oidc.yml`. The OIDC tests
need the backend and Keycloak on one network with a pinned `KC_HOSTNAME` so the
token's `iss` claim matches, which Compose already handles. `make testacc-oidc`
is Compose-only.
