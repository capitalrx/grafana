package sources

import (
	"context"

	"github.com/capitalrx/grafana/pkg/plugins"
)

type Registry interface {
	List(context.Context) []plugins.PluginSource
}
