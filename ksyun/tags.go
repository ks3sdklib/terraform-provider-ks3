package ksyun

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"regexp"
)

func String(v string) *string {
	return &v
}

func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
	}
}

func tagsSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Computed: true,
	}
}

func tagsSchemaWithIgnore() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringDoesNotMatch(regexp.MustCompile(`(^acs:.*)|(^ksyun.*)|(/.*http://.*\.\w+/gm)|(/.*https://.*\.\w+/gm)`), "It cannot begin with \"ksyun\", \"acs:\"; without \"http://\", and \"https://\"."),
		},
	}
}

func parsingTags(d *schema.ResourceData) (map[string]interface{}, []string) {
	oraw, nraw := d.GetChange("tags")
	removedTags := oraw.(map[string]interface{})
	addedTags := nraw.(map[string]interface{})
	// Build the list of what to remove
	removed := make([]string, 0)
	for key, value := range removedTags {
		old, ok := addedTags[key]
		if !ok || old != value {
			// Delete it!
			removed = append(removed, key)
		}
	}

	return addedTags, removed
}

func tagsToMap(tags interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if tags == nil {
		return result
	}
	switch v := tags.(type) {
	case map[string]interface{}:
		for key, value := range tags.(map[string]interface{}) {
			if !tagIgnored(key, value) {
				result[key] = value
			}
		}
	case []interface{}:
		if len(tags.([]interface{})) < 1 {
			return result
		}
		for _, tag := range tags.([]interface{}) {
			t := tag.(map[string]interface{})
			var tagKey string
			var tagValue interface{}
			if v, ok := t["TagKey"]; ok {
				tagKey = v.(string)
				tagValue = t["TagValue"]
			} else if v, ok := t["Key"]; ok {
				tagKey = v.(string)
				tagValue = t["Value"]
			}
			if !tagIgnored(tagKey, tagValue) {
				result[tagKey] = tagValue
			}
		}
	default:
		log.Printf("\u001B[31m[ERROR]\u001B[0m Unknown tags type %s. The tags value is: %v.", v, tags)
	}
	return result
}

func tagIgnored(tagKey string, tagValue interface{}) bool {
	filter := []string{"^ksyun", "^acs:", "^http://", "^https://", "^sae.do.not.delete"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching prefix %v with %v\n", v, tagKey)
		ok, _ := regexp.MatchString(v, tagKey)
		if ok {
			log.Printf("[DEBUG] Found Alibaba Cloud specific tag %s (val: %s), ignoring.\n", tagKey, tagValue)
			return true
		}
	}
	return false
}

func tagsMapEqual(expectMap map[string]interface{}, compareMap map[string]string) bool {
	if len(expectMap) != len(compareMap) {
		return false
	} else {
		for key, eVal := range expectMap {
			if eStr, ok := eVal.(string); !ok {
				// type is mismatch.
				return false
			} else {
				if cStr, ok := compareMap[key]; ok {
					if eStr != cStr {
						return false
					}
				} else {
					return false
				}
			}
		}
	}
	return true
}
