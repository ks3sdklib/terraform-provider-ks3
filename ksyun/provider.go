package ksyun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
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
				DefaultFunc: schema.EnvDefaultFunc("KS3_TEST_ACCESS_KEY_ID", os.Getenv("ALIBABACLOUD_ACCESS_KEY_ID")),
				Description: descriptions["access_key"],
			},
			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KS3_TEST_ACCESS_KEY_SECRET", os.Getenv("ALIBABACLOUD_ACCESS_KEY_SECRET")),
				Description: descriptions["secret_key"],
			},
			"security_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_SECURITY_TOKEN", os.Getenv("ALIBABACLOUD_SECURITY_TOKEN")),
				Description: descriptions["security_token"],
			},
			"ecs_role_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_ECS_ROLE_NAME", os.Getenv("KSYUN_ECS_ROLE_NAME")),
				Description: descriptions["ecs_role_name"],
			},
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("KS3_TEST_REGION", os.Getenv("KS3_TEST_REGION")),
				Description: descriptions["region"],
			},
			"ots_instance_name": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Field 'ots_instance_name' has been deprecated from provider version 1.10.0. New field 'instance_name' of resource 'ksyun_ots_table' instead.",
			},
			"log_endpoint": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Field 'log_endpoint' has been deprecated from provider version 1.28.0. New field 'log' which in nested endpoints instead.",
			},
			"mns_endpoint": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Field 'mns_endpoint' has been deprecated from provider version 1.28.0. New field 'mns' which in nested endpoints instead.",
			},
			"account_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_ACCOUNT_ID", os.Getenv("KSYUN_ACCOUNT_ID")),
				Description: descriptions["account_id"],
			},
			"assume_role": assumeRoleSchema(),
			"fc": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Field 'fc' has been deprecated from provider version 1.28.0. New field 'fc' which in nested endpoints instead.",
			},
			"endpoints": endpointsSchema(),
			"shared_credentials_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["shared_credentials_file"],
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_SHARED_CREDENTIALS_FILE", ""),
			},
			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["profile"],
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_PROFILE", ""),
			},
			"skip_region_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["skip_region_validation"],
			},
			"configuration_source": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  descriptions["configuration_source"],
				ValidateFunc: validation.StringLenBetween(0, 64),
				DefaultFunc:  schema.EnvDefaultFunc("TF_APPEND_USER_AGENT", ""),
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
			"source_ip": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_SOURCE_IP", os.Getenv("KSYUN_SOURCE_IP")),
				Description: descriptions["source_ip"],
			},
			"security_transport": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_SECURITY_TRANSPORT", os.Getenv("KSYUN_SECURITY_TRANSPORT")),
				//Deprecated:  "It has been deprecated from version 1.136.0 and using new field secure_transport instead.",
			},
			"secure_transport": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_SECURE_TRANSPORT", os.Getenv("KSYUN_SECURE_TRANSPORT")),
				Description: descriptions["secure_transport"],
			},
			"credentials_uri": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KSYUN_CREDENTIALS_URI", os.Getenv("KSYUN_CREDENTIALS_URI")),
				Description: descriptions["credentials_uri"],
			},
			"max_retry_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MAX_RETRY_TIMEOUT", 0),
				Description: descriptions["max_retry_timeout"],
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"ksyun_oss_service":        dataSourceKsyunKs3Service(),
			"ksyun_oss_bucket_objects": dataSourceKsyunKs3BucketObjects(),
			"ksyun_oss_buckets":        dataSourceKsyunKs3Buckets(),
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

	var getProviderConfig = func(str string, key string) string {
		if str == "" {
			value, err := getConfigFromProfile(d, key)
			if err == nil && value != nil {
				str = value.(string)
			}
		}
		return str
	}

	accessKey := getProviderConfig(d.Get("access_key").(string), "access_key_id")
	secretKey := getProviderConfig(d.Get("secret_key").(string), "access_key_secret")
	region := getProviderConfig(d.Get("region").(string), "region_id")
	if region == "" {
		region = DEFAULT_REGION
	}
	securityToken := getProviderConfig(d.Get("security_token").(string), "sts_token")

	ecsRoleName := getProviderConfig(d.Get("ecs_role_name").(string), "ram_role_name")

	if accessKey == "" || secretKey == "" {
		if v, ok := d.GetOk("credentials_uri"); ok && v.(string) != "" {
			credentialsURIResp, err := getClientByCredentialsURI(v.(string))
			if err != nil {
				return nil, err
			}
			accessKey = credentialsURIResp.AccessKeyId
			secretKey = credentialsURIResp.AccessKeySecret
			securityToken = credentialsURIResp.SecurityToken
		}
	}

	config := &connectivity.Config{
		AccessKey:            strings.TrimSpace(accessKey),
		SecretKey:            strings.TrimSpace(secretKey),
		EcsRoleName:          strings.TrimSpace(ecsRoleName),
		Region:               connectivity.Region(strings.TrimSpace(region)),
		RegionId:             strings.TrimSpace(region),
		SkipRegionValidation: d.Get("skip_region_validation").(bool),
		ConfigurationSource:  d.Get("configuration_source").(string),
		Protocol:             d.Get("protocol").(string),
		ClientReadTimeout:    d.Get("client_read_timeout").(int),
		ClientConnectTimeout: d.Get("client_connect_timeout").(int),
		SourceIp:             strings.TrimSpace(d.Get("source_ip").(string)),
		SecureTransport:      strings.TrimSpace(d.Get("secure_transport").(string)),
		MaxRetryTimeout:      d.Get("max_retry_timeout").(int),
	}
	if v, ok := d.GetOk("security_transport"); config.SecureTransport == "" && ok && v.(string) != "" {
		config.SecureTransport = v.(string)
	}
	config.SecurityToken = strings.TrimSpace(securityToken)

	config.RamRoleArn = getProviderConfig("", "ram_role_arn")
	config.RamRoleSessionName = getProviderConfig("", "ram_session_name")
	expiredSeconds, err := getConfigFromProfile(d, "expired_seconds")
	if err == nil && expiredSeconds != nil {
		config.RamRoleSessionExpiration = (int)(expiredSeconds.(float64))
	}

	assumeRoleList := d.Get("assume_role").(*schema.Set).List()
	if len(assumeRoleList) == 1 {
		assumeRole := assumeRoleList[0].(map[string]interface{})
		if assumeRole["role_arn"].(string) != "" {
			config.RamRoleArn = assumeRole["role_arn"].(string)
		}
		if assumeRole["session_name"].(string) != "" {
			config.RamRoleSessionName = assumeRole["session_name"].(string)
		}
		if config.RamRoleSessionName == "" {
			config.RamRoleSessionName = "terraform"
		}
		config.RamRolePolicy = assumeRole["policy"].(string)
		if assumeRole["session_expiration"].(int) == 0 {
			if v := os.Getenv("KSYUN_ASSUME_ROLE_SESSION_EXPIRATION"); v != "" {
				if expiredSeconds, err := strconv.Atoi(v); err == nil {
					config.RamRoleSessionExpiration = expiredSeconds
				}
			}
			if config.RamRoleSessionExpiration == 0 {
				config.RamRoleSessionExpiration = 3600
			}
		} else {
			config.RamRoleSessionExpiration = assumeRole["session_expiration"].(int)
		}

		log.Printf("[INFO] assume_role configuration set: (RamRoleArn: %q, RamRoleSessionName: %q, RamRolePolicy: %q, RamRoleSessionExpiration: %d)",
			config.RamRoleArn, config.RamRoleSessionName, config.RamRolePolicy, config.RamRoleSessionExpiration)
	}

	if err := config.MakeConfigByEcsRoleName(); err != nil {
		return nil, err
	}

	endpointsSet := d.Get("endpoints").(*schema.Set)
	var endpointInit sync.Map
	config.Endpoints = &endpointInit

	for _, endpointsSetI := range endpointsSet.List() {
		endpoints := endpointsSetI.(map[string]interface{})
		for key, val := range endpoints {
			endpointInit.Store(key, val)
		}
		config.Ks3Endpoint = strings.TrimSpace(endpoints["ks3"].(string))
	}

	if config.RamRoleArn != "" {
		config.AccessKey, config.SecretKey, config.SecurityToken, err = getAssumeRoleAK(config)
		if err != nil {
			return nil, err
		}
	}

	if ots_instance_name, ok := d.GetOk("ots_instance_name"); ok && ots_instance_name.(string) != "" {
		config.OtsInstanceName = strings.TrimSpace(ots_instance_name.(string))
	}

	if account, ok := d.GetOk("account_id"); ok && account.(string) != "" {
		config.AccountId = strings.TrimSpace(account.(string))
	}

	if config.ConfigurationSource == "" {
		sourceAccessKey := config.AccessKey
		if len(sourceAccessKey) > 25 {
			sourceAccessKey = sourceAccessKey[:25]
		}
		sourceName := fmt.Sprintf("Default/%s:%s", sourceAccessKey, strings.Trim(uuid.New().String(), "-"))
		if len(sourceName) > 64 {
			sourceName = sourceName[:64]
		}
		config.ConfigurationSource = sourceName
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
		"access_key": "The access key for API operations. You can retrieve this from the 'Security Management' section of the Alibaba Cloud console.",

		"secret_key": "The secret key for API operations. You can retrieve this from the 'Security Management' section of the Alibaba Cloud console.",

		"ecs_role_name": "The RAM Role Name attached on a ECS instance for API operations. You can retrieve this from the 'Access Control' section of the Alibaba Cloud console.",

		"region": "The region where Alibaba Cloud operations will take place. Examples are cn-beijing, cn-hangzhou, eu-central-1, etc.",

		"security_token": "security token. A security token is only required if you are using Security Token Service.",

		"account_id": "The account ID for some service API operations. You can retrieve this from the 'Security Settings' section of the Alibaba Cloud console.",

		"profile": "The profile for API operations. If not set, the default profile created with `ksyun configure` will be used.",

		"shared_credentials_file": "The path to the shared credentials file. If not set this defaults to ~/.ksyun/config.json",

		"assume_role_role_arn": "The ARN of a RAM role to assume prior to making API calls.",

		"assume_role_session_name": "The session name to use when assuming the role. If omitted, `terraform` is passed to the AssumeRole call as session name.",

		"assume_role_policy": "The permissions applied when assuming a role. You cannot use, this policy to grant further permissions that are in excess to those of the, role that is being assumed.",

		"assume_role_session_expiration": "The time after which the established session for assuming role expires. Valid value range: [900-3600] seconds. Default to 0 (in this case Ksyun use own default value).",

		"skip_region_validation": "Skip static validation of region ID. Used by users of alternative AlibabaCloud-like APIs or users w/ access to regions that are not public (yet).",

		"configuration_source": "Use this to mark a terraform configuration file source.",

		"client_read_timeout":    "The maximum timeout of the client read request.",
		"client_connect_timeout": "The maximum timeout of the client connection server.",
		"source_ip":              "The source ip for the assume role invoking.",
		"secure_transport":       "The security transport for the assume role invoking.",
		"credentials_uri":        "The URI of sidecar credentials service.",
		"max_retry_timeout":      "The maximum retry timeout of the request.",

		"ecs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ECS endpoints.",

		"rds_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom RDS endpoints.",

		"slb_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom SLB endpoints.",

		"vpc_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom VPC and VPN endpoints.",

		"ess_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Autoscaling endpoints.",

		"ks3_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom KS3 endpoints.",

		"ons_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ONS endpoints.",

		"alikafka_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ALIKAFKA endpoints.",

		"dns_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom DNS endpoints.",

		"ram_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom RAM endpoints.",

		"cs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Container Service endpoints.",

		"cr_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Container Registry endpoints.",

		"cdn_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom CDN endpoints.",

		"kms_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom KMS endpoints.",

		"ots_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Table Store endpoints.",

		"cms_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Cloud Monitor endpoints.",

		"pvtz_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Private Zone endpoints.",

		"sts_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom STS endpoints.",

		"log_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Log Service endpoints.",

		"drds_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom DRDS endpoints.",

		"dds_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom MongoDB endpoints.",

		"polardb_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom PolarDB endpoints.",

		"gpdb_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom GPDB endpoints.",

		"kvstore_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom R-KVStore endpoints.",

		"fc_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Function Computing endpoints.",

		"apigateway_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Api Gateway endpoints.",

		"datahub_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Datahub endpoints.",

		"mns_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom MNS endpoints.",

		"location_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Location Service endpoints.",

		"elasticsearch_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Elasticsearch endpoints.",

		"nas_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom NAS endpoints.",

		"actiontrail_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Actiontrail endpoints.",

		"cas_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom CAS endpoints.",

		"bssopenapi_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom BSSOPENAPI endpoints.",

		"ddoscoo_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom DDOSCOO endpoints.",

		"ddosbgp_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom DDOSBGP endpoints.",

		"emr_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom EMR endpoints.",

		"market_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom Market Place endpoints.",

		"hbase_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom HBase endpoints.",

		"adb_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom AnalyticDB endpoints.",

		"cbn_endpoint":        "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cbn endpoints.",
		"maxcompute_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom MaxCompute endpoints.",

		"dms_enterprise_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dms_enterprise endpoints.",

		"waf_openapi_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom waf_openapi endpoints.",

		"resourcemanager_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom resourcemanager endpoints.",

		"alidns_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom alidns endpoints.",

		"cassandra_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cassandra endpoints.",

		"eci_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom eci endpoints.",

		"oos_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom oos endpoints.",

		"dcdn_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dcdn endpoints.",

		"mse_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom mse endpoints.",

		"config_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom config endpoints.",

		"r_kvstore_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom r_kvstore endpoints.",

		"fnf_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom fnf endpoints.",

		"ros_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ros endpoints.",

		"privatelink_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom privatelink endpoints.",

		"resourcesharing_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom resourcesharing endpoints.",

		"ga_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ga endpoints.",

		"hitsdb_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom hitsdb endpoints.",

		"brain_industrial_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom brain_industrial endpoints.",

		"eipanycast_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom eipanycast endpoints.",

		"ims_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ims endpoints.",

		"quotas_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom quotas endpoints.",

		"sgw_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom sgw endpoints.",

		"dm_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dm endpoints.",

		"eventbridge_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom eventbridge_share endpoints.",

		"onsproxy_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom onsproxy endpoints.",

		"cds_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cds endpoints.",

		"hbr_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom hbr endpoints.",

		"arms_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom arms endpoints.",

		"serverless_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom serverless endpoints.",

		"alb_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom alb endpoints.",

		"redisa_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom redisa endpoints.",

		"gwsecd_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom gwsecd endpoints.",

		"cloudphone_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cloudphone endpoints.",

		"scdn_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom scdn endpoints.",

		"dataworkspublic_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dataworkspublic endpoints.",

		"hcs_sgw_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom hcs_sgw endpoints.",

		"cddc_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cddc endpoints.",

		"mscopensubscription_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom mscopensubscription endpoints.",

		"sddp_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom sddp endpoints.",

		"bastionhost_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom bastionhost endpoints.",

		"sas_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom sas endpoints.",

		"alidfs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom alidfs endpoints.",

		"ehpc_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ehpc endpoints.",

		"ens_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ens endpoints.",

		"iot_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom iot endpoints.",

		"imm_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom imm endpoints.",

		"clickhouse_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom clickhouse endpoints.",

		"dts_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dts endpoints.",

		"dg_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dg endpoints.",

		"cloudsso_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cloudsso endpoints.",

		"waf_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom waf endpoints.",

		"swas_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom swas endpoints.",

		"vs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom vs endpoints.",

		"quickbi_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom quickbi endpoints.",

		"vod_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom vod endpoints.",

		"opensearch_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom opensearch endpoints.",

		"gds_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom gds endpoints.",

		"dbfs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dbfs endpoints.",

		"devopsrdc_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom devopsrdc endpoints.",

		"eais_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom eais endpoints.",

		"cloudauth_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cloudauth endpoints.",

		"imp_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom imp endpoints.",

		"mhub_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom mhub endpoints.",

		"servicemesh_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom servicemesh endpoints.",

		"acr_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom acr endpoints.",

		"edsuser_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom edsuser endpoints.",

		"gaplus_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom gaplus endpoints.",

		"ddosbasic_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ddosbasic endpoints.",

		"smartag_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom smartag endpoints.",

		"tag_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom tag endpoints.",

		"edas_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom edas endpoints.",

		"edasschedulerx_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom edasschedulerx endpoints.",

		"ehs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ehs endpoints.",

		"cloudfw_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cloudfw endpoints.",

		"dysmsapi_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dysmsapi endpoints.",

		"cbs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cbs endpoints.",

		"nlb_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom nlb endpoints.",

		"vpcpeer_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom vpcpeer endpoints.",

		"ebs_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom ebs endpoints.",

		"dmsenterprise_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom dmsenterprise endpoints.",

		"bpstudio_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom bpstudio endpoints.",

		"das_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom das endpoints.",

		"cloudfirewall_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom cloudfirewall endpoints.",

		"srvcatalog_endpoint": "Use this to override the default endpoint URL constructed from the `region`. It's typically used to connect to custom srvcatalog endpoints.",
	}
}

func assumeRoleSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"role_arn": {
					Type:        schema.TypeString,
					Required:    true,
					Description: descriptions["assume_role_role_arn"],
					DefaultFunc: schema.EnvDefaultFunc("KSYUN_ASSUME_ROLE_ARN", ""),
				},
				"session_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["assume_role_session_name"],
					DefaultFunc: schema.EnvDefaultFunc("KSYUN_ASSUME_ROLE_SESSION_NAME", ""),
				},
				"policy": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["assume_role_policy"],
				},
				"session_expiration": {
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  descriptions["assume_role_session_expiration"],
					ValidateFunc: intBetween(900, 3600),
				},
			},
		},
	}
}

func endpointsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"srvcatalog": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["srvcatalog_endpoint"],
				},

				"cloudfirewall": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudfirewall_endpoint"],
				},

				"das": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["das_endpoint"],
				},

				"bpstudio": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["bpstudio_endpoint"],
				},

				"dmsenterprise": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dmsenterprise_endpoint"],
				},

				"ebs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ebs_endpoint"],
				},

				"nlb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["nlb_endpoint"],
				},

				"cbs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cbs_endpoint"],
				},

				"vpcpeer": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["vpcpeer_endpoint"],
				},

				"dysms": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dysms_endpoint"],
				},

				"edas": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["edas_endpoint"],
				},

				"edasschedulerx": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["edasschedulerx_endpoint"],
				},

				"ehs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ehs_endpoint"],
				},

				"tag": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["tag_endpoint"],
				},

				"ddosbasic": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ddosbasic_endpoint"],
				},

				"smartag": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["smartag_endpoint"],
				},

				"gaplus": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["gaplus_endpoint"],
				},

				"cloudfw": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudfw_endpoint"],
				},

				"edsuser": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["edsuser_endpoint"],
				},

				"acr": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["acr_endpoint"],
				},

				"imp": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["imp_endpoint"],
				},
				"eais": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["eais_endpoint"],
				},
				"cloudauth": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudauth_endpoint"],
				},

				"mhub": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["mhub_endpoint"],
				},
				"servicemesh": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["servicemesh_endpoint"],
				},
				"quickbi": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["quickbi_endpoint"],
				},
				"vod": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["vod_endpoint"],
				},
				"opensearch": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["opensearch_endpoint"],
				},
				"gds": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["gds_endpoint"],
				},
				"dbfs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dbfs_endpoint"],
				},
				"devopsrdc": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["devopsrdc_endpoint"],
				},
				"dg": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dg_endpoint"],
				},
				"waf": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["waf_endpoint"],
				},
				"vs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["vs_endpoint"],
				},
				"dts": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dts_endpoint"],
				},
				"cloudsso": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudsso_endpoint"],
				},

				"iot": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["iot_endpoint"],
				},
				"swas": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["swas_endpoint"],
				},

				"imm": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["imm_endpoint"],
				},
				"clickhouse": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["clickhouse_endpoint"],
				},

				"alidfs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["alidfs_endpoint"],
				},

				"ens": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ens_endpoint"],
				},

				"bastionhost": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["bastionhost_endpoint"],
				},
				"cddc": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cddc_endpoint"],
				},
				"sddp": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["sddp_endpoint"],
				},

				"mscopensubscription": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["mscopensubscription_endpoint"],
				},

				"sas": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["sas_endpoint"],
				},

				"ehpc": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ehpc_endpoint"],
				},

				"dataworkspublic": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dataworkspublic_endpoint"],
				},

				"hcs_sgw": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["hcs_sgw_endpoint"],
				},

				"cloudphone": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudphone_endpoint"],
				},

				"alb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["alb_endpoint"],
				},
				"redisa": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["redisa_endpoint"],
				},
				"gwsecd": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["gwsecd_endpoint"],
				},
				"scdn": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["scdn_endpoint"],
				},

				"arms": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["arms_endpoint"],
				},
				"serverless": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["serverless_endpoint"],
				},

				"hbr": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["hbr_endpoint"],
				},

				"onsproxy": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["onsproxy_endpoint"],
				},
				"cds": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cds_endpoint"],
				},

				"dm": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dm_endpoint"],
				},

				"eventbridge": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["eventbridge_endpoint"],
				},

				"sgw": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["sgw_endpoint"],
				},

				"quotas": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["quotas_endpoint"],
				},

				"ims": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ims_endpoint"],
				},

				"brain_industrial": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["brain_industrial_endpoint"],
				},

				"resourcesharing": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["resourcesharing_endpoint"],
				},
				"ga": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ga_endpoint"],
				},

				"hitsdb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["hitsdb_endpoint"],
				},

				"privatelink": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["privatelink_endpoint"],
				},

				"eipanycast": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["eipanycast_endpoint"],
				},

				"fnf": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["fnf_endpoint"],
				},

				"ros": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ros_endpoint"],
				},

				"r_kvstore": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["r_kvstore_endpoint"],
				},

				"config": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["config_endpoint"],
				},

				"dcdn": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dcdn_endpoint"],
				},

				"mse": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["mse_endpoint"],
				},

				"oos": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["oos_endpoint"],
				},

				"eci": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["eci_endpoint"],
				},

				"alidns": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["alidns_endpoint"],
				},

				"resourcemanager": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["resourcemanager_endpoint"],
				},

				"waf_openapi": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["waf_openapi_endpoint"],
				},

				"dms_enterprise": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dms_enterprise_endpoint"],
				},

				"cassandra": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cassandra_endpoint"],
				},

				"cbn": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cbn_endpoint"],
				},

				"ecs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ecs_endpoint"],
				},
				"rds": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["rds_endpoint"],
				},
				"slb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["slb_endpoint"],
				},
				"vpc": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["vpc_endpoint"],
				},
				"ess": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ess_endpoint"],
				},
				"ks3": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ks3_endpoint"],
				},
				"ons": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ons_endpoint"],
				},
				"alikafka": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["alikafka_endpoint"],
				},
				"dns": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dns_endpoint"],
				},
				"ram": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ram_endpoint"],
				},
				"cs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cs_endpoint"],
				},
				"cr": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cr_endpoint"],
				},
				"cdn": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cdn_endpoint"],
				},

				"kms": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["kms_endpoint"],
				},

				"ots": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ots_endpoint"],
				},

				"cms": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cms_endpoint"],
				},

				"pvtz": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["pvtz_endpoint"],
				},

				"sts": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["sts_endpoint"],
				},
				// log service is sls service
				"log": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["log_endpoint"],
				},
				"drds": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["drds_endpoint"],
				},
				"dds": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dds_endpoint"],
				},
				"polardb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["polardb_endpoint"],
				},
				"gpdb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["gpdb_endpoint"],
				},
				"kvstore": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["kvstore_endpoint"],
				},
				"fc": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["fc_endpoint"],
				},
				"apigateway": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["apigateway_endpoint"],
				},
				"datahub": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["datahub_endpoint"],
				},
				"mns": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["mns_endpoint"],
				},
				"location": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["location_endpoint"],
				},
				"elasticsearch": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["elasticsearch_endpoint"],
				},
				"nas": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["nas_endpoint"],
				},
				"actiontrail": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["actiontrail_endpoint"],
				},
				"cas": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cas_endpoint"],
				},
				"bssopenapi": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["bssopenapi_endpoint"],
				},
				"ddoscoo": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ddoscoo_endpoint"],
				},
				"ddosbgp": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ddosbgp_endpoint"],
				},
				"emr": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["emr_endpoint"],
				},
				"market": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["market_endpoint"],
				},
				"adb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["adb_endpoint"],
				},
				"maxcompute": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["maxcompute_endpoint"],
				},
			},
		},
		Set: endpointsToHash,
	}
}

func endpointsToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["ecs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["rds"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["slb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vpc"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ess"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ks3"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ons"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["alikafka"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dns"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ram"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cdn"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["kms"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ots"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cms"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["pvtz"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sts"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["log"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["drds"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dds"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["gpdb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["kvstore"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["polardb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["fc"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["apigateway"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["datahub"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["mns"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["location"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["elasticsearch"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["nas"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["actiontrail"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cas"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["bssopenapi"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ddoscoo"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ddosbgp"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["emr"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["market"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["adb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cbn"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["maxcompute"].(string)))

	buf.WriteString(fmt.Sprintf("%s-", m["dms_enterprise"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["waf_openapi"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["resourcemanager"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["alidns"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cassandra"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["eci"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["oos"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dcdn"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["mse"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["config"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["r_kvstore"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["fnf"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ros"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["privatelink"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["resourcesharing"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ga"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["hitsdb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["brain_industrial"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["eipanycast"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ims"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["quotas"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sgw"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dm"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["eventbridge"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["onsproxy"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cds"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["hbr"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["arms"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["serverless"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["alb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["redisa"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["gwsecd"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudphone"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["scdn"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dataworkspublic"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["hcs_sgw"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cddc"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["mscopensubscription"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sddp"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["bastionhost"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sas"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["alidfs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ehpc"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ens"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["iot"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["imm"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["clickhouse"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dts"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dg"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudsso"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["waf"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["swas"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["quickbi"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vod"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["opensearch"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["gds"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dbfs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["devopsrdc"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["eais"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudauth"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["imp"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["mhub"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["servicemesh"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["acr"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["edsuser"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["gaplus"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ddosbasic"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["smartag"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["tag"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["edas"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["edasschedulerx"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ehs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudfw"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dysms"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cbs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["nlb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vpcpeer"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ebs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dmsenterprise"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["bpstudio"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["das"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudfirewall"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["srvcatalog"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vpcpeer"].(string)))
	return hashcode.String(buf.String())
}

func getConfigFromProfile(d *schema.ResourceData, ProfileKey string) (interface{}, error) {

	if providerConfig == nil {
		if v, ok := d.GetOk("profile"); !ok && v.(string) == "" {
			return nil, nil
		}
		current := d.Get("profile").(string)
		// Set CredsFilename, expanding home directory
		profilePath, err := homedir.Expand(d.Get("shared_credentials_file").(string))
		if err != nil {
			return nil, WrapError(err)
		}
		if profilePath == "" {
			profilePath = fmt.Sprintf("%s/.ksyun/config.json", os.Getenv("HOME"))
			if runtime.GOOS == "windows" {
				profilePath = fmt.Sprintf("%s/.ksyun/config.json", os.Getenv("USERPROFILE"))
			}
		}
		providerConfig = make(map[string]interface{})
		_, err = os.Stat(profilePath)
		if !os.IsNotExist(err) {
			data, err := ioutil.ReadFile(profilePath)
			if err != nil {
				return nil, WrapError(err)
			}
			config := map[string]interface{}{}
			err = json.Unmarshal(data, &config)
			if err != nil {
				return nil, WrapError(err)
			}
			for _, v := range config["profiles"].([]interface{}) {
				if current == v.(map[string]interface{})["name"] {
					providerConfig = v.(map[string]interface{})
				}
			}
		}
	}

	mode := ""
	if v, ok := providerConfig["mode"]; ok {
		mode = v.(string)
	} else {
		return v, nil
	}
	switch ProfileKey {
	case "access_key_id", "access_key_secret":
		if mode == "EcsRamRole" {
			return "", nil
		}
	case "ram_role_name":
		if mode != "EcsRamRole" {
			return "", nil
		}
	case "sts_token":
		if mode != "StsToken" {
			return "", nil
		}
	case "ram_role_arn", "ram_session_name":
		if mode != "RamRoleArn" {
			return "", nil
		}
	case "expired_seconds":
		if mode != "RamRoleArn" {
			return float64(0), nil
		}
	}

	return providerConfig[ProfileKey], nil
}

func getAssumeRoleAK(config *connectivity.Config) (string, string, string, error) {

	request := sts.CreateAssumeRoleRequest()
	request.RoleArn = config.RamRoleArn
	request.RoleSessionName = config.RamRoleSessionName
	request.DurationSeconds = requests.NewInteger(config.RamRoleSessionExpiration)
	request.Policy = config.RamRolePolicy
	request.Scheme = "https"

	var client *sts.Client
	var err error
	if config.SecurityToken == "" {
		client, err = sts.NewClientWithAccessKey(config.RegionId, config.AccessKey, config.SecretKey)
	} else {
		client, err = sts.NewClientWithStsToken(config.RegionId, config.AccessKey, config.SecretKey, config.SecurityToken)
	}

	if err != nil {
		return "", "", "", err
	}

	client.SourceIp = config.SourceIp
	client.SecureTransport = config.SecureTransport
	response, err := client.AssumeRole(request)
	if err != nil {
		return "", "", "", err
	}

	return response.Credentials.AccessKeyId, response.Credentials.AccessKeySecret, response.Credentials.SecurityToken, nil
}

type CredentialsURIResponse struct {
	Code            string
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
	Expiration      string
}

func getClientByCredentialsURI(credentialsURI string) (*CredentialsURIResponse, error) {
	res, err := http.Get(credentialsURI)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("get Credentials from %s failed, status code %d", credentialsURI, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	var response CredentialsURIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal credentials failed, the body %s", string(body))
	}

	if response.Code != "Success" {
		return nil, fmt.Errorf("fetching sts token from %s got an error and its Code is not Success", credentialsURI)
	}

	return &response, nil
}
