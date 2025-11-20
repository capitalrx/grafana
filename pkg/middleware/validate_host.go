package middleware

import (
	"strings"

	contextmodel "github.com/capitalrx/grafana/pkg/services/contexthandler/model"
	"github.com/capitalrx/grafana/pkg/setting"
	"github.com/capitalrx/grafana/pkg/web"
)

func ValidateHostHeader(cfg *setting.Cfg) web.Handler {
	return func(c *contextmodel.ReqContext) {
		// ignore local render calls
		if c.IsRenderCall {
			return
		}

		h := c.Req.Host
		if i := strings.Index(h, ":"); i >= 0 {
			h = h[:i]
		}

		if !strings.EqualFold(h, cfg.Domain) {
			c.Redirect(strings.TrimSuffix(cfg.AppURL, "/")+c.Req.RequestURI, 301)
			return
		}
	}
}
