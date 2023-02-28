package ksyun

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
)

func TestAccKsyunKs3BucketObject_basic(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-ks3-object-test-acc-source")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// first write some data to the tempfile just so it's not 0 bytes.
	err = ioutil.WriteFile(tmpFile.Name(), []byte("{anything will do }"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var v http.Header
	resourceId := "ksyun_ks3_bucket_object.default"
	ra := resourceAttrInit(resourceId, ks3BucketObjectBasicMap)
	testAccCheck := ra.resourceAttrMapUpdateSet()
	rand := acctest.RandIntRange(1000000, 9999999)
	name := fmt.Sprintf("tf-testacc-object-%d", rand)
	testAccConfig := resourceTestAccConfigFunc(resourceId, name, resourceKs3BucketObjectConfigDependence)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		IDRefreshName: resourceId,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKsyunKs3BucketObjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(map[string]interface{}{
					"bucket":       "${ksyun_ks3_bucket.default.bucket}",
					"key":          "test-object-source-key",
					"source":       tmpFile.Name(),
					"content_type": "binary/octet-stream",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKsyunKs3BucketObjectExists(
						"ksyun_ks3_bucket_object.default", name, v),
					testAccCheck(map[string]string{
						"bucket": name,
						"source": tmpFile.Name(),
					}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"source":  REMOVEKEY,
					"content": "some words for test ks3 object content",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKsyunKs3BucketObjectExists(
						"ksyun_ks3_bucket_object.default", name, v),
					testAccCheck(map[string]string{
						"source":  REMOVEKEY,
						"content": "some words for test ks3 object content",
					}),
				),
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
					"server_side_encryption": "KMS",
					"kms_key_id":             "${data.ksyun_kms_keys.enabled.ids.0}",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{}),
				),
			},
			{
				Config: testAccConfig(map[string]interface{}{
					"bucket":                 "${ksyun_ks3_bucket.default.bucket}",
					"server_side_encryption": "AES256",
					"kms_key_id":             REMOVEKEY,
					"key":                    "test-object-source-key",
					"content":                REMOVEKEY,
					"source":                 tmpFile.Name(),
					"content_type":           "binary/octet-stream",
					"acl":                    REMOVEKEY,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKsyunKs3BucketObjectExists(
						"ksyun_ks3_bucket_object.default", name, v),
					testAccCheck(map[string]string{
						"bucket":       name,
						"key":          "test-object-source-key",
						"content":      REMOVEKEY,
						"source":       tmpFile.Name(),
						"content_type": "binary/octet-stream",
						"acl":          "private",
					}),
				),
			},
		},
	})
}

func resourceKs3BucketObjectConfigDependence(name string) string {

	return fmt.Sprintf(`
resource "ksyun_ks3_bucket" "default" {
	bucket = "%s"
}
data "ksyun_kms_keys" "enabled" {
	status = "Enabled"
}
`, name)
}

var ks3BucketObjectBasicMap = map[string]string{
	"bucket":       CHECKSET,
	"key":          "test-object-source-key",
	"source":       CHECKSET,
	"content_type": "binary/octet-stream",
	"acl":          "private",
}

func testAccCheckKsyunKs3BucketObjectExists(n string, bucket string, obj http.Header) resource.TestCheckFunc {
	providers := []*schema.Provider{testAccProvider}
	return testAccCheckKs3BucketObjectExistsWithProviders(n, bucket, obj, &providers)
}
func testAccCheckKs3BucketObjectExistsWithProviders(n string, bucket string, obj http.Header, providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		for _, provider := range *providers {
			// Ignore if Meta is empty, this can happen for validation providers
			if provider.Meta() == nil {
				continue
			}
			client := provider.Meta().(*connectivity.KsyunClient)
			raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
				return ks3Client.Bucket(bucket)
			})
			buck, _ := raw.(*ks3.Bucket)
			if err != nil {
				return fmt.Errorf("Error getting bucket: %#v", err)
			}
			object, err := buck.GetObjectMeta(rs.Primary.ID)
			log.Printf("[WARN]get ks3 bucket object %#v", bucket)
			if err == nil {
				if object != nil {
					obj = object
					return nil
				}
				continue
			} else if err != nil {
				return err

			}
		}

		return fmt.Errorf("Bucket not found")
	}
}
func testAccCheckKsyunKs3BucketObjectDestroy(s *terraform.State) error {
	return testAccCheckKs3BucketObjectDestroyWithProvider(s, testAccProvider)
}

func testAccCheckKs3BucketObjectDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	client := provider.Meta().(*connectivity.KsyunClient)
	var bucket *ks3.Bucket
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ksyun_ks3_bucket" {
			continue
		}
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			return ks3Client.Bucket(rs.Primary.ID)
		})
		if err != nil {
			return fmt.Errorf("Error getting bucket: %#v", err)
		}
		bucket, _ = raw.(*ks3.Bucket)
	}
	if bucket == nil {
		return nil
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ksyun_ks3_bucket_object" {
			continue
		}

		// Try to find the resource
		exist, err := bucket.IsObjectExist(rs.Primary.ID)
		if err != nil {
			if IsExpectedErrors(err, []string{"NoSuchBucket"}) {
				return nil
			}
			return fmt.Errorf("IsObjectExist got an error: %#v", err)
		}

		if !exist {
			return nil
		}

		return fmt.Errorf("Found ks3 object: %s", rs.Primary.ID)
	}

	return nil
}
