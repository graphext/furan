set -e -x

go build
go vet $(go list ./... |grep pkg/)
go test -cover $(go list ./... |grep pkg/)
pushd protos
protoc --go_out=plugins=grpc,paths=source_relative:../pkg/generated/furanrpc api.proto
popd
DOCKER_BUILDKIT=1 docker build -t at .