# Nextcloud Agent Contract

Nextcloud's `occ`, `config.php`, and the LDAP/OnlyOffice app config don't have
a first-class REST API. To make all of that Terraform-manageable, this
provider assumes a small internal service — `nextcloud-agent` — runs on each WFE
and exposes the endpoints below. It's intentionally thin: mostly a wrapper
around `occ`, direct config.php edits, and a couple of active health probes.
This is the one piece of the design you have to build and deploy yourself;
everything else in this repo (the provider) talks to it.

Auth: Bearer token (`Authorization: Bearer <token>`), configured via
`agent_token` / `DISKBG_AGENT_TOKEN`.

| Method | Path | Purpose |
|---|---|---|
| GET/PUT | `/api/v1/nextcloud/config` | config.php-level settings (maintenance mode, trusted domains/proxies, quota, upload limit, log level, background jobs, retention) |
| GET/PUT/DELETE | `/api/v1/ldap/{config_id}` | LDAP config prefix (host, filters, attribute mapping, nested groups, caching) — wraps `occ ldap:set-config` |
| POST | `/api/v1/ldap/{config_id}/test` | Runs `occ ldap:test-config` and returns pass/fail + message |
| GET/PUT | `/api/v1/onlyoffice/config` | OnlyOffice Document Server URL, JWT secret/header, SSL verification, timeout |
| POST | `/api/v1/onlyoffice/validate` | End-to-end check: Document Server reachable, JWT secrets match, Nextcloud↔Document Server round trip, test document opens |
| GET/POST/PUT/DELETE | `/api/v1/users[/{id}]` | Nextcloud Provisioning API passthrough (users, groups, quota, enable/disable) |
| GET/PUT/DELETE | `/api/v1/apps/{id}` | `occ app:install` / `app:enable` / `app:disable` / `config:app:set` |
| POST | `/api/v1/occ/exec` | Runs an arbitrary `occ` command, returns exit code + stdout/stderr |
| GET/PUT | `/api/v1/redis/config` | Redis host/port/password, cache + file-locking toggles |
| GET | `/api/v1/health` | Apache, PHP, Nextcloud, LDAP, PostgreSQL, Redis, Ceph RGW, OnlyOffice health booleans |

HAProxy and Ceph RGW are **not** proxied through the agent — the provider
talks to the HAProxy Data Plane API and the Ceph RGW admin ops API directly,
since both already expose real REST surfaces (see `haproxy_dataplane_url`
and `ceph_rgw_admin_url` in the provider config).

PostgreSQL is deliberately **not** implemented here — use the official
[`cyrilgdn/postgresql`](https://registry.terraform.io/providers/cyrilgdn/postgresql)
provider for databases/roles/grants/extensions, as noted in the original
design doc.
