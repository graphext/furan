package cmd

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	docker "github.com/docker/engine-api/client"

	"github.com/dollarshaveclub/furan/generated/lib"
	"github.com/dollarshaveclub/furan/lib/buildcontext"
	"github.com/dollarshaveclub/furan/lib/builder"
	"github.com/dollarshaveclub/furan/lib/datalayer"
	githubfetch "github.com/dollarshaveclub/furan/lib/github_fetch"
	"github.com/dollarshaveclub/furan/lib/kafka"
	"github.com/dollarshaveclub/furan/lib/s3"
	"github.com/dollarshaveclub/furan/lib/squasher"
	"github.com/dollarshaveclub/furan/lib/tagcheck"
	"github.com/dollarshaveclub/furan/lib/vault"

	"log"

	"github.com/spf13/cobra"
)

// integrationCmd represents the integration command
var integrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "Run a set of integration tests",
	Long: `Run integration tests locally.

Uses secrets from either env vars or JSON file. Both will use --vault-path-prefix for naming.

Env var: <vault path prefix>_<secret name>   Example: SECRET_PRODUCTION_FURAN_AWS_ACCESS_KEY_ID
JSON file (object): { "secret/production/furan/aws/access_key_id": "asdf" }

Pass a JSON file containing options for the integration test (see testdata/integration.json).`,
	Run: integration,
}

var integrationOptionsFile string

type IntegrationOptions struct {
	GitHubRepo   string `json:"github_repo"`
	Ref          string `json:"ref"`
	ImageRepo    string `json:"image_repo"`
	SkipIfExists bool   `json:"skip"`
}

func (iops IntegrationOptions) BuildRequest() *lib.BuildRequest {
	return &lib.BuildRequest{
		Build: &lib.BuildDefinition{
			GithubRepo:       iops.GitHubRepo,
			Ref:              iops.Ref,
			TagWithCommitSha: true,
		},
		Push: &lib.PushDefinition{
			Registry: &lib.PushRegistryDefinition{
				Repo: iops.ImageRepo,
			},
		},
		SkipIfExists: iops.SkipIfExists,
	}
}

func init() {
	integrationCmd.Flags().BoolVar(&vaultConfig.EnvVars, "env-var-secrets", false, "use environment variable secrets (uses vault-path-prefix for naming scheme)")
	integrationCmd.Flags().StringVar(&vaultConfig.JSONFile, "json-secrets-file", "", "JSON secrets file")
	integrationCmd.Flags().StringVar(&integrationOptionsFile, "integration-options-file", "testdata/integration.json", "JSON integration options file")
	RootCmd.AddCommand(integrationCmd)
}

const chanCapacity = 100000

func integration(cmd *cobra.Command, args []string) {
	vaultConfig.TokenAuth = false
	vaultConfig.AppID = ""
	vaultConfig.K8sJWTPath = ""
	vault.SetupVault(&vaultConfig, &awsConfig, &dockerConfig, &gitConfig, &serverConfig, awscredsprefix)

	f, err := os.Open(integrationOptionsFile)
	if err != nil {
		log.Fatalf("error opening integration options file: %v", err)
	}
	defer f.Close()

	intops := map[string]IntegrationOptions{}
	if err := json.NewDecoder(f).Decode(&intops); err != nil {
		log.Fatalf("error unmarshaling integration options file: %v", err)
	}

	logger = log.New(os.Stderr, "", log.LstdFlags)

	mc, err := newDatadogCollector()
	if err != nil {
		log.Fatalf("error creating Datadog collector: %v", err)
	}

	err = getDockercfg()
	if err != nil {
		clierr("error getting dockercfg: %v", err)
	}

	gf := githubfetch.NewGitHubFetcher(gitConfig.Token)
	dc, err := docker.NewEnvClient()
	if err != nil {
		clierr("error creating Docker client: %v", err)
	}

	osm := s3.NewS3StorageManager(awsConfig, mc, logger)
	is := squasher.NewDockerImageSquasher(logger)
	itc := tagcheck.NewRegistryTagChecker(&dockerConfig, logger.Printf)
	s3errcfg := builder.S3ErrorLogConfig{
		PushToS3: false,
	}

	km := kafka.NewFakeEventBusProducer(chanCapacity)
	dl := &datalayer.FakeDataLayer{}

	ib, err := builder.NewImageBuilder(
		km,
		dl,
		gf,
		dc,
		mc,
		osm,
		is,
		itc,
		dockerConfig.DockercfgContents,
		s3errcfg,
		logger)
	if err != nil {
		clierr("error creating image builder: %v", err)
	}

	ib.SetECRConfig(awsConfig.AccessKeyID, awsConfig.SecretAccessKey, awsConfig.ECRRegistryHosts)

	tester := integrationTester{
		DL:  dl,
		IB:  ib,
		EBC: km,
	}

	for name, ops := range intops {
		log.Printf("running test: %v\n", name)
		tester.RunTest(ops.BuildRequest())
	}
}

type integrationTester struct {
	DL  datalayer.DataLayer
	IB  *builder.ImageBuilder
	EBC kafka.EventBusConsumer
}

func streamMessage(rawmsg string) string {
	stream := struct {
		Stream string `json:"stream"`
	}{}
	if err := json.Unmarshal([]byte(rawmsg), &stream); err != nil {
		return rawmsg
	}
	return stream.Stream
}

func (it *integrationTester) RunTest(req *lib.BuildRequest) {
	c := make(chan *lib.BuildEvent, chanCapacity)
	s := make(chan struct{})
	defer close(s)

	go func() {
		for e := range c {
			log.Println(streamMessage(e.Message))
		}
	}()

	ctx := context.Background()

	id, err := it.DL.CreateBuild(ctx, req)
	if err != nil {
		log.Fatalf("error creating build: %v", err)
	}

	ctx = buildcontext.NewBuildIDContext(ctx, id, nil)

	if err := it.EBC.SubscribeToTopic(c, s, id); err != nil {
		log.Fatalf("error subscribing to build events: %v", nil)
	}

	imageid, err := it.IB.Build(ctx, req, id)
	if err != nil {
		if req.SkipIfExists && strings.Contains(err.Error(), "build not necessary") {
			log.Println("push not needed, ending test")
			return
		}
		log.Fatalf("test failed: %v", err)
	}

	switch {
	case req.GetPush().Registry != nil:
		if err := it.IB.PushBuildToRegistry(ctx, req); err != nil {
			log.Fatalf("error pushing to registry: %v", err)
		}
	case req.GetPush().S3 != nil:
		if err := it.IB.PushBuildToS3(ctx, imageid, req); err != nil {
			log.Fatalf("error pushing to S3: %v", err)
		}
	default:
		log.Println("no push defined")
	}
}
