package ksyun

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"log"
	"regexp"
	"time"
)

func dataSourceKsyunKs3Buckets() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKsyunKs3BucketsRead,

		Schema: map[string]*schema.Schema{
			"name_regex": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.ValidateRegexp,
				ForceNew:     true,
			},
			"output_file": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Computed values
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"buckets": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"acl": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"location": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"storage_class": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"creation_date": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"cors_rules": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"allowed_headers": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"allowed_methods": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"allowed_origins": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"expose_headers": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"max_age_seconds": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"logging": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"target_bucket": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"target_prefix": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
							MaxItems: 1,
						},
						"lifecycle_rule": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"filter": {
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"and": {
													Type:     schema.TypeSet,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"prefix": {
																Type:     schema.TypeString,
																Optional: true,
															},
															"tag": {
																Type:     schema.TypeList,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"key": {
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"value": {
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																	},
																},
															},
														},
													},
												},
												"prefix": {
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"expiration": {
										Type:     schema.TypeMap,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"date": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"days": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"tests": {
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"transitions": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"date": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"days": {
													Type:     schema.TypeString,
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
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceKsyunKs3BucketsRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	var requestInfo *ks3.Client
	var allBuckets []ks3.BucketProperties
	nextMarker := ""
	for {
		var options []ks3.Option
		if nextMarker != "" {
			options = append(options, ks3.Marker(nextMarker))
		}

		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return ks3Client.ListBuckets(options...)
		})
		if err != nil {
			return WrapErrorf(err, DataDefaultErrorMsg, "ksyun_ks3_bucket", "CreateBucket", KsyunKs3GoSdk)
		}
		if debugOn() {
			addDebug("ListBuckets", raw, requestInfo, map[string]interface{}{"options": options})
		}
		response, _ := raw.(ks3.ListBucketsResult)

		if response.Buckets == nil || len(response.Buckets) < 1 {
			break
		}

		allBuckets = append(allBuckets, response.Buckets...)

		nextMarker = response.NextMarker
		if nextMarker == "" {
			break
		}
	}

	var filteredBucketsTemp []ks3.BucketProperties
	nameRegex, ok := d.GetOk("name_regex")
	if ok && nameRegex.(string) != "" {
		var ks3BucketNameRegex *regexp.Regexp
		if nameRegex != "" {
			r, err := regexp.Compile(nameRegex.(string))
			if err != nil {
				return WrapError(err)
			}
			ks3BucketNameRegex = r
		}
		for _, bucket := range allBuckets {
			if ks3BucketNameRegex != nil && !ks3BucketNameRegex.MatchString(bucket.Name) {
				continue
			}
			filteredBucketsTemp = append(filteredBucketsTemp, bucket)
		}
	} else {
		filteredBucketsTemp = allBuckets
	}
	return bucketsDescriptionAttributes(d, filteredBucketsTemp, meta)
}

func bucketsDescriptionAttributes(d *schema.ResourceData, buckets []ks3.BucketProperties, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)

	var ids []string
	var s []map[string]interface{}
	var names []string
	var requestInfo *ks3.Client
	for _, bucket := range buckets {
		mapping := map[string]interface{}{
			"name":          bucket.Name,
			"creation_date": bucket.CreationDate.Format("2006-01-02"),
		}

		// Add additional information
		raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return GetBucketInfo(ks3Client, bucket.Name)
		})
		if err == nil {
			if debugOn() {
				addDebug("GetBucketInfo", raw, requestInfo, map[string]string{"bucketName": bucket.Name})
			}
			response, _ := raw.(ks3.GetBucketInfoResult)
			mapping["acl"] = response.BucketInfo.ACL
			mapping["extranet_endpoint"] = response.BucketInfo.ExtranetEndpoint
			mapping["intranet_endpoint"] = response.BucketInfo.IntranetEndpoint
			mapping["owner"] = response.BucketInfo.Owner.ID

			//Add ServerSideEncryption information
			var sseconfig []map[string]interface{}
			if &response.BucketInfo.SseRule != nil {
				if len(response.BucketInfo.SseRule.SSEAlgorithm) > 0 && response.BucketInfo.SseRule.SSEAlgorithm != "None" {
					data := map[string]interface{}{
						"sse_algorithm": response.BucketInfo.SseRule.SSEAlgorithm,
					}
					if response.BucketInfo.SseRule.KMSMasterKeyID != "" {
						data["kms_master_key_id"] = response.BucketInfo.SseRule.KMSMasterKeyID
					}
					sseconfig = make([]map[string]interface{}, 0)
					sseconfig = append(sseconfig, data)
				}
			}
			mapping["server_side_encryption_rule"] = sseconfig

		} else {
			log.Printf("[WARN] Unable to get additional information for the bucket %s: %v", bucket.Name, err)
		}

		// Add CORS rule information
		var ruleMappings []map[string]interface{}
		raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return ks3Client.GetBucketCORS(bucket.Name)
		})
		if err == nil {
			if debugOn() {
				addDebug("GetBucketCORS", raw, requestInfo, map[string]string{"bucketName": bucket.Name})
			}
			cors, _ := raw.(ks3.GetBucketCORSResult)
			if cors.CORSRules != nil {
				for _, rule := range cors.CORSRules {
					ruleMapping := make(map[string]interface{})
					ruleMapping["allowed_headers"] = rule.AllowedHeader
					ruleMapping["allowed_methods"] = rule.AllowedMethod
					ruleMapping["allowed_origins"] = rule.AllowedOrigin
					ruleMapping["expose_headers"] = rule.ExposeHeader
					ruleMapping["max_age_seconds"] = rule.MaxAgeSeconds
					ruleMappings = append(ruleMappings, ruleMapping)
				}
			}
		} else if !IsExpectedErrors(err, []string{"NoSuchCORSConfiguration"}) {
			log.Printf("[WARN] Unable to get CORS information for the bucket %s: %v", bucket.Name, err)
		}
		mapping["cors_rules"] = ruleMappings

		// Add logging information
		var loggingMappings []map[string]interface{}
		raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			return ks3Client.GetBucketLogging(bucket.Name)
		})
		if err == nil {
			addDebug("GetBucketLogging", raw)
			logging, _ := raw.(ks3.GetBucketLoggingResult)
			if logging.LoggingEnabled.TargetBucket != "" || logging.LoggingEnabled.TargetPrefix != "" {
				loggingMapping := map[string]interface{}{
					"target_bucket": logging.LoggingEnabled.TargetBucket,
					"target_prefix": logging.LoggingEnabled.TargetPrefix,
				}
				loggingMappings = append(loggingMappings, loggingMapping)
			}
		} else {
			log.Printf("[WARN] Unable to get logging information for the bucket %s: %v", bucket.Name, err)
		}
		mapping["logging"] = loggingMappings

		// Add lifecycle information
		var lifecycleRuleMappings []map[string]interface{}
		raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return ks3Client.GetBucketLifecycle(bucket.Name)
		})
		if err == nil {
			if debugOn() {
				addDebug("GetBucketLifecycle", raw, requestInfo, map[string]string{"bucketName": bucket.Name})
			}
			lifecycle, _ := raw.(ks3.GetBucketLifecycleResult)
			if len(lifecycle.Rules) > 0 {
				for _, lifecycleRule := range lifecycle.Rules {
					ruleMapping := make(map[string]interface{})
					ruleMapping["id"] = lifecycleRule.ID
					if lifecycleRule.Filter != nil {
						l := make(map[string]interface{})
						if lifecycleRule.Prefix != "" {
							l["prefix"] = lifecycleRule.Prefix
						}
						// and
						if &lifecycleRule.Filter.And != nil {
							if len(lifecycleRule.Filter.And.Tag) != 0 {
								var eSli []interface{}
								for _, tag := range lifecycleRule.Filter.And.Tag {
									e := make(map[string]interface{})
									e["key"] = tag.Key
									e["value"] = tag.Value
									eSli = append(eSli, e)
								}
								l["and"] = eSli
							}
						}
						ruleMapping["filter"] = l
					}
					if LifecycleRuleStatus(lifecycleRule.Status) == ExpirationStatusEnabled {
						ruleMapping["enabled"] = true
					} else {
						ruleMapping["enabled"] = false
					}
					// expiration
					//if lifecycleRule.Expiration != nil {
					//	e := make(map[string]interface{})
					//	if lifecycleRule.Expiration.Date != "" {
					//		t, err := time.Parse(Iso8601DateFormat, lifecycleRule.Expiration.Date)
					//		if err != nil {
					//			return WrapError(err)
					//		}
					//		e["date"] = t.Format("2006-01-02")
					//	}
					//	e["days"] = lifecycleRule.Expiration.Days
					//	ruleMapping["expiration"] = lifecycleRule.Expiration
					//}

					//Expiration
					expirationMapping := make(map[string]interface{})
					if lifecycleRule.Expiration.Date != "" {
						t, err := time.Parse(Iso8601DateFormat, lifecycleRule.Expiration.Date)
						if err != nil {
							return WrapError(err)
						}
						expirationMapping["date"] = t.Format("2006-01-02")
					}
					if &lifecycleRule.Expiration.Days != nil {
						expirationMapping["days"] = lifecycleRule.Expiration.Days
					}
					ruleMapping["expiration"] = expirationMapping
					lifecycleRuleMappings = append(lifecycleRuleMappings, ruleMapping)
				}
			}

		} else {
			log.Printf("[WARN] Unable to get lifecycle information for the bucket %s: %v", bucket.Name, err)
		}
		mapping["lifecycle_rule"] = lifecycleRuleMappings
		ids = append(ids, bucket.Name)
		s = append(s, mapping)
		names = append(names, bucket.Name)
	}

	d.SetId(dataResourceIdHash(ids))
	if err := d.Set("buckets", s); err != nil {
		return WrapError(err)
	}
	if err := d.Set("names", names); err != nil {
		return WrapError(err)
	}
	// create a json file in current directory and write data source to it.
	if output, ok := d.GetOk("output_file"); ok && output.(string) != "" {
		writeToFile(output.(string), s)
	}
	return nil
}

func GetBucketInfo(client *ks3.Client, bucket string) (ks3.GetBucketInfoResult, error) {

	resp, err := client.ListBuckets()
	if err == nil {
		for _, bucketInfo := range resp.Buckets {
			if bucketInfo.Name == bucket {
				return ks3.GetBucketInfoResult{
					BucketInfo: ks3.BucketInfo{
						XMLName: bucketInfo.XMLName,
						Name:    bucket,
						Region:  bucketInfo.Region,
						//GET BUCKETACL
						//ACL:          bucketInfo.Type,
						StorageClass: bucketInfo.Type,
					}}, nil
			}
		}
	}
	return ks3.GetBucketInfoResult{}, err
}
