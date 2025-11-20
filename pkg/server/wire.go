//go:build wireinject
// +build wireinject

// This file should contain wire sets used by both OSS and Enterprise builds.
// Use wireext_oss.go and wireext_enterprise.go for sets that are specific to
// the respective builds.
package server

import (
	"context"

	"github.com/google/wire"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	sdkhttpclient "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/capitalrx/grafana/apps/provisioning/pkg/repository/github"
	"github.com/capitalrx/grafana/pkg/api"
	"github.com/capitalrx/grafana/pkg/api/avatar"
	"github.com/capitalrx/grafana/pkg/api/routing"
	"github.com/capitalrx/grafana/pkg/bus"
	"github.com/capitalrx/grafana/pkg/expr"
	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/infra/httpclient"
	"github.com/capitalrx/grafana/pkg/infra/httpclient/httpclientprovider"
	"github.com/capitalrx/grafana/pkg/infra/kvstore"
	"github.com/capitalrx/grafana/pkg/infra/localcache"
	"github.com/capitalrx/grafana/pkg/infra/log/slogadapter"
	"github.com/capitalrx/grafana/pkg/infra/metrics"
	"github.com/capitalrx/grafana/pkg/infra/remotecache"
	"github.com/capitalrx/grafana/pkg/infra/serverlock"
	"github.com/capitalrx/grafana/pkg/infra/tracing"
	"github.com/capitalrx/grafana/pkg/infra/usagestats"
	uss "github.com/capitalrx/grafana/pkg/infra/usagestats/service"
	"github.com/capitalrx/grafana/pkg/infra/usagestats/statscollector"
	"github.com/capitalrx/grafana/pkg/infra/usagestats/validator"
	"github.com/capitalrx/grafana/pkg/login/social"
	"github.com/capitalrx/grafana/pkg/login/social/connectors"
	"github.com/capitalrx/grafana/pkg/login/social/socialimpl"
	"github.com/capitalrx/grafana/pkg/middleware/csrf"
	"github.com/capitalrx/grafana/pkg/middleware/loggermw"
	apiregistry "github.com/capitalrx/grafana/pkg/registry/apis"
	"github.com/capitalrx/grafana/pkg/registry/apis/dashboard/legacy"
	secretclock "github.com/capitalrx/grafana/pkg/registry/apis/secret/clock"
	secretcontracts "github.com/capitalrx/grafana/pkg/registry/apis/secret/contracts"
	secretdecrypt "github.com/capitalrx/grafana/pkg/registry/apis/secret/decrypt"
	cipher "github.com/capitalrx/grafana/pkg/registry/apis/secret/encryption/cipher/service"
	encryptionManager "github.com/capitalrx/grafana/pkg/registry/apis/secret/encryption/manager"
	secretsgarbagecollectionworker "github.com/capitalrx/grafana/pkg/registry/apis/secret/garbagecollectionworker"
	secretinline "github.com/capitalrx/grafana/pkg/registry/apis/secret/inline"
	secretmutator "github.com/capitalrx/grafana/pkg/registry/apis/secret/mutator"
	secretsecurevalueservice "github.com/capitalrx/grafana/pkg/registry/apis/secret/service"
	secretvalidator "github.com/capitalrx/grafana/pkg/registry/apis/secret/validator"
	appregistry "github.com/capitalrx/grafana/pkg/registry/apps"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/acimpl"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/dualwrite"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/ossaccesscontrol"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/permreg"
	"github.com/capitalrx/grafana/pkg/services/accesscontrol/resourcepermissions"
	"github.com/capitalrx/grafana/pkg/services/annotations"
	"github.com/capitalrx/grafana/pkg/services/annotations/annotationsimpl"
	"github.com/capitalrx/grafana/pkg/services/anonymous/anonimpl/anonstore"
	"github.com/capitalrx/grafana/pkg/services/apikey/apikeyimpl"
	grafanaapiserver "github.com/capitalrx/grafana/pkg/services/apiserver"
	"github.com/capitalrx/grafana/pkg/services/apiserver/standalone"
	"github.com/capitalrx/grafana/pkg/services/auth"
	"github.com/capitalrx/grafana/pkg/services/auth/idimpl"
	"github.com/capitalrx/grafana/pkg/services/auth/jwt"
	"github.com/capitalrx/grafana/pkg/services/authn/authnimpl"
	"github.com/capitalrx/grafana/pkg/services/authz"
	"github.com/capitalrx/grafana/pkg/services/cleanup"
	"github.com/capitalrx/grafana/pkg/services/cloudmigration/cloudmigrationimpl"
	"github.com/capitalrx/grafana/pkg/services/contexthandler"
	"github.com/capitalrx/grafana/pkg/services/correlations"
	"github.com/capitalrx/grafana/pkg/services/dashboardimport"
	dashboardimportservice "github.com/capitalrx/grafana/pkg/services/dashboardimport/service"
	"github.com/capitalrx/grafana/pkg/services/dashboards"
	dashboardstore "github.com/capitalrx/grafana/pkg/services/dashboards/database"
	dashboardservice "github.com/capitalrx/grafana/pkg/services/dashboards/service"
	dashboardclient "github.com/capitalrx/grafana/pkg/services/dashboards/service/client"
	"github.com/capitalrx/grafana/pkg/services/dashboardsnapshots"
	dashsnapstore "github.com/capitalrx/grafana/pkg/services/dashboardsnapshots/database"
	dashsnapsvc "github.com/capitalrx/grafana/pkg/services/dashboardsnapshots/service"
	"github.com/capitalrx/grafana/pkg/services/dashboardversion/dashverimpl"
	"github.com/capitalrx/grafana/pkg/services/datasourceproxy"
	"github.com/capitalrx/grafana/pkg/services/datasources"
	datasourceservice "github.com/capitalrx/grafana/pkg/services/datasources/service"
	"github.com/capitalrx/grafana/pkg/services/dsquerierclient"
	"github.com/capitalrx/grafana/pkg/services/encryption"
	encryptionservice "github.com/capitalrx/grafana/pkg/services/encryption/service"
	"github.com/capitalrx/grafana/pkg/services/extsvcauth"
	extsvcreg "github.com/capitalrx/grafana/pkg/services/extsvcauth/registry"
	"github.com/capitalrx/grafana/pkg/services/featuremgmt"
	"github.com/capitalrx/grafana/pkg/services/folder"
	"github.com/capitalrx/grafana/pkg/services/folder/folderimpl"
	"github.com/capitalrx/grafana/pkg/services/grpcserver"
	grpccontext "github.com/capitalrx/grafana/pkg/services/grpcserver/context"
	"github.com/capitalrx/grafana/pkg/services/grpcserver/interceptors"
	"github.com/capitalrx/grafana/pkg/services/hooks"
	ldapapi "github.com/capitalrx/grafana/pkg/services/ldap/api"
	ldapservice "github.com/capitalrx/grafana/pkg/services/ldap/service"
	"github.com/capitalrx/grafana/pkg/services/libraryelements"
	"github.com/capitalrx/grafana/pkg/services/librarypanels"
	"github.com/capitalrx/grafana/pkg/services/live"
	"github.com/capitalrx/grafana/pkg/services/live/pushhttp"
	"github.com/capitalrx/grafana/pkg/services/login"
	"github.com/capitalrx/grafana/pkg/services/login/authinfoimpl"
	"github.com/capitalrx/grafana/pkg/services/loginattempt"
	"github.com/capitalrx/grafana/pkg/services/loginattempt/loginattemptimpl"
	"github.com/capitalrx/grafana/pkg/services/navtree/navtreeimpl"
	"github.com/capitalrx/grafana/pkg/services/ngalert"
	ngimage "github.com/capitalrx/grafana/pkg/services/ngalert/image"
	ngmetrics "github.com/capitalrx/grafana/pkg/services/ngalert/metrics"
	ngstore "github.com/capitalrx/grafana/pkg/services/ngalert/store"
	"github.com/capitalrx/grafana/pkg/services/notifications"
	"github.com/capitalrx/grafana/pkg/services/oauthtoken"
	"github.com/capitalrx/grafana/pkg/services/oauthtoken/oauthtokentest"
	"github.com/capitalrx/grafana/pkg/services/org/orgimpl"
	"github.com/capitalrx/grafana/pkg/services/playlist/playlistimpl"
	"github.com/capitalrx/grafana/pkg/services/plugindashboards"
	plugindashboardsservice "github.com/capitalrx/grafana/pkg/services/plugindashboards/service"
	"github.com/capitalrx/grafana/pkg/services/pluginsintegration"
	pluginDashboards "github.com/capitalrx/grafana/pkg/services/pluginsintegration/dashboards"
	"github.com/capitalrx/grafana/pkg/services/pluginsintegration/pluginaccesscontrol"
	"github.com/capitalrx/grafana/pkg/services/preference/prefimpl"
	promTypeMigration "github.com/capitalrx/grafana/pkg/services/promtypemigration"
	"github.com/capitalrx/grafana/pkg/services/publicdashboards"
	publicdashboardsApi "github.com/capitalrx/grafana/pkg/services/publicdashboards/api"
	publicdashboardsStore "github.com/capitalrx/grafana/pkg/services/publicdashboards/database"
	publicdashboardsmetric "github.com/capitalrx/grafana/pkg/services/publicdashboards/metric"
	publicdashboardsService "github.com/capitalrx/grafana/pkg/services/publicdashboards/service"
	"github.com/capitalrx/grafana/pkg/services/query"
	"github.com/capitalrx/grafana/pkg/services/queryhistory"
	"github.com/capitalrx/grafana/pkg/services/quota/quotaimpl"
	"github.com/capitalrx/grafana/pkg/services/rendering"
	"github.com/capitalrx/grafana/pkg/services/search"
	"github.com/capitalrx/grafana/pkg/services/search/sort"
	"github.com/capitalrx/grafana/pkg/services/searchV2"
	"github.com/capitalrx/grafana/pkg/services/secrets"
	secretsDatabase "github.com/capitalrx/grafana/pkg/services/secrets/database"
	secretsStore "github.com/capitalrx/grafana/pkg/services/secrets/kvstore"
	secretsMigrations "github.com/capitalrx/grafana/pkg/services/secrets/kvstore/migrations"
	secretsManager "github.com/capitalrx/grafana/pkg/services/secrets/manager"
	"github.com/capitalrx/grafana/pkg/services/serviceaccounts"
	"github.com/capitalrx/grafana/pkg/services/serviceaccounts/extsvcaccounts"
	serviceaccountsmanager "github.com/capitalrx/grafana/pkg/services/serviceaccounts/manager"
	serviceaccountsproxy "github.com/capitalrx/grafana/pkg/services/serviceaccounts/proxy"
	serviceaccountsretriever "github.com/capitalrx/grafana/pkg/services/serviceaccounts/retriever"
	"github.com/capitalrx/grafana/pkg/services/shorturls"
	"github.com/capitalrx/grafana/pkg/services/shorturls/shorturlimpl"
	"github.com/capitalrx/grafana/pkg/services/signingkeys"
	"github.com/capitalrx/grafana/pkg/services/signingkeys/signingkeysimpl"
	"github.com/capitalrx/grafana/pkg/services/sqlstore"
	"github.com/capitalrx/grafana/pkg/services/sqlstore/sqlutil"
	"github.com/capitalrx/grafana/pkg/services/ssosettings"
	ssoSettingsImpl "github.com/capitalrx/grafana/pkg/services/ssosettings/ssosettingsimpl"
	starApi "github.com/capitalrx/grafana/pkg/services/star/api"
	"github.com/capitalrx/grafana/pkg/services/star/starimpl"
	"github.com/capitalrx/grafana/pkg/services/stats/statsimpl"
	"github.com/capitalrx/grafana/pkg/services/store"
	"github.com/capitalrx/grafana/pkg/services/store/resolver"
	"github.com/capitalrx/grafana/pkg/services/supportbundles"
	"github.com/capitalrx/grafana/pkg/services/supportbundles/bundleregistry"
	"github.com/capitalrx/grafana/pkg/services/supportbundles/supportbundlesimpl"
	"github.com/capitalrx/grafana/pkg/services/tag"
	"github.com/capitalrx/grafana/pkg/services/tag/tagimpl"
	"github.com/capitalrx/grafana/pkg/services/team/teamapi"
	"github.com/capitalrx/grafana/pkg/services/team/teamimpl"
	tempuser "github.com/capitalrx/grafana/pkg/services/temp_user"
	"github.com/capitalrx/grafana/pkg/services/temp_user/tempuserimpl"
	"github.com/capitalrx/grafana/pkg/services/updatemanager"
	"github.com/capitalrx/grafana/pkg/services/user"
	"github.com/capitalrx/grafana/pkg/services/user/userimpl"
	"github.com/capitalrx/grafana/pkg/setting"
	legacydualwrite "github.com/capitalrx/grafana/pkg/storage/legacysql/dualwrite"
	secretdatabase "github.com/capitalrx/grafana/pkg/storage/secret/database"
	secretencryption "github.com/capitalrx/grafana/pkg/storage/secret/encryption"
	secretmetadata "github.com/capitalrx/grafana/pkg/storage/secret/metadata"
	secretmigrator "github.com/capitalrx/grafana/pkg/storage/secret/migrator"
	"github.com/capitalrx/grafana/pkg/storage/unified/resource"
	unifiedsearch "github.com/capitalrx/grafana/pkg/storage/unified/search"
	"github.com/capitalrx/grafana/pkg/tsdb/azuremonitor"
	cloudmonitoring "github.com/capitalrx/grafana/pkg/tsdb/cloud-monitoring"
	"github.com/capitalrx/grafana/pkg/tsdb/cloudwatch"
	"github.com/capitalrx/grafana/pkg/tsdb/elasticsearch"
	postgres "github.com/capitalrx/grafana/pkg/tsdb/grafana-postgresql-datasource"
	pyroscope "github.com/capitalrx/grafana/pkg/tsdb/grafana-pyroscope-datasource"
	testdatasource "github.com/capitalrx/grafana/pkg/tsdb/grafana-testdata-datasource"
	"github.com/capitalrx/grafana/pkg/tsdb/grafanads"
	"github.com/capitalrx/grafana/pkg/tsdb/graphite"
	"github.com/capitalrx/grafana/pkg/tsdb/influxdb"
	"github.com/capitalrx/grafana/pkg/tsdb/jaeger"
	"github.com/capitalrx/grafana/pkg/tsdb/loki"
	"github.com/capitalrx/grafana/pkg/tsdb/mssql"
	"github.com/capitalrx/grafana/pkg/tsdb/mysql"
	"github.com/capitalrx/grafana/pkg/tsdb/opentsdb"
	"github.com/capitalrx/grafana/pkg/tsdb/parca"
	"github.com/capitalrx/grafana/pkg/tsdb/prometheus"
	"github.com/capitalrx/grafana/pkg/tsdb/tempo"
	"github.com/capitalrx/grafana/pkg/tsdb/zipkin"
)

func otelTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer("grafana")
}

var withOTelSet = wire.NewSet(
	otelTracer,
	grpcserver.ProvideService,
	interceptors.ProvideAuthenticator,
)

var wireBasicSet = wire.NewSet(
	annotationsimpl.ProvideService,
	wire.Bind(new(annotations.Repository), new(*annotationsimpl.RepositoryImpl)),
	New,
	api.ProvideHTTPServer,
	query.ProvideService,
	wire.Bind(new(query.Service), new(*query.ServiceImpl)),
	bus.ProvideBus,
	wire.Bind(new(bus.Bus), new(*bus.InProcBus)),
	rendering.ProvideService,
	wire.Bind(new(rendering.Service), new(*rendering.RenderingService)),
	routing.ProvideRegister,
	wire.Bind(new(routing.RouteRegister), new(*routing.RouteRegisterImpl)),
	hooks.ProvideService,
	kvstore.ProvideService,
	localcache.ProvideService,
	bundleregistry.ProvideService,
	wire.Bind(new(supportbundles.Service), new(*bundleregistry.Service)),
	updatemanager.ProvideGrafanaService,
	updatemanager.ProvidePluginsService,
	uss.ProvideService,
	wire.Bind(new(usagestats.Service), new(*uss.UsageStats)),
	validator.ProvideService,
	legacy.ProvideLegacyMigrator,
	pluginsintegration.WireSet,
	pluginDashboards.ProvideFileStoreManager,
	wire.Bind(new(pluginDashboards.FileStore), new(*pluginDashboards.FileStoreManager)),
	cloudwatch.ProvideService,
	cloudmonitoring.ProvideService,
	azuremonitor.ProvideService,
	postgres.ProvideService,
	mysql.ProvideService,
	mssql.ProvideService,
	store.ProvideEntityEventsService,
	legacydualwrite.ProvideService,
	httpclientprovider.New,
	wire.Bind(new(httpclient.Provider), new(*sdkhttpclient.Provider)),
	serverlock.ProvideService,
	annotationsimpl.ProvideCleanupService,
	wire.Bind(new(annotations.Cleaner), new(*annotationsimpl.CleanupServiceImpl)),
	cleanup.ProvideService,
	shorturlimpl.ProvideService,
	wire.Bind(new(shorturls.Service), new(*shorturlimpl.ShortURLService)),
	queryhistory.ProvideService,
	wire.Bind(new(queryhistory.Service), new(*queryhistory.QueryHistoryService)),
	correlations.ProvideService,
	wire.Bind(new(correlations.Service), new(*correlations.CorrelationsService)),
	quotaimpl.ProvideService,
	remotecache.ProvideService,
	wire.Bind(new(remotecache.CacheStorage), new(*remotecache.RemoteCache)),
	authinfoimpl.ProvideService,
	wire.Bind(new(login.AuthInfoService), new(*authinfoimpl.Service)),
	authinfoimpl.ProvideStore,
	datasourceproxy.ProvideService,
	sort.ProvideService,
	search.ProvideService,
	searchV2.ProvideService,
	searchV2.ProvideSearchHTTPService,
	store.ProvideService,
	store.ProvideSystemUsersService,
	live.ProvideService,
	pushhttp.ProvideService,
	contexthandler.ProvideService,
	ldapservice.ProvideService,
	wire.Bind(new(ldapservice.LDAP), new(*ldapservice.LDAPImpl)),
	jwt.ProvideService,
	wire.Bind(new(jwt.JWTService), new(*jwt.AuthService)),
	ngstore.ProvideDBStore,
	ngimage.ProvideDeleteExpiredService,
	ngalert.ProvideService,
	librarypanels.ProvideService,
	wire.Bind(new(librarypanels.Service), new(*librarypanels.LibraryPanelService)),
	libraryelements.ProvideService,
	wire.Bind(new(libraryelements.Service), new(*libraryelements.LibraryElementService)),
	notifications.ProvideService,
	notifications.ProvideSmtpService,
	github.ProvideFactory,
	tracing.ProvideService,
	tracing.ProvideTracingConfig,
	wire.Bind(new(tracing.Tracer), new(*tracing.TracingService)),
	withOTelSet,
	testdatasource.ProvideService,
	ldapapi.ProvideService,
	opentsdb.ProvideService,
	socialimpl.ProvideService,
	influxdb.ProvideService,
	wire.Bind(new(social.Service), new(*socialimpl.SocialService)),
	tempo.ProvideService,
	loki.ProvideService,
	graphite.ProvideService,
	prometheus.ProvideService,
	elasticsearch.ProvideService,
	pyroscope.ProvideService,
	parca.ProvideService,
	zipkin.ProvideService,
	jaeger.ProvideService,
	datasourceservice.ProvideCacheService,
	wire.Bind(new(datasources.CacheService), new(*datasourceservice.CacheServiceImpl)),
	encryptionservice.ProvideEncryptionService,
	wire.Bind(new(encryption.Internal), new(*encryptionservice.Service)),
	secretsManager.ProvideSecretsService,
	wire.Bind(new(secrets.Service), new(*secretsManager.SecretsService)),
	secretsDatabase.ProvideSecretsStore,
	wire.Bind(new(secrets.Store), new(*secretsDatabase.SecretsStoreImpl)),
	secretsgarbagecollectionworker.ProvideWorker,
	grafanads.ProvideService,
	wire.Bind(new(dashboardsnapshots.Store), new(*dashsnapstore.DashboardSnapshotStore)),
	dashsnapstore.ProvideStore,
	wire.Bind(new(dashboardsnapshots.Service), new(*dashsnapsvc.ServiceImpl)),
	dashsnapsvc.ProvideService,
	datasourceservice.ProvideService,
	wire.Bind(new(datasources.DataSourceService), new(*datasourceservice.Service)),
	datasourceservice.ProvideLegacyDataSourceLookup,
	serviceaccountsretriever.ProvideService,
	wire.Bind(new(serviceaccounts.ServiceAccountRetriever), new(*serviceaccountsretriever.Service)),
	ossaccesscontrol.ProvideServiceAccountPermissions,
	wire.Bind(new(accesscontrol.ServiceAccountPermissionsService), new(*ossaccesscontrol.ServiceAccountPermissionsService)),
	serviceaccountsmanager.ProvideServiceAccountsService,
	serviceaccountsproxy.ProvideServiceAccountsProxy,
	wire.Bind(new(serviceaccounts.Service), new(*serviceaccountsproxy.ServiceAccountsProxy)),
	dsquerierclient.NewNullQSDatasourceClientBuilder,
	expr.ProvideService,
	featuremgmt.ProvideManagerService,
	featuremgmt.ProvideToggles,
	dashboardservice.ProvideDashboardServiceImpl,
	wire.Bind(new(dashboards.PermissionsRegistrationService), new(*dashboardservice.DashboardServiceImpl)),
	dashboardservice.ProvideDashboardService,
	dashboardservice.ProvideDashboardProvisioningService,
	dashboardservice.ProvideDashboardPluginService,
	dashboardstore.ProvideDashboardStore,
	folderimpl.ProvideService,
	wire.Bind(new(folder.Service), new(*folderimpl.Service)),
	wire.Bind(new(folder.LegacyService), new(*folderimpl.Service)),
	folderimpl.ProvideStore,
	wire.Bind(new(folder.Store), new(*folderimpl.FolderStoreImpl)),
	dashboardimportservice.ProvideService,
	wire.Bind(new(dashboardimport.Service), new(*dashboardimportservice.ImportDashboardService)),
	plugindashboardsservice.ProvideService,
	wire.Bind(new(plugindashboards.Service), new(*plugindashboardsservice.Service)),
	plugindashboardsservice.ProvideDashboardUpdater,
	secretsStore.ProvideService,
	avatar.ProvideAvatarCacheServer,
	statscollector.ProvideService,
	csrf.ProvideCSRFFilter,
	wire.Bind(new(csrf.Service), new(*csrf.CSRF)),
	ossaccesscontrol.ProvideTeamPermissions,
	wire.Bind(new(accesscontrol.TeamPermissionsService), new(*ossaccesscontrol.TeamPermissionsService)),
	ossaccesscontrol.ProvideFolderPermissions,
	wire.Bind(new(accesscontrol.FolderPermissionsService), new(*ossaccesscontrol.FolderPermissionsService)),
	ossaccesscontrol.ProvideDashboardPermissions,
	wire.Bind(new(accesscontrol.DashboardPermissionsService), new(*ossaccesscontrol.DashboardPermissionsService)),
	ossaccesscontrol.ProvideReceiverPermissionsService,
	wire.Bind(new(accesscontrol.ReceiverPermissionsService), new(*ossaccesscontrol.ReceiverPermissionsService)),
	starimpl.ProvideService,
	playlistimpl.ProvideService,
	apikeyimpl.ProvideService,
	dashverimpl.ProvideService,
	publicdashboardsService.ProvideService,
	wire.Bind(new(publicdashboards.Service), new(*publicdashboardsService.PublicDashboardServiceImpl)),
	publicdashboardsStore.ProvideStore,
	wire.Bind(new(publicdashboards.Store), new(*publicdashboardsStore.PublicDashboardStoreImpl)),
	publicdashboardsmetric.ProvideService,
	publicdashboardsApi.ProvideApi,
	starApi.ProvideApi,
	userimpl.ProvideService,
	orgimpl.ProvideService,
	orgimpl.ProvideDeletionService,
	statsimpl.ProvideService,
	grpccontext.ProvideContextHandler,
	grpcserver.ProvideHealthService,
	grpcserver.ProvideReflectionService,
	resolver.ProvideEntityReferenceResolver,
	teamimpl.ProvideService,
	teamapi.ProvideTeamAPI,
	tempuserimpl.ProvideService,
	loginattemptimpl.ProvideService,
	wire.Bind(new(loginattempt.Service), new(*loginattemptimpl.Service)),
	secretsMigrations.ProvideDataSourceMigrationService,
	secretsMigrations.ProvideSecretMigrationProvider,
	wire.Bind(new(secretsMigrations.SecretMigrationProvider), new(*secretsMigrations.SecretMigrationProviderImpl)),
	promTypeMigration.ProvideAzurePromMigrationService,
	promTypeMigration.ProvideAmazonPromMigrationService,
	promTypeMigration.ProvidePromTypeMigrationProvider,
	wire.Bind(new(promTypeMigration.PromTypeMigrationProvider), new(*promTypeMigration.PromTypeMigrationProviderImpl)),
	resourcepermissions.NewActionSetService,
	wire.Bind(new(accesscontrol.ActionResolver), new(resourcepermissions.ActionSetService)),
	wire.Bind(new(pluginaccesscontrol.ActionSetRegistry), new(resourcepermissions.ActionSetService)),
	permreg.ProvidePermissionRegistry,
	acimpl.ProvideAccessControl,
	dualwrite.ProvideZanzanaReconciler,
	navtreeimpl.ProvideService,
	wire.Bind(new(accesscontrol.AccessControl), new(*acimpl.AccessControl)),
	wire.Bind(new(notifications.TempUserStore), new(tempuser.Service)),
	tagimpl.ProvideService,
	wire.Bind(new(tag.Service), new(*tagimpl.Service)),
	authnimpl.ProvideService,
	authnimpl.ProvideIdentitySynchronizer,
	authnimpl.ProvideAuthnService,
	authnimpl.ProvideAuthnServiceAuthenticateOnly,
	authnimpl.ProvideRegistration,
	supportbundlesimpl.ProvideService,
	extsvcaccounts.ProvideExtSvcAccountsService,
	wire.Bind(new(serviceaccounts.ExtSvcAccountsService), new(*extsvcaccounts.ExtSvcAccountsService)),
	extsvcreg.ProvideExtSvcRegistry,
	wire.Bind(new(extsvcauth.ExternalServiceRegistry), new(*extsvcreg.Registry)),
	anonstore.ProvideAnonDBStore,
	wire.Bind(new(anonstore.AnonStore), new(*anonstore.AnonDBStore)),
	loggermw.Provide,
	slogadapter.Provide,
	signingkeysimpl.ProvideEmbeddedSigningKeysService,
	wire.Bind(new(signingkeys.Service), new(*signingkeysimpl.Service)),
	ssoSettingsImpl.ProvideService,
	wire.Bind(new(ssosettings.Service), new(*ssoSettingsImpl.Service)),
	idimpl.ProvideService,
	wire.Bind(new(auth.IDService), new(*idimpl.Service)),
	cloudmigrationimpl.ProvideService,
	userimpl.ProvideVerifier,
	connectors.ProvideOrgRoleMapper,
	wire.Bind(new(user.Verifier), new(*userimpl.Verifier)),
	authz.WireSet,
	// Secrets Manager
	secretmetadata.ProvideSecureValueMetadataStorage,
	secretmetadata.ProvideKeeperMetadataStorage,
	secretmetadata.ProvideDecryptStorage,
	secretdecrypt.ProvideDecryptAuthorizer,
	secretdecrypt.ProvideDecryptService,
	secretinline.ProvideInlineSecureValueService,
	secretencryption.ProvideDataKeyStorage,
	secretencryption.ProvideGlobalDataKeyStorage,
	secretencryption.ProvideEncryptedValueStorage,
	secretencryption.ProvideGlobalEncryptedValueStorage,
	secretsecurevalueservice.ProvideSecureValueService,
	secretvalidator.ProvideKeeperValidator,
	secretvalidator.ProvideSecureValueValidator,
	secretmutator.ProvideKeeperMutator,
	secretmutator.ProvideSecureValueMutator,
	secretmigrator.NewWithEngine,
	secretdatabase.ProvideDatabase,
	secretclock.ProvideClock,
	wire.Bind(new(secretcontracts.Database), new(*secretdatabase.Database)),
	wire.Bind(new(secretcontracts.Clock), new(*secretclock.Clock)),
	encryptionManager.ProvideEncryptionManager,
	cipher.ProvideAESGCMCipherService,
	// Unified storage
	resource.ProvideStorageMetrics,
	resource.ProvideIndexMetrics,
	// Kubernetes API server
	grafanaapiserver.WireSet,
	apiregistry.WireSet,
	appregistry.WireSet,
	// Dashboard Kubernetes helpers
	dashboardclient.ProvideK8sClientWithFallback,
)

var wireSet = wire.NewSet(
	wireBasicSet,
	metrics.WireSet,
	sqlstore.ProvideService,
	ngmetrics.ProvideService,
	wire.Bind(new(notifications.Service), new(*notifications.NotificationService)),
	wire.Bind(new(notifications.WebhookSender), new(*notifications.NotificationService)),
	wire.Bind(new(notifications.EmailSender), new(*notifications.NotificationService)),
	wire.Bind(new(db.DB), new(*sqlstore.SQLStore)),
	prefimpl.ProvideService,
	oauthtoken.ProvideService,
	wire.Bind(new(oauthtoken.OAuthTokenService), new(*oauthtoken.Service)),
	wire.Bind(new(cleanup.AlertRuleService), new(*ngstore.DBstore)),
)

var wireCLISet = wire.NewSet(
	NewRunner,
	wireBasicSet,
	metrics.WireSet,
	sqlstore.ProvideService,
	ngmetrics.ProvideService,
	wire.Bind(new(notifications.Service), new(*notifications.NotificationService)),
	wire.Bind(new(notifications.WebhookSender), new(*notifications.NotificationService)),
	wire.Bind(new(notifications.EmailSender), new(*notifications.NotificationService)),
	wire.Bind(new(db.DB), new(*sqlstore.SQLStore)),
	prefimpl.ProvideService,
	oauthtoken.ProvideService,
	wire.Bind(new(oauthtoken.OAuthTokenService), new(*oauthtoken.Service)),
)

var wireTestSet = wire.NewSet(
	wireBasicSet,
	ProvideTestEnv,
	metrics.WireSetForTest,
	sqlstore.ProvideServiceForTests,
	ngmetrics.ProvideServiceForTest,
	notifications.MockNotificationService,
	wire.Bind(new(notifications.Service), new(*notifications.NotificationServiceMock)),
	wire.Bind(new(notifications.WebhookSender), new(*notifications.NotificationServiceMock)),
	wire.Bind(new(notifications.EmailSender), new(*notifications.NotificationServiceMock)),
	wire.Bind(new(db.DB), new(*sqlstore.SQLStore)),
	prefimpl.ProvideService,
	oauthtoken.ProvideService,
	oauthtokentest.ProvideService,
	wire.Bind(new(oauthtoken.OAuthTokenService), new(*oauthtokentest.Service)),
	wire.Bind(new(cleanup.AlertRuleService), new(*ngstore.DBstore)),
)

func Initialize(ctx context.Context, cfg *setting.Cfg, opts Options, apiOpts api.ServerOptions) (*Server, error) {
	wire.Build(wireExtsSet)
	return &Server{}, nil
}

func InitializeForTest(ctx context.Context, t sqlutil.ITestDB, testingT interface {
	mock.TestingT
	Cleanup(func())
}, cfg *setting.Cfg, opts Options, apiOpts api.ServerOptions,
) (*TestEnv, error) {
	wire.Build(wireExtsTestSet)
	return &TestEnv{Server: &Server{}, TestingT: testingT, SQLStore: &sqlstore.SQLStore{}, Cfg: &setting.Cfg{}}, nil
}

func InitializeForCLI(ctx context.Context, cfg *setting.Cfg) (Runner, error) {
	wire.Build(wireExtsCLISet)
	return Runner{}, nil
}

// InitializeForCLITarget is a simplified set of dependencies for the CLI, used
// by the server target subcommand to launch specific dskit modules.
func InitializeForCLITarget(ctx context.Context, cfg *setting.Cfg) (ModuleRunner, error) {
	wire.Build(wireExtsBaseCLISet)
	return ModuleRunner{}, nil
}

// InitializeModuleServer is a simplified set of dependencies for the CLI,
// suitable for running background services and targeting dskit modules.
func InitializeModuleServer(cfg *setting.Cfg, opts Options, apiOpts api.ServerOptions) (*ModuleServer, error) {
	wire.Build(wireExtsModuleServerSet)
	return &ModuleServer{}, nil
}

// Initialize the standalone APIServer factory
func InitializeAPIServerFactory() (standalone.APIServerFactory, error) {
	wire.Build(wireExtsStandaloneAPIServerSet)
	return &standalone.NoOpAPIServerFactory{}, nil // Wire will replace this with a real interface
}

func InitializeDocumentBuilders(cfg *setting.Cfg) (resource.DocumentBuilderSupplier, error) {
	wire.Build(wireExtsSet)
	return &unifiedsearch.StandardDocumentBuilders{}, nil
}
