package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/Natiq0/terraform-provider-sonarqube/sonarqube"
)

func main() {

	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	plugin.Serve(
		&plugin.ServeOpts{
			Debug:        debug,
			ProviderAddr: "Natiq0/sonarqube",
			ProviderFunc: sonarqube.Provider,
		},
	)
}
