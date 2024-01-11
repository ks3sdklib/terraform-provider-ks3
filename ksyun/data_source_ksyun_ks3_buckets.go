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
							MaxItems: 1,
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
						},

						"lifecycle_rule": {
							Type:     schema.TypeList,
							Computed: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"filter": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"prefix": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"and": {
													Type:     schema.TypeList,
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
											},
										},
									},
									"expiration": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"date": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"days": {
													Type:         schema.TypeInt,
													Optional:     true,
													ValidateFunc: validation.IntAtLeast(0),
												},
											},
										},
									},
									"transition": {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"date": {
													Type:     schema.TypeString,
													Optional: true,
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
								},
							},
						},

						"policy": {
							Type:     schema.TypeString,
							Optional: true,
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
			mapping["location"] = response.BucketInfo.Region
			mapping["storage_class"] = response.BucketInfo.StorageClass
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
					if lifecycleRule.Prefix != "" {
						ruleMapping["prefix"] = lifecycleRule.Prefix
					}
					if lifecycleRule.Filter != nil {
						filter := make(map[string]interface{})
						if lifecycleRule.Filter.Prefix != "" {
							filter["prefix"] = lifecycleRule.Filter.Prefix
						}
						// and
						if lifecycleRule.Filter.And != nil {
							and := make(map[string]interface{})
							if lifecycleRule.Filter.And.Prefix != "" {
								and["prefix"] = lifecycleRule.Filter.And.Prefix
							}
							if len(lifecycleRule.Filter.And.Tag) != 0 {
								var tags []interface{}
								for _, tag := range lifecycleRule.Filter.And.Tag {
									e := make(map[string]interface{})
									e["key"] = tag.Key
									e["value"] = tag.Value
									tags = append(tags, e)
								}
								and["tag"] = tags
							}
							filter["and"] = []interface{}{and}
						}
						ruleMapping["filter"] = []interface{}{filter}
					}
					if LifecycleRuleStatus(lifecycleRule.Status) == ExpirationStatusEnabled {
						ruleMapping["enabled"] = true
					} else {
						ruleMapping["enabled"] = false
					}
					// Expiration
					expirationMapping := make(map[string]interface{})
					if lifecycleRule.Expiration.Date != "" {
						t, err := time.Parse(Iso8601DateFormat, lifecycleRule.Expiration.Date)
						if err != nil {
							return WrapError(err)
						}
						expirationMapping["date"] = t.Format("2006-01-02")
					}
					if lifecycleRule.Expiration.Days != 0 {
						expirationMapping["days"] = lifecycleRule.Expiration.Days
					}
					ruleMapping["expiration"] = []interface{}{expirationMapping}
					lifecycleRuleMappings = append(lifecycleRuleMappings, ruleMapping)
					// Transition
					var transitionList []interface{}
					if len(lifecycleRule.Transitions) > 0 {
						for _, transition := range lifecycleRule.Transitions {
							transitionMapping := make(map[string]interface{})
							if transition.Date != "" {
								t, err := time.Parse(Iso8601DateFormat, transition.Date)
								if err != nil {
									return WrapError(err)
								}
								transitionMapping["date"] = t.Format("2006-01-02")
							}
							if transition.Days != 0 {
								transitionMapping["days"] = transition.Days
							}
							if transition.StorageClass != "" {
								transitionMapping["storage_class"] = transition.StorageClass
							}
							transitionList = append(transitionList, transitionMapping)
						}
					}
					ruleMapping["transition"] = transitionList
				}
			}
		} else {
			log.Printf("[WARN] Unable to get lifecycle information for the bucket %s: %v", bucket.Name, err)
		}
		mapping["lifecycle_rule"] = lifecycleRuleMappings

		// Add policy information
		var policy string
		raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			requestInfo = ks3Client
			return ks3Client.GetBucketPolicy(bucket.Name)
		})

		if err == nil {
			if debugOn() {
				addDebug("GetBucketPolicy", raw, requestInfo, map[string]string{"bucketName": bucket.Name})
			}
			policy = raw.(string)
		} else {
			log.Printf("[WARN] Unable to get policy information for the bucket %s: %v", bucket.Name, err)
		}
		mapping["policy"] = policy

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
	acl, err := client.GetBucketACL(bucket)
	if err != nil {
		return ks3.GetBucketInfoResult{}, err
	}
	resp, err := client.ListBuckets()
	if err != nil {
		return ks3.GetBucketInfoResult{}, err
	}
	for _, bucketInfo := range resp.Buckets {
		if bucketInfo.Name == bucket {
			return ks3.GetBucketInfoResult{
				BucketInfo: ks3.BucketInfo{
					XMLName:      bucketInfo.XMLName,
					Name:         bucket,
					Region:       bucketInfo.Region,
					ACL:          string(acl.GetCannedACL()),
					StorageClass: bucketInfo.Type,
				}}, nil
		}
	}
	return ks3.GetBucketInfoResult{}, nil
}
