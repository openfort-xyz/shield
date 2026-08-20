FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS builder
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
