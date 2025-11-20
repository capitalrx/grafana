package testutil

import (
	"github.com/capitalrx/grafana/pkg/api/routing"
	"github.com/capitalrx/grafana/pkg/bus"
	"github.com/capitalrx/grafana/pkg/infra/localcache"
	"github.com/capitalrx/grafana/pkg/infra/tracing"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/acimpl"
	acdb "github.com/capitalrx/grafana/pkg/services/accesscontrol/database"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/ossaccesscontrol"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/permreg"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/resourcepermissions"
	"github.com/capitalrx/grafana/pkg/services/apiserver"
	"github.com/capitalrx/grafana/pkg/services/dashboards/database"
	"github.com/capitalrx/grafana/pkg/services/featuremgmt"
	"github.com/capitalrx/grafana/pkg/services/folder/folderimpl"
	"github.com/capitalrx/grafana/pkg/services/licensing/licensingtest"
	"github.com/capitalrx/grafana/pkg/services/org/orgimpl"
	"github.com/capitalrx/grafana/pkg/services/quota/quotatest"
	"github.com/capitalrx/grafana/pkg/services/search/sort"
	"github.com/capitalrx/grafana/pkg/services/sqlstore"
	"github.com/capitalrx/grafana/pkg/services/supportbundles/bundleregistry"
	"github.com/capitalrx/grafana/pkg/services/supportbundles/supportbundlestest"
	"github.com/capitalrx/grafana/pkg/services/tag/tagimpl"
	"github.com/capitalrx/grafana/pkg/services/team/teamimpl"
	"github.com/capitalrx/grafana/pkg/services/user/userimpl"
	"github.com/capitalrx/grafana/pkg/setting"
	"github.com/capitalrx/grafana/pkg/storage/legacysql/dualwrite"
)

func ProvideFolderPermissions(
	features featuremgmt.FeatureToggles,
	cfg *setting.Cfg,
	sqlStore *sqlstore.SQLStore,
) (*ossaccesscontrol.FolderPermissionsService, error) {
	actionSets := resourcepermissions.NewActionSetService()

	license := licensingtest.NewFakeLicensing()
	license.On("FeatureEnabled", "accesscontrol.enforcement").Return(true).Maybe()

	ac := acimpl.ProvideAccessControl(featuremgmt.WithFeatures())

	quotaService := quotatest.New(false, nil)
	dashboardStore, err := database.ProvideDashboardStore(sqlStore, cfg, features, tagimpl.ProvideService(sqlStore))
	if err != nil {
		return nil, err
	}

	fStore := folderimpl.ProvideStore(sqlStore)
	fService := folderimpl.ProvideService(
		fStore, ac, bus.ProvideBus(tracing.InitializeTracerForTest()), dashboardStore,
		nil, sqlStore, features, supportbundlestest.NewFakeBundleService(), nil, cfg, nil, tracing.InitializeTracerForTest(), nil, dualwrite.ProvideTestService(), sort.ProvideService(), apiserver.WithoutRestConfig)

	acSvc := acimpl.ProvideOSSService(
		cfg, acdb.ProvideService(sqlStore), actionSets, localcache.ProvideService(),
		features, tracing.InitializeTracerForTest(), sqlStore, permreg.ProvidePermissionRegistry(),
		nil,
	)

	orgService, err := orgimpl.ProvideService(sqlStore, cfg, quotaService)
	if err != nil {
		return nil, err
	}
	teamSvc, err := teamimpl.ProvideService(sqlStore, cfg, tracing.InitializeTracerForTest())
	if err != nil {
		return nil, err
	}
	cache := localcache.ProvideService()

	userSvc, err := userimpl.ProvideService(
		sqlStore,
		orgService,
		cfg,
		teamSvc,
		cache,
		tracing.InitializeTracerForTest(),
		quotaService,
		bundleregistry.ProvideService(),
	)
	if err != nil {
		return nil, err
	}

	return ossaccesscontrol.ProvideFolderPermissions(
		cfg,
		features,
		routing.NewRouteRegister(),
		sqlStore,
		ac,
		license,
		fService,
		acSvc,
		teamSvc,
		userSvc,
		actionSets,
	)
}
