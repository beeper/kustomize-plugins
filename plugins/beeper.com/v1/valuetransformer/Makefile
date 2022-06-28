default: all

all: linux mac

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../../../../../stack/tools/Linux-x86_64/valuetransformer

mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ../../../../../stack/tools/Darwin-x86_64/valuetransformer
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ../../../../../stack/tools/Darwin-arm64/valuetransformer
