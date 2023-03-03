package connectivity

import (
	"sync"
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
	MaxRetryTimeout      int
	ConfigurationSource  string
	Endpoints            *sync.Map
	Ks3Endpoint          string
}
