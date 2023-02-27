package ksyun

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"log"
	"os"
	"testing"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider
var defaultRegionToTest = os.Getenv("KS3_TEST_REGION")

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"ksyun": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("KS3_TEST_ACCESS_KEY_ID"); v == "" {
		t.Fatal("KS3_TEST_ACCESS_KEY_ID must be set for acceptance tests")
	}
	if v := os.Getenv("KS3_TEST_ACCESS_KEY_SECRET"); v == "" {
		t.Fatal("KS3_TEST_ACCESS_KEY_SECRET must be set for acceptance tests")
	}
	if v := os.Getenv("KS3_TEST_REGION"); v == "" {
		log.Println("[INFO] Test: Using cn-beijing as test region")
		os.Setenv("KS3_TEST_REGION", "cn-beijing")
	} else {
		defaultRegionToTest = v
	}
}

// currently not all account site type support create PostPaid resources, PayByBandwidth and other limits.
// The setting of account site type can skip some unsupported cases automatically.

// Skip automatically the testcases which does not support some known regions.
// If supported is true, the regions should a list of supporting the service regions.
// If supported is false, the regions should a list of unsupporting the service regions.
// If the region is unsupported and has backend region, the backend region will instead
func testAccPreCheckWithRegions(t *testing.T, supported bool, regions []connectivity.Region) {
	if v := os.Getenv("KS3_TEST_ACCESS_KEY_ID"); v == "" {
		t.Fatal("KS3_TEST_ACCESS_KEY_ID must be set for acceptance tests")
	}
	if v := os.Getenv("KS3_TEST_ACCESS_KEY_SECRET"); v == "" {
		t.Fatal("KS3_TEST_ACCESS_KEY_SECRET must be set for acceptance tests")
	}
	if v := os.Getenv("KS3_TEST_REGION"); v == "" {
		t.Logf("[WARNING] The region is not set and using cn-beijing as test region")
		os.Setenv("KS3_TEST_REGION", "cn-beijing")
	}
	checkoutSupportedRegions(t, supported, regions)
}

func checkoutSupportedRegions(t *testing.T, supported bool, regions []connectivity.Region) {
	region := os.Getenv("KS3_TEST_REGION")
	find := false
	backupRegion := string(connectivity.APSouthEast1)
	if region == string(connectivity.APSouthEast1) {
		backupRegion = string(connectivity.EUCentral1)
	}

	checkoutRegion := os.Getenv("CHECKOUT_REGION")
	if checkoutRegion == "true" {
		if region == string(connectivity.Hangzhou) {
			region = string(connectivity.EUCentral1)
			os.Setenv("KS3_TEST_REGION", region)
		}
	}
	backupRegionFind := false
	hangzhouRegionFind := false
	for _, r := range regions {
		if region == string(r) {
			find = true
			break
		}
		if string(r) == backupRegion {
			backupRegionFind = true
		}
		if string(connectivity.Hangzhou) == string(r) {
			hangzhouRegionFind = true
		}
	}

	if (find && !supported) || (!find && supported) {
		if supported {
			if backupRegionFind {
				t.Logf("Skipping unsupported region %s. Supported regions: %s. Using %s as this test region", region, regions, backupRegion)
				os.Setenv("KS3_TEST_REGION", backupRegion)
				defaultRegionToTest = backupRegion
				return
			}
			if hangzhouRegionFind {
				t.Logf("Skipping unsupported region %s. Supported regions: %s. Using %s as this test region", region, regions, connectivity.Hangzhou)
				os.Setenv("KS3_TEST_REGION", string(connectivity.Hangzhou))
				defaultRegionToTest = string(connectivity.Hangzhou)
				return
			}
			t.Skipf("Skipping unsupported region %s. Supported regions: %s.", region, regions)
		} else {
			if !backupRegionFind {
				t.Logf("Skipping unsupported region %s. Unsupported regions: %s. Using %s as this test region", region, regions, backupRegion)
				os.Setenv("KS3_TEST_REGION", backupRegion)
				defaultRegionToTest = backupRegion
				return
			}
			if !hangzhouRegionFind {
				t.Logf("Skipping unsupported region %s. Supported regions: %s. Using %s as this test region", region, regions, connectivity.Hangzhou)
				os.Setenv("KS3_TEST_REGION", string(connectivity.Hangzhou))
				defaultRegionToTest = string(connectivity.Hangzhou)
				return
			}
			t.Skipf("Skipping unsupported region %s. Unsupported regions: %s.", region, regions)
		}
		t.Skipped()
	}
}

// Skip automatically the sweep testcases which does not support some known regions.
// If supported is true, the regions should a list of supporting the service regions.
// If supported is false, the regions should a list of unsupporting the service regions.

func testAccCheckAlicloudDataSourceID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("data source ID not set")
		}
		return nil
	}
}

var providerCommon = `
provider "ksyun" {
	assume_role {}
}
`

func TestAccAlicloudProviderOss(t *testing.T) {
	//var v ks3.GetBucketInfoResult

	//resourceId := "ksyun_ks3_bucket.default"
	//ra := resourceAttrInit(resourceId, ks3BucketBasicMap)
	//
	//serviceFunc := func() interface{} {
	//	return &Ks3Service{testAccProvider.Meta().(*connectivity.KsyunClient)}
	//}
	//rc := resourceCheckInit(resourceId, &v, serviceFunc)
	//
	//rac := resourceAttrCheckInit(rc, ra)
	//
	//testAccCheck := rac.resourceAttrMapUpdateSet()
	//rand := acctest.RandIntRange(1000, 9999)
	//name := fmt.Sprintf("tf-testacc%sbucket-%d", defaultRegionToTest, rand)
	//testAccConfig := resourceTestAccConfigFunc(resourceId, name, func(name string) string {
	//	return providerCommon + resourceOssBucketConfigDependence(name)
	//})
	//
	//resource.Test(t, resource.TestCase{
	//	PreCheck: func() {
	//		testAccPreCheck(t)
	//	},
	//	// module name
	//	IDRefreshName: resourceId,
	//	Providers:     testAccProviders,
	//	CheckDestroy:  rac.checkResourceDestroy(),
	//	Steps: []resource.TestStep{
	//		{
	//			Config: testAccConfig(map[string]interface{}{
	//				"bucket": name,
	//			}),
	//			Check: resource.ComposeTestCheckFunc(
	//				testAccCheck(map[string]string{
	//					"bucket": name,
	//				}),
	//			),
	//		},
	//	},
	//})
}
