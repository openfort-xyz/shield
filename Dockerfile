FROM golang:1.22.0-alpine as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o app cmd/main.go

FROM scratch
WORKDIR /app
COPY --from=builder /app/app /usr/bin/
COPY internal/infrastructure/repositories/sql/migrations /app/internal/infrastructure/repositories/sql/migrations
ENTRYPOINT ["app"]