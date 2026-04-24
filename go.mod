module github.com/openfort-xyz/shield

go 1.26.0

require (
	github.com/MicahParks/keyfunc/v3 v3.7.0
	github.com/caarlos0/env/v10 v10.0.0
	github.com/codahale/sss v0.0.0-20160501174526-0cb9f6d3f7f1
	github.com/go-sql-driver/mysql v1.9.3
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/mileusna/useragent v1.3.5
	github.com/openfort-xyz/metrics v0.0.8
	github.com/openfort-xyz/shamir-secret-sharing-go v0.0.2
	github.com/pressly/goose v2.7.0+incompatible
	github.com/resend/resend-go/v3 v3.0.0
	github.com/rs/cors v1.11.1
	github.com/smsapi/smsapi-go v1.5.0
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.11.1
	github.com/tidwall/buntdb v1.3.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.68.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	go.uber.org/ratelimit v0.3.1
	golang.org/x/crypto v0.49.0
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.31.0
)

require (
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/MicahParks/jwkset v0.11.0 // indirect
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/openfort-xyz/jsonrpc v0.0.7 // indirect
	github.com/openfort-xyz/pubsub v0.1.25 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.61.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tidwall/btree v1.7.0 // indirect
	github.com/tidwall/gjson v1.17.1 // indirect
	github.com/tidwall/grect v0.1.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/rtred v0.1.2 // indirect
	github.com/tidwall/tinyqueue v0.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

tool github.com/google/wire/cmd/wire
