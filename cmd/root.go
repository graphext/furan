package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dollarshaveclub/furan/v2/pkg/config"
	"github.com/dollarshaveclub/furan/v2/pkg/secrets"
)

var awsConfig config.AWSConfig
var vaultConfig config.VaultConfig

var secretsbackend string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:          "furan",
	Short:        "Docker image builder",
	Long:         `API application to build Docker images on command`,
	SilenceUsage: true,
}

// Execute is the entry point for the app
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

var sf = &secrets.Fetcher{}

func init() {
	// Add global flags here
}

func secretsSetup() {
	sf.VaultOptions = vaultConfig
	switch secretsbackend {
	case "vault":
		sf.Backend = secrets.VaultBackend
	case "env":
		sf.Backend = secrets.EnvVarBackend
	case "json":
		sf.Backend = secrets.JSONBackend
	case "filetree":
		sf.Backend = secrets.FileTreeBackend
	}
}

var ghConfig config.GitHubConfig

func gitHubSecrets() {
	secretsSetup()
	if err := sf.GitHub(&ghConfig); err != nil {
		clierr("error getting github secrets: %v", err)
	}
}

var quayConfig config.QuayConfig

func quaySecrets() {
	secretsSetup()
	if err := sf.Quay(&quayConfig); err != nil {
		clierr("error getting quay secrets: %v", err)
	}
}

func awsSecrets() {
	secretsSetup()
	if err := sf.AWS(&awsConfig); err != nil {
		clierr("error getting aws secrets: %v", err)
	}
}

var dbConfig config.DBConfig

func dbSecrets() {
	secretsSetup()
	if err := sf.Database(&dbConfig); err != nil {
		clierr("error getting db secrets: %v", err)
	}
}

func clierr(msg string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", params...)
	os.Exit(1)
}
