package ksyun

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
)

// Generates a hash for the set hash function used by the ID
func dataResourceIdHash(ids []string) string {
	var buf bytes.Buffer

	for _, id := range ids {
		buf.WriteString(fmt.Sprintf("%s-", id))
	}

	return fmt.Sprintf("%d", hashcode.String(buf.String()))
}
