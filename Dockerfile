FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder
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
