output "bucket-cors" {
  value = ksyun_ks3_bucket.bucket-cors.id
}

output "bucket-cors-rule" {
  value = ksyun_ks3_bucket.bucket-cors.cors_rule
}

