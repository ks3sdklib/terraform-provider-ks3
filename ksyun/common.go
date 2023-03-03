package ksyun

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Status string

const (
	Deleted = Status("Deleted")
)

const DefaultTimeoutMedium = 500

// Protocol represents network protocol
type Protocol string

// Constants of protocol definition
const (
	Http  = Protocol("http")
	Https = Protocol("https")
	Tcp   = Protocol("tcp")
	Udp   = Protocol("udp")
)

// ValidProtocols network protocol list
var ValidProtocols = []Protocol{Http, Https, Tcp, Udp}

// simple array value check method, support string type only

// default region for all resource
const DEFAULT_REGION = "BEIJING"

const COLON_SEPARATED = ":"

const ServerSideEncryptionAes256 = "AES256"
const ServerSideEncryptionKMS = "KMS"

type TagResourceType string

func GetUserHomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("Get current user got an error: %#v.", err)
	}
	return usr.HomeDir, nil
}

func writeToFile(filePath string, data interface{}) error {
	var out string
	switch data.(type) {
	case string:
		out = data.(string)
		break
	case nil:
		return nil
	default:
		bs, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			return fmt.Errorf("MarshalIndent data %#v got an error: %#v", data, err)
		}
		out = string(bs)
	}

	if strings.HasPrefix(filePath, "~") {
		home, err := GetUserHomeDir()
		if err != nil {
			return err
		}
		if home != "" {
			filePath = strings.Replace(filePath, "~", home, 1)
		}
	}

	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			return err
		}
	}

	return ioutil.WriteFile(filePath, []byte(out), 422)
}

type Invoker struct {
	catchers []*Catcher
}

type Catcher struct {
	Reason           string
	RetryCount       int
	RetryWaitSeconds int
}

var ClientErrorCatcher = Catcher{ClientFailure, 10, 5}
var ServiceBusyCatcher = Catcher{"ServiceUnavailable", 10, 5}
var ThrottlingCatcher = Catcher{Throttling, 50, 2}

func NewInvoker() Invoker {
	i := Invoker{}
	i.AddCatcher(ClientErrorCatcher)
	i.AddCatcher(ServiceBusyCatcher)
	i.AddCatcher(ThrottlingCatcher)
	return i
}

func (a *Invoker) AddCatcher(catcher Catcher) {
	a.catchers = append(a.catchers, &catcher)
}

func (a *Invoker) Run(f func() error) error {
	err := f()

	if err == nil {
		return nil
	}

	for _, catcher := range a.catchers {
		if IsExpectedErrors(err, []string{catcher.Reason}) {
			catcher.RetryCount--

			if catcher.RetryCount <= 0 {
				return fmt.Errorf("Retry timeout and got an error: %#v.", err)
			} else {
				time.Sleep(time.Duration(catcher.RetryWaitSeconds) * time.Second)
				return a.Run(f)
			}
		}
	}
	return err
}

func debugOn() bool {
	for _, part := range strings.Split(os.Getenv("DEBUG"), ",") {
		if strings.TrimSpace(part) == "terraform" {
			return true
		}
	}
	return false
}

func addDebug(action, content interface{}, requestInfo ...interface{}) {
	if debugOn() {
		trace := "[DEBUG TRACE]:\n"
		for skip := 1; skip < 5; skip++ {
			_, filepath, line, _ := runtime.Caller(skip)
			trace += fmt.Sprintf("%s:%d\n", filepath, line)
		}

		if len(requestInfo) > 0 {
			var request = struct {
				Domain     string
				Version    string
				UserAgent  string
				ActionName string
				Method     string
				Product    string
				Region     string
				AK         string
			}{}
			switch requestInfo[0].(type) {
			case *requests.RpcRequest:
				tmp := requestInfo[0].(*requests.RpcRequest)
				request.Domain = tmp.GetDomain()
				request.Version = tmp.GetVersion()
				request.ActionName = tmp.GetActionName()
				request.Method = tmp.GetMethod()
				request.Product = tmp.GetProduct()
				request.Region = tmp.GetRegionId()
			case *requests.RoaRequest:
				tmp := requestInfo[0].(*requests.RoaRequest)
				request.Domain = tmp.GetDomain()
				request.Version = tmp.GetVersion()
				request.ActionName = tmp.GetActionName()
				request.Method = tmp.GetMethod()
				request.Product = tmp.GetProduct()
				request.Region = tmp.GetRegionId()
			case *requests.CommonRequest:
				tmp := requestInfo[0].(*requests.CommonRequest)
				request.Domain = tmp.GetDomain()
				request.Version = tmp.GetVersion()
				request.ActionName = tmp.GetActionName()
				request.Method = tmp.GetMethod()
				request.Product = tmp.GetProduct()
				request.Region = tmp.GetRegionId()
			case *ks3.Client:
				request.Product = "KS3"
				request.ActionName = fmt.Sprintf("%s", action)
			}
			requestContent := ""
			if len(requestInfo) > 1 {
				requestContent = fmt.Sprintf("%#v", requestInfo[1])
			}

			if len(requestInfo) == 1 {
				if v, ok := requestInfo[0].(map[string]interface{}); ok {
					if res, err := json.Marshal(&v); err == nil {
						requestContent = string(res)
					}
					if res, err := json.Marshal(&content); err == nil {
						content = string(res)
					}
				}
			}

			content = fmt.Sprintf("%vDomain:%v, Version:%v, ActionName:%v, Method:%v, Product:%v, Region:%v\n\n"+
				"*************** %s Request ***************\n%#v\n",
				content, request.Domain, request.Version, request.ActionName,
				request.Method, request.Product, request.Region, request.ActionName, requestContent)
		}

		log.Printf(DefaultDebugMsg, action, content, trace)
	}
}

// Return a ComplexError which including extra error message, error occurred file and path
func GetFunc(level int) string {
	pc, _, _, ok := runtime.Caller(level)
	if !ok {
		log.Printf("[ERROR] runtime.Caller error in GetFuncName.")
		return ""
	}
	return strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(pc).Name()), ".")
}

func ParseResourceId(id string, length int) (parts []string, err error) {
	parts = strings.Split(id, ":")

	if len(parts) != length {
		err = WrapError(fmt.Errorf("Invalid Resource Id %s. Expected parts' length %d, got %d", id, length, len(parts)))
	}
	return parts, err
}
