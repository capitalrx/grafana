package queryaudit

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	rpb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

const (
	// flushThresholdRecords requests an early upload so memory stays bounded between scheduled
	// flushes on busy instances.
	flushThresholdRecords = 20_000

	// maxBufferedRecords is a hard cap for when uploads keep failing. Past it new records are
	// dropped and counted, since running out of memory would lose the whole buffer.
	maxBufferedRecords = 5 * flushThresholdRecords
)

type s3PutObjectAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type scopeKey struct {
	resource attribute.Distinct
	scope    instrumentation.Scope
}

// s3Exporter is an sdklog.Exporter that buffers records in memory and uploads them as a single
// zstd-compressed OTLP ExportLogsServiceRequest (.binpb.zst) per flush. Export only converts and
// appends, so it is cheap enough to run synchronously on the query path.
type s3Exporter struct {
	client   s3PutObjectAPI
	bucket   string
	prefix   string
	instance string
	encoder  *zstd.Encoder
	now      func() time.Time

	// full is signalled (without blocking) when the buffer reaches flushThresholdRecords.
	full chan struct{}

	mu       sync.Mutex
	resLogs  []*lpb.ResourceLogs
	byRes    map[attribute.Distinct]*lpb.ResourceLogs
	byScope  map[scopeKey]*lpb.ScopeLogs
	buffered int
	dropped  int
	stopped  bool

	// flushMu serializes uploads so a failed batch can be requeued without reordering races.
	flushMu sync.Mutex
}

func newS3Exporter(client s3PutObjectAPI, bucket, prefix, instance string) (*s3Exporter, error) {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("creating zstd encoder: %w", err)
	}
	e := &s3Exporter{
		client:   client,
		bucket:   bucket,
		prefix:   prefix,
		instance: instance,
		encoder:  enc,
		now:      time.Now,
		full:     make(chan struct{}, 1),
	}
	e.reset()
	return e, nil
}

func (e *s3Exporter) reset() {
	e.resLogs = nil
	e.byRes = map[attribute.Distinct]*lpb.ResourceLogs{}
	e.byScope = map[scopeKey]*lpb.ScopeLogs{}
	e.buffered = 0
}

func (e *s3Exporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopped {
		return nil
	}

	for i := range records {
		if e.buffered >= maxBufferedRecords {
			e.dropped += len(records) - i
			break
		}
		r := &records[i]
		res := r.Resource()
		resKey := res.Equivalent()
		rl, ok := e.byRes[resKey]
		if !ok {
			rl = &lpb.ResourceLogs{SchemaUrl: res.SchemaURL()}
			if res.Len() > 0 {
				rl.Resource = &rpb.Resource{Attributes: attrIter(res.Iter())}
			}
			e.byRes[resKey] = rl
			e.resLogs = append(e.resLogs, rl)
		}

		scope := r.InstrumentationScope()
		sk := scopeKey{resource: resKey, scope: scope}
		sl, ok := e.byScope[sk]
		if !ok {
			sl = &lpb.ScopeLogs{
				SchemaUrl: scope.SchemaURL,
				Scope: &cpb.InstrumentationScope{
					Name:       scope.Name,
					Version:    scope.Version,
					Attributes: attrIter(scope.Attributes.Iter()),
				},
			}
			e.byScope[sk] = sl
			rl.ScopeLogs = append(rl.ScopeLogs, sl)
		}

		sl.LogRecords = append(sl.LogRecords, logRecord(r))
		e.buffered++
	}

	if e.buffered >= flushThresholdRecords {
		select {
		case e.full <- struct{}{}:
		default:
		}
	}
	return nil
}

// ForceFlush uploads everything buffered so far. On failure the batch is put back in the buffer
// so the next flush retries it.
func (e *s3Exporter) ForceFlush(ctx context.Context) error {
	e.flushMu.Lock()
	defer e.flushMu.Unlock()

	e.mu.Lock()
	batch, n, dropped := e.resLogs, e.buffered, e.dropped
	e.reset()
	e.dropped = 0
	e.mu.Unlock()

	var errs []error
	if dropped > 0 {
		errs = append(errs, fmt.Errorf("dropped %d query audit records because the buffer was full", dropped))
	}
	if n > 0 {
		if err := e.upload(ctx, batch); err != nil {
			e.requeue(batch)
			errs = append(errs, fmt.Errorf("uploading %d query audit records: %w", n, err))
		}
	}
	return errors.Join(errs...)
}

func (e *s3Exporter) Shutdown(ctx context.Context) error {
	err := e.ForceFlush(ctx)
	e.mu.Lock()
	e.stopped = true
	e.mu.Unlock()
	return err
}

// requeue puts a failed batch in front of anything recorded while the upload was in flight.
func (e *s3Exporter) requeue(batch []*lpb.ResourceLogs) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pending := e.resLogs
	e.reset()
	for _, rl := range append(batch, pending...) {
		e.resLogs = append(e.resLogs, rl)
		for _, sl := range rl.ScopeLogs {
			e.buffered += len(sl.LogRecords)
		}
	}
	// Grouping maps are only an optimisation for Export; leaving them empty after a requeue just
	// means new records start new ResourceLogs/ScopeLogs entries, which OTLP allows.
}

func (e *s3Exporter) upload(ctx context.Context, batch []*lpb.ResourceLogs) error {
	raw, err := proto.Marshal(&collogspb.ExportLogsServiceRequest{ResourceLogs: batch})
	if err != nil {
		return fmt.Errorf("marshalling OTLP logs: %w", err)
	}
	body := e.encoder.EncodeAll(raw, make([]byte, 0, len(raw)/4))

	key, err := e.objectKey()
	if err != nil {
		return err
	}
	_, err = e.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(e.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
		ContentType:   aws.String("application/x-protobuf"),
	})
	return err
}

// objectKey follows the layout used by the OpenTelemetry Collector awss3 exporter so these files
// are indexed the same way as every other service's logs.
func (e *s3Exporter) objectKey() (string, error) {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generating object key: %w", err)
	}
	t := e.now().UTC()
	name := fmt.Sprintf("%s_%d_%s.binpb.zst", e.instance, t.UnixNano(), hex.EncodeToString(suffix[:]))
	partition := t.Format("year=2006/month=01/day=02/hour=15")
	return path.Join(e.prefix, partition, name), nil
}

func logRecord(r *sdklog.Record) *lpb.LogRecord {
	out := &lpb.LogRecord{
		TimeUnixNano:         unixNano(r.Timestamp()),
		ObservedTimeUnixNano: unixNano(r.ObservedTimestamp()),
		EventName:            r.EventName(),
		// The OTel API severity values are defined to match the OTLP enum.
		SeverityNumber: lpb.SeverityNumber(r.Severity()), //nolint:gosec
		SeverityText:   r.SeverityText(),
		Body:           anyValue(r.Body()),
		Attributes:     make([]*cpb.KeyValue, 0, r.AttributesLen()),
		Flags:          uint32(r.TraceFlags()),
	}
	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		out.Attributes = append(out.Attributes, keyValue(kv))
		return true
	})
	if id := r.TraceID(); id.IsValid() {
		out.TraceId = id[:]
	}
	if id := r.SpanID(); id.IsValid() {
		out.SpanId = id[:]
	}
	return out
}

func unixNano(t time.Time) uint64 {
	n := t.UnixNano()
	if n < 0 {
		return 0
	}
	return uint64(n)
}

func attrIter(it attribute.Iterator) []*cpb.KeyValue {
	if it.Len() == 0 {
		return nil
	}
	out := make([]*cpb.KeyValue, 0, it.Len())
	for it.Next() {
		out = append(out, keyValue(it.Attribute()))
	}
	return out
}

func keyValue(kv attribute.KeyValue) *cpb.KeyValue {
	return &cpb.KeyValue{Key: string(kv.Key), Value: anyValue(kv.Value)}
}

func anyValue(v attribute.Value) *cpb.AnyValue {
	switch v.Type() {
	case attribute.BOOL:
		return &cpb.AnyValue{Value: &cpb.AnyValue_BoolValue{BoolValue: v.AsBool()}}
	case attribute.INT64:
		return &cpb.AnyValue{Value: &cpb.AnyValue_IntValue{IntValue: v.AsInt64()}}
	case attribute.FLOAT64:
		return &cpb.AnyValue{Value: &cpb.AnyValue_DoubleValue{DoubleValue: v.AsFloat64()}}
	case attribute.STRING:
		return &cpb.AnyValue{Value: &cpb.AnyValue_StringValue{StringValue: v.AsString()}}
	case attribute.BYTESLICE:
		return &cpb.AnyValue{Value: &cpb.AnyValue_BytesValue{BytesValue: v.AsByteSlice()}}
	case attribute.BOOLSLICE:
		return arrayValue(v.AsBoolSlice(), attribute.BoolValue)
	case attribute.INT64SLICE:
		return arrayValue(v.AsInt64Slice(), attribute.Int64Value)
	case attribute.FLOAT64SLICE:
		return arrayValue(v.AsFloat64Slice(), attribute.Float64Value)
	case attribute.STRINGSLICE:
		return arrayValue(v.AsStringSlice(), attribute.StringValue)
	case attribute.SLICE:
		return arrayValue(v.AsSlice(), func(v attribute.Value) attribute.Value { return v })
	case attribute.MAP:
		kvs := v.AsMap()
		out := make([]*cpb.KeyValue, 0, len(kvs))
		for _, kv := range kvs {
			out = append(out, keyValue(kv))
		}
		return &cpb.AnyValue{Value: &cpb.AnyValue_KvlistValue{KvlistValue: &cpb.KeyValueList{Values: out}}}
	default:
		return &cpb.AnyValue{}
	}
}

func arrayValue[T any](vals []T, wrap func(T) attribute.Value) *cpb.AnyValue {
	out := make([]*cpb.AnyValue, len(vals))
	for i, v := range vals {
		out[i] = anyValue(wrap(v))
	}
	return &cpb.AnyValue{Value: &cpb.AnyValue_ArrayValue{ArrayValue: &cpb.ArrayValue{Values: out}}}
}
