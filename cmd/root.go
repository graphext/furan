package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dollarshaveclub/furan/pkg/config"
	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/pkg/secrets"
)

var awsConfig config.AWSConfig
var vaultConfig config.VaultConfig

var tags string
var secretsbackend string

// used by build and trigger commands
var cliBuildRequest = furanrpc.BuildRequest{
	Build: &furanrpc.BuildDefinition{},
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "furan",
	Short: "Docker image builder",
	Long:  `API application to build Docker images on command`,
}

// Execute is the entry point for the app
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

var sf = &secrets.Fetcher{}

func init() {
	// Secrets
	RootCmd.PersistentFlags().StringVar(&vaultConfig.Addr, "vault-addr", os.Getenv("VAULT_ADDR"), "Vault URL (if using Vault secret backend)")
	RootCmd.PersistentFlags().StringVar(&vaultConfig.Token, "vault-token", os.Getenv("VAULT_TOKEN"), "Vault token (if using token auth & Vault secret backend)")
	RootCmd.PersistentFlags().BoolVar(&vaultConfig.TokenAuth, "vault-token-auth", false, "Use Vault token-based auth (if using Vault secret backend)")
	RootCmd.PersistentFlags().BoolVar(&vaultConfig.K8sAuth, "vault-k8s-auth", false, "Use Vault k8s auth (if using Vault secret backend)")
	RootCmd.PersistentFlags().StringVar(&vaultConfig.K8sJWTPath, "vault-k8s-jwt-path", "/var/run/secrets/kubernetes.io/serviceaccount/token", "Vault k8s JWT file path (if using k8s auth & Vault secret backend)")
	RootCmd.PersistentFlags().StringVar(&vaultConfig.K8sRole, "vault-k8s-role", "", "Vault k8s role (if using k8s auth & Vault secret backend)")
	RootCmd.PersistentFlags().StringVar(&vaultConfig.K8sAuthPath, "vault-k8s-auth-path", "kubernetes", "Vault k8s auth path (if using k8s auth & Vault secret backend)")
	RootCmd.PersistentFlags().StringVar(&secretsbackend, "secrets-backend", "vault", "Secret backend (one of: vault,env,json,filetree)")
	RootCmd.PersistentFlags().StringVar(&sf.JSONFile, "secrets-json-file", "secrets.json", "Secret JSON file path (if using json backend)")
	RootCmd.PersistentFlags().StringVar(&sf.FileTreeRoot, "secrets-filetree-root", "/vault/secrets/", "Secrets filetree root path (if using filetree backend)")
	RootCmd.PersistentFlags().StringVar(&sf.Mapping, "secrets-mapping", "", "Secrets mapping template string (required)")

	// AWS S3 Cache
	RootCmd.PersistentFlags().StringVar(&awsConfig.Region, "aws-region", "us-west-2", "AWS region")
	RootCmd.PersistentFlags().StringVar(&awsConfig.CacheBucket, "s3-cache-bucket", "", "AWS S3 cache bucket")
	RootCmd.PersistentFlags().StringVar(&awsConfig.CacheKeyPrefix, "s3-cache-key-pfx", "", "AWS S3 cache key prefix")

	// ECR
	RootCmd.PersistentFlags().BoolVar(&awsConfig.EnableECR, "ecr", false, "Enable AWS ECR support")
	RootCmd.PersistentFlags().StringSliceVar(&awsConfig.ECRRegistryHosts, "ecr-registry-hosts", []string{}, "ECR registry hosts (ex: 123456789.dkr.ecr.us-west-2.amazonaws.com) to authorize for base images")
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
