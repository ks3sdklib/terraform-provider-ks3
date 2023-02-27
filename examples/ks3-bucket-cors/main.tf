provider "ksyun" {
  alias  = "us-prod"
  region = "us-west-1"
}

resource "ksyun_ks3_bucket" "bucket-cors" {
  provider = ksyun.us-prod

  bucket = var.name
  acl    = var.acl

  cors_rule {
    allowed_origins = [var.allow-origins-star]
    allowed_methods = split(",", var.allow-methods-put)
    allowed_headers = [var.allowed_headers]
  }

  cors_rule {
    allowed_origins = [var.allow-origins-aliyun]
    allowed_methods = split(",", var.allow-methods-get)
    allowed_headers = [var.allowed_headers]
    expose_headers  = [var.expose_headers]
    max_age_seconds = var.max_age_seconds
  }
}

