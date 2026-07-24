terraform {
  required_providers {
    diskbg = {
      source  = "Masumoou/nextcloud"
      version = "~> 0.1"
    }
  }
}

provider "nextcloud" {
  agent_url = "https://wfe-01.example.com:8443"
  # agent_token, haproxy_*, ceph_rgw_* are best set via environment
  # variables instead of hardcoding them here:
  #   DISKBG_AGENT_TOKEN, DISKBG_HAPROXY_URL, DISKBG_HAPROXY_USERNAME,
  #   DISKBG_HAPROXY_PASSWORD, DISKBG_CEPH_RGW_URL,
  #   DISKBG_CEPH_RGW_ACCESS_KEY, DISKBG_CEPH_RGW_SECRET_KEY
}
