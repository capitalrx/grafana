package ossaccesscontrol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/setting"
)

func TestDatasourcePermissionsServiceTeamsAndBuiltInRoles(t *testing.T) {
	sql := db.InitTestDB(t)
	svc := ProvideDatasourcePermissionsService(setting.NewCfg(), featuremgmt.WithFeatures(), sql)
	ctx := context.Background()
	require.NoError(t, sql.WithDbSession(ctx, func(sess *db.Session) error {
		_, err := sess.Exec("INSERT INTO team (id, org_id, name, uid, email, created, updated) VALUES (7, 1, 'sre', 'sre-uid', '', '2026-01-01', '2026-01-01')")
		return err
	}))
	requester := accesscontrol.BackgroundUser("test", 1, org.RoleAdmin, []accesscontrol.Permission{
		{Action: accesscontrol.ActionTeamsRead, Scope: accesscontrol.ScopeTeamsAll},
		{Action: accesscontrol.ActionOrgUsersRead, Scope: accesscontrol.ScopeUsersAll},
	})

	_, err := svc.SetPermissions(ctx, 1, "ds1",
		accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Viewer", Permission: "Query"},
		accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Editor", Permission: "Query"},
		accesscontrol.SetResourcePermissionCommand{TeamID: 7, Permission: "Query"},
	)
	require.NoError(t, err)

	perms, err := svc.GetPermissions(ctx, requester, "ds1")
	require.NoError(t, err)
	require.Len(t, perms, 3)
	for _, p := range perms {
		require.ElementsMatch(t, DatasourceQueryActions, p.Actions)
	}

	// Restrict to the team only.
	_, err = svc.SetPermissions(ctx, 1, "ds1",
		accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Viewer", Permission: ""},
		accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Editor", Permission: ""},
	)
	require.NoError(t, err)

	perms, err = svc.GetPermissions(ctx, requester, "ds1")
	require.NoError(t, err)
	require.Len(t, perms, 1)
	require.Equal(t, int64(7), perms[0].TeamID)

	// Other data sources are untouched.
	_, err = svc.SetPermissions(ctx, 1, "ds2", accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Viewer", Permission: "Query"})
	require.NoError(t, err)
	perms, err = svc.GetPermissions(ctx, requester, "ds1")
	require.NoError(t, err)
	require.Len(t, perms, 1)
}
