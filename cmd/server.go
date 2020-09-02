// +build linux darwin freebsd netbsd openbsd

package cmd

import (
	"log"
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

func init() {
	serverCmd.PersistentFlags().StringVar(&serverConfig.HTTPSAddr, "https-addr", "0.0.0.0:4001", "REST HTTPS listen address")
	serverCmd.PersistentFlags().StringVar(&serverConfig.GRPCAddr, "grpc-addr", "0.0.0.0:4000", "gRPC listen address")
	serverCmd.PersistentFlags().StringVar(&tracesvcname, "trace-svc", "furan2", "APM trace service name (optional)")

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

	jr, err := jobrunner.NewInClusterRunner(dl)
	if err != nil {
		clierr("error setting up in-cluster job runner: %v", err)
	}
	jr.JobFunc = jobrunner.FuranJobFunc

	bm := &builder.Manager{
		JRunner: jr,
		DL:      dl,
	}

	gopts := grpc.Options{
		TraceSvcName:            tracesvcname,
		CredentialDecryptionKey: dbConfig.CredEncKeyArray,
		Cache:                   cachedefaults(),
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
