terraform {
  required_providers {
    diskbg = {
      source = "diskbg/diskbg"
    }
  }
}

provider "diskbg" {
  agent_url           = "http://localhost"
  ceph_rgw_admin_url  = "http://192.168.1.100:8081"
  ceph_rgw_access_key = "DUMMY_ACCESS_KEY"
  ceph_rgw_secret_key = "DUMMY_SECRET_KEY"
}

resource "nextcloud_ceph_rgw_user" "test" {
  user_id      = "tf-test-user"
  display_name = "Terraform Test User"
}
