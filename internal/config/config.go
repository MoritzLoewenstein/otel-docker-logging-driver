package config

import (
	"os"
	"strings"
)

type Config struct {
	// OTLP gRPC endpoint, e.g. https://collector:4317
	Endpoint string
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
		Insecure:    strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_LOGS_INSECURE"), "true") || strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"), "true"),
		Headers:     parseHeaders(getenvDefault("OTEL_EXPORTER_OTLP_LOGS_HEADERS", os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))),
		Compression: os.Getenv("OTEL_EXPORTER_OTLP_LOGS_COMPRESSION"),
	}
	return c
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
