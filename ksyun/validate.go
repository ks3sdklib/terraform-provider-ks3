package ksyun

import (
	"fmt"
	"time"
)

func validateKs3BucketDateTimestamp(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := time.Parse("yyyy-MM-ddT00:00:00+08:00", value)
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q cannot be parsed as date YYYY-MM-DD Format", value))
	}
	return
}
