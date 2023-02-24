package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/ksyun/terraform-provider-ks3/ksyun"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ksyun.Provider})
}
