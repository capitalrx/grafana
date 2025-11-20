package routing

import (
	"net/http"

	"github.com/capitalrx/grafana/pkg/api/response"
	contextmodel "github.com/capitalrx/grafana/pkg/services/contexthandler/model"
	"github.com/capitalrx/grafana/pkg/web"
)

var (
	ServerError = func(err error) response.Response {
		return response.Error(http.StatusInternalServerError, "Server error", err)
	}
)

func Wrap(handler func(c *contextmodel.ReqContext) response.Response) web.Handler {
	return func(c *contextmodel.ReqContext) {
		if res := handler(c); res != nil {
			res.WriteTo(c)
		}
	}
}
