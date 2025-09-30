generate:
	go generate ./...

up:
	docker compose up --build --watch

dev:
	docker compose --profile dev up --build waiting_room-dev valkey

test:
	GOEXPERIMENT=synctest go test -race -parallel $(shell nproc) ./...

lint:
	GOEXPERIMENT=synctest go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix ./...
