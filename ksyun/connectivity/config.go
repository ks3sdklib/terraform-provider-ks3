package connectivity

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"sync"

	credential "github.com/aliyun/credentials-go/credentials"
)

// Config of ksyun
type Config struct {
	AccessKey            string
	SecretKey            string
	Region               Region
	SecurityToken        string
	Protocol             string
	ClientReadTimeout    int
	ClientConnectTimeout int
	SkipRegionValidation bool
	SourceIp             string
	SecureTransport      string
	MaxRetryTimeout      int
	ConfigurationSource  string
	Endpoints            *sync.Map
	Ks3Endpoint          string
}

func (c *Config) getAuthCredential(stsSupported bool) auth.Credential {
	if c.AccessKey != "" && c.SecretKey != "" {
		if stsSupported && c.SecurityToken != "" {
			return credentials.NewStsTokenCredential(c.AccessKey, c.SecretKey, c.SecurityToken)
		}
		return credentials.NewAccessKeyCredential(c.AccessKey, c.SecretKey)
	}
	return credentials.NewAccessKeyCredential(c.AccessKey, c.SecretKey)
}

func (c *Config) getCredentialConfig() *credential.Config {
	credentialType := ""
	credentialConfig := &credential.Config{}
	if c.AccessKey != "" && c.SecretKey != "" {
		credentialType = "access_key"
		credentialConfig.AccessKeyId = &c.AccessKey     // AccessKeyId
		credentialConfig.AccessKeySecret = &c.SecretKey // AccessKeySecret
	}

	credentialConfig.Type = &credentialType
	return credentialConfig
}
