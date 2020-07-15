module github.com/dollarshaveclub/furan

go 1.14

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20190717161051-705d9623b7c1+incompatible

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/containerd/containerd v1.4.0-0 => github.com/containerd/containerd v1.4.0-beta.1.0.20200624184620-1127ffc7400e

require (
	github.com/aws/aws-sdk-go v1.33.5
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20200626212615-d883bee51b56
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v1.13.1 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.11.0
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/jackc/pgconn v1.5.0
	github.com/jackc/pgtype v1.3.0
	github.com/jackc/pgx/v4 v4.6.0
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mholt/archiver/v3 v3.3.0
	github.com/moby/buildkit v0.7.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	golang.org/x/crypto v0.0.0-20200707235045-ab33eee955e0
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20200610104632-a5b850bcf112 // indirect
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/api v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/client-go v0.18.4
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // indirect
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19 // indirect
)
