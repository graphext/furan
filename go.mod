module github.com/dollarshaveclub/furan

go 1.14

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20190717161051-705d9623b7c1+incompatible

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/containerd/containerd v1.4.0-0 => github.com/containerd/containerd v1.4.0-beta.1.0.20200624184620-1127ffc7400e

replace github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305

require (
	github.com/DataDog/datadog-go v3.7.2+incompatible // indirect
	github.com/Microsoft/hcsshim v0.8.10 // indirect
	github.com/aws/aws-sdk-go v1.33.7
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20200626212615-d883bee51b56
	github.com/containerd/cgroups v0.0.0-20200824123100-0b889c03f102 // indirect
	github.com/containerd/console v1.0.0 // indirect
	github.com/containerd/containerd v1.4.1 // indirect
	github.com/containerd/continuity v0.0.0-20200710164510-efbc4488d8fe // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v1.13.1 // indirect
	github.com/dollarshaveclub/pvc v1.0.0
	github.com/gofrs/flock v0.8.0 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.11.0
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/jackc/pgconn v1.6.4
	github.com/jackc/pgproto3/v2 v2.0.4 // indirect
	github.com/jackc/pgtype v1.4.2
	github.com/jackc/pgx/v4 v4.8.1
	github.com/jaguilar/vt100 v0.0.0-20150826170717-2703a27b14ea // indirect
	github.com/jhump/protoreflect v1.7.0
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/mholt/archiver/v3 v3.3.0
	github.com/moby/buildkit v0.7.2
	github.com/opencontainers/go-digest v1.0.0
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/tonistiigi/fsutil v0.0.0-20200803195102-c92e8e0e204a // indirect
	go.opencensus.io v0.22.4 // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/net v0.0.0-20200923182212-328152dc79b1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208 // indirect
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20200924141100-a14c0a98937d // indirect
	google.golang.org/grpc v1.32.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/DataDog/dd-trace-go.v1 v1.25.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/api v0.19.1
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v0.19.1
)
