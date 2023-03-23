package connectivity

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"

	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type KsyunClient struct {
	Region    Region
	AccessKey string
	SecretKey string
	config    *Config
	ks3conn   *ks3.Client
	Endpoint  string
}

const DefaultClientRetryCountSmall = 5

const Terraform = "HashiCorp-Terraform"

const Provider = "Terraform-Provider"

const Module = "Terraform-Module"

var goSdkMutex = sync.RWMutex{} // The Go SDK is not thread-safe
var loadSdkfromRemoteMutex = sync.Mutex{}
var loadSdkEndpointMutex = sync.Mutex{}

// The main version number that is being run at the moment.
var providerVersion = "1.198.0"
var terraformVersion = strings.TrimSuffix(schema.Provider{}.TerraformVersion, "-dev")

// Client for KsyunClient
func (c *Config) Client() (*KsyunClient, error) {

	return &KsyunClient{
		config:    c,
		Region:    c.Region,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
		Endpoint:  c.Ks3Endpoint,
	}, nil
}

func (client *KsyunClient) GetRetryTimeout(defaultTimeout time.Duration) time.Duration {

	maxRetryTimeout := client.config.MaxRetryTimeout
	if maxRetryTimeout != 0 {
		return time.Duration(maxRetryTimeout) * time.Second
	}

	return defaultTimeout
}

func (client *KsyunClient) WithKs3Client(do func(*ks3.Client) (interface{}, error)) (interface{}, error) {
	goSdkMutex.Lock()
	defer goSdkMutex.Unlock()
	// Initialize the KS3 client if necessary
	if client.ks3conn == nil {
		ks3conn, err := ks3.New(client.Endpoint, client.AccessKey, client.SecretKey)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the KS3 client: %#v", err)
		}
		client.ks3conn = ks3conn
	}
	return do(client.ks3conn)
}

func (client *KsyunClient) WithKs3BucketByName(bucketName string, do func(*ks3.Bucket) (interface{}, error)) (interface{}, error) {
	return client.WithKs3Client(func(ks3Client *ks3.Client) (interface{}, error) {
		bucket, err := client.ks3conn.Bucket(bucketName)
		if err != nil {
			return nil, fmt.Errorf("unable to get the bucket %s: %#v", bucketName, err)
		}
		return do(bucket)
	})
}

func (client *KsyunClient) getSdkConfig() *sdk.Config {
	return sdk.NewConfig().
		WithMaxRetryTime(DefaultClientRetryCountSmall).
		WithTimeout(time.Duration(30) * time.Second).
		WithEnableAsync(false).
		WithGoRoutinePoolSize(100).
		WithMaxTaskQueueSize(10000).
		WithDebug(true).
		WithHttpTransport(client.getTransport()).
		WithScheme(client.config.Protocol)
}

func (client *KsyunClient) getUserAgent() string {
	return fmt.Sprintf("%s/%s %s/%s %s/%s", Terraform, terraformVersion, Provider, providerVersion, Module, client.config.ConfigurationSource)
}

func (client *KsyunClient) getTransport() *http.Transport {
	handshakeTimeout, err := strconv.Atoi(os.Getenv("TLSHandshakeTimeout"))
	if err != nil {
		handshakeTimeout = 120
	}
	transport := &http.Transport{}
	transport.TLSHandshakeTimeout = time.Duration(handshakeTimeout) * time.Second

	return transport
}
