terraform {
  required_providers {
    nextcloud = {
      source = "Masumoou/nextcloud"
    }
  }
}

provider "nextcloud" {
  agent_url              = "http://localhost"
  haproxy_dataplane_url  = "http://192.168.1.100:5555/v3"
  haproxy_username       = "admin"
  haproxy_password = ""
}

resource "nextcloud_haproxy_backend" "test" {
  name    = "tf-test-backend"
  mode    = "http"
  balance = "roundrobin"

  server {
    name    = "test-server-1"
    address = "192.0.2.1"
    port    = 8080
    check   = true
  }
}
