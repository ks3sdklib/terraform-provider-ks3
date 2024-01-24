package ksyun

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"log"
	"os"
	"testing"
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"ksyun": testAccProvider,
	}
}

func TestKsyunKs3BucketACL(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		// 资源销毁后校验
		CheckDestroy: rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: bucketACLConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket": "terraform-test-bucket-acl",
						"acl":    "public-read",
					}),
				),
			},
		},
	})
}

func TestKsyunKs3BucketStorageClass(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		// 资源销毁后校验
		CheckDestroy: rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: bucketStorageClassConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket":        "terraform-test-bucket-storage-class",
						"storage_class": "IA",
					}),
				),
			},
		},
	})
}

func TestKsyunKs3BucketPolicy(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		// 资源销毁后校验
		CheckDestroy: rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				// 创建资源配置
				Config: bucketPolicyConfig,
				// 资源创建后校验是否一致
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket": "terraform-test-bucket-policy",
						"acl":    "private",
						"policy": policyStr,
					}),
				),
			},
		},
	})
}

func TestKsyunKs3BucketCORS(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		// 资源销毁后校验
		CheckDestroy: rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: bucketCORSConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket":                        "terraform-test-bucket-cors",
						"acl":                           "private",
						"cors_rule.#":                   "1",
						"cors_rule.0.allowed_headers.0": "*",
					}),
				),
			},
		},
	})
}

func TestKsyunKs3BucketLogging(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		// 资源销毁后校验
		CheckDestroy: rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: bucketLoggingConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket":                  "terraform-test-bucket-logging",
						"acl":                     "private",
						"logging.#":               "1",
						"logging.0.target_bucket": "test-target-bucket",
						"logging.0.target_prefix": "log/",
					}),
				),
			},
		},
	})
}

func TestKsyunKs3BucketLifecycle(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)
	testAccCheck := rac.resourceAttrMapUpdateSet()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		// 资源销毁后校验
		CheckDestroy: rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: bucketLifecycleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket":                             "terraform-test-bucket-lifecycle",
						"acl":                                "private",
						"lifecycle_rule.#":                   "1",
						"lifecycle_rule.0.id":                "id1",
						"lifecycle_rule.0.enabled":           "true",
						"lifecycle_rule.0.expiration.#":      "1",
						"lifecycle_rule.0.expiration.0.date": "2023-04-10",
					}),
				),
			},
		},
	})
}

func TestKsyunKs3BucketLifecycleComplexConfig(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)
	testAccCheck := rac.resourceAttrMapUpdateSet()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		// 资源销毁后校验
		CheckDestroy: rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: bucketLifecycleComplexConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket":                                      "terraform-test-bucket-complex-lifecycle",
						"acl":                                         "private",
						"lifecycle_rule.#":                            "3",
						"lifecycle_rule.0.id":                         "id1",
						"lifecycle_rule.0.enabled":                    "true",
						"lifecycle_rule.0.filter.#":                   "1",
						"lifecycle_rule.0.filter.0.prefix":            "documents",
						"lifecycle_rule.0.expiration.#":               "1",
						"lifecycle_rule.0.expiration.0.date":          "2023-04-10",
						"lifecycle_rule.1.id":                         "id2",
						"lifecycle_rule.1.enabled":                    "true",
						"lifecycle_rule.1.filter.#":                   "1",
						"lifecycle_rule.1.filter.0.prefix":            "logs",
						"lifecycle_rule.1.expiration.#":               "1",
						"lifecycle_rule.1.expiration.0.days":          "130",
						"lifecycle_rule.1.transition.#":               "2",
						"lifecycle_rule.1.transition.0.days":          "10",
						"lifecycle_rule.1.transition.0.storage_class": "STANDARD_IA",
						"lifecycle_rule.1.transition.1.days":          "40",
						"lifecycle_rule.1.transition.1.storage_class": "ARCHIVE",
						"lifecycle_rule.2.id":                         "id3",
						"lifecycle_rule.2.enabled":                    "true",
						"lifecycle_rule.2.filter.#":                   "1",
						"lifecycle_rule.2.filter.0.and.#":             "1",
						"lifecycle_rule.2.filter.0.and.0.prefix":      "docs",
						"lifecycle_rule.2.filter.0.and.0.tag.#":       "2",
						"lifecycle_rule.2.filter.0.and.0.tag.0.key":   "age",
						"lifecycle_rule.2.filter.0.and.0.tag.0.value": "21",
						"lifecycle_rule.2.filter.0.and.0.tag.1.key":   "name",
						"lifecycle_rule.2.filter.0.and.0.tag.1.value": "li",
						"lifecycle_rule.2.expiration.#":               "1",
						"lifecycle_rule.2.expiration.0.date":          "2021-01-01",
					}),
				),
			},
		},
	})
}

const bucketACLConfig = `
resource "ksyun_ks3_bucket" "default"{
  bucket = "terraform-test-bucket-acl"
  acl = "public-read"
}
`

const bucketStorageClassConfig = `
resource "ksyun_ks3_bucket" "default"{
  bucket = "terraform-test-bucket-storage-class"
  storage_class = "IA"
}
`

const bucketPolicyConfig = `
resource "ksyun_ks3_bucket" "default"{
  bucket = "terraform-test-bucket-policy"
  policy = <<-EOT
  {
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "ks3:*"
        ],
        "Principal": {
          "KSC": [
            "*"
          ]
        },
        "Resource": [
          "krn:ksc:ks3:::terraform-test-bucket-policy",
          "krn:ksc:ks3:::terraform-test-bucket-policy/*"
        ]
      }
    ]
  }
  EOT
}
`

const policyStr = `{
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ks3:*"
      ],
      "Principal": {
        "KSC": [
          "*"
        ]
      },
      "Resource": [
        "krn:ksc:ks3:::terraform-test-bucket-policy",
        "krn:ksc:ks3:::terraform-test-bucket-policy/*"
      ]
    }
  ]
}
`

const bucketCORSConfig = `
resource "ksyun_ks3_bucket" "default" {
  bucket = "terraform-test-bucket-cors"
  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET","PUT","DELETE"]
    allowed_origins = ["*"]
  }
}
`

const bucketLoggingConfig = `
resource "ksyun_ks3_bucket" "default" {
  bucket = "terraform-test-bucket-logging"
  logging {
    target_bucket = "test-target-bucket"
    target_prefix = "log/"
  }
}
`

const bucketLifecycleConfig = `
resource "ksyun_ks3_bucket" "default" {
  bucket = "terraform-test-bucket-lifecycle"
  lifecycle_rule {
    id = "id1"
    enabled = true
    expiration {
      date = "2023-04-10"
    }
  }
}
`

const bucketLifecycleComplexConfig = `
resource "ksyun_ks3_bucket" "default" {
  bucket = "terraform-test-bucket-complex-lifecycle"
  lifecycle_rule {
    id = "id1"
    enabled = true
	filter {
	  prefix = "documents"
	}
    expiration {
      date = "2023-04-10"
    }
  }
  lifecycle_rule {
    id = "id2"
    enabled = true
	filter {
	  prefix = "logs"
	}
    expiration {
      days = 130
    }
    transition {
      days = 10
	  storage_class = "STANDARD_IA"
    }
    transition {
      days = 40
	  storage_class = "ARCHIVE"
    }
  }
  lifecycle_rule {
    id = "id3"
    enabled = true
	filter {
	  and {
		prefix = "docs"
		tag {
		  key = "age"
		  value = "21"
		}
		tag {
		  key = "name"
		  value = "li"
		}
	  }
	}
    expiration {
      date = "2021-01-01"
    }
  }
}
`

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("KS3_TEST_ACCESS_KEY_ID"); v == "" {
		t.Fatal("KS3_TEST_ACCESS_KEY_ID must be set for acceptance tests")
	}
	if v := os.Getenv("KS3_TEST_ACCESS_KEY_SECRET"); v == "" {
		t.Fatal("KS3_TEST_ACCESS_KEY_SECRET must be set for acceptance tests")
	}
	if v := os.Getenv("KS3_TEST_REGION"); v == "" {
		log.Println("[INFO] Test: Using BEIJING as test region")
		os.Setenv("KS3_TEST_REGION", "BEIJING")
	} else {
		defaultRegionToTest = v
	}
}
