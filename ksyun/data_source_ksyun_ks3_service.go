package ksyun

import (
	"fmt"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/ksyun/terraform-provider-ks3/ksyun/connectivity"
)

func dataSourceKsyunKs3Service() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKsyunKs3ServiceRead,

		Schema: map[string]*schema.Schema{
			"enable": {
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{"On", "Off"}, false),
				Optional:     true,
				Default:      "Off",
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
func dataSourceKsyunKs3ServiceRead(d *schema.ResourceData, meta interface{}) error {
	if v, ok := d.GetOk("enable"); !ok || v.(string) != "On" {
		d.SetId("OssServiceHasNotBeenOpened")
		d.Set("status", "")
		return nil
	}

	conn, err := meta.(*connectivity.KsyunClient).NewTeaCommonClient(connectivity.OpenOssService)
	if err != nil {
		return WrapError(err)
	}
	response, err := conn.DoRequest(StringPointer("OpenOssService"), nil, StringPointer("POST"), StringPointer("2019-04-22"), StringPointer("AK"), nil, nil, &util.RuntimeOptions{})

	addDebug("OpenOssService", response, nil)
	if err != nil {
		if IsExpectedErrors(err, []string{"SYSTEM.SALE_VALIDATE_NO_SPECIFIC_CODE_FAILEDError", "ORDER.OPEND"}) {
			d.SetId("OssServicHasBeenOpened")
			d.Set("status", "Opened")
			return nil
		}
		return WrapErrorf(err, DataDefaultErrorMsg, "alicloud_ks3_service", "OpenOssService", AlibabaCloudSdkGoERROR)
	}

	d.SetId(fmt.Sprintf("%v", response["OrderId"]))
	d.Set("status", "Opened")

	return nil
}
