package dashboardsnapshots

import (
	"github.com/capitalrx/grafana/pkg/apimachinery/errutil"
)

var ErrBaseNotFound = errutil.NotFound("dashboardsnapshots.not-found", errutil.WithPublicMessage("Snapshot not found"))
