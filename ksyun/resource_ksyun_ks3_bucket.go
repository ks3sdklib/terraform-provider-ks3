package ksyun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"log"
	"strconv"
	"strings"
	"time"
)

func resourceKsyunKs3Bucket() *schema.Resource {
	return &schema.Resource{
		Create: resourceKsyunKs3BucketCreate,
		Read:   resourceKsyunKs3BucketRead,
		Update: resourceKsyunKs3BucketUpdate,
		Delete: resourceKsyunKs3BucketDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(3, 63),
				Default:      resource.PrefixedUniqueId("ks3-bucket-"),
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
						},
						"filter": {
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 1,
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
													Required: true,
												},
												"value": {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
								},
							},
						},

						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
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
										Type:     schema.TypeInt,
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
				MaxItems: 10,
			},

			"storage_class": {
				Type:     schema.TypeString,
				Default:  ks3.TypeNormal,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(ks3.TypeNormal),
					string(ks3.TypeIA),
					string(ks3.TypeArchive),
				}, false),
			},
		},
	}
}

func resourceKsyunKs3BucketCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	request := map[string]string{"bucketName": d.Get("bucket").(string)}
	var requestInfo *ks3.Client
	type Request struct {
		BucketName         string
		StorageClassOption ks3.Option
		AclTypeOption      ks3.Option
	}

	req := Request{
		d.Get("bucket").(string),
		ks3.BucketTypeClass(ks3.BucketType(d.Get("storage_class").(string))),
		ks3.ACL(ks3.ACLType(d.Get("acl").(string))),
	}
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		return nil, ks3Client.CreateBucket(req.BucketName, req.StorageClassOption, req.AclTypeOption)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "ksyun_ks3_bucket", "CreateBucket", KsyunKs3GoSdk)
	}
	addDebug("CreateBucket", raw, requestInfo, req)
	d.SetId(request["bucketName"])

	return resourceKsyunKs3BucketUpdate(d, meta)
}

func resourceKsyunKs3BucketRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	ks3Service := Ks3Service{client}
	object, err := ks3Service.DescribeKs3Bucket(d.Id())
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
	d.Set("location", object.BucketInfo.Location)
	d.Set("owner", object.BucketInfo.Owner.ID)
	d.Set("storage_class", object.BucketInfo.StorageClass)

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
				lifecycleRule.Expiration.Date = strings.ReplaceAll(lifecycleRule.Expiration.Date, ".000", "")
				t, err := time.Parse(Iso8601DateFormat, lifecycleRule.Expiration.Date)
				if err != nil {
					return WrapError(err)
				}
				e["date"] = t.Format("2006-01-02")
			}
			e["days"] = strconv.Itoa(lifecycleRule.Expiration.Days)
			rule["expiration"] = e
		}
		// transitions
		if len(lifecycleRule.Transitions) != 0 {
			var eSli []interface{}
			for _, transition := range lifecycleRule.Transitions {
				e := make(map[string]interface{})
				if transition.Date != "" {
					transition.Date = strings.ReplaceAll(transition.Date, ".000", "")
					t, err := time.Parse(Iso8601DateFormat, transition.Date)
					if err != nil {
						return WrapError(err)
					}
					e["date"] = t.Format("2006-01-02")
				}
				e["days"] = transition.Days
				e["storage_class"] = string(transition.StorageClass)
				eSli = append(eSli, e)
			}
			rule["transitions"] = schema.NewSet(transitionsHash, eSli)
		}
		lrules = append(lrules, rule)
	}

	if err := d.Set("lifecycle_rule", lrules); err != nil {
		return WrapError(err)
	}

	return nil
}

func resourceKsyunKs3BucketUpdate(d *schema.ResourceData, meta interface{}) error {
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
		if err := resourceKsyunKs3BucketCorsUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("cors_rule")
	}
	if d.HasChange("logging") {
		if err := resourceKsyunKs3BucketLoggingUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("logging")
	}

	if d.HasChange("lifecycle_rule") {
		if err := resourceKsyunKs3BucketLifecycleRuleUpdate(client, d); err != nil {
			return WrapError(err)
		}
		d.SetPartial("lifecycle_rule")
	}

	d.Partial(false)
	return resourceKsyunKs3BucketRead(d, meta)
}

func resourceKsyunKs3BucketCorsUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
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
			log.Printf("[DEBUG] KS3 bucket: %s, put CORS: %#v, %#v", d.Id(), k, v)
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

	log.Printf("[DEBUG] Ks3 bucket: %s, put CORS: %#v", d.Id(), cors)
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
func resourceKsyunKs3BucketLoggingUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
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

func resourceKsyunKs3BucketLifecycleRuleUpdate(client *connectivity.KsyunClient, d *schema.ResourceData) error {
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

	for _, lifecycleRule := range lifecycleRules {
		r := lifecycleRule.(map[string]interface{})
		rule := ks3.LifecycleRule{}
		// ID--有值
		if val, ok := r["id"].(string); ok && val != "" {
			rule.ID = val
		}

		// Enabled
		if val, ok := r["enabled"].(bool); ok && val {
			rule.Status = string(ExpirationStatusEnabled)
		} else {
			rule.Status = string(ExpirationStatusDisabled)
		}
		if filterSet, ok := r["filter"].(*schema.Set); ok {
			if filterSet.Len() > 0 {
				if filter, ok := filterSet.List()[0].(map[string]interface{}); ok {
					filterModel := &ks3.LifecycleFilter{
						And: ks3.LifecycleAnd{},
					}
					filterModel.And.Prefix = filter["prefix"].(string)
					tagList := filter["tag"].([]interface{})
					for _, tag := range tagList {
						tagMap := tag.(map[string]interface{})
						key := tagMap["key"].(string)
						value := tagMap["value"].(string)
						filterModel.And.Tag = append(filterModel.And.Tag, ks3.Tag{
							Key:   key,
							Value: value,
						})
					}
					rule.Filter = filterModel
				}
			}
		}
		json_p, _ := json.Marshal(rule.Filter)
		fmt.Printf("rule.filter=%s\n", json_p)

		// Expiration
		expirationMap, ok := r["expiration"].(map[string]interface{})
		if ok {
			expirationTmp := ks3.LifecycleExpiration{}
			valDate, _ := expirationMap["date"].(string)
			valDays, _ := expirationMap["days"].(int)
			fmt.Printf("---valDate:%s", valDate)
			fmt.Printf("---valDays:%d", valDays)
			cnt := 0
			if valDate != "" {
				expirationTmp.Date = fmt.Sprintf("%sT00:00:00+08:00", valDate)
				cnt++
			}
			if valDays > 0 {
				expirationTmp.Days = valDays
				cnt++
			}
			if cnt != 1 {
				return WrapError(Error("One and only one of 'date', 'date' and 'days' can be specified in one expiration configuration."))
			}
			rule.Expiration = &expirationTmp
		}
		json_p, _ = json.Marshal(rule.Expiration)
		fmt.Printf("rule.Expiration=%s\n", json_p)

		// Transitions
		transitionsRaw := r["transitions"]
		if transitionsRaw != nil {
			transitions := transitionsRaw.(*schema.Set).List()
			if len(transitions) > 0 {
				for _, transition := range transitions {
					transitionTmp := ks3.LifecycleTransition{}
					valDays := transition.(map[string]interface{})["days"].(int)
					valStorageClass := transition.(map[string]interface{})["storage_class"].(string)
					date := transition.(map[string]interface{})["date"].(string)
					if date != "" {
						transitionTmp.Date = fmt.Sprintf("%sT00:00:00+08:00", date)
					}
					if valDays > 0 {
						transitionTmp.Days = valDays
					}

					if valStorageClass != "" {
						transitionTmp.StorageClass = ks3.StorageClassType(valStorageClass)
					}
					rule.Transitions = append(rule.Transitions, transitionTmp)
				}
			}
		}
		json_p, _ = json.Marshal(rule.Transitions)
		fmt.Printf("rule.Transitions=%s\n", json_p)
		fmt.Println("---------------------------------")
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

func resourceKsyunKs3BucketDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	var requestInfo *ks3.Client
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return ks3Client.IsBucketExist(d.Id())
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "IsBucketExist", KsyunKs3GoSdk)
	}
	addDebug("IsBucketExist", raw, requestInfo, map[string]string{"bucketName": d.Id()})

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		raw, err = client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
			return nil, ks3Client.DeleteBucket(d.Id())
		})
		if err != nil {
			if IsExpectedErrors(err, []string{"BucketNotEmpty"}) {
				raw, er := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
					bucket, _ := ks3Client.Bucket(d.Get("bucket").(string))
					marker := ""
					retryTime := 3
					for retryTime > 0 {
						lsRes, err := bucket.ListObjects(ks3.Marker(marker))
						if err != nil {
							retryTime--
							time.Sleep(time.Duration(1) * time.Second)
							continue
						}
						for _, object := range lsRes.Objects {
							bucket.DeleteObject(object.Key)
						}
						if lsRes.IsTruncated {
							marker = lsRes.NextMarker
						} else {
							return true, nil
						}
					}
					return false, nil
				})
				if er != nil {
					return resource.NonRetryableError(er)
				}
				addDebug("DeleteObjects", raw, requestInfo, map[string]string{"bucketName": d.Id()})
				return resource.RetryableError(err)
			}
		}
		return resource.NonRetryableError(err)
		addDebug("DeleteBucket", raw, requestInfo, map[string]string{"bucketName": d.Id()})
		return nil
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteBucket", KsyunKs3GoSdk)
	}
	return nil
}

func transitionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["date"]; ok {
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

func expirationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["date"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["days"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	return hashcode.String(buf.String())
}
