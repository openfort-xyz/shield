FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder
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
