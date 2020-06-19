module github.com/dollarshaveclub/furan

go 1.14

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20190717161051-705d9623b7c1+incompatible

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v1.13.1
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.11.0
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.4.1 // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/gotestyourself/gotestyourself v1.4.0 // indirect
	github.com/jackc/pgconn v1.5.0
	github.com/jackc/pgtype v1.3.0
	github.com/jackc/pgx/v4 v4.6.0
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/spf13/cobra v1.0.0
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9 // indirect
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20200610104632-a5b850bcf112 // indirect
	google.golang.org/grpc v1.29.1
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gotest.tools v1.4.0 // indirect
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v0.17.7
	k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89 // indirect
)
