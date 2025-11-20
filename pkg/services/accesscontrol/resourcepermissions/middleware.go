package resourcepermissions

import (
	contextmodel "github.com/capitalrx/grafana/pkg/services/contexthandler/model"
)

func nopMiddleware(c *contextmodel.ReqContext) {}
