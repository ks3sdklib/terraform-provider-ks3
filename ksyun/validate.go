package ksyun

import (
	"fmt"
	"time"
)

const Iso8601DateFormat = "2006-01-02T00:00:00+08:00"

func validateKs3BucketDateTimestamp(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := time.Parse(Iso8601DateFormat, value)
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q cannot be parsed as date %s Format", value, Iso8601DateFormat))
	}
	return
}
