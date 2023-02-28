package ksyun

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Status string

const (
	Deleted = Status("Deleted")
)

// timeout for common product, ecs e.g.
const DefaultTimeout = 120
const Timeout5Minute = 300
const DefaultTimeoutMedium = 500

// Protocol represents network protocol
type Protocol string

// Constants of protocol definition
const (
	Http  = Protocol("http")
	Https = Protocol("https")
	Tcp   = Protocol("tcp")
	Udp   = Protocol("udp")
	All   = Protocol("all")
	Icmp  = Protocol("icmp")
	Gre   = Protocol("gre")
)

// ValidProtocols network protocol list
var ValidProtocols = []Protocol{Http, Https, Tcp, Udp}

// simple array value check method, support string type only

// default region for all resource
const DEFAULT_REGION = "cn-beijing"

const INT_MAX = 2147483647

// symbol of multiIZ
const MULTI_IZ_SYMBOL = "MAZ"

const COMMA_SEPARATED = ","

const COLON_SEPARATED = ":"

const SLASH_SEPARATED = "/"

const LOCAL_HOST_IP = "127.0.0.1"

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		if v == nil {
			continue
		}
		vs = append(vs, v.(string))
	}
	return vs
}

// Takes list of string to strings. Expand to an array
// of raw strings and returns a []interface{}
func convertListStringToListInterface(list []string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, v)
	}
	return vs
}

func expandIntList(configured []interface{}) []int {
	vs := make([]int, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, v.(int))
	}
	return vs
}

// Convert the result for an array and returns a Json string
func convertListToJsonString(configured []interface{}) string {
	if len(configured) < 1 {
		return ""
	}
	result := "["
	for i, v := range configured {
		if v == nil {
			continue
		}
		result += "\"" + v.(string) + "\""
		if i < len(configured)-1 {
			result += ","
		}
	}
	result += "]"
	return result
}

func convertJsonStringToStringList(src interface{}) (result []interface{}) {
	if err, ok := src.([]interface{}); !ok {
		panic(err)
	}
	for _, v := range src.([]interface{}) {
		result = append(result, fmt.Sprint(formatInt(v)))
	}
	return
}

func encodeToBase64String(configured []string) string {
	result := ""
	for i, v := range configured {
		result += v
		if i < len(configured)-1 {
			result += ","
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(result))
}

func decodeFromBase64String(configured string) (result []string, err error) {

	decodeString, err := base64.StdEncoding.DecodeString(configured)
	if err != nil {
		return result, err
	}

	result = strings.Split(string(decodeString), ",")
	return result, nil
}

func convertJsonStringToMap(configured string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configured), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Convert the result for an array and returns a comma separate
func convertListToCommaSeparate(configured []interface{}) string {
	if len(configured) < 1 {
		return ""
	}
	result := ""
	for i, v := range configured {
		rail := ","
		if i == len(configured)-1 {
			rail = ""
		}
		result += v.(string) + rail
	}
	return result
}

func convertBoolToString(configured bool) string {
	return strconv.FormatBool(configured)
}

func convertStringToBool(configured string) bool {
	v, _ := strconv.ParseBool(configured)
	return v
}

func convertIntergerToString(configured int) string {
	return strconv.Itoa(configured)
}

func convertFloat64ToString(configured float64) string {
	return strconv.FormatFloat(configured, 'E', -1, 64)
}

func convertJsonStringToList(configured string) ([]interface{}, error) {
	result := make([]interface{}, 0)
	if err := json.Unmarshal([]byte(configured), &result); err != nil {
		return nil, err
	}

	return result, nil
}

func convertMaptoJsonString(m map[string]interface{}) (string, error) {
	//sm := make(map[string]string, len(m))
	//for k, v := range m {
	//	sm[k] = v.(string)
	//}

	if result, err := json.Marshal(m); err != nil {
		return "", err
	} else {
		return string(result), nil
	}
}

func convertListMapToJsonString(configured []map[string]interface{}) (string, error) {
	if len(configured) < 1 {
		return "[]", nil
	}

	result := "["
	for i, m := range configured {
		if m == nil {
			continue
		}

		sm := make(map[string]interface{}, len(m))
		for k, v := range m {
			sm[k] = v
		}

		item, err := json.Marshal(sm)
		if err == nil {
			result += string(item)
			if i < len(configured)-1 {
				result += ","
			}
		}
	}
	result += "]"
	return result, nil
}

func convertMapFloat64ToJsonString(m map[string]interface{}) (string, error) {
	sm := make(map[string]json.Number, len(m))

	for k, v := range m {
		sm[k] = v.(json.Number)
	}

	if result, err := json.Marshal(sm); err != nil {
		return "", err
	} else {
		return string(result), nil
	}
}

func StringPointer(s string) *string {
	return &s
}

func BoolPointer(b bool) *bool {
	return &b
}

func Int32Pointer(i int32) *int32 {
	return &i
}

func Int64Pointer(i int64) *int64 {
	return &i
}

func IntMin(x, y int) int {
	if x < y {
		return x
	}
	return y
}

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

		//fmt.Printf(DefaultDebugMsg, action, content, trace)
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

// When  using teadsl, we need to convert float, int64 and int32 to int for comparison.
func formatInt(src interface{}) int {
	if src == nil {
		return 0
	}
	attrType := reflect.TypeOf(src)
	switch attrType.String() {
	case "float64":
		return int(src.(float64))
	case "float32":
		return int(src.(float32))
	case "int64":
		return int(src.(int64))
	case "int32":
		return int(src.(int32))
	case "int":
		return src.(int)
	case "string":
		v, err := strconv.Atoi(src.(string))
		if err != nil {
			panic(err)
		}
		return v
	case "json.Number":
		v, err := strconv.Atoi(src.(json.Number).String())
		if err != nil {
			panic(err)
		}
		return v
	default:
		panic(fmt.Sprintf("Not support type %s", attrType.String()))
	}
}
