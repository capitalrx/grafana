package queryaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	claims "github.com/grafana/authlib/types"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"go.opentelemetry.io/otel/attribute"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/contexthandler"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
)

const (
	eventName = "grafana.datasource.query"

	// maxAttributeValueLength caps query bodies, which can be arbitrarily large SQL or JSON.
	maxAttributeValueLength = 32 * 1024

	// fromAlertHeader matches ngalert's models.FromAlertHeaderName. It is duplicated rather than
	// imported because ngalert depends on the plugin client that this package is wired into.
	fromAlertHeader = "FromAlert"
)

const (
	InitiatorUser           = "user"
	InitiatorServiceAccount = "service_account"
	InitiatorAnonymous      = "anonymous"
	// InitiatorGrafana marks queries Grafana runs on its own, such as scheduled alert rule
	// evaluation, where no HTTP request or user is involved.
	InitiatorGrafana = "grafana"
)

const (
	SourceExplore         = "explore"
	SourceDashboard       = "dashboard"
	SourcePublicDashboard = "public_dashboard"
	SourceDrilldown       = "drilldown"
	SourceApp             = "app"
	SourceAlerting        = "alerting"
	SourceAPI             = "api"
	SourceInternal        = "internal"
)

// drilldownApps are the app plugins behind the Drilldown section of the navigation.
var drilldownApps = map[string]bool{
	"grafana-lokiexplore-app":      true,
	"grafana-exploretraces-app":    true,
	"grafana-metricsdrilldown-app": true,
	"grafana-pyroscope-app":        true,
}

const (
	kindQuery    = "query"
	kindResource = "resource"
	kindStream   = "stream"
)

// RecordQueryData records one audit event per query in req.
func (s *Service) RecordQueryData(ctx context.Context, req *backend.QueryDataRequest, resp *backend.QueryDataResponse, err error, start time.Time) {
	if s.IsDisabled() || req == nil {
		return
	}
	headers := func(key string) string { return req.GetHTTPHeader(key) }
	common := s.commonAttributes(ctx, req.PluginContext, req.Headers[fromAlertHeader] == "true", headers)
	common = append(common,
		attribute.String("grafana.query.kind", kindQuery),
		attribute.Int64("grafana.query.duration_ms", time.Since(start).Milliseconds()),
	)
	subject := subjectOf(ctx, req.PluginContext)

	for _, q := range req.Queries {
		attrs := append([]attribute.KeyValue{}, common...)
		attrs = append(attrs,
			attribute.String("grafana.query.ref_id", q.RefID),
			attribute.String("grafana.query.time_range.from", q.TimeRange.From.UTC().Format(time.RFC3339)),
			attribute.String("grafana.query.time_range.to", q.TimeRange.To.UTC().Format(time.RFC3339)),
			queryModel(q.JSON),
		)
		if q.QueryType != "" {
			attrs = append(attrs, attribute.String("grafana.query.type", q.QueryType))
		}

		qErr := err
		if qErr == nil && resp != nil {
			if dr, ok := resp.Responses[q.RefID]; ok && dr.Error != nil {
				qErr = dr.Error
			}
		}
		attrs = appendStatus(attrs, qErr)
		s.emit(ctx, start, qErr != nil, describe(subject, kindQuery, req.PluginContext), attrs)
	}
}

// RecordResource records a datasource resource call, which backs things like label and field
// lookups in query editors and the Drilldown apps.
func (s *Service) RecordResource(ctx context.Context, req *backend.CallResourceRequest, statusCode int, err error, start time.Time) {
	if s.IsDisabled() || req == nil || req.PluginContext.DataSourceInstanceSettings == nil {
		return
	}
	headers := func(key string) string { return firstHeader(req.Headers, key) }
	attrs := s.commonAttributes(ctx, req.PluginContext, firstHeader(req.Headers, fromAlertHeader) == "true", headers)
	attrs = append(attrs,
		attribute.String("grafana.query.kind", kindResource),
		attribute.Int64("grafana.query.duration_ms", time.Since(start).Milliseconds()),
		attribute.String("grafana.resource.path", req.Path),
		attribute.String("http.request.method", req.Method),
	)
	if req.URL != "" {
		attrs = append(attrs, attribute.String("grafana.resource.url", req.URL))
	}
	if len(req.Body) > 0 {
		attrs = append(attrs, queryModel(req.Body))
	}
	if statusCode > 0 {
		attrs = append(attrs, attribute.Int("http.response.status_code", statusCode))
	}
	if err == nil && statusCode >= 400 {
		err = fmt.Errorf("resource call returned status %d", statusCode)
	}
	attrs = appendStatus(attrs, err)
	s.emit(ctx, start, err != nil, describe(subjectOf(ctx, req.PluginContext), kindResource, req.PluginContext), attrs)
}

// RecordStream records a datasource stream subscription, e.g. live tailing logs in Explore.
func (s *Service) RecordStream(ctx context.Context, req *backend.SubscribeStreamRequest, err error, start time.Time) {
	if s.IsDisabled() || req == nil || req.PluginContext.DataSourceInstanceSettings == nil {
		return
	}
	headers := func(key string) string { return req.GetHTTPHeader(key) }
	attrs := s.commonAttributes(ctx, req.PluginContext, false, headers)
	attrs = append(attrs,
		attribute.String("grafana.query.kind", kindStream),
		attribute.String("grafana.stream.path", req.Path),
	)
	if len(req.Data) > 0 {
		attrs = append(attrs, queryModel(req.Data))
	}
	attrs = appendStatus(attrs, err)
	s.emit(ctx, start, err != nil, describe(subjectOf(ctx, req.PluginContext), kindStream, req.PluginContext), attrs)
}

func (s *Service) commonAttributes(ctx context.Context, pCtx backend.PluginContext, fromAlert bool, header func(string) string) []attribute.KeyValue {
	reqCtx := contexthandler.FromContext(ctx)
	requester, _ := identity.GetRequester(ctx)
	if requester != nil && requester.IsNil() {
		requester = nil
	}

	attrs := make([]attribute.KeyValue, 0, 32)
	attrs = append(attrs,
		attribute.String("grafana.query.initiator", initiatorOf(reqCtx, requester)),
		attribute.String("grafana.plugin.id", pCtx.PluginID),
	)
	attrs = appendIfSet(attrs, "grafana.namespace", pCtx.Namespace)
	if requester != nil {
		attrs = append(attrs, attribute.Int64("grafana.org.id", requester.GetOrgID()))
	}
	attrs = append(attrs, userAttributes(requester, pCtx.User)...)

	if ds := pCtx.DataSourceInstanceSettings; ds != nil {
		attrs = append(attrs,
			attribute.String("grafana.datasource.uid", ds.UID),
			attribute.String("grafana.datasource.name", ds.Name),
			attribute.String("grafana.datasource.type", ds.Type),
		)
	}

	source, appID, pagePath := sourceOf(reqCtx, fromAlert, header)
	attrs = append(attrs, attribute.String("grafana.query.source", source))
	attrs = appendIfSet(attrs, "grafana.app.id", appID)
	attrs = appendIfSet(attrs, "grafana.page.path", pagePath)
	attrs = appendIfSet(attrs, "grafana.dashboard.uid", header("X-Dashboard-Uid"))
	attrs = appendIfSet(attrs, "grafana.dashboard.title", header("X-Dashboard-Title"))
	attrs = appendIfSet(attrs, "grafana.panel.id", header("X-Panel-Id"))
	attrs = appendIfSet(attrs, "grafana.panel.plugin_id", header("X-Panel-Plugin-Id"))
	attrs = appendIfSet(attrs, "grafana.query.group_id", header("X-Query-Group-Id"))

	if fromAlert {
		for _, h := range []struct{ header, key string }{
			{"X-Rule-Uid", "grafana.alert.rule.uid"},
			{"X-Rule-Name", "grafana.alert.rule.name"},
			{"X-Rule-Folder", "grafana.alert.rule.folder"},
			{"X-Rule-Type", "grafana.alert.rule.type"},
			{"X-Rule-Source", "grafana.alert.rule.source"},
		} {
			// ngalert query-escapes rule metadata before putting it in headers.
			v := header(h.header)
			if unescaped, err := url.QueryUnescape(v); err == nil {
				v = unescaped
			}
			attrs = appendIfSet(attrs, h.key, v)
		}
	}

	if reqCtx != nil && reqCtx.Req != nil {
		attrs = appendIfSet(attrs, "client.address", reqCtx.RemoteAddr())
		attrs = appendIfSet(attrs, "user_agent.original", reqCtx.Req.UserAgent())
	}
	return attrs
}

func initiatorOf(reqCtx *contextmodel.ReqContext, requester identity.Requester) string {
	switch {
	case requester != nil && requester.IsIdentityType(claims.TypeUser):
		return InitiatorUser
	case reqCtx == nil:
		// Background work such as the alert scheduler runs without an HTTP request, under a
		// synthetic service identity rather than a person.
		return InitiatorGrafana
	case requester == nil || requester.IsIdentityType(claims.TypeAnonymous):
		return InitiatorAnonymous
	case requester.IsIdentityType(claims.TypeServiceAccount, claims.TypeAPIKey):
		return InitiatorServiceAccount
	default:
		return InitiatorGrafana
	}
}

func userAttributes(requester identity.Requester, user *backend.User) []attribute.KeyValue {
	var login, email, name, role, id string
	if user != nil {
		login, email, name, role = user.Login, user.Email, user.Name, user.Role
	}
	if requester != nil {
		id = requester.GetUID()
		login = firstNonEmpty(requester.GetLogin(), login)
		email = firstNonEmpty(requester.GetEmail(), email)
		name = firstNonEmpty(requester.GetName(), name)
		role = firstNonEmpty(string(requester.GetOrgRole()), role)
	}

	var attrs []attribute.KeyValue
	attrs = appendIfSet(attrs, "user.id", id)
	attrs = appendIfSet(attrs, "user.name", login)
	attrs = appendIfSet(attrs, "user.email", email)
	attrs = appendIfSet(attrs, "user.full_name", name)
	attrs = appendIfSet(attrs, "user.roles", role)
	return attrs
}

// subjectOf names whoever ran the query for the human-readable log body.
func subjectOf(ctx context.Context, pCtx backend.PluginContext) string {
	if requester, err := identity.GetRequester(ctx); err == nil && !requester.IsNil() {
		if v := firstNonEmpty(requester.GetEmail(), requester.GetLogin()); v != "" {
			return v
		}
	}
	if pCtx.User != nil {
		if v := firstNonEmpty(pCtx.User.Email, pCtx.User.Login); v != "" {
			return v
		}
	}
	return InitiatorAnonymous
}

func describe(subject, kind string, pCtx backend.PluginContext) string {
	target := pCtx.PluginID
	if ds := pCtx.DataSourceInstanceSettings; ds != nil {
		target = fmt.Sprintf("%s (%s)", ds.Name, ds.UID)
	}
	return fmt.Sprintf("%s ran a datasource %s against %s", subject, kind, target)
}

// sourceOf classifies where in Grafana a query came from. The browser sets the Referer of every
// query request to the page that issued it, which is the only signal shared by Explore,
// dashboards and app plugins.
func sourceOf(reqCtx *contextmodel.ReqContext, fromAlert bool, header func(string) string) (source, appID, pagePath string) {
	if fromAlert {
		source = SourceAlerting
	}
	if reqCtx == nil || reqCtx.Req == nil {
		if source == "" {
			source = SourceInternal
		}
		return source, "", ""
	}

	if ref, err := url.Parse(reqCtx.Req.Referer()); err == nil {
		pagePath = ref.Path
	}
	if source != "" {
		return source, "", pagePath
	}
	if reqCtx.IsPublicDashboardView() {
		return SourcePublicDashboard, "", pagePath
	}

	segments := strings.Split(strings.Trim(pagePath, "/"), "/")
	for i, seg := range segments {
		switch seg {
		case "explore":
			if i+1 < len(segments) && segments[i+1] == "metrics" {
				return SourceDrilldown, "", pagePath
			}
			return SourceExplore, "", pagePath
		case "drilldown":
			return SourceDrilldown, "", pagePath
		case "d", "d-solo", "dashboard":
			return SourceDashboard, "", pagePath
		case "public-dashboards":
			return SourcePublicDashboard, "", pagePath
		case "alerting":
			return SourceAlerting, "", pagePath
		case "a":
			if i+1 < len(segments) {
				appID = segments[i+1]
				if drilldownApps[appID] {
					return SourceDrilldown, appID, pagePath
				}
				return SourceApp, appID, pagePath
			}
		}
	}

	if header("X-Dashboard-Uid") != "" {
		return SourceDashboard, "", pagePath
	}
	return SourceAPI, "", pagePath
}

func appendStatus(attrs []attribute.KeyValue, err error) []attribute.KeyValue {
	if err != nil {
		return append(attrs,
			attribute.String("grafana.query.status", "error"),
			attribute.String("error.message", err.Error()),
		)
	}
	return append(attrs, attribute.String("grafana.query.status", "ok"))
}

func appendIfSet(attrs []attribute.KeyValue, key, value string) []attribute.KeyValue {
	if value == "" {
		return attrs
	}
	return append(attrs, attribute.String(key, value))
}

func firstHeader(headers map[string][]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// queryModel records a query as a structured map rather than a JSON string so it is indexed as a
// nested object and individual fields (expr, rawSql, datasource.uid, ...) can be searched. Anything
// that is not a JSON object, or is too large to index sensibly, is kept as a (truncated) string.
func queryModel(raw []byte) attribute.KeyValue {
	const key = "grafana.query.model"
	if len(raw) <= maxAttributeValueLength {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var obj map[string]any
		if err := dec.Decode(&obj); err == nil && obj != nil && !dec.More() {
			return attribute.KeyValue{Key: key, Value: jsonValue(obj)}
		}
	}
	return attribute.String(key, string(raw))
}

func jsonValue(v any) attribute.Value {
	switch v := v.(type) {
	case map[string]any:
		kvs := make([]attribute.KeyValue, 0, len(v))
		for _, k := range slices.Sorted(maps.Keys(v)) {
			kvs = append(kvs, attribute.KeyValue{Key: attribute.Key(k), Value: jsonValue(v[k])})
		}
		return attribute.MapValue(kvs...)
	case []any:
		vals := make([]attribute.Value, len(v))
		for i, e := range v {
			vals[i] = jsonValue(e)
		}
		return attribute.SliceValue(vals...)
	case string:
		return attribute.StringValue(v)
	case bool:
		return attribute.BoolValue(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return attribute.Int64Value(i)
		}
		if f, err := v.Float64(); err == nil {
			return attribute.Float64Value(f)
		}
		return attribute.StringValue(v.String())
	default:
		// JSON null has no OTLP equivalent; Quickwit drops empty values.
		return attribute.Value{}
	}
}
