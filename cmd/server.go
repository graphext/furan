package cmd

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	// Import pprof handlers into http.DefaultServeMux
	_ "net/http/pprof"

	"github.com/spf13/cobra"

	"github.com/dollarshaveclub/furan/pkg/builder"
	"github.com/dollarshaveclub/furan/pkg/config"
	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/github"
	"github.com/dollarshaveclub/furan/pkg/grpc"
	"github.com/dollarshaveclub/furan/pkg/jobrunner"
	"github.com/dollarshaveclub/furan/pkg/models"
)

var serverConfig config.ServerConfig

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run Furan server",
	Long:  `Furan API server (see docs)`,
	PreRun: func(cmd *cobra.Command, args []string) {
		gitHubSecrets()
		awsSecrets()
		quaySecrets()
		dbSecrets()
	},
	Run: server,
}

var defaultcachetype, tracesvcname string
var defaultcachemaxmode bool
var tlscert, tlskey string
var jcinterval, jcage time.Duration

func init() {
	serverCmd.PersistentFlags().StringVar(&serverConfig.HTTPSAddr, "https-addr", "0.0.0.0:4001", "REST HTTPS listen address")
	serverCmd.PersistentFlags().StringVar(&serverConfig.GRPCAddr, "grpc-addr", "0.0.0.0:4000", "gRPC listen address")
	serverCmd.PersistentFlags().StringVar(&tracesvcname, "trace-svc", "furan2", "APM trace service name (optional)")

	// Job cleanup
	serverCmd.PersistentFlags().DurationVar(&jcinterval, "job-cleanup-interval", 30*time.Minute, "Build job cleanup check interval")
	serverCmd.PersistentFlags().DurationVar(&jcage, "job-cleanup-min-age", 48*time.Hour, "Minimum age for build jobs to be eligible for cleanup (deletion)")

	// TLS cert
	serverCmd.PersistentFlags().StringVar(&tlscert, "tls-cert-file", "", "Path to TLS certificate (optional, overrides secrets provider)")
	serverCmd.PersistentFlags().StringVar(&tlskey, "tls-key-file", "", "Path to TLS key (optional, overrides secrets provider)")

	// Default cache options
	serverCmd.PersistentFlags().StringVar(&defaultcachetype, "default-cache-type", "disabled", "Default cache type (if not specified in build request): s3, inline, disabled")
	serverCmd.PersistentFlags().BoolVar(&defaultcachemaxmode, "default-cache-max-mode", false, "Cache max mode by default")

	RootCmd.AddCommand(serverCmd)
}

func cachedefaults() models.CacheOpts {
	out := models.CacheOpts{}
	switch defaultcachetype {
	case "s3":
		out.Type = models.S3CacheType
	case "inline":
		out.Type = models.InlineCacheType
	case "disabled":
		out.Type = models.DisabledCacheType
	default:
		log.Printf("unknown cache type: %v; ignoring", defaultcachetype)
		// leave as unset/unknown
	}
	out.MaxMode = defaultcachemaxmode
	return out
}

func server(cmd *cobra.Command, args []string) {
	dl, err := datalayer.NewPostgresDBLayer(dbConfig.PostgresURI)
	if err != nil {
		clierr("error configuring database: %v", err)
	}

	ctx := context.Background()

	jr, err := jobrunner.NewInClusterRunner(dl)
	if err != nil {
		clierr("error setting up in-cluster job runner: %v", err)
	}
	jr.JobFunc = jobrunner.FuranJobFunc
	jr.LogFunc = log.Printf
	if err := jr.StartCleanup(ctx, jcinterval, jcage, jobrunner.JobLabel); err != nil {
		clierr("error starting job cleanup: %v", err)
	}

	bm := &builder.Manager{
		JRunner: jr,
		DL:      dl,
	}

	var cert tls.Certificate
	if tlskey != "" && tlscert != "" {
		c, err := tls.LoadX509KeyPair(tlscert, tlskey)
		if err != nil {
			clierr("error loading tls cert/key: %v", err)
		}
		cert = c
	}

	gopts := grpc.Options{
		TraceSvcName:            tracesvcname,
		CredentialDecryptionKey: dbConfig.CredEncKeyArray,
		Cache:                   cachedefaults(),
		TLSCertificate:          cert,
		LogFunc:                 log.Printf,
	}

	grs := grpc.Server{
		DL:        dl,
		BM:        bm,
		CFFactory: func(tkn string) models.CodeFetcher { return github.NewGitHubFetcher(tkn) },
		Opts:      gopts,
	}

	log.Println(grs.Listen(serverConfig.GRPCAddr))

}
