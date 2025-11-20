package appregistry

import (
	"github.com/google/wire"

	"github.com/capitalrx/grafana/pkg/registry/apps/advisor"
	"github.com/capitalrx/grafana/pkg/registry/apps/alerting/notifications"
	"github.com/capitalrx/grafana/pkg/registry/apps/investigations"
	"github.com/capitalrx/grafana/pkg/registry/apps/playlist"
	"github.com/capitalrx/grafana/pkg/registry/apps/plugins"
	"github.com/capitalrx/grafana/pkg/registry/apps/shorturl"
)

var WireSet = wire.NewSet(
	ProvideAppInstallers,
	ProvideBuilderRunners,
	playlist.RegisterAppInstaller,
	investigations.RegisterApp,
	advisor.RegisterApp,
	notifications.RegisterApp,
	plugins.RegisterAppInstaller,
	shorturl.RegisterAppInstaller,
)
