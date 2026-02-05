VERSION := $(shell git describe --tags --always --dirty)

wasm:
	GOOS=js GOARCH=wasm go build -ldflags="-X 'main.Version=${VERSION}'" -o gojs/well-jsnet.wasm ./jsnet
wasm-dev:
	GOOS=js GOARCH=wasm go build -ldflags="-X 'main.Version=${VERSION}'" -o ../well.remoon.net/static/well-jsnet/well-jsnet.wasm ./jsnet
