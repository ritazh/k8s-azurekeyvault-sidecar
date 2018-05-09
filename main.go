// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"github.com/golang/glog"
	
	// kvmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	// kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	
)

const (
	program		= "k8s-azurekeyvault-sidecar"
	version     = "0.0.1"
	configFilePath  = "azure.json"
)

// Option is a collection of configs
type Option struct {
	// the name of the Azure Key Vault instance
	vaultName string
	// version flag
	showVersion bool
	// azure configs 
	azConfig *AzureAuthConfig
}

var (
	options Option
)

func main() {
	sigChan := make(chan os.Signal, 1)
	// register for SIGTERM (docker)
	signal.Notify(sigChan, syscall.SIGTERM)

	if err := parseConfigs(); err != nil {
		showUsage("invalid config, %s", err)
	}
	if options.showVersion {
		fmt.Printf("%s %s\n", program, version)
		fmt.Printf("%s \n", options.azConfig.SubscriptionID)
	}
	glog.Infof("starting the %s, %s", program, version)

	for {
		s := <-sigChan
		if s == syscall.SIGTERM {
			glog.Infof("Received SIGTERM. Exit program")
			os.Exit(0)
		} 
	}

}

func parseConfigs() error {
	options.vaultName = *flag.String("vaultName", getEnv("VAULT_NAME", ""), "Name of Azure Key Vault instance.")
	options.showVersion = *flag.Bool("version", true, "Show version.")

	flag.Parse()

	if options.vaultName == "" {
		return fmt.Errorf("env VAULT_NAME is unset")
	}

	options.azConfig, _ = GetAzureAuthConfig(configFilePath)
	if options.azConfig.SubscriptionID == "" {
		return fmt.Errorf("Missing SubscriptionID in azure config")
	}
	return nil
}

func getEnv(variable, value string) string {
	if v := os.Getenv(variable); v != "" {
		return v
	}

	return value
}

func showUsage(message string, args ...interface{}) {
	flag.PrintDefaults()
	if message != "" {
		fmt.Printf("\n[error] "+message+"\n", args...)
		os.Exit(1)
	}

	os.Exit(0)
}