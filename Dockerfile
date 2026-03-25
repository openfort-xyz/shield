FROM golang:1.24.5-alpine AS builder
RUN apk add --no-cache ca-certificates git

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=secret,id=GITHUB_TOKEN \
    GITHUB_TOKEN="$(cat /run/secrets/GITHUB_TOKEN)" && \
    git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com".insteadOf "https://github.com" && \
    go env -w GOPRIVATE=github.com/openfort-xyz/* && \
    go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o app cmd/main.go

FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/app /usr/bin/
COPY internal/adapters/repositories/sql/migrations /app/internal/adapters/repositories/sql/migrations
ENTRYPOINT ["app"]