provider "ksyun" {
  alias  = "bj-prod"
  region = "beijing"
}

resource "ksyun_ks3_bucket" "bucket-new" {
  provider = ksyun.bj-prod

  bucket = var.bucket-new
  acl    = var.acl-bj
}

resource "ksyun_ks3_bucket" "bucket-attr" {
  provider = ksyun.bj-prod

  bucket = var.bucket-attr

  website {
    index_document = var.index-doc
    error_document = var.error-doc
  }

  logging {
    target_bucket = ksyun_ks3_bucket.bucket-new.id
    target_prefix = var.target-prefix
  }

  lifecycle_rule {
    id      = var.rule-days
    prefix  = "${var.rule-prefix}/${var.role-days}"
    enabled = true

    expiration {
      days = var.rule-days
    }
  }

  lifecycle_rule {
    id      = var.role-date
    prefix  = "${var.rule-prefix}/${var.role-date}"
    enabled = true

    expiration {
      date = var.rule-date
    }
  }

  referer_config {
    allow_empty = var.allow-empty
    referers    = [var.referers]
  }
}

