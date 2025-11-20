package correlationstest

import (
	"github.com/capitalrx/grafana/pkg/api/routing"
	"github.com/capitalrx/grafana/pkg/bus"
	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/acimpl"
	"github.com/capitalrx/grafana/pkg/services/correlations"
	"github.com/capitalrx/grafana/pkg/services/datasources"
	fakeDatasources "github.com/capitalrx/grafana/pkg/services/datasources/fakes"
	"github.com/capitalrx/grafana/pkg/services/featuremgmt"
	"github.com/capitalrx/grafana/pkg/services/quota/quotatest"
	"github.com/capitalrx/grafana/pkg/setting"
)

func New(db db.DB, cfg *setting.Cfg, bus bus.Bus) *correlations.CorrelationsService {
	ds := &fakeDatasources.FakeDataSourceService{
		DataSources: []*datasources.DataSource{
			{ID: 1, UID: "graphite", Type: datasources.DS_GRAPHITE},
		},
	}

	correlationsSvc, _ := correlations.ProvideService(db, routing.NewRouteRegister(), ds, acimpl.ProvideAccessControl(featuremgmt.WithFeatures()), bus, quotatest.New(false, nil), cfg)
	return correlationsSvc
}
