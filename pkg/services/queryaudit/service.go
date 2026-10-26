// Package queryaudit records every datasource query as an OpenTelemetry log record and ships the
// records to S3 as zstd-compressed OTLP protobuf files.
package queryaudit

import (
	"context"
	"fmt"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/setting"
)

const (
	scopeName = "github.com/grafana/grafana/pkg/services/queryaudit"

	// shutdownFlushTimeout must stay below the server's 30s shutdown deadline so the final upload
	// finishes before the process exits.
	shutdownFlushTimeout = 20 * time.Second

	fullFlushBackoff = time.Minute
)

type Service struct {
	cfg      setting.QueryAuditSettings
	log      log.Logger
	provider *sdklog.LoggerProvider
	logger   otellog.Logger
	exporter *s3Exporter
}

func ProvideService(cfg *setting.Cfg) (*Service, error) {
	s := &Service{cfg: cfg.QueryAudit, log: log.New("query-audit")}
	if !s.cfg.Enabled {
		return s, nil
	}

	var opts []func(*awsconfig.LoadOptions) error
	if s.cfg.S3Region != "" {
		opts = append(opts, awsconfig.WithRegion(s.cfg.S3Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("query audit: loading AWS config: %w", err)
	}

	return newService(cfg, s3.NewFromConfig(awsCfg), s.log)
}

func newService(cfg *setting.Cfg, client s3PutObjectAPI, logger log.Logger) (*Service, error) {
	instance, err := os.Hostname()
	if err != nil || instance == "" {
		instance = "grafana"
	}

	exporter, err := newS3Exporter(client, cfg.QueryAudit.S3Bucket, cfg.QueryAudit.S3Prefix, instance)
	if err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}

	res := resource.NewSchemaless(
		attribute.String("service.name", "grafana"),
		attribute.String("service.version", cfg.BuildVersion),
		attribute.String("service.instance.id", instance),
	)

	// The exporter only buffers in memory, so the simple processor is safe on the query path and,
	// unlike the batch processor, never drops records when a queue fills up.
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
		sdklog.WithAttributeValueLengthLimit(maxAttributeValueLength),
	)

	return &Service{
		cfg:      cfg.QueryAudit,
		log:      logger,
		provider: provider,
		logger:   provider.Logger(scopeName),
		exporter: exporter,
	}, nil
}

func (s *Service) IsDisabled() bool {
	return s == nil || !s.cfg.Enabled
}

func (s *Service) Run(ctx context.Context) error {
	s.log.Info("Query audit logging enabled", "bucket", s.cfg.S3Bucket, "prefix", s.cfg.S3Prefix, "flushInterval", s.cfg.FlushInterval)

	ticker := time.NewTicker(s.cfg.FlushInterval)
	defer ticker.Stop()

	var lastFailure time.Time
	for {
		select {
		case <-ctx.Done():
			// ctx is already cancelled here, so the final upload needs its own deadline.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownFlushTimeout)
			defer cancel()
			if err := s.provider.Shutdown(shutdownCtx); err != nil {
				s.log.Error("Failed to flush query audit logs on shutdown", "error", err)
			} else {
				s.log.Info("Flushed query audit logs on shutdown")
			}
			return nil
		case <-ticker.C:
			if !s.flush(ctx) {
				lastFailure = time.Now()
			}
		case <-s.exporter.full:
			// While S3 is failing, every new record past the threshold signals again; back off so
			// that does not turn into an upload attempt per query.
			if time.Since(lastFailure) < fullFlushBackoff {
				continue
			}
			if !s.flush(ctx) {
				lastFailure = time.Now()
			}
		}
	}
}

func (s *Service) flush(ctx context.Context) bool {
	if err := s.provider.ForceFlush(ctx); err != nil {
		s.log.Error("Failed to upload query audit logs, will retry on next flush", "error", err)
		return false
	}
	return true
}

// emit records a single audit event. It is a no-op when auditing is disabled.
func (s *Service) emit(ctx context.Context, ts time.Time, failed bool, body string, attrs []attribute.KeyValue) {
	if s.IsDisabled() {
		return
	}
	var r otellog.Record
	r.SetEventName(eventName)
	r.SetTimestamp(ts)
	r.SetObservedTimestamp(time.Now())
	if failed {
		r.SetSeverity(otellog.SeverityError)
		r.SetSeverityText("ERROR")
	} else {
		r.SetSeverity(otellog.SeverityInfo)
		r.SetSeverityText("INFO")
	}
	r.SetBody(attribute.StringValue(body))
	r.AddAttributes(attrs...)
	s.logger.Emit(ctx, r)
}
