build:
	go build ./cmd/bvv_moneymaker
debug:
	~/go/bin/dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient ./cmd/bvv_moneymaker/
run:
	./bvv_moneymaker
fmt:
	gofmt -w .
	goimports -w .
	gci write .

test: sec lint

sec:
	gosec ./...
lint:
	golangci-lint run
