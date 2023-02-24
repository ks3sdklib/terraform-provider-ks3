package ksyun

import (
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
	"strings"
)

type LifecycleRuleStatus string

const (
	ExpirationStatusEnabled  = LifecycleRuleStatus("Enabled")
	ExpirationStatusDisabled = LifecycleRuleStatus("Disabled")
)

func ks3NotFoundError(err error) bool {
	if e, ok := err.(ks3.ServiceError); ok &&
		(e.StatusCode == 404 || strings.HasPrefix(e.Code, "NoSuch") || strings.HasPrefix(e.Message, "No Row found")) {
		return true
	}
	return false
}

type ListenerErr struct {
	ErrType string
	Err     error
}

func (e *ListenerErr) Error() string {
	return e.ErrType + " " + e.Err.Error()

}
