package config

import (
	"os"
	"testing"
)

func TestNormalizeProtocol(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"grpc":            "grpc",
		"GRPC":            "grpc",
		"http":            "http",
		"HTTP/PROTOBUF":   "http",
		"http-protobuf":   "http",
		"http_proto":      "http",
		"something-else":  "",
	}
	for in, want := range cases {
		if got := normalizeProtocol(in); got != want {
			t.Fatalf("normalizeProtocol(%q)=%q want %q", in, got, want)
		}
	}
}

func TestParseHeaders(t *testing.T) {
	m := parseHeaders("")
	if len(m) != 0 { t.Fatalf("expected empty map, got %v", m) }

	m = parseHeaders("a=b, c=d ,e=f=g")
	if m["a"] != "b" || m["c"] != "d" || m["e"] != "f=g" {
		t.Fatalf("unexpected headers parse: %v", m)
	}
}

func TestFromEnv(t *testing.T) {
	// Save and restore env
	save := func(k string) (string, func()) {
		old := os.Getenv(k)
		return old, func() { _ = os.Setenv(k, old) }
	}

	restores := []func(){}
	for _, k := range []string{
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_LOGS_INSECURE",
		"OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_EXPORTER_OTLP_LOGS_HEADERS",
		"OTEL_EXPORTER_OTLP_HEADERS",
		"OTEL_EXPORTER_OTLP_LOGS_COMPRESSION",
	} {
		_, r := save(k)
		restores = append(restores, r)
		_ = os.Unsetenv(k)
	}
	defer func() { for _, r := range restores { r() } }()

	// Defaults
	cfg := FromEnv()
	if cfg.Endpoint == "" || cfg.Endpoint == "http://" {
		t.Fatalf("unexpected default endpoint: %q", cfg.Endpoint)
	}

	// Explicit LOGS_* override
	os.Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "https://collector:4318")
	os.Setenv("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "http/protobuf")
	os.Setenv("OTEL_EXPORTER_OTLP_LOGS_INSECURE", "true")
	os.Setenv("OTEL_EXPORTER_OTLP_LOGS_HEADERS", "k=v,x=y")
	os.Setenv("OTEL_EXPORTER_OTLP_LOGS_COMPRESSION", "gzip")
	cfg = FromEnv()
	if cfg.Endpoint != "https://collector:4318" { t.Fatalf("endpoint=%q", cfg.Endpoint) }
	if cfg.Protocol != "http" { t.Fatalf("protocol=%q", cfg.Protocol) }
	if !cfg.Insecure { t.Fatalf("insecure expected true") }
	if len(cfg.Headers) != 2 || cfg.Headers["k"] != "v" || cfg.Headers["x"] != "y" { t.Fatalf("headers=%v", cfg.Headers) }
	if cfg.Compression != "gzip" { t.Fatalf("compression=%q", cfg.Compression) }

	// Fallback to generic vars
	os.Unsetenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
	os.Unsetenv("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL")
	os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	os.Unsetenv("OTEL_EXPORTER_OTLP_LOGS_INSECURE")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "TRUE")
	os.Unsetenv("OTEL_EXPORTER_OTLP_LOGS_HEADERS")
	os.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "a=b")
	cfg = FromEnv()
	if cfg.Endpoint != "collector:4317" { t.Fatalf("endpoint=%q", cfg.Endpoint) }
	if cfg.Protocol != "grpc" { t.Fatalf("protocol=%q", cfg.Protocol) }
	if !cfg.Insecure { t.Fatalf("insecure expected true") }
	if cfg.Headers["a"] != "b" { t.Fatalf("headers=%v", cfg.Headers) }
}
