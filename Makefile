generate:
	go generate ./...

up:
	docker compose up --build --watch

test:
	GOEXPERIMENT=synctest go test -race -parallel $(shell nproc) ./...

lint:
	GOEXPERIMENT=synctest go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix ./...