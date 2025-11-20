package investigations

import (
	"github.com/grafana/grafana-app-sdk/app"
	"github.com/grafana/grafana-app-sdk/simple"
	"github.com/capitalrx/grafana/apps/investigations/pkg/apis"
	investigationv0alpha1 "github.com/capitalrx/grafana/apps/investigations/pkg/apis/investigations/v0alpha1"
	investigationapp "github.com/capitalrx/grafana/apps/investigations/pkg/app"
	"github.com/capitalrx/grafana/pkg/services/apiserver/builder"
	"github.com/capitalrx/grafana/pkg/services/apiserver/builder/runner"
	"github.com/capitalrx/grafana/pkg/setting"
)

type InvestigationsAppProvider struct {
	app.Provider
	cfg *setting.Cfg
}

func RegisterApp(
	cfg *setting.Cfg,
) *InvestigationsAppProvider {
	provider := &InvestigationsAppProvider{
		cfg: cfg,
	}
	appCfg := &runner.AppBuilderConfig{
		OpenAPIDefGetter:         investigationv0alpha1.GetOpenAPIDefinitions,
		ManagedKinds:             investigationapp.GetKinds(),
		Authorizer:               GetAuthorizer(),
		AllowedV0Alpha1Resources: []string{builder.AllResourcesAllowed},
	}
	provider.Provider = simple.NewAppProvider(apis.LocalManifest(), appCfg, investigationapp.New)
	return provider
}
