.PHONY: run
run:
	clear
	go run cmd/server/main.go $(ARGS)

.PHONY: build
build:
	clear
	go build ./...

.PHONY: wip
wip:
	clear
	go build -tags wip ./...

.PHONY: fmt
fmt:
	gofmt -w .
