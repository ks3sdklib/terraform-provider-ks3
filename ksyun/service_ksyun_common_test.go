package ksyun

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ResourceTestAccConfigFunc func(map[string]interface{}) string

// be used to check attribute map value
const (
	NOSET      = "#NOSET"       // be equivalent to method "TestCheckNoResourceAttrSet"
	CHECKSET   = "#CHECKSET"    // "TestCheckResourceAttrSet"
	REMOVEKEY  = "#REMOVEKEY"   // remove checkMap key
	REGEXMATCH = "#REGEXMATCH:" // "TestMatchResourceAttr" ,the map name/key like `"attribute" : REGEXMATCH + "attributeString"`
	ForceSleep = "force_sleep"
)

const (
	// indentation symbol
	INDENTATIONSYMBOL = " "

	// child field indend number
	CHILDINDEND = 2
)

var defaultRegionToTest = os.Getenv("KS3_REGION")

// be used for generate testcase step config
type resourceConfig struct {
	// the resource name
	name string

	resourceId string

	// store attribute value that primary resource
	attributeMap map[string]interface{}

	// generate assistant test config
	configDependence func(name string) string
}

// check attribute only
type resourceAttr struct {
	resourceId string
	checkMap   map[string]string
}

var ks3BucketBasicMap = map[string]string{
	"creation_date":    CHECKSET,
	"lifecycle_rule.#": "0",
}

// check the existence of resource
type resourceCheck struct {
	// IDRefreshName, like "ksyun_instance.foo"
	resourceId string

	// The response of the service method DescribeXXX
	resourceObject interface{}

	// The resource service client type, like DnsService, VpcService
	serviceFunc func() interface{}

	// service describe method name
	describeMethod string

	// additional attributes
	additionalAttrs []string

	// additional attributes type
	additionalAttrsType map[string]schema.ValueType
}

// check the existence and attribute of the resource at the same time
type resourceAttrCheck struct {
	*resourceCheck
	*resourceAttr
}

var testAccProviders map[string]terraform.ResourceProvider

type resourceAttrMapUpdate func(map[string]string) resource.TestCheckFunc

// get resourceAttrMapUpdate for a series test step and check resource exist
func (rac *resourceAttrCheck) resourceAttrMapUpdateSet() resourceAttrMapUpdate {
	return func(changeMap map[string]string) resource.TestCheckFunc {
		callback := func() {
			rac.updateCheckMapPair(changeMap)
		}
		return rac.resourceAttrMapCheckWithCallback(callback)
	}
}

// make a new map and copy from the old field checkMap, then update it according to the changeMap
func (ra *resourceAttr) updateCheckMapPair(changeMap map[string]string) {
	if interval, ok := changeMap[ForceSleep]; ok {
		intervalInt, err := strconv.Atoi(interval)
		if err == nil {
			time.Sleep(time.Duration(intervalInt) * time.Second)
			delete(changeMap, ForceSleep)
		}
	}
	newCheckMap := make(map[string]string, len(ra.checkMap))
	for k, v := range ra.checkMap {
		newCheckMap[k] = v
	}
	ra.checkMap = newCheckMap
	if changeMap != nil && len(changeMap) > 0 {
		for rk, rv := range changeMap {
			_, ok := ra.checkMap[rk]
			if rv == REMOVEKEY && ok {
				delete(ra.checkMap, rk)
			} else if ok {
				delete(ra.checkMap, rk)
				ra.checkMap[rk] = rv
			} else {
				ra.checkMap[rk] = rv
			}
		}
	}
}

var testAccProvider *schema.Provider

func resourceTestAccConfigFunc(resourceId string,
	name string,
	configDependence func(name string) string) ResourceTestAccConfigFunc {
	basicInfo := resourceConfig{
		name:             name,
		resourceId:       resourceId,
		attributeMap:     make(map[string]interface{}),
		configDependence: configDependence,
	}
	return basicInfo.configBuild(false)
}

func resourceCheckInit(resourceId string, resourceObject interface{}, serviceFunc func() interface{}, additionalAttrs ...string) *resourceCheck {
	rc := &resourceCheck{
		resourceId:      resourceId,
		resourceObject:  resourceObject,
		serviceFunc:     serviceFunc,
		additionalAttrs: additionalAttrs,
	}
	if len(rc.additionalAttrs) > 0 {
		rc.setAdditionalAttrsType()
	}
	return rc
}
func resourceAttrCheckInit(rc *resourceCheck, ra *resourceAttr) *resourceAttrCheck {
	return &resourceAttrCheck{
		resourceCheck: rc,
		resourceAttr:  ra,
	}
}

// check the resource existence by invoking DescribeXXX method of service and assign *resourceCheck.resourceObject value,
// the service is returned by invoking *resourceCheck.serviceFunc
func (rc *resourceCheck) checkResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var err error
		rs, ok := s.RootModule().Resources[rc.resourceId]
		if !ok {
			return WrapError(fmt.Errorf("can't find resource by id: %s", rc.resourceId))

		}
		outValue, err := rc.callDescribeMethod(rs)
		if err != nil {
			return WrapError(err)
		}
		errorValue := outValue[1]
		if !errorValue.IsNil() {
			return WrapError(fmt.Errorf("Checking resource %s %s exists error:%s ", rc.resourceId, rs.Primary.ID, errorValue.Interface().(error).Error()))
		}
		/*if reflect.TypeOf(rc.resourceObject).Elem().String() == outValue[0].Type().String() {
			reflect.ValueOf(rc.resourceObject).Elem().Set(outValue[0])
			return nil
		} else {
			return WrapError(fmt.Errorf("The response object type expected *%s, got %s ", outValue[0].Type().String(), reflect.TypeOf(rc.resourceObject).String()))
		}*/
		return nil
	}
}

// execute the callback before check attribute and check resource exist
func (rac *resourceAttrCheck) resourceAttrMapCheckWithCallback(callback func()) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		err := rac.resourceCheck.checkResourceExists()(s)
		if err != nil {
			return WrapError(err)
		}
		return rac.resourceAttr.resourceAttrMapCheckWithCallback(callback)(s)
	}
}

// execute the callback before check attribute
func (ra *resourceAttr) resourceAttrMapCheckWithCallback(callback func()) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		callback()
		return ra.resourceAttrMapCheck()(s)
	}
}

// check attribute func
func (ra *resourceAttr) resourceAttrMapCheck() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[ra.resourceId]
		if !ok {
			return WrapError(fmt.Errorf("can't find resource by id: %s", ra.resourceId))
		}
		if rs.Primary.ID == "" {
			return WrapError(fmt.Errorf("resource ID is not set"))
		}

		if ra.checkMap == nil || len(ra.checkMap) == 0 {
			return WrapError(fmt.Errorf("the parameter \"checkMap\" is nil or empty"))
		}

		var errorStrSlice []string
		errorStrSlice = append(errorStrSlice, "")
		for key, value := range ra.checkMap {
			var err error
			if strings.HasPrefix(value, REGEXMATCH) {
				var regex *regexp.Regexp
				regex, err = regexp.Compile(value[len(REGEXMATCH):])
				if err == nil {
					err = resource.TestMatchResourceAttr(ra.resourceId, key, regex)(s)
				} else {
					err = nil
				}
			} else if value == NOSET {
				err = resource.TestCheckNoResourceAttr(ra.resourceId, key)(s)
			} else if value == CHECKSET {
				err = resource.TestCheckResourceAttrSet(ra.resourceId, key)(s)
			} else {
				err = resource.TestCheckResourceAttr(ra.resourceId, key, value)(s)
			}
			if err != nil {
				errorStrSlice = append(errorStrSlice, err.Error())
			}
		}
		if len(errorStrSlice) == 1 {
			return nil
		}
		return WrapError(fmt.Errorf(strings.Join(errorStrSlice, "\n")))
	}
}

// caching the additional attribute type used to convert the addition attribute value type before calling Get method
func (rc *resourceCheck) setAdditionalAttrsType() {
	provider := Provider().(*schema.Provider)
	resourceType, ok := provider.ResourcesMap[strings.Split(rc.resourceId, ".")[0]]
	if !ok {
		log.Panicf("invalid resource type: %s", strings.Split(rc.resourceId, ".")[0])
	}
	if rc.additionalAttrsType == nil {
		rc.additionalAttrsType = make(map[string]schema.ValueType)
	}
	for _, attr := range rc.additionalAttrs {
		if s, ok := resourceType.Schema[attr]; !ok {
			log.Panicf("invalid resource attribute: %s", attr)
		} else {
			rc.additionalAttrsType[attr] = s.Type
		}
	}
	return
}

func resourceAttrInit(resourceId string, checkMap map[string]string) *resourceAttr {
	if checkMap == nil {
		checkMap = make(map[string]string)
	}
	return &resourceAttr{
		resourceId: resourceId,
		checkMap:   checkMap,
	}
}

// get BasicConfigFunc for resource a series test step
// overwrite: if true ,the attributeMap will be replace by changMap , other will be update
func (b *resourceConfig) configBuild(overwrite bool) ResourceTestAccConfigFunc {
	return func(changeMap map[string]interface{}) string {
		if overwrite {
			b.attributeMap = changeMap
		} else {
			b.configUpdate(changeMap)
		}
		strs := strings.Split(b.resourceId, ".")
		assistantConfig := b.configDependence(b.name)
		var primaryConfig string
		if strings.Compare("data", strs[0]) == 0 {
			primaryConfig = fmt.Sprintf("\n\ndata \"%s\" \"%s\" ", strs[1], strs[2])
		} else {
			primaryConfig = fmt.Sprintf("\n\nresource \"%s\" \"%s\" ", strs[0], strs[1])
		}
		return assistantConfig + primaryConfig + fmt.Sprint(valueConvert(0, reflect.ValueOf(b.attributeMap)))
	}
}

// according to changeMap to change the attributeMap value
func (b *resourceConfig) configUpdate(changeMap map[string]interface{}) {
	newMap := make(map[string]interface{}, len(b.attributeMap))
	for k, v := range b.attributeMap {
		newMap[k] = v
	}
	b.attributeMap = newMap
	if changeMap != nil && len(changeMap) > 0 {
		for rk, rv := range changeMap {
			_, ok := b.attributeMap[rk]
			if strValue, isCost := rv.(string); ok && isCost && strValue == REMOVEKEY {
				delete(b.attributeMap, rk)
			} else if ok {
				delete(b.attributeMap, rk)
				b.attributeMap[rk] = rv
			} else {
				b.attributeMap[rk] = rv
			}
		}
	}
}

// deal with the parameter common method
func valueConvert(indentation int, val reflect.Value) interface{} {
	switch val.Kind() {
	case reflect.Interface:
		return valueConvert(indentation, reflect.ValueOf(val.Interface()))
	case reflect.String:
		return fmt.Sprintf("\"%s\"", val.String())
	case reflect.Slice:
		return listValue(indentation, val)
	case reflect.Map:
		return mapValue(indentation, val)
	case reflect.Bool:
		return val.Bool()
	case reflect.Int:
		return val.Int()
	default:
		log.Panicf("invalid attribute value type: %#v", val)
	}
	return ""
}

// deal with list parameter
func listValue(indentation int, val reflect.Value) string {
	var valList []string
	for i := 0; i < val.Len(); i++ {
		valList = append(valList, addIndentation(indentation+CHILDINDEND)+
			fmt.Sprint(valueConvert(indentation+CHILDINDEND, val.Index(i))))
	}

	return fmt.Sprintf("[\n%s\n%s]", strings.Join(valList, ",\n"), addIndentation(indentation))
}

func addIndentation(indentation int) string {
	return strings.Repeat(INDENTATIONSYMBOL, indentation)
}

func getRealValueType(value reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Interface:
		return getRealValueType(reflect.ValueOf(value.Interface()))
	default:
		return value
	}
}

// deal with list parameter that child element is map
func listValueMapChild(indentation int, key string, val reflect.Value) string {
	var valList []string
	for i := 0; i < val.Len(); i++ {
		valList = append(valList, addIndentation(indentation)+key+" "+
			mapValue(indentation, getRealValueType(val.Index(i))))
	}

	return fmt.Sprintf("%s\n%s", strings.Join(valList, "\n"), addIndentation(indentation))
}

// deal with map parameter
func mapValue(indentation int, val reflect.Value) string {
	var valList []string
	for _, keyV := range val.MapKeys() {
		mapVal := getRealValueType(val.MapIndex(keyV))
		var line string
		if mapVal.Kind() == reflect.Slice && mapVal.Len() > 0 {
			eleVal := getRealValueType(mapVal.Index(0))
			if eleVal.Kind() == reflect.Map {
				line = fmt.Sprintf(`%s%s`, addIndentation(indentation),
					listValueMapChild(indentation+CHILDINDEND, keyV.String(), mapVal))
				valList = append(valList, line)
				continue
			}
		}
		value := valueConvert(indentation+len(keyV.String())+CHILDINDEND+3, val.MapIndex(keyV))
		switch value.(type) {
		case bool:
			line = fmt.Sprintf(`%s%s = %t`, addIndentation(indentation+CHILDINDEND), keyV.String(), value)
		case int:
			line = fmt.Sprintf(`%s%s = %d`, addIndentation(indentation+CHILDINDEND), keyV.String(), value)
		default:
			line = fmt.Sprintf(`%s%s = %s`, addIndentation(indentation+CHILDINDEND), keyV.String(), value)
		}

		valList = append(valList, line)
	}
	return fmt.Sprintf("{\n%s\n%s}", strings.Join(valList, "\n"), addIndentation(indentation))
}

//invoking DescribeXXX method of service
func (rc *resourceCheck) callDescribeMethod(rs *terraform.ResourceState) ([]reflect.Value, error) {
	var err error
	if rs.Primary.ID == "" {
		return nil, WrapError(fmt.Errorf("resource ID is not set"))
	}
	serviceP := rc.serviceFunc()
	if rc.describeMethod == "" {
		rc.describeMethod, err = getResourceDescribeMethod(rc.resourceId)
		if err != nil {
			return nil, WrapError(err)
		}
	}
	value := reflect.ValueOf(serviceP)
	typeName := value.Type().String()
	value = value.MethodByName(rc.describeMethod)
	if !value.IsValid() {
		return nil, WrapError(Error("The service type %s does not have method %s", typeName, rc.describeMethod))
	}
	inValue := []reflect.Value{reflect.ValueOf(rs.Primary.ID)}
	for _, attr := range rc.additionalAttrs {
		if attrValue, ok := rs.Primary.Attributes[attr]; ok {
			if attrType, o := rc.additionalAttrsType[attr]; o {
				switch attrType {
				case schema.TypeBool:
					v, _ := strconv.ParseBool(attrValue)
					inValue = append(inValue, reflect.ValueOf(v))
					continue
				case schema.TypeInt:
					v, _ := strconv.ParseInt(attrValue, 10, 64)
					inValue = append(inValue, reflect.ValueOf(v))
					continue
				}
			}
			inValue = append(inValue, reflect.ValueOf(attrValue))
		}
	}
	return value.Call(inValue), nil
}
func getResourceDescribeMethod(resourceId string) (string, error) {
	start := strings.Index(resourceId, "ksyun_")
	if start < 0 {
		return "", WrapError(fmt.Errorf("the parameter \"name\" don't contain string \"ksyun_\""))
	}
	start += len("ksyun_")
	end := strings.Index(resourceId[start:], ".") + start
	if end < 0 {
		return "", WrapError(fmt.Errorf("the parameter \"name\" don't contain string \".\""))
	}
	strs := strings.Split(resourceId[start:end], "_")
	describeName := "Describe"
	for _, str := range strs {
		describeName = describeName + strings.Title(str)
	}
	return describeName, nil
}

// check the resource destroy
func (rc *resourceCheck) checkResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		strs := strings.Split(rc.resourceId, ".")
		var resourceType string
		for _, str := range strs {
			if strings.Contains(str, "ksyun_") {
				resourceType = strings.Trim(str, " ")
				break
			}
		}

		if resourceType == "" {
			return WrapError(Error("The resourceId %s is not correct and it should prefix with ksyun_", rc.resourceId))
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}
			outValue, err := rc.callDescribeMethod(rs)
			errorValue := outValue[1]
			if !errorValue.IsNil() {
				err = errorValue.Interface().(error)
				if err != nil {
					if NotFoundError(err) {
						continue
					}
					return WrapError(err)
				}
			} else {
				return WrapError(Error("the resource %s %s was not destroyed ! ", rc.resourceId, rs.Primary.ID))
			}
		}
		return nil
	}
}
