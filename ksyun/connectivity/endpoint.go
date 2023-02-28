package connectivity

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// ServiceCode Load endpoints from endpoints.xml or environment variables to meet specified application scenario, like private cloud.
type ServiceCode string

const (
	KS3Code = ServiceCode("KS3")
)

type Endpoints struct {
	Endpoint []Endpoint `xml:"Endpoint"`
}

type Endpoint struct {
	Name      string    `xml:"name,attr"`
	RegionIds RegionIds `xml:"RegionIds"`
	Products  Products  `xml:"Products"`
}

type RegionIds struct {
	RegionId string `xml:"RegionId"`
}

type Products struct {
	Product []Product `xml:"Product"`
}

type Product struct {
	ProductName string `xml:"ProductName"`
	DomainName  string `xml:"DomainName"`
}

var localEndpointPath = "./endpoints.xml"
var localEndpointPathEnv = "TF_ENDPOINT_PATH"
var loadLocalEndpoint = false

func hasLocalEndpoint() bool {
	data, err := ioutil.ReadFile(localEndpointPath)
	if err != nil || len(data) <= 0 {
		d, e := ioutil.ReadFile(os.Getenv(localEndpointPathEnv))
		if e != nil {
			return false
		}
		data = d
	}
	return len(data) > 0
}

func loadEndpoint(region string, serviceCode ServiceCode) string {
	endpoint := strings.TrimSpace(os.Getenv(fmt.Sprintf("%s_ENDPOINT", string(serviceCode))))
	if endpoint != "" {
		return endpoint
	}

	// Load current path endpoint file endpoints.xml, if failed, it will load from environment variables TF_ENDPOINT_PATH
	if !loadLocalEndpoint {
		return ""
	}
	data, err := ioutil.ReadFile(localEndpointPath)
	if err != nil || len(data) <= 0 {
		d, e := ioutil.ReadFile(os.Getenv(localEndpointPathEnv))
		if e != nil {
			return ""
		}
		data = d
	}
	var endpoints Endpoints
	err = xml.Unmarshal(data, &endpoints)
	if err != nil {
		return ""
	}
	for _, endpoint := range endpoints.Endpoint {
		if endpoint.RegionIds.RegionId == string(region) {
			for _, product := range endpoint.Products.Product {
				if strings.ToLower(product.ProductName) == strings.ToLower(string(serviceCode)) {
					return strings.TrimSpace(product.DomainName)
				}
			}
		}
	}

	return ""
}

// NOTE: The productCode must be lower.
func (client *KsyunClient) loadEndpoint(productCode string) error {
	// Firstly, load endpoint from environment variables
	endpoint := strings.TrimSpace(os.Getenv(fmt.Sprintf("%s_ENDPOINT", strings.ToUpper(productCode))))
	if endpoint != "" {
		client.config.Endpoints.Store(productCode, endpoint)
		return nil
	}

	// Secondly, load endpoint from known rules
	// Currently, this way is not pass.
	// if _, ok := irregularProductCode[productCode]; !ok {
	// 	client.config.Endpoints[productCode] = regularEndpoint
	// 	return nil
	// }

	// Thirdly, load endpoint from location
	serviceCode := serviceCodeMapping[productCode]
	if serviceCode == "" {
		serviceCode = productCode
	}
	endpoint, err := client.describeEndpointForService(serviceCode)
	if err == nil {
		client.config.Endpoints.Store(strings.ToLower(serviceCode), endpoint)
	}
	return err
}

// Load current path endpoint file endpoints.xml, if failed, it will load from environment variables TF_ENDPOINT_PATH
func (config *Config) loadEndpointFromLocal() error {
	data, err := ioutil.ReadFile(localEndpointPath)
	if err != nil || len(data) <= 0 {
		d, e := ioutil.ReadFile(os.Getenv(localEndpointPathEnv))
		if e != nil {
			return e
		}
		data = d
	}
	var endpoints Endpoints
	err = xml.Unmarshal(data, &endpoints)
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints.Endpoint {
		if endpoint.RegionIds.RegionId == string(config.RegionId) {
			for _, product := range endpoint.Products.Product {
				config.Endpoints.Store(strings.ToLower(product.ProductName), strings.TrimSpace(product.DomainName))
			}
		}
	}
	return nil
}

func incrementalWait(firstDuration time.Duration, increaseDuration time.Duration) func() {
	retryCount := 1
	return func() {
		var waitTime time.Duration
		if retryCount == 1 {
			waitTime = firstDuration
		} else if retryCount > 1 {
			waitTime += increaseDuration
		}
		time.Sleep(waitTime)
		retryCount++
	}
}
func (client *KsyunClient) describeEndpointForService(serviceCode string) (string, error) {
	//args := location.CreateDescribeEndpointsRequest()
	//args.ServiceCode = serviceCode
	//args.Id = client.config.RegionId
	//args.Domain = "ks3-cn-beijing.ksyuncs.com"

	//locationClient, err := location.NewClientWithOptions(client.config.RegionId, client.getSdkConfig(), client.config.getAuthCredential(true))
	//if err != nil {
	//	return "", fmt.Errorf("Unable to initialize the location client: %#v", err)
	//
	//}
	//defer locationClient.Shutdown()
	//locationClient.AppendUserAgent(Terraform, terraformVersion)
	//locationClient.AppendUserAgent(Provider, providerVersion)
	//locationClient.AppendUserAgent(Module, client.config.ConfigurationSource)
	//wait := incrementalWait(3*time.Second, 5*time.Second)
	//var endpointResult string
	//err = resource.Retry(5*time.Minute, func() *resource.RetryError {
	//	endpointsResponse, err := locationClient.DescribeEndpoints(args)
	//	if err != nil {
	//		re := regexp.MustCompile("^Post [\"]*https://.*")
	//		if err.Error() != "" && re.MatchString(err.Error()) {
	//			wait()
	//			return resource.RetryableError(err)
	//		}
	//		return resource.NonRetryableError(err)
	//	}
	//	if endpointsResponse != nil && len(endpointsResponse.Endpoints.Endpoint) > 0 {
	//		for _, e := range endpointsResponse.Endpoints.Endpoint {
	//			if e.Type == "openAPI" {
	//				endpointResult = e.Endpoint
	//				return nil
	//			}
	//		}
	//	}
	//	return nil
	//})
	//if err != nil {
	//	return "", fmt.Errorf("Describe %s endpoint using region: %#v got an error: %#v.", serviceCode, client.RegionId, err)
	//}
	//if endpointResult == "" {
	//	return "", fmt.Errorf("There is no any available endpoint for %s in region %s.", serviceCode, client.RegionId)
	//}
	return "ks3-cn-beijing.ksyuncs.com", nil
}

var serviceCodeMapping = map[string]string{
	"cloudapi": "apigateway",
}

const (
	OpenKS3Service = "ks3-cn-beijing.ksyuncs.com"
)
