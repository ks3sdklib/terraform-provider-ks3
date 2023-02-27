output "bucket-new" {
  value = ksyun_ks3_bucket.bucket-new.id
}

output "bucket-attr" {
  value = ksyun_ks3_bucket.bucket-attr.id
}

output "bucket-attr-website" {
  value = ksyun_ks3_bucket.bucket-attr.website
}

output "bucket-attr-logging" {
  value = ksyun_ks3_bucket.bucket-attr.logging
}

output "bucket-attr-lifecycle" {
  value = ksyun_ks3_bucket.bucket-attr.lifecycle_rule
}

output "bucket-attr-referers" {
  value = ksyun_ks3_bucket.bucket-attr.referer_config
}

