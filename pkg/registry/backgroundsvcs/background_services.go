package backgroundsvcs

import (
	"github.com/capitalrx/grafana/pkg/api"
	"github.com/capitalrx/grafana/pkg/infra/metrics"
	"github.com/capitalrx/grafana/pkg/infra/remotecache"
	"github.com/capitalrx/grafana/pkg/infra/tracing"
	uss "github.com/capitalrx/grafana/pkg/infra/usagestats/service"
	"github.com/capitalrx/grafana/pkg/infra/usagestats/statscollector"
	"github.com/capitalrx/grafana/pkg/registry"
	apiregistry "github.com/capitalrx/grafana/pkg/registry/apis"
	secretsgarbagecollectionworker "github.com/capitalrx/grafana/pkg/registry/apis/secret/garbagecollectionworker"
	appregistry "github.com/capitalrx/grafana/pkg/registry/apps"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/dualwrite"
	"github.com/capitalrx/grafana/pkg/services/anonymous/anonimpl"
	grafanaapiserver "github.com/capitalrx/grafana/pkg/services/apiserver"
	"github.com/capitalrx/grafana/pkg/services/auth"
	"github.com/capitalrx/grafana/pkg/services/authn/authnimpl"
	"github.com/capitalrx/grafana/pkg/services/cleanup"
	"github.com/capitalrx/grafana/pkg/services/cloudmigration"
	"github.com/capitalrx/grafana/pkg/services/dashboards/service"
	"github.com/capitalrx/grafana/pkg/services/dashboardsnapshots"
	"github.com/capitalrx/grafana/pkg/services/grpcserver"
	ldapapi "github.com/capitalrx/grafana/pkg/services/ldap/api"
	"github.com/capitalrx/grafana/pkg/services/live"
	"github.com/capitalrx/grafana/pkg/services/live/pushhttp"
	"github.com/capitalrx/grafana/pkg/services/loginattempt/loginattemptimpl"
	"github.com/capitalrx/grafana/pkg/services/ngalert"
	"github.com/capitalrx/grafana/pkg/services/notifications"
	plugindashboardsservice "github.com/capitalrx/grafana/pkg/services/plugindashboards/service"
	"github.com/capitalrx/grafana/pkg/services/pluginsintegration/angulardetectorsprovider"
	"github.com/capitalrx/grafana/pkg/services/pluginsintegration/keyretriever/dynamic"
	"github.com/capitalrx/grafana/pkg/services/pluginsintegration/pluginexternal"
	"github.com/capitalrx/grafana/pkg/services/pluginsintegration/plugininstaller"
	pluginStore "github.com/capitalrx/grafana/pkg/services/pluginsintegration/pluginstore"
	"github.com/capitalrx/grafana/pkg/services/provisioning"
	publicdashboardsmetric "github.com/capitalrx/grafana/pkg/services/publicdashboards/metric"
	"github.com/capitalrx/grafana/pkg/services/rendering"
	"github.com/capitalrx/grafana/pkg/services/searchV2"
	secretsMigrations "github.com/capitalrx/grafana/pkg/services/secrets/kvstore/migrations"
	secretsManager "github.com/capitalrx/grafana/pkg/services/secrets/manager"
	"github.com/capitalrx/grafana/pkg/services/serviceaccounts"
	samanager "github.com/capitalrx/grafana/pkg/services/serviceaccounts/manager"
	"github.com/capitalrx/grafana/pkg/services/ssosettings"
	"github.com/capitalrx/grafana/pkg/services/ssosettings/ssosettingsimpl"
	"github.com/capitalrx/grafana/pkg/services/store"
	"github.com/capitalrx/grafana/pkg/services/supportbundles/supportbundlesimpl"
	"github.com/capitalrx/grafana/pkg/services/team/teamapi"
	"github.com/capitalrx/grafana/pkg/services/updatemanager"
)

func ProvideBackgroundServiceRegistry(
	httpServer *api.HTTPServer, ng *ngalert.AlertNG, cleanup *cleanup.CleanUpService, live *live.GrafanaLive,
	pushGateway *pushhttp.Gateway, notifications *notifications.NotificationService, pluginStore *pluginStore.Service,
	rendering *rendering.RenderingService, tokenService auth.UserTokenBackgroundService, tracing *tracing.TracingService,
	provisioning *provisioning.ProvisioningServiceImpl, usageStats *uss.UsageStats,
	statsCollector *statscollector.Service, grafanaUpdateChecker *updatemanager.GrafanaService,
	pluginsUpdateChecker *updatemanager.PluginsService, metrics *metrics.InternalMetricsService,
	secretsService *secretsManager.SecretsService, remoteCache *remotecache.RemoteCache, StorageService store.StorageService, searchService searchV2.SearchService, entityEventsService store.EntityEventsService,
	saService *samanager.ServiceAccountsService, grpcServerProvider grpcserver.Provider,
	secretMigrationProvider secretsMigrations.SecretMigrationProvider, loginAttemptService *loginattemptimpl.Service,
	bundleService *supportbundlesimpl.Service, publicDashboardsMetric *publicdashboardsmetric.Service,
	keyRetriever *dynamic.KeyRetriever, dynamicAngularDetectorsProvider *angulardetectorsprovider.Dynamic,
	grafanaAPIServer grafanaapiserver.Service,
	anon *anonimpl.AnonDeviceService,
	ssoSettings *ssosettingsimpl.Service,
	pluginExternal *pluginexternal.Service,
	pluginInstaller *plugininstaller.Service,
	zanzanaReconciler *dualwrite.ZanzanaReconciler,
	appRegistry *appregistry.Service,
	pluginDashboardUpdater *plugindashboardsservice.DashboardUpdater,
	dashboardServiceImpl *service.DashboardServiceImpl,
	secretsGarbageCollectionWorker *secretsgarbagecollectionworker.Worker,
	// Need to make sure these are initialized, is there a better place to put them?
	_ dashboardsnapshots.Service,
	_ serviceaccounts.Service,
	_ *grpcserver.HealthService, _ *grpcserver.ReflectionService,
	_ *ldapapi.Service, _ *apiregistry.Service, _ auth.IDService, _ *teamapi.TeamAPI, _ ssosettings.Service,
	_ cloudmigration.Service, _ authnimpl.Registration,
) *BackgroundServiceRegistry {
	return NewBackgroundServiceRegistry(
		httpServer,
		ng,
		cleanup,
		live,
		pushGateway,
		notifications,
		rendering,
		tokenService,
		provisioning,
		grafanaUpdateChecker,
		pluginsUpdateChecker,
		metrics,
		usageStats,
		statsCollector,
		tracing,
		remoteCache,
		secretsService,
		StorageService,
		searchService,
		entityEventsService,
		grpcServerProvider,
		saService,
		pluginStore,
		secretMigrationProvider,
		loginAttemptService,
		bundleService,
		publicDashboardsMetric,
		keyRetriever,
		dynamicAngularDetectorsProvider,
		grafanaAPIServer,
		anon,
		ssoSettings,
		pluginExternal,
		pluginInstaller,
		zanzanaReconciler,
		appRegistry,
		pluginDashboardUpdater,
		dashboardServiceImpl,
		secretsGarbageCollectionWorker,
	)
}

// BackgroundServiceRegistry provides background services.
type BackgroundServiceRegistry struct {
	Services []registry.BackgroundService
}

func NewBackgroundServiceRegistry(services ...registry.BackgroundService) *BackgroundServiceRegistry {
	return &BackgroundServiceRegistry{services}
}

func (r *BackgroundServiceRegistry) GetServices() []registry.BackgroundService {
	return r.Services
}
