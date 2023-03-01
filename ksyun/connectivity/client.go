package connectivity

import (
	rpc "github.com/alibabacloud-go/tea-rpc/client"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
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
	Region          Region
	SourceIp        string
	SecureTransport string
	AccessKey       string
	SecretKey       string
	SecurityToken   string
	accountIdMutex  sync.RWMutex
	config          *Config
	teaSdkConfig    rpc.Config
	accountId       string
	ks3conn         *ks3.Client
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
	accessKey := os.Getenv("KS3_ACCESS_KEY_ID")
	secretKey := os.Getenv("KS3_ACCESS_KEY_SECRET")
	region := os.Getenv("KS3_REGION")
	securityToken := os.Getenv("KSYUN_SECURITY_TOKEN")

	return &KsyunClient{
		config:        c,
		Region:        Region(region),
		AccessKey:     accessKey,
		SecretKey:     secretKey,
		SecurityToken: securityToken,
	}, nil
}

type ks3Credentials struct {
	client *KsyunClient
}

func (defCre *ks3Credentials) GetAccessKeyID() string {

	accessKey := os.Getenv("KS3_ACCESS_KEY_ID")
	return accessKey
}

func (defCre *ks3Credentials) GetAccessKeySecret() string {
	secretKey := os.Getenv("KS3_ACCESS_KEY_SECRET")
	return secretKey
}

func (defCre *ks3Credentials) GetSecurityToken() string {
	return ""
}

type ks3CredentialsProvider struct {
	client *KsyunClient
}

func (defBuild *ks3CredentialsProvider) GetCredentials() ks3.Credentials {
	return &ks3Credentials{client: defBuild.client}
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
		schema := strings.ToLower(client.config.Protocol)
		endpoint := client.config.Ks3Endpoint
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = fmt.Sprintf("%s://%s", schema, endpoint)
		}
		if endpoint == "" {
			endpoint = ""
		}

		clientOptions := []ks3.ClientOption{ks3.UserAgent(client.getUserAgent())}
		clientOptions = append(clientOptions, ks3.SetCredentialsProvider(&ks3CredentialsProvider{client: client}))

		ks3conn, err := ks3.New(endpoint, "", "", clientOptions...)
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

func (client *KsyunClient) NewTeaCommonClient(endpoint string) (*rpc.Client, error) {
	sdkConfig := client.teaSdkConfig
	sdkConfig.SetEndpoint(endpoint)

	conn, err := rpc.NewClient(&sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the tea client: %#v", err)
	}

	return conn, nil
}

func (client *KsyunClient) NewCommonRequest(schema string) (*requests.CommonRequest, error) {
	endpoint := ""
	request := requests.NewCommonRequest()
	// Use product code to find product domain
	if endpoint != "" {
		request.Domain = endpoint
	} else {
		// When getting endpoint failed by location, using custom endpoint instead
		request.Domain = "ks3-cn-beijing.ksyuncs.com"
	}
	request.Scheme = schema
	request.SetReadTimeout(time.Duration(client.config.ClientReadTimeout) * time.Millisecond)
	request.SetConnectTimeout(time.Duration(client.config.ClientConnectTimeout) * time.Millisecond)
	request.AppendUserAgent(Terraform, terraformVersion)
	request.AppendUserAgent(Provider, providerVersion)
	request.AppendUserAgent(Module, client.config.ConfigurationSource)
	return request, nil
}

func (client *KsyunClient) getSdkConfig() *sdk.Config {
	return sdk.NewConfig().
		WithMaxRetryTime(DefaultClientRetryCountSmall).
		WithTimeout(time.Duration(30) * time.Second).
		WithEnableAsync(false).
		WithGoRoutinePoolSize(100).
		WithMaxTaskQueueSize(10000).
		WithDebug(false).
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
