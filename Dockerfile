FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -installsuffix cgo -tags "netgo osusergo" -trimpath -buildvcs=false -ldflags "-w -s" -o app cmd/main.go


FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/app /usr/bin/
COPY internal/adapters/repositories/sql/migrations /app/internal/adapters/repositories/sql/migrations
ENTRYPOINT ["app"]
CMD ["server"]
