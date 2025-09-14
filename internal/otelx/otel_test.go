package otelx

import (
	"testing"
	"time"

	olog "go.opentelemetry.io/otel/log"
)

func TestBuildRecord(t *testing.T) {
	ts := time.Unix(123, 456)
	rec := BuildRecord(ts, "hello", olog.SeverityInfo, olog.String("a", "b"))

	if got := rec.Body().AsString(); got != "hello" {
		t.Fatalf("body=%q", got)
	}
	if got := rec.Severity(); got != olog.SeverityInfo {
		t.Fatalf("severity=%v", got)
	}
	if got := rec.Timestamp(); !got.Equal(ts) {
		t.Fatalf("timestamp=%v want %v", got, ts)
	}
	attrs := map[string]string{}
	rec.WalkAttributes(func(kv olog.KeyValue) bool {
		if v := kv.Value.AsString(); v != "" {
			attrs[kv.Key] = v
		}
		return true
	})
	if attrs["a"] != "b" {
		t.Fatalf("attrs=%v", attrs)
	}
}
