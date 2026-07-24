# terraform-provider-nextcloud

A Terraform provider for your Nextcloud platform and its
supporting infrastructure — built from the "Terraform
Provider Vision" design doc. It manages Nextcloud config, LDAP, OnlyOffice,
users, apps, `occ` commands, Redis, HAProxy backends, and Ceph RGW users as
code, instead of hand-run `occ`/config.php edits.

## What's implemented

| Resource / data source | Covers |
|---|---|
| `nextcloud_config` | Maintenance mode, trusted domains/proxies, default language/quota, upload limit, log level, background jobs, trash/versions retention |
| `nextcloud_ldap_config` | Host, base/bind DN, filters, attribute mapping, nested groups, caching, connection test on apply |
| `nextcloud_onlyoffice` | Document Server URL, JWT secret/header, SSL verification, timeout, end-to-end validation on apply (matches the `nextcloud_onlyoffice.main` example in the design doc) |
| `nextcloud_user` | Nextcloud Provisioning API: users, groups, quota, enable/disable |
| `nextcloud_app` | Install/enable/disable an app + `config:app:set` values |
| `nextcloud_occ_command` | Runs an arbitrary `occ` command, re-running via a `triggers` map (e.g. `maintenance:repair`, `files:scan`, `db:add-missing-indices`) |
| `nextcloud_redis_config` | Host/port/password, cache + file-locking toggles |
| `nextcloud_haproxy_backend` | Backend + servers via the HAProxy Data Plane API |
| `nextcloud_ceph_rgw_user` | S3 user, access/secret key, bucket quota, suspension, via the Ceph RGW admin ops API |
| `data.nextcloud_health` | Apache/PHP/Nextcloud/LDAP/PostgreSQL/Redis/Ceph RGW/OnlyOffice health, plus a combined `healthy` bool |

PostgreSQL is intentionally left to the official
[`cyrilgdn/postgresql`](https://registry.terraform.io/providers/cyrilgdn/postgresql)
provider, per the design doc's own recommendation.

## The one assumption: the agent

Nextcloud doesn't expose `occ`/config.php/LDAP/OnlyOffice config over REST,
so this provider assumes a small internal agent runs on each WFE and
provides that surface. **You'll need to build and deploy that agent** — see
[`docs/agent.md`](docs/agent.md) for the exact endpoint contract it needs to
implement. HAProxy and Ceph RGW are talked to directly (no agent needed)
since they already have real APIs.

## Building

Requires Go 1.21+ and normal internet access (this was scaffolded in a
network-restricted sandbox that can reach GitHub but not the Go module
proxy/checksum database, so dependency resolution had to be verified
against source rather than a full `go build` — run the two commands below
in a normal dev environment to fetch dependencies and build cleanly):

```bash
go mod tidy
go build -o bin/terraform-provider-nextcloud .
```

Or, with the included Makefile:

```bash
make build     # builds bin/terraform-provider-nextcloud
make install   # builds and installs into ~/.terraform.d/plugins for local testing
make fmt vet   # formatting + static checks
```

All Go source in this repo has been checked with `gofmt` and parses
cleanly; what wasn't verifiable offline is the third-party dependency graph
(`terraform-plugin-framework` and its transitive deps), which resolves fine
directly against GitHub but needs `sum.golang.org`/`proxy.golang.org` (or
`GOFLAGS=-insecure GONOSUMCHECK=1`/`GOSUMDB=off` as a workaround) for a
fully offline-sandbox build.

## Using it locally without publishing

Add a dev override so Terraform uses your local build instead of fetching
from a registry — put this in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "Masumoou/nextcloud" = "/absolute/path/to/terraform-provider-nextcloud/bin"
  }
  direct {}
}
```

Then `terraform plan`/`apply` in `examples/resources/` will use your local
binary directly (no `terraform init` needed while the override is active).

## Configuration

```hcl
provider "nextcloud" {
  agent_url = "https://wfe-01.example.com:8443" # or DISKBG_AGENT_URL
  # agent_token, haproxy_*, ceph_rgw_* -> set via env vars, see below
}
```

| Provider argument | Env var fallback |
|---|---|
| `agent_url` | `DISKBG_AGENT_URL` |
| `agent_token` | `DISKBG_AGENT_TOKEN` |
| `haproxy_dataplane_url` | `DISKBG_HAPROXY_URL` |
| `haproxy_username` | `DISKBG_HAPROXY_USERNAME` |
| `haproxy_password` | `DISKBG_HAPROXY_PASSWORD` |
| `ceph_rgw_admin_url` | `DISKBG_CEPH_RGW_URL` |
| `ceph_rgw_access_key` | `DISKBG_CEPH_RGW_ACCESS_KEY` |
| `ceph_rgw_secret_key` | `DISKBG_CEPH_RGW_SECRET_KEY` |
| `insecure_skip_verify` | — (staging/self-signed certs only) |

See `examples/resources/full_platform_example.tf` for a full worked example
(Nextcloud config, LDAP, OnlyOffice, Redis, an app install, an HAProxy
backend with two WFE servers, a Ceph RGW user, an `occ maintenance:repair`
run, and a final health-check gate).

## Extending

Every resource follows the same shape: a `client.*Config`/`client.*` struct
in `internal/client/`, and a `resource.Resource` implementation in
`internal/provider/` with `toAPI`/`fromAPI` conversion helpers. The design
doc calls out several sections not yet built out as full resources —
Apache vhosts, PHP-level config (memory limit, OPcache, extensions),
HAProxy frontends/ACLs/SSL certs, disaster-recovery/rebuild-a-WFE
workflows, drift detection, and blue/green or canary deployment
orchestration. Adding any of them means: extend `internal/client/`,
add a `resource_*.go` following the existing pattern, and register it in
`internal/provider/provider.go`'s `Resources()` slice.
