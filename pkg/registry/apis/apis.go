package apiregistry

import (
	dashboardinternal "github.com/capitalrx/grafana/pkg/registry/apis/dashboard"
	"github.com/capitalrx/grafana/pkg/registry/apis/dashboardsnapshot"
	"github.com/capitalrx/grafana/pkg/registry/apis/datasource"
	"github.com/capitalrx/grafana/pkg/registry/apis/featuretoggle"
	"github.com/capitalrx/grafana/pkg/registry/apis/folders"
	"github.com/capitalrx/grafana/pkg/registry/apis/iam"
	"github.com/capitalrx/grafana/pkg/registry/apis/ofrep"
	"github.com/capitalrx/grafana/pkg/registry/apis/preferences"
	"github.com/capitalrx/grafana/pkg/registry/apis/provisioning"
	"github.com/capitalrx/grafana/pkg/registry/apis/query"
	"github.com/capitalrx/grafana/pkg/registry/apis/secret"
	"github.com/capitalrx/grafana/pkg/registry/apis/userstorage"
)

type Service struct{}

// ProvideRegistryServiceSink is an entry point for each service that will force initialization
// and give each builder the chance to register itself with the main server
func ProvideRegistryServiceSink(
	_ *dashboardinternal.DashboardsAPIBuilder,
	_ *dashboardsnapshot.SnapshotsAPIBuilder,
	_ *featuretoggle.FeatureFlagAPIBuilder,
	_ *datasource.DataSourceAPIBuilder,
	_ *folders.FolderAPIBuilder,
	_ *iam.IdentityAccessManagementAPIBuilder,
	_ *query.QueryAPIBuilder,
	_ *userstorage.UserStorageAPIBuilder,
	_ *preferences.APIBuilder,
	_ *provisioning.APIBuilder,
	_ *ofrep.APIBuilder,
	_ *secret.DependencyRegisterer,
) *Service {
	return &Service{}
}
