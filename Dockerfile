FROM golang:1.24.5-alpine AS builder

WORKDIR /build

ENV GOCACHE=/root/.cache/go-build

RUN --mount=type=cache,target="/root/.cache/go-build" \
  go install github.com/mikefarah/yq/v4@latest

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=0 go build -ldflags "-s -w" -o /app ./cmd/waiting-room/main.go

FROM golang:1.24.5-alpine AS dev

WORKDIR /app

ENV GOCACHE=/root/.cache/go-build

RUN --mount=type=cache,target="/root/.cache/go-build" \
  apk add --no-cache git && \
  go install github.com/cosmtrek/air@latest

CMD [ "air", "-c", ".air.toml" ]

FROM gcr.io/distroless/static-debian12:nonroot AS prod

ENV TZ=Asia/Tokyo

USER nonroot

COPY --from=builder --chown=nonroot:nonroot /app /app

CMD [ "/app" ]
