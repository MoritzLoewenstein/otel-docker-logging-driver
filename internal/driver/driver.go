package driver

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/fifo"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/go-plugins-helpers/sdk"
	protoio "github.com/gogo/protobuf/io"

	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"

	"github.com/moritzloewenstein/otel-docker-logging-driver/internal/config"
	"github.com/moritzloewenstein/otel-docker-logging-driver/internal/otelx"
)

type StartLoggingRequest struct {
	File string
	Info logger.Info
}

type StopLoggingRequest struct {
	File string
}

type CapabilitiesResponse struct {
	Err string
	Cap logger.Capability
}

type ReadLogsRequest struct {
	Info   logger.Info
	Config logger.ReadConfig
}

type response struct{ Err string }

// Driver is the core logging driver implementation.
type Driver struct {
	mu   sync.Mutex
	logs map[string]*dockerInput
	cfg  config.Config
}

type dockerInput struct {
	stream io.ReadCloser
	info   logger.Info
	cancel context.CancelFunc
}

func New(cfg config.Config, _ any) *Driver {
	return &Driver{logs: make(map[string]*dockerInput), cfg: cfg}
}

func RegisterHandlers(h *sdk.Handler, d *Driver) {
	h.HandleFunc("/LogDriver.StartLogging", func(w http.ResponseWriter, r *http.Request) {
		var req StartLoggingRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		fmt.Fprintf(os.Stdout, "StartLogging: container=%s file=%s\n", req.Info.ContainerID, req.File)
		err := d.StartLogging(req.File, req.Info)
		writeResp(w, err)
	})

	h.HandleFunc("/LogDriver.StopLogging", func(w http.ResponseWriter, r *http.Request) {
		var req StopLoggingRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		fmt.Fprintf(os.Stdout, "StopLogging: file=%s\n", req.File)
		err := d.StopLogging(req.File)
		writeResp(w, err)
	})

	h.HandleFunc("/LogDriver.Capabilities", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(&CapabilitiesResponse{Cap: logger.Capability{ReadLogs: false}})
	})

	h.HandleFunc("/LogDriver.ReadLogs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"Err":"not implemented"}`))
	})
}

func writeResp(w http.ResponseWriter, err error) {
	var res response
	if err != nil {
		res.Err = err.Error()
	}
	_ = json.NewEncoder(w).Encode(&res)
}

func (d *Driver) StartLogging(file string, info logger.Info) error {
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("logger for %q already exists", file)
	}
	d.mu.Unlock()

	f, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return fmt.Errorf("open fifo %q: %w", file, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	d.mu.Lock()
	d.logs[file] = &dockerInput{stream: f, info: info, cancel: cancel}
	d.mu.Unlock()

	go d.consume(ctx, f, info)
	return nil
}

func (d *Driver) StopLogging(file string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	lf, ok := d.logs[file]
	if ok {
		lf.cancel()
		_ = lf.stream.Close()
		delete(d.logs, file)
	}
	return nil
}

func (d *Driver) consume(ctx context.Context, r io.ReadCloser, info logger.Info) {
	dec := protoio.NewUint32DelimitedReader(r, binary.BigEndian, 1e6)
	defer dec.Close()
	var entry logdriver.LogEntry

	otelLogger := global.Logger("otel-docker-logging-driver")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := dec.ReadMsg(&entry); err != nil {
			if err == io.EOF || err == io.ErrClosedPipe {
				return
			}
			// Recreate reader on transient error.
			dec = protoio.NewUint32DelimitedReader(r, binary.BigEndian, 1e6)
			continue
		}

		// Map Docker entry to OTEL log record.
		severity := olog.SeverityInfo
		if entry.Source == "stderr" {
			severity = olog.SeverityError
		}
		// Base attributes
		attrs := []olog.KeyValue{
			olog.String("docker.container.id", info.ContainerID),
			olog.String("docker.container.name", info.Name()),
			olog.String("docker.image.name", info.ContainerImageName),
			olog.String("docker.stream", entry.Source),
		}

		// Per-container options from --log-opt
		if v, ok := info.Config["include-labels"]; ok && (v == "1" || v == "true" || v == "yes") {
			for k, val := range info.ContainerLabels {
				attrs = append(attrs, olog.String("docker.label."+k, val))
			}
		}
		// TODO: include-env (Docker does not pass env by default to logging drivers)

		// Warn if unsupported per-container transport overrides are set
		if _, ok := info.Config["endpoint"]; ok {
			fmt.Fprintln(os.Stderr, "per-container endpoint override not yet supported; using plugin-level endpoint")
		}
		if _, ok := info.Config["headers"]; ok {
			fmt.Fprintln(os.Stderr, "per-container headers override not yet supported; using plugin-level headers")
		}

		rec := otelx.BuildRecord(time.Unix(0, entry.TimeNano), string(entry.Line), severity, attrs...)
		otelLogger.Emit(context.Background(), rec)
		entry.Reset()
	}
}
