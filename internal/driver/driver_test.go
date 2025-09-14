package driver

import (
	"context"
	"encoding/binary"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	protoio "github.com/gogo/protobuf/io"

	"github.com/moritzloewenstein/otel-docker-logging-driver/internal/config"

	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	logsdk "go.opentelemetry.io/otel/sdk/log"
)

// captureExporter implements a logsdk.Exporter to capture records synchronously.
type captureExporter struct {
	mu   sync.Mutex
	recs []logsdk.Record
}

func (e *captureExporter) Export(ctx context.Context, records []logsdk.Record) error {
	e.mu.Lock()
	e.recs = append(e.recs, records...)
	e.mu.Unlock()
	return nil
}
func (e *captureExporter) Shutdown(context.Context) error   { return nil }
func (e *captureExporter) ForceFlush(context.Context) error { return nil }

func TestConsume_MappingAndLabels(t *testing.T) {
	// Install a capturing logger provider.
	exp := &captureExporter{}
	proc := logsdk.NewSimpleProcessor(exp)
	provider := logsdk.NewLoggerProvider(logsdk.WithProcessor(proc))
	global.SetLoggerProvider(provider)

	// Prepare a pipe with two docker log entries: stdout and stderr.
	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()

	info := logger.Info{
		ContainerID:        "cid123",
		ContainerImageName: "busybox",
		ContainerName:      "demo",
		Config:             map[string]string{"include-labels": "true"},
		ContainerLabels:    map[string]string{"test.label": "demo"},
	}

	d := New(config.Config{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.consume(ctx, pr, info)

	w := protoio.NewUint32DelimitedWriter(pw, binary.BigEndian)
	write := func(src, body string, ts int64) {
		entry := &logdriver.LogEntry{
			Source:   src,
			Line:     []byte(body),
			TimeNano: ts,
		}
		_ = w.WriteMsg(entry)
	}
	write("stdout", "hello", time.Now().UnixNano())
	write("stderr", "oops", time.Now().UnixNano())
	_ = pw.Close()

	// Wait for records to be captured.
	deadline := time.Now().Add(2 * time.Second)
	for {
		exp.mu.Lock()
		n := len(exp.recs)
		exp.mu.Unlock()
		if n >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for records, got %d", n)
		}
		time.Sleep(10 * time.Millisecond)
	}

	exp.mu.Lock()
	recs := append([]logsdk.Record(nil), exp.recs...)
	exp.mu.Unlock()

	// Validate mapping of first record (stdout)
	b := reccStr(recs[0].Body())
	if b != "hello" {
		t.Fatalf("body0=%q", b)
	}
	if reccSev(recs[0]) != olog.SeverityInfo {
		t.Fatalf("sev0=%v", reccSev(recs[0]))
	}

	attrs := recAttrs(recs[0])
	if attrs["docker.container.id"] != "cid123" {
		t.Fatalf("attr id=%q", attrs["docker.container.id"])
	}
	if attrs["docker.container.name"] == "" {
		t.Fatalf("missing container name attr")
	}
	if attrs["docker.image.name"] != "busybox" {
		t.Fatalf("image=%q", attrs["docker.image.name"])
	}
	if attrs["docker.stream"] != "stdout" {
		t.Fatalf("stream=%q", attrs["docker.stream"])
	}
	if attrs["docker.label.test.label"] != "demo" {
		t.Fatalf("label missing: %v", attrs)
	}

	// Second record should map stderr to error severity
	if reccSev(recs[1]) != olog.SeverityError {
		t.Fatalf("sev1=%v", reccSev(recs[1]))
	}
}

// helpers to read values from sdk/log.Record

func reccStr(v olog.Value) string {
	return v.AsString()
}

func reccSev(r logsdk.Record) olog.Severity { return r.Severity() }

func recAttrs(r logsdk.Record) map[string]string {
	m := map[string]string{}
	r.WalkAttributes(func(kv olog.KeyValue) bool {
		if s := kv.Value.AsString(); s != "" {
			m[kv.Key] = s
		}
		return true
	})
	return m
}
