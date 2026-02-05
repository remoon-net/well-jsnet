wasm:
	GOOS=js GOARCH=wasm go build -o gojs/well-jsnet.wasm ./jsnet
wasm-dev:
	GOOS=js GOARCH=wasm go build -o ../well.remoon.net/static/well-jsnet/well-jsnet.wasm ./jsnet
