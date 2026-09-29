module github.com/kamalyes/go-logger-benchmark

go 1.25.0

replace github.com/kamalyes/go-logger => ../go-logger

require (
	github.com/kamalyes/go-logger v0.6.3-0.20260929050357-19f6ce565150
	github.com/rs/zerolog v1.34.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	google.golang.org/grpc v1.77.0 // indirect
)
