set -e -x

go build
go vet $(go list ./... |grep pkg/)
go test -cover $(go list ./... |grep pkg/)
DOCKER_BUILDKIT=1 docker build -t at .