package api

import (
	"net/http"

	"github.com/capitalrx/grafana/pkg/api/response"
	"github.com/capitalrx/grafana/pkg/infra/metrics"
	contextmodel "github.com/capitalrx/grafana/pkg/services/contexthandler/model"
	"github.com/capitalrx/grafana/pkg/web"
)

func (hs *HTTPServer) PostFrontendMetrics(c *contextmodel.ReqContext) response.Response {
	cmd := metrics.PostFrontendMetricsCommand{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	for _, event := range cmd.Events {
		if recorder, ok := metrics.FrontendMetrics[event.Name]; ok {
			recorder(event)
		} else {
			c.Logger.Debug("Received unknown frontend metric", "metric", event.Name)
		}
	}
	return response.Empty(200)
}
