package config

import (
	"os"
	"strings"
)

type Config struct {
	// OTLP endpoint (collector): http(s)://host:port or host:port
	Endpoint string
	// Protocol: "grpc" or "http" (maps from OTEL "grpc" or "http/protobuf")
	Protocol string
	// If true, disable TLS when endpoint has no scheme (back-compat)
	Insecure bool
	// Optional headers: k=v,k2=v2
	Headers map[string]string
	// Compression: "gzip" or ""
	Compression string
}

func FromEnv() Config {
	c := Config{
		Endpoint:    getenvDefault("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", getenvDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")),
		Protocol:    normalizeProtocol(getenvDefault("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"))),
		Insecure:    strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_LOGS_INSECURE"), "true") || strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"), "true"),
		Headers:     parseHeaders(getenvDefault("OTEL_EXPORTER_OTLP_LOGS_HEADERS", os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))),
		Compression: os.Getenv("OTEL_EXPORTER_OTLP_LOGS_COMPRESSION"),
	}
	return c
}

func normalizeProtocol(p string) string {
	p = strings.TrimSpace(strings.ToLower(p))
	switch p {
	case "grpc":
		return "grpc"
	case "http", "http/protobuf", "http_proto", "http-protobuf":
		return "http"
	default:
		return "" // auto-detect by endpoint scheme later
	}
}

func parseHeaders(s string) map[string]string {
	m := map[string]string{}
	if s == "" {
		return m
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return m
}

func getenvDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
