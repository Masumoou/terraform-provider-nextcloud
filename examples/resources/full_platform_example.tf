# Full example: brings up the platform config on a single WFE the
# way it's described in the "Terraform Provider Vision" doc.

resource "nextcloud_config" "main" {
  maintenance_mode = false
  trusted_domains  = ["example.com", "www.example.com"]
  default_language = "bg"
  default_quota    = "10 GB"
  upload_limit_mb  = 4096
  log_level        = 2
  background_jobs  = "cron"
}

resource "nextcloud_ldap_config" "main" {
  config_id     = "s01"
  host          = "ldaps://ldap.example.com"
  base_dn       = "dc=example,dc=com"
  bind_dn       = "cn=nextcloud,ou=services,dc=example,dc=com"
  bind_password = var.ldap_bind_password

  user_filter   = "(&(objectClass=inetOrgPerson))"
  login_filter  = "(&(objectClass=inetOrgPerson)(|(uid=%uid)(mail=%uid)))"
  nested_groups = true
}

resource "nextcloud_onlyoffice" "main" {
  document_server = "https://office.example.com"
  jwt_secret      = var.onlyoffice_jwt_secret
  verify_ssl      = true
}

resource "nextcloud_redis_config" "main" {
  host                  = "redis-01.example.com"
  port                  = 6379
  password              = var.redis_password
  use_for_cache         = true
  use_for_file_locking  = true
}

resource "nextcloud_app" "onlyoffice" {
  app_id  = "onlyoffice"
  enabled = true

  depends_on = [nextcloud_onlyoffice.main]
}

resource "nextcloud_haproxy_backend" "nextcloud" {
  name    = "bk_nextcloud"
  mode    = "http"
  balance = "leastconn"

  server {
    name    = "wfe-01"
    address = "192.168.1.101"
    port    = 443
    check   = true
  }

  server {
    name    = "wfe-02"
    address = "192.168.1.102"
    port    = 443
    check   = true
  }
}

resource "nextcloud_ceph_rgw_user" "nextcloud_primary" {
  user_id      = "nextcloud-primary"
  display_name = "Nextcloud primary storage user"
  max_buckets  = 10
}

resource "nextcloud_occ_command" "repair" {
  command = "maintenance:repair"

  # Re-run whenever the Nextcloud version changes.
  triggers = {
    nextcloud_version = "30.0.2"
  }

  depends_on = [
    nextcloud_config.main,
    nextcloud_ldap_config.main,
  ]
}

data "nextcloud_health" "check" {
  depends_on = [nextcloud_occ_command.repair]
}

output "platform_healthy" {
  value = data.nextcloud_health.check.healthy
}

variable "ldap_bind_password" {
  type      = string
  sensitive = true
}

variable "onlyoffice_jwt_secret" {
  type      = string
  sensitive = true
}

variable "redis_password" {
  type      = string
  sensitive = true
}
