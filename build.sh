
set -x

VERSION=$1



GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o webhook-amd64 main.go

docker build -t allenhaozi/webhook.tar:v0.0.${VERSION} -f Dockerfile.local .


kind load docker-image allenhaozi/webhook.tar:v0.0.${VERSION} --name fluid-dev
