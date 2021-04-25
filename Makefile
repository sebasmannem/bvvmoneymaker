build:
	go build ./cmd/bvv_moneymaker
debug:
	~/go/bin/dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient ./cmd/bvv_moneymaker/
