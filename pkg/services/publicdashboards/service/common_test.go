package service

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/infra/log"
	"github.com/capitalrx/grafana/pkg/infra/tracing"
	"github.com/capitalrx/grafana/pkg/services/annotations"
	"github.com/capitalrx/grafana/pkg/services/annotations/annotationsimpl"
	"github.com/capitalrx/grafana/pkg/services/dashboards"
	"github.com/capitalrx/grafana/pkg/services/featuremgmt"
	"github.com/capitalrx/grafana/pkg/services/licensing/licensingtest"
	"github.com/capitalrx/grafana/pkg/services/publicdashboards"
	"github.com/capitalrx/grafana/pkg/services/publicdashboards/database"
	. "github.com/capitalrx/grafana/pkg/services/publicdashboards/models"
	"github.com/capitalrx/grafana/pkg/services/publicdashboards/service/intervalv2"
	"github.com/capitalrx/grafana/pkg/services/sqlstore"
	"github.com/capitalrx/grafana/pkg/services/tag/tagimpl"
	"github.com/capitalrx/grafana/pkg/setting"
)

func newPublicDashboardServiceImpl(
	t *testing.T,
	store *sqlstore.SQLStore,
	cfg *setting.Cfg,
	publicDashboardStore publicdashboards.Store,
	dashboardService dashboards.DashboardService,
	annotationsRepo annotations.Repository,
) (*PublicDashboardServiceImpl, db.DB, *setting.Cfg) {
	t.Helper()

	if store == nil {
		store, cfg = db.InitTestDBWithCfg(t)
	}
	tagService := tagimpl.ProvideService(store)
	if annotationsRepo == nil {
		annotationsRepo = annotationsimpl.ProvideService(store, cfg, featuremgmt.WithFeatures(), tagService, tracing.InitializeTracerForTest(), nil, dashboardService, prometheus.NewPedanticRegistry())
	}

	if publicDashboardStore == nil {
		publicDashboardStore = database.ProvideStore(store, cfg, featuremgmt.WithFeatures())
	}
	serviceWrapper := ProvideServiceWrapper(publicDashboardStore)

	license := licensingtest.NewFakeLicensing()
	license.On("FeatureEnabled", FeaturePublicDashboardsEmailSharing).Return(false)

	return &PublicDashboardServiceImpl{
		AnnotationsRepo:    annotationsRepo,
		log:                log.New("test.logger"),
		intervalCalculator: intervalv2.NewCalculator(),
		dashboardService:   dashboardService,
		store:              publicDashboardStore,
		serviceWrapper:     serviceWrapper,
		license:            license,
		features:           featuremgmt.WithFeatures(),
	}, store, cfg
}
