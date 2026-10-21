package guardian

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/accesscontrol/acimpl"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/setting"
)

func TestRBACGuardian(t *testing.T) {
	restricted := datasources.DataSource{ID: 1, UID: "restricted"}
	open := datasources.DataSource{ID: 2, UID: "open"}

	cfg := setting.NewCfg()
	cfg.RBAC.DatasourcePermissionsEnforced = true
	provider := ProvideGuardian(cfg, acimpl.ProvideAccessControlTest())

	viewer := accesscontrol.BackgroundUser("viewer", 1, org.RoleViewer, []accesscontrol.Permission{
		{Action: datasources.ActionRead, Scope: "datasources:uid:open"},
		{Action: datasources.ActionQuery, Scope: "datasources:uid:open"},
	})
	admin := accesscontrol.BackgroundUser("admin", 1, org.RoleAdmin, []accesscontrol.Permission{
		{Action: datasources.ActionRead, Scope: datasources.ScopeAll},
		{Action: datasources.ActionQuery, Scope: datasources.ScopeAll},
	})

	t.Run("viewer without a grant cannot query", func(t *testing.T) {
		ok, err := provider.New(1, viewer, restricted).CanQuery(restricted.ID)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("viewer with a grant can query", func(t *testing.T) {
		ok, err := provider.New(1, viewer, open).CanQuery(open.ID)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("admin can query everything", func(t *testing.T) {
		ok, err := provider.New(1, admin, restricted).CanQuery(restricted.ID)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("filters the data source list", func(t *testing.T) {
		got, err := provider.New(1, viewer).FilterDatasourcesByReadPermissions([]*datasources.DataSource{&restricted, &open})
		require.NoError(t, err)
		require.Equal(t, []*datasources.DataSource{&open}, got)
	})

	t.Run("not enforced allows everything", func(t *testing.T) {
		ok, err := ProvideGuardian(setting.NewCfg(), acimpl.ProvideAccessControlTest()).New(1, viewer, restricted).CanQuery(restricted.ID)
		require.NoError(t, err)
		require.True(t, ok)
	})
}
