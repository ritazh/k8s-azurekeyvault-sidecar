// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"syscall"
	"github.com/golang/glog"
	
	kvmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	
)

const (
	program					= "k8s-azurekeyvault-sidecar"
	version     			= "0.0.1"
	configFilePath  		= "azure.json"
	permission  os.FileMode = 0400
)

// Option is a collection of configs
type Option struct {
	// the name of the Azure Key Vault instance
	vaultName string
	// the name of the Azure Key Vault secret
	secretName string
	// directory to save data
	dir string
	// version flag
	showVersion bool
	// azure configs 
	azConfig *AzureAuthConfig
}

var (
	options Option
)

func main() {
	ctx := context.Background()
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

	kvClient := kv.New()

	vaultUrl, err := getVault(ctx, options.azConfig.SubscriptionID, options.vaultName)
	if err != nil {
		showError("failed to get key vault, error: %s", err)
	}

	token, err := GetKeyvaultToken(AuthGrantType(), configFilePath)
	if err != nil {
		showError("failed to get token, error: %s", err)
	}
	
	kvClient.Authorizer = token

	go func() {
		for {
			s := <-sigChan
			if s == syscall.SIGTERM {
				glog.Infof("Received SIGTERM. Exit program")
				os.Exit(0)
			} 
		}
	}()
	
	/// TODO: make this a loop later
	secret, err := kvClient.GetSecret(ctx, *vaultUrl, options.secretName, "")
	if err != nil {
		showError("failed to get secret, error: %s", err)
	}
	glog.Infof("secret: %s", *secret.Value)

	_, err = os.Lstat(options.dir)
	if err != nil {
		showError("failed to get directory %s, error: %s", options.dir, err)
	}

	fileInfo, err := os.Lstat(path.Join(options.dir, options.secretName))
	if fileInfo != nil {
		glog.V(0).Infof("secret %s already exists in %s", options.secretName,options.dir)
	} else {
		if err = ioutil.WriteFile(path.Join(options.dir, options.secretName), []byte(*secret.Value), permission); err != nil {
			showError("azure KeyVault failed to write secret %s at %s with err %s", options.secretName, options.dir, err)
		}
		glog.V(0).Infof("azure KeyVault wrote secret %s at %s", options.secretName,options.dir)
	}
}

func parseConfigs() error {
	options.vaultName = *flag.String("vaultName", getEnv("VAULT_NAME", ""), "Name of Azure Key Vault instance.")
	options.secretName = *flag.String("secretName", getEnv("SECRET_NAME", ""), "Name of Azure Key Vault secret.")
	options.dir = *flag.String("dir", getEnv("DIR", ""), "Directory path to write data.")
	options.showVersion = *flag.Bool("version", true, "Show version.")

	flag.Parse()

	if options.vaultName == "" {
		return fmt.Errorf("env VAULT_NAME is unset")
	}
	if options.secretName == "" {
		return fmt.Errorf("env SECRET_NAME is unset")
	}
	if options.dir == "" {
		return fmt.Errorf("env DIR is unset")
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

func showError(message string, args ...interface{}) {
	if message != "" {
		fmt.Printf("\n[error] "+message+"\n", args...)
		os.Exit(1)
	}

	os.Exit(0)
}

func getVault(ctx context.Context, subscriptionID string, vaultName string) (vaultUrl *string, err error) {
	vaultsClient := kvmgmt.NewVaultsClient(subscriptionID)
	token, _ := GetManagementToken(AuthGrantType(), configFilePath)
	vaultsClient.Authorizer = token
	resourceGroup, err := GetResourceGroup(configFilePath)
	vault, err := vaultsClient.Get(ctx, *resourceGroup, vaultName)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault, error: %v", err)
	}
	return vault.Properties.VaultURI, nil
}