package connectivity

import (
	rpc "github.com/alibabacloud-go/tea-rpc/client"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"

	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type KsyunClient struct {
	Region          Region
	RegionId        string
	SourceIp        string
	SecureTransport string
	//In order to build ots table client, add accesskey and secretkey in ksyunclient temporarily.
	AccessKey       string
	SecretKey       string
	SecurityToken   string
	OtsInstanceName string
	accountIdMutex  sync.RWMutex
	config          *Config
	teaSdkConfig    rpc.Config
	accountId       string
	ks3conn         *ks3.Client
	teaConn         *rpc.Client
}

type ApiVersion string

const (
	ApiVersion20140515 = ApiVersion("2014-05-15")
)

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
	loadLocalEndpoint = hasLocalEndpoint()
	if hasLocalEndpoint() {
		if err := c.loadEndpointFromLocal(); err != nil {
			return nil, err
		}
	}
	return &KsyunClient{
		config:          c,
		SourceIp:        c.SourceIp,
		Region:          c.Region,
		RegionId:        c.RegionId,
		AccessKey:       c.AccessKey,
		SecretKey:       c.SecretKey,
		SecurityToken:   c.SecurityToken,
		OtsInstanceName: c.OtsInstanceName,
		accountId:       c.AccountId,
	}, nil
}

type ks3Credentials struct {
	client *KsyunClient
}

func (defCre *ks3Credentials) GetAccessKeyID() string {
	value, err := defCre.client.teaSdkConfig.Credential.GetAccessKeyId()
	if err == nil && value != nil {
		return *value
	}
	return defCre.client.config.AccessKey
}

func (defCre *ks3Credentials) GetAccessKeySecret() string {
	value, err := defCre.client.teaSdkConfig.Credential.GetAccessKeySecret()
	if err == nil && value != nil {
		return *value
	}
	return defCre.client.config.SecretKey
}

func (defCre *ks3Credentials) GetSecurityToken() string {
	value, err := defCre.client.teaSdkConfig.Credential.GetSecurityToken()
	if err == nil && value != nil {
		return *value
	}
	return defCre.client.config.SecurityToken
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
		schma := strings.ToLower(client.config.Protocol)
		endpoint := client.config.Ks3Endpoint
		if endpoint == "" {
			endpoint = loadEndpoint(client.config.RegionId, KS3Code)
		}
		if endpoint == "" {
			endpointItem, err := client.describeEndpointForService(strings.ToLower(string(KS3Code)))
			if err != nil {
				log.Printf("describeEndpointForService got an error: %#v.", err)
			}
			endpoint = endpointItem
			if endpoint == "" {
				endpoint = fmt.Sprintf("ks3-%s.ksyuncs.com", client.RegionId)
			}
		}
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = fmt.Sprintf("%s://%s", schma, endpoint)
		}

		clientOptions := []ks3.ClientOption{ks3.UserAgent(client.getUserAgent())}
		proxy, err := client.getHttpProxy()
		if proxy != nil {
			skip, err := client.skipProxy(endpoint)
			if err != nil {
				return nil, err
			}
			if !skip {
				clientOptions = append(clientOptions, ks3.Proxy(proxy.String()))
			}
		}

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

func (client *KsyunClient) NewCommonRequest(product, serviceCode, schema string, apiVersion ApiVersion) (*requests.CommonRequest, error) {
	endpoint := ""
	product = strings.ToLower(product)
	if _, exist := client.config.Endpoints.Load(product); !exist {
		if err := client.loadEndpoint(product); err != nil {
			return nil, err
		}
	}
	if v, exist := client.config.Endpoints.Load(product); exist && v.(string) != "" {
		endpoint = v.(string)
	}
	request := requests.NewCommonRequest()
	// Use product code to find product domain
	if endpoint != "" {
		request.Domain = endpoint
	} else {
		// When getting endpoint failed by location, using custom endpoint instead
		request.Domain = fmt.Sprintf("%s.%s.ksyuncs.com", strings.ToLower(serviceCode), client.RegionId)
	}
	request.Version = string(apiVersion)
	request.RegionId = client.RegionId
	request.Product = product
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

func (client *KsyunClient) getHttpProxy() (proxy *url.URL, err error) {
	if client.config.Protocol == "HTTPS" {
		if rawurl := os.Getenv("HTTPS_PROXY"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		} else if rawurl := os.Getenv("https_proxy"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		}
	} else {
		if rawurl := os.Getenv("HTTP_PROXY"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		} else if rawurl := os.Getenv("http_proxy"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		}
	}
	return proxy, err
}

func (client *KsyunClient) skipProxy(endpoint string) (bool, error) {
	var urls []string
	if rawurl := os.Getenv("NO_PROXY"); rawurl != "" {
		urls = strings.Split(rawurl, ",")
	} else if rawurl := os.Getenv("no_proxy"); rawurl != "" {
		urls = strings.Split(rawurl, ",")
	}
	for _, value := range urls {
		if strings.HasPrefix(value, "*") {
			value = fmt.Sprintf(".%s", value)
		}
		noProxyReg, err := regexp.Compile(value)
		if err != nil {
			return false, err
		}
		if noProxyReg.MatchString(endpoint) {
			return true, nil
		}
	}
	return false, nil
}
