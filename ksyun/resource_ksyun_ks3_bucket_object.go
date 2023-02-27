package ksyun

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/mitchellh/go-homedir"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"github.com/wilac-pv/terraform-provider-ks3/ksyun/connectivity"
	"io"
	"log"
	"strings"
	"time"
)

func resourceKsyunKs3BucketObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceKsyunKs3BucketObjectPut,
		Read:   resourceKsyunKs3BucketObjectRead,
		Update: resourceKsyunKs3BucketObjectPut,
		Delete: resourceKsyunKs3BucketObjectDelete,

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"content"},
			},

			"content": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"source"},
			},

			"acl": {
				Type:         schema.TypeString,
				Default:      ks3.ACLPrivate,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"private", "public-read", "public-read-write"}, false),
			},

			"content_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"content_length": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cache_control": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_disposition": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_encoding": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_md5": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"expires": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"server_side_encryption": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(ServerSideEncryptionKMS), string(ServerSideEncryptionAes256),
				}, false),
				Default: ServerSideEncryptionAes256,
			},

			"kms_key_id": {
				Type:     schema.TypeString,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return ServerSideEncryptionKMS != d.Get("server_side_encryption").(string)
				},
			},

			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"version_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceKsyunKs3BucketObjectPut(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	var requestInfo *ks3.Client
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return ks3Client.Bucket(d.Get("bucket").(string))
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "ksyun_ks3_bucket_object", "Bucket", KsyunKs3GoSdk)
	}
	addDebug("Bucket", raw, requestInfo, map[string]string{"bucketName": d.Get("bucket").(string)})
	bucket, _ := raw.(*ks3.Bucket)
	var filePath string
	var body io.Reader

	if v, ok := d.GetOk("source"); ok {
		source := v.(string)
		path, err := homedir.Expand(source)
		if err != nil {
			return WrapError(err)
		}

		filePath = path
	} else if v, ok := d.GetOk("content"); ok {
		content := v.(string)
		body = bytes.NewReader([]byte(content))
	} else {
		return WrapError(Error("[ERROR] Must specify \"source\" or \"content\" field"))
	}

	key := d.Get("key").(string)
	options, err := buildObjectHeaderOptions(d)

	if v, ok := d.GetOk("server_side_encryption"); ok {
		options = append(options, ks3.ServerSideEncryption(v.(string)))
	}

	if v, ok := d.GetOk("kms_key_id"); ok {
		options = append(options, ks3.ServerSideEncryptionKeyID(v.(string)))
	}

	if err != nil {
		return WrapError(err)
	}
	if filePath != "" {
		err = bucket.PutObjectFromFile(key, filePath, options...)
	}

	if body != nil {
		err = bucket.PutObject(key, body, options...)
	}

	if err != nil {
		return WrapError(Error("Error putting object in Oss bucket (%#v): %s", bucket, err))
	}

	d.SetId(key)
	return resourceKsyunKs3BucketObjectRead(d, meta)
}

func resourceKsyunKs3BucketObjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	var requestInfo *ks3.Client
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return ks3Client.Bucket(d.Get("bucket").(string))
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "Bucket", KsyunKs3GoSdk)
	}
	addDebug("Bucket", raw, requestInfo, map[string]string{"bucketName": d.Get("bucket").(string)})
	bucket, _ := raw.(*ks3.Bucket)
	options, err := buildObjectHeaderOptions(d)
	if err != nil {
		return WrapError(err)
	}

	object, err := bucket.GetObjectDetailedMeta(d.Get("key").(string), options...)
	if err != nil {
		if IsExpectedErrors(err, []string{"404 Not Found"}) {
			d.SetId("")
			return WrapError(Error("To get the Object: %#v but it is not exist in the specified bucket %s.", d.Get("key").(string), d.Get("bucket").(string)))
		}
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "GetObjectDetailedMeta", KsyunKs3GoSdk)
	}
	addDebug("GetObjectDetailedMeta", object, requestInfo, map[string]interface{}{
		"objectKey": d.Get("key").(string),
		"options":   options,
	})

	d.Set("content_type", object.Get("Content-Type"))
	d.Set("content_length", object.Get("Content-Length"))
	d.Set("cache_control", object.Get("Cache-Control"))
	d.Set("content_disposition", object.Get("Content-Disposition"))
	d.Set("content_encoding", object.Get("Content-Encoding"))
	d.Set("expires", object.Get("Expires"))
	d.Set("etag", strings.Trim(object.Get("ETag"), `"`))
	d.Set("version_id", object.Get("x-ks3-version-id"))

	return nil
}

func resourceKsyunKs3BucketObjectDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.KsyunClient)
	ks3Service := Ks3Service{client}
	var requestInfo *ks3.Client
	raw, err := client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		requestInfo = ks3Client
		return ks3Client.Bucket(d.Get("bucket").(string))
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "Bucket", KsyunKs3GoSdk)
	}
	addDebug("Bucket", raw, requestInfo, map[string]string{"bucketName": d.Get("bucket").(string)})
	bucket, _ := raw.(*ks3.Bucket)

	err = bucket.DeleteObject(d.Id())
	if err != nil {
		if IsExpectedErrors(err, []string{"No Content", "Not Found"}) {
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), "DeleteObject", KsyunKs3GoSdk)
	}

	return WrapError(ks3Service.WaitForOssBucketObject(bucket, d.Id(), Deleted, DefaultTimeoutMedium))

}

func buildObjectHeaderOptions(d *schema.ResourceData) (options []ks3.Option, err error) {

	if v, ok := d.GetOk("acl"); ok {
		options = append(options, ks3.ObjectACL(ks3.ACLType(v.(string))))
	}

	if v, ok := d.GetOk("content_type"); ok {
		options = append(options, ks3.ContentType(v.(string)))
	}

	if v, ok := d.GetOk("cache_control"); ok {
		options = append(options, ks3.CacheControl(v.(string)))
	}

	if v, ok := d.GetOk("content_disposition"); ok {
		options = append(options, ks3.ContentDisposition(v.(string)))
	}

	if v, ok := d.GetOk("content_encoding"); ok {
		options = append(options, ks3.ContentEncoding(v.(string)))
	}

	if v, ok := d.GetOk("content_md5"); ok {
		options = append(options, ks3.ContentMD5(v.(string)))
	}

	if v, ok := d.GetOk("expires"); ok {
		expires := v.(string)
		expiresTime, err := time.Parse(time.RFC1123, expires)
		if err != nil {
			return nil, fmt.Errorf("expires format must respect the RFC1123 standard (current value: %s)", expires)
		}
		options = append(options, ks3.Expires(expiresTime))
	}

	if options == nil || len(options) == 0 {
		log.Printf("[WARN] Object header options is nil.")
	}
	return options, nil
}
