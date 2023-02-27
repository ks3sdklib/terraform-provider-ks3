provider "ksyun" {
  alias  = "bj-prod"
  region = "cn-beijing"
}

resource "ksyun_ks3_bucket" "bucket-new" {
  bucket = var.bucket-new
  acl    = var.acl
}

resource "ksyun_ks3_bucket_object" "content" {
  bucket  = ksyun_ks3_bucket.bucket-new.bucket
  key     = var.object-key
  content = var.object-content
}

