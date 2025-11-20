package process

import (
	"context"

	"github.com/capitalrx/grafana/pkg/plugins"
)

type Manager interface {
	// Start executes a backend plugin process.
	Start(ctx context.Context, p *plugins.Plugin) error
	// Stop terminates a backend plugin process.
	Stop(ctx context.Context, p *plugins.Plugin) error
}
