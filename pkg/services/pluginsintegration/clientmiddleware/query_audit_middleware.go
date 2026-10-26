package clientmiddleware

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"

	"github.com/grafana/grafana/pkg/services/queryaudit"
)

// NewQueryAuditMiddleware creates a backend.HandlerMiddleware that records every datasource query,
// resource call and stream subscription in the query audit log. Every query path (the query API
// used by Explore, dashboards and app plugins, expressions, and alert rule evaluation) goes
// through the plugin client, so this is the one place that sees all of them.
func NewQueryAuditMiddleware(auditor *queryaudit.Service) backend.HandlerMiddleware {
	return backend.HandlerMiddlewareFunc(func(next backend.Handler) backend.Handler {
		if auditor.IsDisabled() {
			return next
		}
		return &QueryAuditMiddleware{
			BaseHandler: backend.NewBaseHandler(next),
			auditor:     auditor,
		}
	})
}

type QueryAuditMiddleware struct {
	backend.BaseHandler
	auditor *queryaudit.Service
}

func (m *QueryAuditMiddleware) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	start := time.Now()
	resp, err := m.BaseHandler.QueryData(ctx, req)
	m.auditor.RecordQueryData(ctx, req, resp, err, start)
	return resp, err
}

func (m *QueryAuditMiddleware) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	start := time.Now()
	statusCode := 0
	wrapped := backend.CallResourceResponseSenderFunc(func(res *backend.CallResourceResponse) error {
		if statusCode == 0 && res != nil {
			statusCode = res.Status
		}
		return sender.Send(res)
	})
	err := m.BaseHandler.CallResource(ctx, req, wrapped)
	m.auditor.RecordResource(ctx, req, statusCode, err, start)
	return err
}

func (m *QueryAuditMiddleware) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	start := time.Now()
	resp, err := m.BaseHandler.SubscribeStream(ctx, req)
	m.auditor.RecordStream(ctx, req, err, start)
	return resp, err
}
