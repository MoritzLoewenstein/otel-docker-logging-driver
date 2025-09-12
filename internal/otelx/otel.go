package otelx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc/credentials"

	"github.com/moritzloewenstein/otel-docker-logging-driver/internal/config"
)

type Exporter = logsdk.Exporter

type Provider = logsdk.LoggerProvider

func SetupProvider(ctx context.Context, cfg config.Config) (Exporter, *Provider, error) {
	protocol := cfg.Protocol
	if protocol == "" {
		// Backwards-compatible default is gRPC, even if endpoint has http(s) scheme
		protocol = "grpc"
	}

	var exp Exporter
	var err error

	switch strings.ToLower(protocol) {
	case "http":
		opts := []otlploghttp.Option{}
		// Endpoint
		if cfg.Endpoint != "" {
			if u, err := url.Parse(cfg.Endpoint); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
				// If no path is specified, default to logs resource path
				if u.Path == "" || u.Path == "/" {
					u.Path = "/v1/logs"
				}
				opts = append(opts, otlploghttp.WithEndpointURL(u.String()))
			} else {
				opts = append(opts, otlploghttp.WithEndpoint(cfg.Endpoint))
			}
		}
		// Insecure
		if cfg.Insecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		// Headers
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlploghttp.WithHeaders(cfg.Headers))
		}
		// Note: compression and file-based TLS are currently applied only for gRPC
		exp, err = otlploghttp.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("create otlp http logs exporter: %w", err)
		}
	default: // grpc
		opts := []otlploggrpc.Option{}
		// Endpoint
		if cfg.Endpoint != "" {
			if u, err := url.Parse(cfg.Endpoint); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
				opts = append(opts, otlploggrpc.WithEndpointURL(cfg.Endpoint))
			} else {
				opts = append(opts, otlploggrpc.WithEndpoint(cfg.Endpoint))
			}
		}
		// Insecure
		if cfg.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		// Headers
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlploggrpc.WithHeaders(cfg.Headers))
		}
		// Compression
		switch cfg.Compression {
		case "gzip", "GZIP":
			opts = append(opts, otlploggrpc.WithCompressor("gzip"))
		}
		// Optional TLS via files (env-based): prefer LOGS_* vars, fall back to generic OTLP_* vars
		ca := os.Getenv("OTEL_EXPORTER_OTLP_LOGS_CERTIFICATE")
		if ca == "" {
			ca = os.Getenv("OTEL_EXPORTER_OTLP_CERTIFICATE")
		}
		if ca != "" {
			clientCert := os.Getenv("OTEL_EXPORTER_OTLP_LOGS_CLIENT_CERTIFICATE")
			if clientCert == "" {
				clientCert = os.Getenv("OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE")
			}
			clientKey := os.Getenv("OTEL_EXPORTER_OTLP_LOGS_CLIENT_KEY")
			if clientKey == "" {
				clientKey = os.Getenv("OTEL_EXPORTER_OTLP_CLIENT_KEY")
			}
			creds, lerr := loadTLSCreds(ca, clientCert, clientKey)
			if lerr == nil {
				opts = append(opts, otlploggrpc.WithTLSCredentials(creds))
			}
		}
		exp, err = otlploggrpc.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("create otlp grpc logs exporter: %w", err)
		}
	}

	proc := logsdk.NewBatchProcessor(exp)
	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("otel-docker-logging-driver"),
		attribute.String("process.executable.name", os.Args[0]),
	))
	provider := logsdk.NewLoggerProvider(
		logsdk.WithProcessor(proc),
		logsdk.WithResource(res),
	)
	global.SetLoggerProvider(provider)
	return exp, provider, nil
}

func loadTLSCreds(caFile, certFile, keyFile string) (credentials.TransportCredentials, error) {
	certPool := x509.NewCertPool()
	pemServerCA, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	if ok := certPool.AppendCertsFromPEM(pemServerCA); !ok {
		return nil, fmt.Errorf("failed to add server CA")
	}
	var clientCerts []tls.Certificate
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		clientCerts = []tls.Certificate{cert}
	}
	config := &tls.Config{RootCAs: certPool, Certificates: clientCerts, MinVersion: tls.VersionTLS12}
	return credentials.NewTLS(config), nil
}

// BuildRecord constructs a log record with standard mapping.
func BuildRecord(ts time.Time, body string, severity olog.Severity, attrs ...olog.KeyValue) olog.Record {
	var rec olog.Record
	rec.SetTimestamp(ts)
	rec.SetObservedTimestamp(time.Now())
	rec.SetSeverity(severity)
	rec.SetBody(olog.StringValue(body))
	rec.AddAttributes(attrs...)
	return rec
}
