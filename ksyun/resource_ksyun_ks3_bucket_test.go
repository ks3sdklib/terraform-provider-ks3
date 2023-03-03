package ksyun

import (
	"fmt"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"strconv"
	"strings"
	"time"
)

func init() {
	resource.AddTestSweepers("ksyun_ks3_bucket", &resource.Sweeper{
		Name: "ksyun_ks3_bucket",
		F:    testSweepKS3Buckets,
	})
}

// sharedClientForRegion returns a common AlicloudClient setup needed for the sweeper
// functions for a give n region
func sharedClientForRegion() (interface{}, error) {
	var accessKey, secretKey string
	if accessKey = os.Getenv("KS3_ACCESS_KEY_ID"); accessKey == "" {
		return nil, fmt.Errorf("empty KS3_ACCESS_KEY_ID")
	}

	if secretKey = os.Getenv("KS3_ACCESS_KEY_SECRET"); secretKey == "" {
		return nil, fmt.Errorf("empty KS3_ACCESS_KEY_SECRET")
	}
	region := os.Getenv("KS3_REGION")
	conf := connectivity.Config{
		Region:    connectivity.Region(region),
		AccessKey: accessKey,
		SecretKey: secretKey,
		Protocol:  "HTTP",
	}
	// configures a default client for the region, using the above env vars
	client, err := conf.Client()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func testSweepKS3Buckets(string) error {
	rawClient, err := sharedClientForRegion()
	if err != nil {
		return fmt.Errorf("error getting KSYUN client: %s", err)
	}
	client := rawClient.(*connectivity.KsyunClient)

	prefixes := []string{
		"tf-testacc",
		"tf-test-",
	}

	var options []ks3.Option
	options = append(options, ks3.MaxKeys(1000))

	raw, err := client.WithKs3Client(func(ossClient *ks3.Client) (interface{}, error) {
		return ossClient.ListBuckets(options...)
	})
	if err != nil {
		return fmt.Errorf("Error retrieving OSS buckets: %s", err)
	}
	resp, _ := raw.(ks3.ListBucketsResult)
	sweeped := false

	for _, v := range resp.Buckets {
		name := v.Name
		skip := true
		for _, prefix := range prefixes {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				skip = false
				break
			}
		}
		if skip {
			log.Printf("[INFO] Skipping OSS bucket: %s", name)
			continue
		}
		sweeped = true
		raw, err := client.WithKs3Client(func(ossClient *ks3.Client) (interface{}, error) {
			return ossClient.Bucket(name)
		})
		if err != nil {
			return fmt.Errorf("Error getting bucket (%s): %#v", name, err)
		}
		bucket, _ := raw.(*ks3.Bucket)
		if objects, err := bucket.ListObjects(options...); err != nil {
			log.Printf("[ERROR] Failed to list objects: %s", err)
		} else if len(objects.Objects) > 0 {
			for _, o := range objects.Objects {
				if err := bucket.DeleteObject(o.Key); err != nil {
					log.Printf("[ERROR] Failed to delete object (%s): %s.", o.Key, err)
				}
			}

		}

		log.Printf("[INFO] Deleting OSS bucket: %s", name)

		_, err = client.WithKs3Client(func(ossClient *ks3.Client) (interface{}, error) {
			return nil, ossClient.DeleteBucket(name)
		})
		if err != nil {
			log.Printf("[ERROR] Failed to delete OSS bucket (%s): %s", name, err)
		}
	}
	if sweeped {
		time.Sleep(5 * time.Second)
	}
	return nil
}

func TestKsyunKS3BucketBasic(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	rand := acctest.RandIntRange(1000000, 9999999)
	name := fmt.Sprintf("tf-testacc-bucket-%d", rand)
	testAccConfig := resourceTestAccConfigFunc(resourceId, name, resourceOssBucketConfigDependence)
	hashcode1 := strconv.Itoa(expirationHash(map[string]interface{}{
		"days":                         365,
		"date":                         "",
		"created_before_date":          "",
		"expired_object_delete_marker": false,
	}))
	hashcode2 := strconv.Itoa(expirationHash(map[string]interface{}{
		"days":                         0,
		"date":                         "2018-01-12",
		"created_before_date":          "",
		"expired_object_delete_marker": false,
	}))
	hashcode3 := strconv.Itoa(transitionsHash(map[string]interface{}{
		"days":                3,
		"created_before_date": "",
		"storage_class":       "IA",
	}))
	hashcode4 := strconv.Itoa(transitionsHash(map[string]interface{}{
		"days":                30,
		"created_before_date": "",
		"storage_class":       "Archive",
	}))
	hashcode5 := strconv.Itoa(transitionsHash(map[string]interface{}{
		"days":                0,
		"created_before_date": "2023-11-11",
		"storage_class":       "IA",
	}))
	hashcode6 := strconv.Itoa(transitionsHash(map[string]interface{}{
		"days":                0,
		"created_before_date": "2023-11-10",
		"storage_class":       "Archive",
	}))
	hashcode7 := strconv.Itoa(expirationHash(map[string]interface{}{
		"days":                         0,
		"date":                         "",
		"created_before_date":          "2018-01-12",
		"expired_object_delete_marker": false,
	}))
	hashcode8 := strconv.Itoa(abortMultipartUploadHash(map[string]interface{}{
		"days":                0,
		"created_before_date": "2018-01-22",
	}))
	hashcode9 := strconv.Itoa(abortMultipartUploadHash(map[string]interface{}{
		"days":                10,
		"created_before_date": "",
	}))
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		// module name
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		CheckDestroy:  rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(map[string]interface{}{
					"bucket": name,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket": name,
					}),
				),
			},
			{
				ResourceName:            resourceId,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"acl": "public-read",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"acl": "public-read",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"acl": "public-read-write",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"acl": "public-read-write",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"cors_rule": []map[string]interface{}{
						{
							"allowed_origins": []string{"*"},
							"allowed_methods": []string{"PUT", "GET"},
							"allowed_headers": []string{"authorization"},
						},
						{
							"allowed_origins": []string{"http://www.a.com", "http://www.b.com"},
							"allowed_methods": []string{"GET"},
							"allowed_headers": []string{"authorization"},
							"expose_headers":  []string{"x-oss-test", "x-oss-test1"},
							"max_age_seconds": "100",
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"cors_rule.#":                   "2",
						"cors_rule.0.allowed_headers.0": "authorization",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"website": []map[string]interface{}{
						{
							"index_document": "index.html",
							"error_document": "error.html",
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"website.#": "1",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"logging": []map[string]interface{}{
						{
							"target_bucket": "${ksyun_ks3_bucket.target.id}",
							"target_prefix": "log/",
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"logging.#": "1",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"referer_config": []map[string]interface{}{
						{
							"allow_empty": "false",
							"referers": []string{
								"http://www.ksyun.com",
								"https://www.ksyun.com",
							},
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"referer_config.#": "1",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"lifecycle_rule": []map[string]interface{}{
						{
							"id":      "rule1",
							"prefix":  "path1/",
							"enabled": "true",
							"expiration": []map[string]string{
								{
									"days": "365",
								},
							},
						},
						{
							"id":      "rule2",
							"prefix":  "path2/",
							"enabled": "true",
							"expiration": []map[string]string{
								{
									"date": "2018-01-12",
								},
							},
						},
						{
							"id":      "rule3",
							"prefix":  "path3/",
							"enabled": "true",
							"transitions": []map[string]interface{}{
								{
									"days":          "3",
									"storage_class": "IA",
								},
								{
									"days":          "30",
									"storage_class": "Archive",
								},
							},
						},
						{
							"id":      "rule4",
							"prefix":  "path4/",
							"enabled": "true",
							"transitions": []map[string]interface{}{
								{
									"created_before_date": "2023-11-11",
									"storage_class":       "IA",
								},
								{
									"created_before_date": "2023-11-10",
									"storage_class":       "Archive",
								},
							},
						},
						{
							"id":      "rule5",
							"prefix":  "path5/",
							"enabled": "true",
							"expiration": []map[string]string{
								{
									"created_before_date": "2018-01-12",
								},
							},
							"abort_multipart_upload": []map[string]string{
								{
									"created_before_date": "2018-01-22",
								},
							},
						},
						{
							"id":      "rule6",
							"prefix":  "path6/",
							"enabled": "true",
							"abort_multipart_upload": []map[string]string{
								{
									"days": "10",
								},
							},
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"lifecycle_rule.#":                                   "6",
						"lifecycle_rule.0.id":                                "rule1",
						"lifecycle_rule.0.prefix":                            "path1/",
						"lifecycle_rule.0.enabled":                           "true",
						"lifecycle_rule.0.expiration." + hashcode1 + ".days": "365",
						"lifecycle_rule.1.id":                                "rule2",
						"lifecycle_rule.1.prefix":                            "path2/",
						"lifecycle_rule.1.enabled":                           "true",
						"lifecycle_rule.1.expiration." + hashcode2 + ".date": "2018-01-12",

						"lifecycle_rule.2.id":                                          "rule3",
						"lifecycle_rule.2.prefix":                                      "path3/",
						"lifecycle_rule.2.enabled":                                     "true",
						"lifecycle_rule.2.transitions." + hashcode3 + ".days":          "3",
						"lifecycle_rule.2.transitions." + hashcode3 + ".storage_class": string(ks3.StorageIA),
						"lifecycle_rule.2.transitions." + hashcode4 + ".days":          "30",
						"lifecycle_rule.2.transitions." + hashcode4 + ".storage_class": string(ks3.StorageArchive),

						"lifecycle_rule.3.id":      "rule4",
						"lifecycle_rule.3.prefix":  "path4/",
						"lifecycle_rule.3.enabled": "true",
						"lifecycle_rule.3.transitions." + hashcode5 + ".created_before_date": "2023-11-11",
						"lifecycle_rule.3.transitions." + hashcode5 + ".storage_class":       string(ks3.StorageIA),
						"lifecycle_rule.3.transitions." + hashcode6 + ".created_before_date": "2023-11-10",
						"lifecycle_rule.3.transitions." + hashcode6 + ".storage_class":       string(ks3.StorageArchive),

						"lifecycle_rule.4.id":      "rule5",
						"lifecycle_rule.4.prefix":  "path5/",
						"lifecycle_rule.4.enabled": "true",
						"lifecycle_rule.4.expiration." + hashcode7 + ".created_before_date":             "2018-01-12",
						"lifecycle_rule.4.abort_multipart_upload." + hashcode8 + ".created_before_date": "2018-01-22",

						"lifecycle_rule.5.id":      "rule6",
						"lifecycle_rule.5.prefix":  "path6/",
						"lifecycle_rule.5.enabled": "true",
						"lifecycle_rule.5.abort_multipart_upload." + hashcode9 + ".days": "10",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"policy": `{\"Statement\":[{\"Action\":[\"oss:*\"],\"Effect\":\"Allow\",\"Resource\":[\"acs:oss:*:*:*\"]}],\"Version\":\"1\"}`,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(nil),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"tags": map[string]string{
						"key1": "value1",
						"Key2": "Value2",
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"tags.%":    "2",
						"tags.key1": "value1",
						"tags.Key2": "Value2",
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"tags": map[string]string{
						"key1-update": "value1-update",
						"Key2-update": "Value2-update",
						"key3-new":    "value3-new",
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"tags.%":           "3",
						"tags.key1-update": "value1-update",
						"tags.Key2-update": "Value2-update",
						"tags.key3-new":    "value3-new",
						"tags.key1":        REMOVEKEY,
						"tags.Key2":        REMOVEKEY,
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"acl":            "public-read",
					"cors_rule":      REMOVEKEY,
					"tags":           REMOVEKEY,
					"website":        REMOVEKEY,
					"logging":        REMOVEKEY,
					"referer_config": REMOVEKEY,
					"lifecycle_rule": REMOVEKEY,
					"policy":         REMOVEKEY,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"acl":                           "public-read",
						"cors_rule.#":                   "0",
						"cors_rule.0.allowed_headers.0": REMOVEKEY,
						"website.#":                     "0",
						"logging.#":                     "0",
						"referer_config.#":              "0",
						"lifecycle_rule.#":              "0",
						"lifecycle_rule.0.id":           REMOVEKEY,
						"lifecycle_rule.0.prefix":       REMOVEKEY,
						"lifecycle_rule.0.enabled":      REMOVEKEY,
						"lifecycle_rule.0.expiration." + hashcode1 + ".days": REMOVEKEY,
						"lifecycle_rule.1.id":                                REMOVEKEY,
						"lifecycle_rule.1.prefix":                            REMOVEKEY,
						"lifecycle_rule.1.enabled":                           REMOVEKEY,
						"lifecycle_rule.1.expiration." + hashcode2 + ".date": REMOVEKEY,

						"lifecycle_rule.2.id":                                          REMOVEKEY,
						"lifecycle_rule.2.prefix":                                      REMOVEKEY,
						"lifecycle_rule.2.enabled":                                     REMOVEKEY,
						"lifecycle_rule.2.transitions." + hashcode3 + ".days":          REMOVEKEY,
						"lifecycle_rule.2.transitions." + hashcode3 + ".storage_class": REMOVEKEY,
						"lifecycle_rule.2.transitions." + hashcode4 + ".days":          REMOVEKEY,
						"lifecycle_rule.2.transitions." + hashcode4 + ".storage_class": REMOVEKEY,

						"lifecycle_rule.3.id":      REMOVEKEY,
						"lifecycle_rule.3.prefix":  REMOVEKEY,
						"lifecycle_rule.3.enabled": REMOVEKEY,
						"lifecycle_rule.3.transitions." + hashcode5 + ".created_before_date": REMOVEKEY,
						"lifecycle_rule.3.transitions." + hashcode5 + ".storage_class":       REMOVEKEY,
						"lifecycle_rule.3.transitions." + hashcode6 + ".created_before_date": REMOVEKEY,
						"lifecycle_rule.3.transitions." + hashcode6 + ".storage_class":       REMOVEKEY,

						"lifecycle_rule.4.id":      REMOVEKEY,
						"lifecycle_rule.4.prefix":  REMOVEKEY,
						"lifecycle_rule.4.enabled": REMOVEKEY,
						"lifecycle_rule.4.expiration." + hashcode7 + ".created_before_date":             REMOVEKEY,
						"lifecycle_rule.4.abort_multipart_upload." + hashcode8 + ".created_before_date": REMOVEKEY,

						"lifecycle_rule.5.id":      REMOVEKEY,
						"lifecycle_rule.5.prefix":  REMOVEKEY,
						"lifecycle_rule.5.enabled": REMOVEKEY,
						"lifecycle_rule.5.abort_multipart_upload." + hashcode9 + ".days": REMOVEKEY,

						"tags.%":           "0",
						"tags.key1-update": REMOVEKEY,
						"tags.Key2-update": REMOVEKEY,
						"tags.key3-new":    REMOVEKEY,
					}),
				),
			},
		},
	})
}

func resourceOssBucketConfigDependence(name string) string {
	return fmt.Sprintf(`
resource "ksyun_ks3_bucket" "target"{
	bucket = "%s-t"
}
`, name)
}

func TestAccAlicloudOssBucketBasic1(t *testing.T) {
	var v ks3.GetBucketInfoResult

	resourceId := "ksyun_ks3_bucket.default"
	ra := resourceAttrInit(resourceId, ks3BucketBasicMap)

	serviceFunc := func() interface{} {
		return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	}
	rc := resourceCheckInit(resourceId, &v, serviceFunc)

	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	rand := acctest.RandIntRange(1000000, 9999999)
	name := fmt.Sprintf("tf-testacc-bucket-%d", rand)
	testAccConfig := resourceTestAccConfigFunc(resourceId, name, resourceKs3BucketConfigBasic)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		// module name
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		CheckDestroy:  rac.checkResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(map[string]interface{}{
					"bucket": name,
					"acl":    "public-read",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"bucket": name,
						"acl":    "public-read",
					}),
				),
			},
			{
				ResourceName:            resourceId,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
		},
	})
}

func resourceKs3BucketConfigBasic(name string) string {
	return fmt.Sprintf("")
}
