terraform {
  required_providers {
    nextcloud = {
      source  = "Masumoou/nextcloud"
      version = "~> 0.1"
    }
  }
}

provider "nextcloud" {
  agent_url = "https://wfe-01.example.com:8443"
  # agent_token, haproxy_*, ceph_rgw_* are best set via environment
  # variables instead of hardcoding them here:
  #   NEXTCLOUD_AGENT_TOKEN, NEXTCLOUD_HAPROXY_URL, NEXTCLOUD_HAPROXY_USERNAME,
  #   NEXTCLOUD_HAPROXY_PASSWORD, NEXTCLOUD_CEPH_RGW_URL,
  #   NEXTCLOUD_CEPH_RGW_ACCESS_KEY, NEXTCLOUD_CEPH_RGW_SECRET_KEY
}
