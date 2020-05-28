set -e -x

export GO111MODULE=off

go build
dep check
go test -cover $(go list ./... |grep lib/)
docker build -t at .