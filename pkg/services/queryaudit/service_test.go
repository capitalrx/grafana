package queryaudit

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/contexthandler/ctxkey"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/web"
)

type fakeS3 struct {
	mu      sync.Mutex
	err     error
	uploads []*s3.PutObjectInput
	bodies  [][]byte
}

func (f *fakeS3) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	body, err := io.ReadAll(in.Body)
	if err != nil {
		return nil, err
	}
	f.uploads = append(f.uploads, in)
	f.bodies = append(f.bodies, body)
	return &s3.PutObjectOutput{}, nil
}

func newTestService(t *testing.T, client *fakeS3) *Service {
	t.Helper()
	cfg := setting.NewCfg()
	cfg.BuildVersion = "13.0.0"
	cfg.QueryAudit = setting.QueryAuditSettings{
		Enabled:       true,
		S3Bucket:      "observability-logs-cap-rx-com",
		S3Prefix:      "services/grafana_audit",
		FlushInterval: time.Hour,
	}
	s, err := newService(cfg, client, log.NewNopLogger())
	require.NoError(t, err)
	return s
}

func decodeUpload(t *testing.T, body []byte) []*lpb.LogRecord {
	t.Helper()
	dec, err := zstd.NewReader(nil)
	require.NoError(t, err)
	defer dec.Close()
	raw, err := dec.DecodeAll(body, nil)
	require.NoError(t, err)

	var req collogspb.ExportLogsServiceRequest
	require.NoError(t, proto.Unmarshal(raw, &req))

	var out []*lpb.LogRecord
	for _, rl := range req.ResourceLogs {
		for _, sl := range rl.ScopeLogs {
			out = append(out, sl.LogRecords...)
		}
	}
	return out
}

func attrs(r *lpb.LogRecord) map[string]string {
	out := map[string]string{}
	for _, kv := range r.Attributes {
		if sv, ok := kv.Value.Value.(*cpb.AnyValue_StringValue); ok {
			out[kv.Key] = sv.StringValue
		} else {
			out[kv.Key] = kv.Value.String()
		}
	}
	return out
}

func attrValue(r *lpb.LogRecord, key string) *cpb.AnyValue {
	for _, kv := range r.Attributes {
		if kv.Key == key {
			return kv.Value
		}
	}
	return nil
}

func withBrowserRequest(t *testing.T, referer string, u *user.SignedInUser) context.Context {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/ds/query", nil)
	req.Header.Set("Referer", referer)
	reqCtx := &contextmodel.ReqContext{
		Context:      &web.Context{Req: req},
		SignedInUser: u,
	}
	ctx := ctxkey.Set(req.Context(), reqCtx)
	return identity.WithRequester(ctx, u)
}

func lokiPluginContext(u *user.SignedInUser) backend.PluginContext {
	pCtx := backend.PluginContext{
		OrgID:    1,
		PluginID: "loki",
		DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
			UID:  "loki-uid",
			Name: "Loki",
			Type: "loki",
		},
	}
	if u != nil {
		pCtx.User = &backend.User{Login: u.Login, Email: u.Email, Name: u.Name, Role: string(u.OrgRole)}
	}
	return pCtx
}

func TestRecordQueryData_UploadsOTLPProtobufZstd(t *testing.T) {
	client := &fakeS3{}
	s := newTestService(t, client)
	alice := &user.SignedInUser{UserID: 7, UserUID: "alice-uid", Login: "alice", Email: "alice@cap-rx.com", OrgID: 1, OrgRole: "Viewer"}
	ctx := withBrowserRequest(t, "https://grafana.cap-rx.com/explore?left=abc", alice)

	from := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	req := &backend.QueryDataRequest{
		PluginContext: lokiPluginContext(alice),
		Headers:       map[string]string{},
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: []byte(`{"expr":"{app=\"api\"}"}`), TimeRange: backend.TimeRange{From: from, To: from.Add(time.Hour)}},
			{RefID: "B", JSON: []byte(`{"expr":"{app=\"web\"}"}`), TimeRange: backend.TimeRange{From: from, To: from.Add(time.Hour)}},
		},
	}
	resp := &backend.QueryDataResponse{Responses: backend.Responses{
		"A": {},
		"B": {Error: errors.New("boom")},
	}}

	s.RecordQueryData(ctx, req, resp, nil, time.Now())
	require.NoError(t, s.provider.ForceFlush(context.Background()))

	require.Len(t, client.uploads, 1)
	in := client.uploads[0]
	require.Equal(t, "observability-logs-cap-rx-com", *in.Bucket)
	require.Regexp(t, `^services/grafana_audit/year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/.+\.binpb\.zst$`, *in.Key)

	records := decodeUpload(t, client.bodies[0])
	require.Len(t, records, 2)

	a := attrs(records[0])
	require.Equal(t, eventName, records[0].EventName)
	require.Equal(t, lpb.SeverityNumber_SEVERITY_NUMBER_INFO, records[0].SeverityNumber)
	require.Equal(t, InitiatorUser, a["grafana.query.initiator"])
	require.Equal(t, SourceExplore, a["grafana.query.source"])
	require.Equal(t, "alice@cap-rx.com", a["user.email"])
	require.Equal(t, "alice", a["user.name"])
	require.Equal(t, "loki-uid", a["grafana.datasource.uid"])
	require.Equal(t, "A", a["grafana.query.ref_id"])
	model := attrValue(records[0], "grafana.query.model").GetKvlistValue()
	require.NotNil(t, model, "query model should be a structured map, not a string")
	require.Equal(t, `{app="api"}`, model.Values[0].Value.GetStringValue())
	require.Equal(t, "ok", a["grafana.query.status"])
	require.Contains(t, records[0].Body.GetStringValue(), "alice@cap-rx.com")

	b := attrs(records[1])
	require.Equal(t, lpb.SeverityNumber_SEVERITY_NUMBER_ERROR, records[1].SeverityNumber)
	require.Equal(t, "error", b["grafana.query.status"])
	require.Equal(t, "boom", b["error.message"])
}

func TestRecordQueryData_ScheduledAlertIsGrafanaInitiated(t *testing.T) {
	client := &fakeS3{}
	s := newTestService(t, client)

	// The alert scheduler runs without an HTTP request, as a synthetic service account.
	req := &backend.QueryDataRequest{
		PluginContext: backend.PluginContext{
			OrgID:                      1,
			PluginID:                   "prometheus",
			User:                       &backend.User{Login: "grafana_scheduler", Role: "Admin"},
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "mimir", Name: "Mimir", Type: "prometheus"},
		},
		Headers: map[string]string{
			fromAlertHeader:      "true",
			"http_X-Rule-Uid":    "rule-1",
			"http_X-Rule-Name":   "High%20error%20rate",
			"http_X-Rule-Folder": "Platform",
		},
		Queries: []backend.DataQuery{{RefID: "A", JSON: []byte(`{"expr":"up"}`)}},
	}

	s.RecordQueryData(context.Background(), req, &backend.QueryDataResponse{}, nil, time.Now())
	require.NoError(t, s.provider.ForceFlush(context.Background()))

	records := decodeUpload(t, client.bodies[0])
	require.Len(t, records, 1)
	a := attrs(records[0])
	require.Equal(t, InitiatorGrafana, a["grafana.query.initiator"])
	require.Equal(t, SourceAlerting, a["grafana.query.source"])
	require.Equal(t, "grafana_scheduler", a["user.name"])
	require.Equal(t, "rule-1", a["grafana.alert.rule.uid"])
	require.Equal(t, "High error rate", a["grafana.alert.rule.name"])
}

func TestRecordResource_OnlyDatasources(t *testing.T) {
	client := &fakeS3{}
	s := newTestService(t, client)
	bob := &user.SignedInUser{UserID: 8, UserUID: "bob-uid", Login: "bob", Email: "bob@cap-rx.com", OrgID: 1}
	ctx := withBrowserRequest(t, "https://grafana.cap-rx.com/a/grafana-lokiexplore-app/explore", bob)

	s.RecordResource(ctx, &backend.CallResourceRequest{
		PluginContext: backend.PluginContext{PluginID: "grafana-lokiexplore-app"},
		Path:          "settings",
	}, 200, nil, time.Now())
	s.RecordResource(ctx, &backend.CallResourceRequest{
		PluginContext: lokiPluginContext(bob),
		Path:          "labels",
		Method:        http.MethodGet,
	}, 500, nil, time.Now())
	require.NoError(t, s.provider.ForceFlush(context.Background()))

	records := decodeUpload(t, client.bodies[0])
	require.Len(t, records, 1)
	a := attrs(records[0])
	require.Equal(t, SourceDrilldown, a["grafana.query.source"])
	require.Equal(t, "grafana-lokiexplore-app", a["grafana.app.id"])
	require.Equal(t, "labels", a["grafana.resource.path"])
	require.Equal(t, "error", a["grafana.query.status"])
}

func TestForceFlush_RequeuesFailedUploads(t *testing.T) {
	client := &fakeS3{err: errors.New("s3 unavailable")}
	s := newTestService(t, client)
	ctx := withBrowserRequest(t, "https://grafana.cap-rx.com/d/abc/my-dash", &user.SignedInUser{UserID: 9, Login: "carol", OrgID: 1})
	q := func(ref string) *backend.QueryDataRequest {
		return &backend.QueryDataRequest{PluginContext: lokiPluginContext(nil), Queries: []backend.DataQuery{{RefID: ref}}}
	}

	s.RecordQueryData(ctx, q("A"), nil, nil, time.Now())
	require.Error(t, s.provider.ForceFlush(context.Background()))

	s.RecordQueryData(ctx, q("B"), nil, nil, time.Now())
	client.err = nil
	require.NoError(t, s.provider.ForceFlush(context.Background()))

	require.Len(t, client.uploads, 1)
	records := decodeUpload(t, client.bodies[0])
	require.Len(t, records, 2)
	require.Equal(t, "A", attrs(records[0])["grafana.query.ref_id"])
	require.Equal(t, "B", attrs(records[1])["grafana.query.ref_id"])
	require.Equal(t, SourceDashboard, attrs(records[0])["grafana.query.source"])

	// Nothing left to upload.
	require.NoError(t, s.provider.ForceFlush(context.Background()))
	require.Len(t, client.uploads, 1)
}

func TestRun_FlushesOnShutdown(t *testing.T) {
	client := &fakeS3{}
	s := newTestService(t, client)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	s.RecordQueryData(context.Background(), &backend.QueryDataRequest{
		PluginContext: lokiPluginContext(nil),
		Queries:       []backend.DataQuery{{RefID: "A"}},
	}, nil, nil, time.Now())
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}
	require.Len(t, client.uploads, 1)

	// Records emitted after shutdown are dropped rather than panicking.
	s.RecordQueryData(context.Background(), &backend.QueryDataRequest{
		PluginContext: lokiPluginContext(nil),
		Queries:       []backend.DataQuery{{RefID: "B"}},
	}, nil, nil, time.Now())
}

func TestDisabledServiceIsNoop(t *testing.T) {
	cfg := setting.NewCfg()
	s, err := ProvideService(cfg)
	require.NoError(t, err)
	require.True(t, s.IsDisabled())
	s.RecordQueryData(context.Background(), &backend.QueryDataRequest{Queries: []backend.DataQuery{{RefID: "A"}}}, nil, nil, time.Now())

	var nilService *Service
	require.True(t, nilService.IsDisabled())
}

func TestSourceOf(t *testing.T) {
	noHeaders := func(string) string { return "" }
	dashHeader := func(k string) string {
		if k == "X-Dashboard-Uid" {
			return "abc"
		}
		return ""
	}

	tests := []struct {
		name      string
		referer   string
		fromAlert bool
		headers   func(string) string
		source    string
		appID     string
	}{
		{"explore", "https://g.example.com/explore?left=1", false, noHeaders, SourceExplore, ""},
		{"explore under sub path", "https://g.example.com/grafana/explore", false, noHeaders, SourceExplore, ""},
		{"metrics drilldown", "https://g.example.com/explore/metrics/trail", false, noHeaders, SourceDrilldown, ""},
		{"traces drilldown app", "https://g.example.com/a/grafana-exploretraces-app/explore", false, noHeaders, SourceDrilldown, "grafana-exploretraces-app"},
		{"other app", "https://g.example.com/a/my-app/page", false, noHeaders, SourceApp, "my-app"},
		{"dashboard", "https://g.example.com/d/abc/my-dash?orgId=1", false, noHeaders, SourceDashboard, ""},
		{"solo panel", "https://g.example.com/d-solo/abc/my-dash", false, noHeaders, SourceDashboard, ""},
		{"alert rule editor", "https://g.example.com/alerting/new", false, noHeaders, SourceAlerting, ""},
		{"alert preview header wins", "https://g.example.com/explore", true, noHeaders, SourceAlerting, ""},
		{"api with dashboard header", "", false, dashHeader, SourceDashboard, ""},
		{"api", "", false, noHeaders, SourceAPI, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/ds/query", nil)
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			reqCtx := &contextmodel.ReqContext{Context: &web.Context{Req: req}, SignedInUser: &user.SignedInUser{}}
			source, appID, _ := sourceOf(reqCtx, tt.fromAlert, tt.headers)
			require.Equal(t, tt.source, source)
			require.Equal(t, tt.appID, appID)
		})
	}

	t.Run("no http request", func(t *testing.T) {
		source, _, _ := sourceOf(nil, false, noHeaders)
		require.Equal(t, SourceInternal, source)
	})
}

func TestObjectKeyIsUniquePerUpload(t *testing.T) {
	e, err := newS3Exporter(&fakeS3{}, "b", "services/grafana_audit", "grafana-0")
	require.NoError(t, err)
	e.now = func() time.Time { return time.Date(2026, 10, 25, 9, 30, 0, 0, time.UTC) }

	k1, err := e.objectKey()
	require.NoError(t, err)
	k2, err := e.objectKey()
	require.NoError(t, err)
	require.NotEqual(t, k1, k2)
	require.True(t, strings.HasPrefix(k1, "services/grafana_audit/year=2026/month=10/day=25/hour=09/grafana-0_"), k1)
	require.True(t, strings.HasSuffix(k1, ".binpb.zst"), k1)
}

func TestExport_DropsPastHardCap(t *testing.T) {
	client := &fakeS3{err: errors.New("s3 unavailable")}
	e, err := newS3Exporter(client, "b", "p", "grafana-0")
	require.NoError(t, err)
	e.buffered = maxBufferedRecords

	require.NoError(t, e.Export(context.Background(), make([]sdklog.Record, 3)))
	require.Equal(t, maxBufferedRecords, e.buffered)
	require.Equal(t, 3, e.dropped)

	err = e.ForceFlush(context.Background())
	require.ErrorContains(t, err, "dropped 3 query audit records")
	require.Zero(t, e.dropped)
}

func TestQueryModel(t *testing.T) {
	t.Run("nested JSON becomes a structured map", func(t *testing.T) {
		kv := queryModel([]byte(`{"refId":"A","intervalMs":30000,"ratio":0.5,"hide":false,"alias":null,` +
			`"datasource":{"type":"quickwit-quickwit-datasource","uid":"adjudication-api-logs"},` +
			`"bucketAggs":[{"id":"2","settings":{"interval":"auto"}}]}`))
		pb := anyValue(kv.Value)

		got := map[string]*cpb.AnyValue{}
		for _, e := range pb.GetKvlistValue().GetValues() {
			got[e.Key] = e.Value
		}
		require.Equal(t, "A", got["refId"].GetStringValue())
		require.Equal(t, int64(30000), got["intervalMs"].GetIntValue())
		require.Equal(t, 0.5, got["ratio"].GetDoubleValue())
		require.False(t, got["hide"].GetBoolValue())
		require.Nil(t, got["alias"].GetValue(), "null should be an empty value")

		ds := map[string]string{}
		for _, e := range got["datasource"].GetKvlistValue().GetValues() {
			ds[e.Key] = e.Value.GetStringValue()
		}
		require.Equal(t, map[string]string{"type": "quickwit-quickwit-datasource", "uid": "adjudication-api-logs"}, ds)

		aggs := got["bucketAggs"].GetArrayValue().GetValues()
		require.Len(t, aggs, 1)
		require.NotNil(t, aggs[0].GetKvlistValue())
	})

	for name, raw := range map[string]string{
		"not JSON":          "rawSql=select 1",
		"JSON array":        `[1,2]`,
		"trailing garbage":  `{"a":1} {"b":2}`,
		"over the size cap": `{"q":"` + strings.Repeat("x", maxAttributeValueLength) + `"}`,
	} {
		t.Run(name+" stays a string", func(t *testing.T) {
			kv := queryModel([]byte(raw))
			require.Equal(t, attribute.STRING, kv.Value.Type())
			require.Equal(t, raw, kv.Value.AsString())
		})
	}
}
