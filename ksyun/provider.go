package ksyun

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/mutexkv"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
)

// Provider returns a schema.Provider for ksyun
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KS3_ACCESS_KEY_ID", os.Getenv("KS3_ACCESS_KEY_ID")),
				Description: descriptions["access_key"],
			},
			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KS3_ACCESS_KEY_SECRET", os.Getenv("KS3_ACCESS_KEY_SECRET")),
				Description: descriptions["secret_key"],
			},
			"security_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_SECURITY_TOKEN", os.Getenv("KSYUN_SECURITY_TOKEN")),
				Description: descriptions["security_token"],
			},
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("KS3_REGION", os.Getenv("KS3_REGION")),
				Description: descriptions["region"],
			},
			"protocol": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "HTTPS",
				Description:  descriptions["protocol"],
				ValidateFunc: validation.StringInSlice([]string{"HTTP", "HTTPS"}, false),
			},
			"client_read_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLIENT_READ_TIMEOUT", 60000),
				Description: descriptions["client_read_timeout"],
			},
			"client_connect_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLIENT_CONNECT_TIMEOUT", 60000),
				Description: descriptions["client_connect_timeout"],
			},
			"max_retry_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MAX_RETRY_TIMEOUT", 0),
				Description: descriptions["max_retry_timeout"],
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"ksyun_ks3_service":        dataSourceKsyunKs3Service(),
			"ksyun_ks3_bucket_objects": dataSourceKsyunKs3BucketObjects(),
			"ksyun_ks3_buckets":        dataSourceKsyunKs3Buckets(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"ksyun_ks3_bucket":             resourceKsyunKs3Bucket(),
			"ksyun_ks3_bucket_object":      resourceKsyunKs3BucketObject(),
			"ksyun_ks3_bucket_replication": resourceKsyunKs3BucketReplication(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var providerConfig map[string]interface{}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	accessKey := os.Getenv("KS3_ACCESS_KEY_ID")
	secretKey := os.Getenv("KS3_ACCESS_KEY_SECRET")
	region := os.Getenv("KS3_REGION")
	if region == "" {
		region = DEFAULT_REGION
	}
	securityToken := os.Getenv("KSYUN_SECURITY_TOKEN")
	config := &connectivity.Config{
		AccessKey:     strings.TrimSpace(accessKey),
		SecretKey:     strings.TrimSpace(secretKey),
		SecurityToken: securityToken,
		Region:        connectivity.Region(strings.TrimSpace(region)),
	}

	client, err := config.Client()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// This is a global MutexKV for use within this plugin.
var ksyunMutexKV = mutexkv.NewMutexKV()

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"access_key": "The access key for API operations. You can retrieve this from the 'Security Management' section of the Ksyun Cloud console.",

		"secret_key": "The secret key for API operations. You can retrieve this from the 'Security Management' section of the Ksyun Cloud console.",

		"region": "The region where Ksyun-ks3 operations will take place. Examples are  BEIJING etc.",

		"security_token": "security token. A security token is only required if you are using Security Token Service.",

		"client_read_timeout": "The maximum timeout of the client read request.",

		"client_connect_timeout": "The maximum timeout of the client connection server.",

		"max_retry_timeout": "The maximum retry timeout of the request.",

		"ks3_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom KS3 endpoints.",
	}
}
