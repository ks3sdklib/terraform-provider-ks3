package ksyun

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"io/ioutil"
	"log"
	"time"
)

func resourceKsyunKs3Bucket() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudOssBucketCreate,
		Read:   resourceAlicloudOssBucketRead,
		Update: resourceAlicloudOssBucketUpdate,
		Delete: resourceAlicloudOssBucketDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(3, 63),
				Default:      resource.PrefixedUniqueId("tf-ks3-bucket-"),
			},

			"acl": {
				Type:         schema.TypeString,
				Default:      ks3.ACLPrivate,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"private", "public-read", "public-read-write"}, false),
			},

			"cors_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_headers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"allowed_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"allowed_origins": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"expose_headers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"max_age_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				MaxItems: 10,
			},

			"website": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"index_document": {
							Type:     schema.TypeString,
							Required: true,
						},

						"error_document": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				MaxItems: 1,
			},

			"logging": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_bucket": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				MaxItems: 1,
			},

			"logging_isenable": {
				Type:       schema.TypeBool,
				Optional:   true,
				Deprecated: "Deprecated from 1.37.0. When `logging` is set, the bucket logging will be able.",
			},

			"referer_config": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow_empty": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"referers": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				MaxItems: 1,
			},

			"lifecycle_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringLenBetween(0, 255),
						},
						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"expiration": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      expirationHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateOssBucketDateTimestamp,
									},
									"created_before_date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateOssBucketDateTimestamp,
									},
									"days": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"expired_object_delete_marker": {
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},
						"transitions": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      transitionsHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"created_before_date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateOssBucketDateTimestamp,
									},
									"days": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"storage_class": {
										Type:     schema.TypeString,
										Default:  ks3.StorageStandard,
										Optional: true,
										ValidateFunc: validation.StringInSlice([]string{
											string(ks3.StorageStandard),
											string(ks3.StorageIA),
											string(ks3.StorageArchive),
										}, false),
									},
								},
							},
						},
						"abort_multipart_upload": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      abortMultipartUploadHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"created_before_date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateOssBucketDateTimestamp,
									},
									"days": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
						"noncurrent_version_expiration": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      expirationHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"days": {
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
						},
						"noncurrent_version_transition": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      transitionsHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"days": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"storage_class": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											string(ks3.StorageStandard),
											string(ks3.StorageIA),
											string(ks3.StorageArchive),
										}, false),
									},
								},
							},
						},
					},
				},
				MaxItems: 1000,
			},

			"policy": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"extranet_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"intranet_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"location": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_class": {
				Type:     schema.TypeString,
				Default:  ks3.StorageStandard,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(ks3.StorageStandard),
					string(ks3.StorageIA),
					string(ks3.StorageArchive),
				}, false),
			},
			"server_side_encryption_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sse_algorithm": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								ServerSideEncryptionAes256,
								ServerSideEncryptionKMS,
							}, false),
						},
						"kms_master_key_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				MaxItems: 1,
			},

			"tags": tagsSchema(),

			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"versioning": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"status": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								"Enabled",
								"Suspended",
							}, false),
						},
					},
				},
				MaxItems: 1,
			},

			"transfer_acceleration": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				MaxItems: 1,
			},
		},
	}
}

func resourceAlicloudOssBucketCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	request := map[string]string{"bucketName": d.Get("bucket").(string)}
	var requestInfo *ks3.Client
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return ks3Client.IsBucketExist(request["bucketName"])
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "ksyun_ks3_bucket", "IsBucketExist", KsyunKs3GoSdk)
	}
	addDebug("IsBucketExist", raw, requestInfo, request)
	isExist, _ := raw.(bool)
	if isExist {
		return WrapError(Error("[ERROR] The specified bucket name: %#v is not available. The bucket namespace is shared by all users of the OSS system. Please select a different name and try again.", request["bucketName"]))
	}
	type Request struct {
		BucketName         string
		StorageClassOption ks3.Option
		AclTypeOption      ks3.Option
	}

	req := Request{
		d.Get("bucket").(string),
		ks3.StorageClass(ks3.StorageClassType(d.Get("storage_class").(string))),
		ks3.ACL(ks3.ACLType(d.Get("acl").(string))),
	}
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return nil, ks3Client.CreateBucket(req.BucketName, req.StorageClassOption, req.AclTypeOption)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "ksyun_ks3_bucket", "CreateBucket", KsyunKs3GoSdk)
	}
	addDebug("CreateBucket", raw, requestInfo, req)
	err = resource.Retry(3*time.Minute, func() *resource.RetryError {
		raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			return ks3Client.IsBucketExist(request["bucketName"])
		})

		if err != nil {
			return resource.NonRetryableError(err)
		}
		isExist, _ := raw.(bool)
		if !isExist {
			return resource.RetryableError(Error("Trying to ensure new OSS bucket %#v has been created successfully.", request["bucketName"]))
		}
		addDebug("IsBucketExist", raw, requestInfo, request)
		return nil
	})

	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "ksyun_ks3_bucket", "IsBucketExist", KsyunKs3GoSdk)
	}

	// Assign the bucket name as the resource ID
	d.SetId(request["bucketName"])

	return resourceAlicloudOssBucketUpdate(d, meta)
}

func resourceAlicloudOssBucketRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	ks3Service := Ks3Service{client}
	object, err := ks3Service.DescribeOssBucket(d.Id())
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return WrapError(err)
	}

	d.Set("bucket", d.Id())

	d.Set("acl", object.BucketInfo.ACL)
	d.Set("creation_date", object.BucketInfo.CreationDate.Format("2006-01-02"))
	d.Set("extranet_endpoint", object.BucketInfo.ExtranetEndpoint)
	d.Set("intranet_endpoint", object.BucketInfo.IntranetEndpoint)
	d.Set("location", object.BucketInfo.Location)
	d.Set("owner", object.BucketInfo.Owner.ID)
	d.Set("storage_class", object.BucketInfo.StorageClass)
	d.Set("redundancy_type", object.BucketInfo.RedundancyType)

	if &object.BucketInfo.SseRule != nil {
		if len(object.BucketInfo.SseRule.SSEAlgorithm) > 0 && object.BucketInfo.SseRule.SSEAlgorithm != "None" {
			rule := make(map[string]interface{})
			rule["sse_algorithm"] = object.BucketInfo.SseRule.SSEAlgorithm
			if object.BucketInfo.SseRule.KMSMasterKeyID != "" {
				rule["kms_master_key_id"] = object.BucketInfo.SseRule.KMSMasterKeyID
			}
			data := make([]map[string]interface{}, 0)
			data = append(data, rule)
			d.Set("server_side_encryption_rule", data)
		}
	}

	if object.BucketInfo.Versioning != "" {
		data := map[string]interface{}{
			"status": object.BucketInfo.Versioning,
		}
		versioning := make([]map[string]interface{}, 0)
		versioning = append(versioning, data)
		d.Set("versioning", versioning)
	}
	request := map[string]string{"bucketName": d.Id()}
	var requestInfo *ks3.Client

	// Read the CORS
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return ks3Client.GetBucketCORS(request["bucketName"])
	})
	if err != nil && !IsExpectedErrors(err, []string{"NoSuchCORSConfiguration"}) {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetBucketCORS", KsyunKs3GoSdk)
	}
	addDebug("GetBucketCORS", raw, requestInfo, request)
	cors, _ := raw.(ks3.GetBucketCORSResult)
	rules := make([]map[string]interface{}, 0, len(cors.CORSRules))
	for _, r := range cors.CORSRules {
		rule := make(map[string]interface{})
		rule["allowed_headers"] = r.AllowedHeader
		rule["allowed_methods"] = r.AllowedMethod
		rule["allowed_origins"] = r.AllowedOrigin
		rule["expose_headers"] = r.ExposeHeader
		rule["max_age_seconds"] = r.MaxAgeSeconds

		rules = append(rules, rule)
	}
	if err := d.Set("cors_rule", rules); err != nil {
		return WrapError(err)
	}

	// Read the website configuration
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return ks3Client.GetBucketWebsite(d.Id())
	})
	if err != nil && !IsExpectedErrors(err, []string{"NoSuchWebsiteConfiguration"}) {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetBucketWebsite", KsyunKs3GoSdk)
	}
	addDebug("GetBucketWebsite", raw, requestInfo, request)
	ws, _ := raw.(ks3.GetBucketWebsiteResult)
	websites := make([]map[string]interface{}, 0)
	if err == nil && &ws != nil {
		w := make(map[string]interface{})

		if v := &ws.IndexDocument; v != nil {
			w["index_document"] = v.Suffix
		}

		if v := &ws.ErrorDocument; v != nil {
			w["error_document"] = v.Key
		}
		websites = append(websites, w)
	}
	if err := d.Set("website", websites); err != nil {
		return WrapError(err)
	}

	// Read the logging configuration
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return ks3Client.GetBucketLogging(d.Id())
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetBucketLogging", KsyunKs3GoSdk)
	}
	addDebug("GetBucketLogging", raw, requestInfo, request)
	logging, _ := raw.(ks3.GetBucketLoggingResult)

	if &logging != nil {
		enable := logging.LoggingEnabled
		if &enable != nil {
			lgs := make([]map[string]interface{}, 0)
			tb := logging.LoggingEnabled.TargetBucket
			tp := logging.LoggingEnabled.TargetPrefix
			if tb != "" || tp != "" {
				lgs = append(lgs, map[string]interface{}{
					"target_bucket": tb,
					"target_prefix": tp,
				})
			}
			if err := d.Set("logging", lgs); err != nil {
				return WrapError(err)
			}
		}
	}

	// Read the bucket referer
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return ks3Client.GetBucketReferer(d.Id())
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetBucketReferer", KsyunKs3GoSdk)
	}
	addDebug("GetBucketReferer", raw, requestInfo, request)
	referers := make([]map[string]interface{}, 0)
	referer, _ := raw.(ks3.GetBucketRefererResult)
	if len(referer.RefererList) > 0 {
		referers = append(referers, map[string]interface{}{
			"allow_empty": referer.AllowEmptyReferer,
			"referers":    referer.RefererList,
		})
		if err := d.Set("referer_config", referers); err != nil {
			return WrapError(err)
		}
	}

	// Read the lifecycle rule configuration
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return ks3Client.GetBucketLifecycle(d.Id())
	})
	if err != nil && !ks3NotFoundError(err) {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetBucketLifecycle", KsyunKs3GoSdk)
	}
	addDebug("GetBucketLifecycle", raw, requestInfo, request)
	lrules := make([]map[string]interface{}, 0)
	lifecycle, _ := raw.(ks3.GetBucketLifecycleResult)
	for _, lifecycleRule := range lifecycle.Rules {
		rule := make(map[string]interface{})
		rule["id"] = lifecycleRule.ID
		rule["prefix"] = lifecycleRule.Prefix
		if LifecycleRuleStatus(lifecycleRule.Status) == ExpirationStatusEnabled {
			rule["enabled"] = true
		} else {
			rule["enabled"] = false
		}
		// expiration
		if lifecycleRule.Expiration != nil {
			e := make(map[string]interface{})
			if lifecycleRule.Expiration.Date != "" {
				t, err := time.Parse("2006-01-02T15:04:05.000Z", lifecycleRule.Expiration.Date)
				if err != nil {
					return WrapError(err)
				}
				e["date"] = t.Format("2006-01-02")
			}
			if lifecycleRule.Expiration.CreatedBeforeDate != "" {
				t, err := time.Parse("2006-01-02T15:04:05.000Z", lifecycleRule.Expiration.CreatedBeforeDate)
				if err != nil {
					return WrapError(err)
				}
				e["created_before_date"] = t.Format("2006-01-02")
			}
			if lifecycleRule.Expiration.ExpiredObjectDeleteMarker != nil {
				e["expired_object_delete_marker"] = *lifecycleRule.Expiration.ExpiredObjectDeleteMarker
			}
			e["days"] = int(lifecycleRule.Expiration.Days)
			rule["expiration"] = schema.NewSet(expirationHash, []interface{}{e})
		}
		// transitions
		if len(lifecycleRule.Transitions) != 0 {
			var eSli []interface{}
			for _, transition := range lifecycleRule.Transitions {
				e := make(map[string]interface{})
				if transition.CreatedBeforeDate != "" {
					t, err := time.Parse("2006-01-02T15:04:05.000Z", transition.CreatedBeforeDate)
					if err != nil {
						return WrapError(err)
					}
					e["created_before_date"] = t.Format("2006-01-02")
				}
				e["days"] = transition.Days
				e["storage_class"] = string(transition.StorageClass)
				eSli = append(eSli, e)
			}
			rule["transitions"] = schema.NewSet(transitionsHash, eSli)
		}
		// abort_multipart_upload
		if lifecycleRule.AbortMultipartUpload != nil {
			e := make(map[string]interface{})
			if lifecycleRule.AbortMultipartUpload.CreatedBeforeDate != "" {
				t, err := time.Parse("2006-01-02T15:04:05.000Z", lifecycleRule.AbortMultipartUpload.CreatedBeforeDate)
				if err != nil {
					return WrapError(err)
				}
				e["created_before_date"] = t.Format("2006-01-02")
			}
			valDays := int(lifecycleRule.AbortMultipartUpload.Days)
			if valDays > 0 {
				e["days"] = int(lifecycleRule.AbortMultipartUpload.Days)
			}
			rule["abort_multipart_upload"] = schema.NewSet(abortMultipartUploadHash, []interface{}{e})
		}
		// NoncurrentVersionExpiration
		if lifecycleRule.NonVersionExpiration != nil {
			e := make(map[string]interface{})
			e["days"] = int(lifecycleRule.NonVersionExpiration.NoncurrentDays)
			rule["noncurrent_version_expiration"] = schema.NewSet(expirationHash, []interface{}{e})
		}
		// NoncurrentVersionTransitions
		if len(lifecycleRule.NonVersionTransitions) != 0 {
			var eSli []interface{}
			for _, transition := range lifecycleRule.NonVersionTransitions {
				e := make(map[string]interface{})
				e["days"] = transition.NoncurrentDays
				e["storage_class"] = string(transition.StorageClass)
				eSli = append(eSli, e)
			}
			rule["noncurrent_version_transition"] = schema.NewSet(transitionsHash, eSli)
		}
		lrules = append(lrules, rule)
	}

	if err := d.Set("lifecycle_rule", lrules); err != nil {
		return WrapError(err)
	}

	// Read Policy
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		params := map[string]interface{}{}
		params["policy"] = nil
		return ks3Client.Conn.Do("GET", d.Id(), "", params, nil, nil, 0, nil)
	})

	if err != nil && !ks3NotFoundError(err) {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetPolicyByConn", KsyunKs3GoSdk)
	}
	addDebug("GetPolicyByConn", raw, requestInfo, request)
	policy := ""
	if err == nil {
		rawResp := raw.(*ks3.Response)
		defer rawResp.Body.Close()
		rawData, err := ioutil.ReadAll(rawResp.Body)
		if err != nil {
			return WrapError(err)
		}
		policy = string(rawData)
	}

	if err := d.Set("policy", policy); err != nil {
		return WrapError(err)
	}

	// Read tags
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return ks3Client.GetBucketTagging(d.Id())
	})

	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetBucketTagging", KsyunKs3GoSdk)
	}
	addDebug("GetBucketTagging", raw, requestInfo, request)
	tagging, _ := raw.(ks3.GetBucketTaggingResult)
	tagsMap := make(map[string]string)
	if len(tagging.Tags) > 0 {
		for _, t := range tagging.Tags {
			tagsMap[t.Key] = t.Value
		}
	}
	if err := d.Set("tags", tagsMap); err != nil {
		return WrapError(err)
	}

	// Read the bucket transfer acceleration status
	raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return ks3Client.GetBucketTransferAcc(d.Id())
	})
	if err != nil && !ks3NotFoundError(err) && !IsExpectedErrors(err, []string{"TransferAccelerationDisabled"}) {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetBucketTransferAcc", KsyunKs3GoSdk)
	}
	acc, _ := raw.(ks3.TransferAccConfiguration)
	accMap := make([]map[string]interface{}, 0)
	if err == nil && &acc != nil {
		data := map[string]interface{}{
			"enabled": acc.Enabled,
		}
		accMap = append(accMap, data)
	}
	if err := d.Set("transfer_acceleration", accMap); err != nil {
		return WrapError(err)
	}

	return nil
}

func resourceAlicloudOssBucketUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)

	d.Partial(true)

	if d.HasChange("acl") && !d.IsNewResource() {
		request := map[string]string{"bucketName": d.Id(), "bucketACL": d.Get("acl").(string)}
		var requestInfo *ks3.Client
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.SetBucketACL(d.Id(), ks3.ACLType(d.Get("acl").(string)))
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketACL", KsyunKs3GoSdk)
		}
		addDebug("SetBucketACL", raw, requestInfo, request)
		d.SetPartial("acl")
	}

	if d.HasChange("cors_rule") {
		if err := resourceAlicloudOssBucketCorsUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("cors_rule")
	}

	if d.HasChange("website") {
		if err := resourceAlicloudOssBucketWebsiteUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("website")
	}

	if d.HasChange("logging") {
		if err := resourceAlicloudOssBucketLoggingUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("logging")
	}

	if d.HasChange("referer_config") {
		if err := resourceAlicloudOssBucketRefererUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("referer_config")
	}

	if d.HasChange("lifecycle_rule") {
		if err := resourceAlicloudOssBucketLifecycleRuleUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("lifecycle_rule")
	}

	if d.HasChange("policy") {
		if err := resourceAlicloudOssBucketPolicyUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("policy")
	}

	if d.HasChange("server_side_encryption_rule") {
		if err := resourceAlicloudOssBucketEncryptionUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("server_side_encryption_rule")
	}

	if d.HasChange("tags") {
		if err := resourceAlicloudOssBucketTaggingUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("tags")
	}

	if d.HasChange("versioning") {
		if err := resourceAlicloudOssBucketVersioningUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("versioning")
	}

	if d.HasChange("transfer_acceleration") {
		if err := resourceAlicloudOssBucketTransferAccUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("transfer_acceleration")
	}

	d.Partial(false)
	return resourceAlicloudOssBucketRead(d, meta)
}

func resourceAlicloudOssBucketCorsUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	cors := d.Get("cors_rule").([]interface{})
	var requestInfo *ks3.Client
	if cors == nil || len(cors) == 0 {
		err := resource.Retry(3*time.Minute, func() *resource.RetryError {
			raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
				requestInfo = ks3Client
				return nil, ks3Client.DeleteBucketCORS(d.Id())
			})
			if err != nil {
				return resource.NonRetryableError(err)
			}
			addDebug("DeleteBucketCORS", raw, requestInfo, map[string]string{"bucketName": d.Id()})
			return nil
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucketCORS", KsyunKs3GoSdk)
		}
		return nil
	}
	// Put CORS
	rules := make([]ks3.CORSRule, 0, len(cors))
	for _, c := range cors {
		corsMap := c.(map[string]interface{})
		rule := ks3.CORSRule{}
		for k, v := range corsMap {
			log.Printf("[DEBUG] OSS bucket: %s, put CORS: %#v, %#v", d.Id(), k, v)
			if k == "max_age_seconds" {
				rule.MaxAgeSeconds = v.(int)
			} else {
				rMap := make([]string, len(v.([]interface{})))
				for i, vv := range v.([]interface{}) {
					rMap[i] = vv.(string)
				}
				switch k {
				case "allowed_headers":
					rule.AllowedHeader = rMap
				case "allowed_methods":
					rule.AllowedMethod = rMap
				case "allowed_origins":
					rule.AllowedOrigin = rMap
				case "expose_headers":
					rule.ExposeHeader = rMap
				}
			}
		}
		rules = append(rules, rule)
	}

	log.Printf("[DEBUG] Oss bucket: %s, put CORS: %#v", d.Id(), cors)
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return nil, ks3Client.SetBucketCORS(d.Id(), rules)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketCORS", KsyunKs3GoSdk)
	}
	addDebug("SetBucketCORS", raw, requestInfo, map[string]interface{}{
		"bucketName": d.Id(),
		"corsRules":  rules,
	})
	return nil
}
func resourceAlicloudOssBucketWebsiteUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	ws := d.Get("website").([]interface{})
	var requestInfo *ks3.Client
	if ws == nil || len(ws) == 0 {
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.DeleteBucketWebsite(d.Id())
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucketWebsite", KsyunKs3GoSdk)
		}
		addDebug("DeleteBucketWebsite", raw, requestInfo, map[string]string{"bucketName": d.Id()})
		return nil
	}

	var index_document, error_document string
	w := ws[0].(map[string]interface{})

	if v, ok := w["index_document"]; ok {
		index_document = v.(string)
	}
	if v, ok := w["error_document"]; ok {
		error_document = v.(string)
	}
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return nil, ks3Client.SetBucketWebsite(d.Id(), index_document, error_document)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketWebsite", KsyunKs3GoSdk)
	}
	addDebug("SetBucketWebsite", raw, requestInfo, map[string]interface{}{
		"bucketName":    d.Id(),
		"indexDocument": index_document,
		"errorDocument": error_document,
	})
	return nil
}

func resourceAlicloudOssBucketLoggingUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	logging := d.Get("logging").([]interface{})
	var requestInfo *ks3.Client
	if logging == nil || len(logging) == 0 {
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.DeleteBucketLogging(d.Id())
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucketLogging", KsyunKs3GoSdk)
		}
		addDebug("DeleteBucketLogging", raw, requestInfo, map[string]string{"bucketName": d.Id()})
		return nil
	}

	c := logging[0].(map[string]interface{})
	var target_bucket, target_prefix string
	if v, ok := c["target_bucket"]; ok {
		target_bucket = v.(string)
	}
	if v, ok := c["target_prefix"]; ok {
		target_prefix = v.(string)
	}
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return nil, ks3Client.SetBucketLogging(d.Id(), target_bucket, target_prefix, target_bucket != "" || target_prefix != "")
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketLogging", KsyunKs3GoSdk)
	}
	addDebug("SetBucketLogging", raw, requestInfo, map[string]interface{}{
		"bucketName":   d.Id(),
		"targetBucket": target_bucket,
		"targetPrefix": target_prefix,
		"isEnable":     target_bucket != "",
	})
	return nil
}

func resourceAlicloudOssBucketRefererUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	config := d.Get("referer_config").([]interface{})
	var requestInfo *ks3.Client
	if config == nil || len(config) < 1 {
		log.Printf("[DEBUG] OSS set bucket referer as nil")
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.SetBucketReferer(d.Id(), nil, true)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketReferer", KsyunKs3GoSdk)
		}
		addDebug("SetBucketReferer", raw, requestInfo, map[string]interface{}{
			"allowEmptyReferer": true,
			"bucketName":        d.Id(),
		})
		return nil
	}

	c := config[0].(map[string]interface{})

	var allow bool
	var referers []string
	if v, ok := c["allow_empty"]; ok {
		allow = v.(bool)
	}
	if v, ok := c["referers"]; ok {
		for _, referer := range v.([]interface{}) {
			referers = append(referers, referer.(string))
		}
	}
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return nil, ks3Client.SetBucketReferer(d.Id(), referers, allow)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketReferer", KsyunKs3GoSdk)
	}
	addDebug("SetBucketReferer", raw, requestInfo, map[string]interface{}{
		"bucketName":        d.Id(),
		"referers":          referers,
		"allowEmptyReferer": allow,
	})
	return nil
}

func resourceAlicloudOssBucketLifecycleRuleUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	bucket := d.Id()
	lifecycleRules := d.Get("lifecycle_rule").([]interface{})
	var requestInfo *ks3.Client
	if lifecycleRules == nil || len(lifecycleRules) == 0 {
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.DeleteBucketLifecycle(bucket)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucketLifecycle", KsyunKs3GoSdk)

		}
		addDebug("DeleteBucketLifecycle", raw, requestInfo, map[string]interface{}{
			"bucketName": bucket,
		})
		return nil
	}

	rules := make([]ks3.LifecycleRule, 0, len(lifecycleRules))

	for i, lifecycleRule := range lifecycleRules {
		r := lifecycleRule.(map[string]interface{})

		rule := ks3.LifecycleRule{
			Prefix: r["prefix"].(string),
		}

		// ID
		if val, ok := r["id"].(string); ok && val != "" {
			rule.ID = val
		}

		// Enabled
		if val, ok := r["enabled"].(bool); ok && val {
			rule.Status = string(ExpirationStatusEnabled)
		} else {
			rule.Status = string(ExpirationStatusDisabled)
		}

		// Expiration
		expiration := d.Get(fmt.Sprintf("lifecycle_rule.%d.expiration", i)).(*schema.Set).List()
		if len(expiration) > 0 {
			e := expiration[0].(map[string]interface{})
			i := ks3.LifecycleExpiration{}
			valDate, _ := e["date"].(string)
			valCreatedBeforeDate, _ := e["created_before_date"].(string)
			valDays, _ := e["days"].(int)

			if val, ok := e["expired_object_delete_marker"].(bool); ok && val {
				if valDays > 0 || valDate != "" || valCreatedBeforeDate != "" {
					return WrapError(Error("'date/created_before_date/days' conflicts with 'expired_object_delete_marker'. One and only one of them can be specified in one expiration configuration."))
				}
				i.ExpiredObjectDeleteMarker = &val
			} else {
				cnt := 0
				if valDate != "" {
					i.Date = fmt.Sprintf("%sT00:00:00.000Z", valDate)
					cnt++
				}
				if valCreatedBeforeDate != "" {
					i.CreatedBeforeDate = fmt.Sprintf("%sT00:00:00.000Z", valCreatedBeforeDate)
					cnt++
				}
				if valDays > 0 {
					i.Days = valDays
					cnt++
				}

				if cnt != 1 {
					return WrapError(Error("One and only one of 'date', 'created_before_date' and 'days' can be specified in one expiration configuration."))
				}
			}
			rule.Expiration = &i
		}

		// Transitions
		transitions := d.Get(fmt.Sprintf("lifecycle_rule.%d.transitions", i)).(*schema.Set).List()
		if len(transitions) > 0 {
			for _, transition := range transitions {
				i := ks3.LifecycleTransition{}

				valCreatedBeforeDate := transition.(map[string]interface{})["created_before_date"].(string)
				valDays := transition.(map[string]interface{})["days"].(int)
				valStorageClass := transition.(map[string]interface{})["storage_class"].(string)

				if (valCreatedBeforeDate != "" && valDays > 0) || (valCreatedBeforeDate == "" && valDays <= 0) || (valStorageClass == "") {
					return WrapError(Error("'CreatedBeforeDate' conflicts with 'Days'. One and only one of them can be specified in one transition configuration. 'storage_class' must be set."))
				}

				if valCreatedBeforeDate != "" {
					i.CreatedBeforeDate = fmt.Sprintf("%sT00:00:00.000Z", valCreatedBeforeDate)
				}
				if valDays > 0 {
					i.Days = valDays
				}

				if valStorageClass != "" {
					i.StorageClass = ks3.StorageClassType(valStorageClass)
				}
				rule.Transitions = append(rule.Transitions, i)
			}
		}

		// AbortMultipartUpload
		abortMultipartUpload := d.Get(fmt.Sprintf("lifecycle_rule.%d.abort_multipart_upload", i)).(*schema.Set).List()
		if len(abortMultipartUpload) > 0 {
			e := abortMultipartUpload[0].(map[string]interface{})
			i := ks3.LifecycleAbortMultipartUpload{}
			valCreatedBeforeDate, _ := e["created_before_date"].(string)
			valDays, _ := e["days"].(int)

			if (valCreatedBeforeDate != "" && valDays > 0) || (valCreatedBeforeDate == "" && valDays <= 0) {
				return WrapError(Error("'CreatedBeforeDate' conflicts with 'days'. One and only one of them can be specified in one abort_multipart_upload configuration."))
			}

			if valCreatedBeforeDate != "" {
				i.CreatedBeforeDate = fmt.Sprintf("%sT00:00:00.000Z", valCreatedBeforeDate)
			}
			if valDays > 0 {
				i.Days = valDays
			}
			rule.AbortMultipartUpload = &i
		}

		// Noncurrent Version Expiration
		noncurrentVersionExpiration := d.Get(fmt.Sprintf("lifecycle_rule.%d.noncurrent_version_expiration", i)).(*schema.Set).List()
		if len(noncurrentVersionExpiration) > 0 {
			e := noncurrentVersionExpiration[0].(map[string]interface{})
			i := ks3.LifecycleVersionExpiration{}
			valDays, _ := e["days"].(int)
			i.NoncurrentDays = valDays
			rule.NonVersionExpiration = &i
		}

		// Noncurrent Version Transitions
		noncurrentVersionTransitions := d.Get(fmt.Sprintf("lifecycle_rule.%d.noncurrent_version_transition", i)).(*schema.Set).List()
		if len(noncurrentVersionTransitions) > 0 {
			for _, transition := range noncurrentVersionTransitions {
				i := ks3.LifecycleVersionTransition{}

				valDays := transition.(map[string]interface{})["days"].(int)
				valStorageClass := transition.(map[string]interface{})["storage_class"].(string)

				i.NoncurrentDays = valDays
				i.StorageClass = ks3.StorageClassType(valStorageClass)

				rule.NonVersionTransitions = append(rule.NonVersionTransitions, i)
			}
		}

		rules = append(rules, rule)
	}

	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return nil, ks3Client.SetBucketLifecycle(bucket, rules)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketLifecycle", KsyunKs3GoSdk)
	}
	addDebug("SetBucketLifecycle", raw, requestInfo, map[string]interface{}{
		"bucketName": bucket,
		"rules":      rules,
	})
	return nil
}

func resourceAlicloudOssBucketPolicyUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	bucket := d.Id()
	policy := d.Get("policy").(string)
	var requestInfo *ks3.Client
	if len(policy) == 0 {
		params := map[string]interface{}{}
		params["policy"] = nil
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return ks3Client.Conn.Do("DELETE", bucket, "", params, nil, nil, 0, nil)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeletePolicyByConn", KsyunKs3GoSdk)
		}
		addDebug("DeletePolicyByConn", raw, requestInfo, params)
		return nil
	}
	params := map[string]interface{}{}
	params["policy"] = nil
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		buffer := new(bytes.Buffer)
		buffer.Write([]byte(policy))
		return ks3Client.Conn.Do("PUT", bucket, "", params, nil, buffer, 0, nil)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "PutPolicyByConn", KsyunKs3GoSdk)
	}
	addDebug("PutPolicyByConn", raw, requestInfo, params)
	return nil
}

func resourceAlicloudOssBucketEncryptionUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	encryption_rule := d.Get("server_side_encryption_rule").([]interface{})
	var requestInfo *ks3.Client
	if encryption_rule == nil || len(encryption_rule) == 0 {
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.DeleteBucketEncryption(d.Id())
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucketEncryption", KsyunKs3GoSdk)
		}
		addDebug("DeleteBucketEncryption", raw, requestInfo, map[string]string{"bucketName": d.Id()})
		return nil
	}

	var sseRule ks3.ServerEncryptionRule
	c := encryption_rule[0].(map[string]interface{})
	if v, ok := c["sse_algorithm"]; ok {
		sseRule.SSEDefault.SSEAlgorithm = v.(string)
	}

	if v, ok := c["kms_master_key_id"]; ok {
		sseRule.SSEDefault.KMSMasterKeyID = v.(string)
	}

	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return nil, ks3Client.SetBucketEncryption(d.Id(), sseRule)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketEncryption", KsyunKs3GoSdk)
	}
	addDebug("SetBucketEncryption", raw, requestInfo, map[string]interface{}{
		"bucketName":     d.Id(),
		"encryptionRule": sseRule,
	})
	return nil
}

func resourceAlicloudOssBucketTaggingUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	tagsMap := d.Get("tags").(map[string]interface{})
	var requestInfo *ks3.Client
	if tagsMap == nil || len(tagsMap) == 0 {
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.DeleteBucketTagging(d.Id())
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucketTagging", KsyunKs3GoSdk)
		}
		addDebug("DeleteBucketTagging", raw, requestInfo, map[string]string{"bucketName": d.Id()})
		return nil
	}

	// Put tagging
	var bTagging ks3.Tagging
	for k, v := range tagsMap {
		bTagging.Tags = append(bTagging.Tags, ks3.Tag{
			Key:   k,
			Value: v.(string),
		})
	}
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return nil, ks3Client.SetBucketTagging(d.Id(), bTagging)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketTagging", KsyunKs3GoSdk)
	}
	addDebug("SetBucketTagging", raw, requestInfo, map[string]interface{}{
		"bucketName": d.Id(),
		"tagging":    bTagging,
	})
	return nil
}

func resourceAlicloudOssBucketVersioningUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	versioning := d.Get("versioning").([]interface{})
	if len(versioning) == 1 {
		var status string
		c := versioning[0].(map[string]interface{})
		if v, ok := c["status"]; ok {
			status = v.(string)
		}

		versioningCfg := ks3.VersioningConfig{}
		versioningCfg.Status = status
		var requestInfo *ks3.Client
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.SetBucketVersioning(d.Id(), versioningCfg)
		})

		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketVersioning", KsyunKs3GoSdk)
		}
		addDebug("SetBucketVersioning", raw, requestInfo, map[string]interface{}{
			"bucketName":       d.Id(),
			"versioningConfig": versioningCfg,
		})
	}

	return nil
}

func resourceAlicloudOssBucketTransferAccUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
	acc := d.Get("transfer_acceleration").([]interface{})
	if len(acc) == 1 {
		var requestInfo *ks3.Client
		var aacCfg ks3.TransferAccConfiguration
		c := acc[0].(map[string]interface{})
		if v, ok := c["enabled"]; ok {
			aacCfg.Enabled = v.(bool)
		}

		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return nil, ks3Client.SetBucketTransferAcc(d.Id(), aacCfg)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), "SetBucketTransferAcc", KsyunKs3GoSdk)
		}
		addDebug("SetBucketTransferAcc", raw, requestInfo, map[string]interface{}{
			"bucketName":               d.Id(),
			"TransferAccConfiguration": aacCfg,
		})
	}

	return nil
}

func resourceAlicloudOssBucketDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	ks3Service := Ks3Service{client}
	var requestInfo *ks3.Client
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return ks3Client.IsBucketExist(d.Id())
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "IsBucketExist", KsyunKs3GoSdk)
	}
	addDebug("IsBucketExist", raw, requestInfo, map[string]string{"bucketName": d.Id()})

	exist, _ := raw.(bool)
	if !exist {
		return nil
	}

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			return nil, ks3Client.DeleteBucket(d.Id())
		})
		if err != nil {
			if IsExpectedErrors(err, []string{"BucketNotEmpty"}) {
				if d.Get("force_destroy").(bool) {
					raw, er := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
						bucket, _ := ks3Client.Bucket(d.Get("bucket").(string))
						lor, err := bucket.ListObjectVersions()
						if err != nil {
							return nil, WrapErrorf(err, DefaultErrorMsg, d.Id(), "ListObjectVersions", KsyunKs3GoSdk)
						}
						addDebug("ListObjectVersions", lor, requestInfo)
						objectsToDelete := make([]ks3.DeleteObject, 0)
						for _, object := range lor.ObjectDeleteMarkers {
							objectsToDelete = append(objectsToDelete, ks3.DeleteObject{
								Key:       object.Key,
								VersionId: object.VersionId,
							})
						}

						for _, object := range lor.ObjectVersions {
							objectsToDelete = append(objectsToDelete, ks3.DeleteObject{
								Key:       object.Key,
								VersionId: object.VersionId,
							})
						}
						return bucket.DeleteObjectVersions(objectsToDelete)
					})
					if er != nil {
						return resource.NonRetryableError(er)
					}
					addDebug("DeleteObjectVersions", raw, requestInfo, map[string]string{"bucketName": d.Id()})
					return resource.RetryableError(err)
				}
			}
			return resource.NonRetryableError(err)
		}
		addDebug("DeleteBucket", raw, requestInfo, map[string]string{"bucketName": d.Id()})
		return nil
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucket", KsyunKs3GoSdk)
	}
	return WrapError(ks3Service.WaitForOssBucket(d.Id(), Deleted, DefaultTimeoutMedium))
}

func expirationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["date"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["created_before_date"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["days"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["expired_object_delete_marker"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}
	return hashcode.String(buf.String())
}

func transitionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["created_before_date"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["storage_class"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["days"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	return hashcode.String(buf.String())
}

func abortMultipartUploadHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["created_before_date"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["days"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	return hashcode.String(buf.String())
}
