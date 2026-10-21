package provisioning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/team"
)

type accessTeams struct {
	team.Service
}

func (accessTeams) SearchTeams(_ context.Context, q *team.SearchTeamsQuery) (team.SearchTeamQueryResult, error) {
	ids := map[string]int64{"sre": 1, "dev": 2}
	if id, ok := ids[q.Name]; ok {
		return team.SearchTeamQueryResult{Teams: []*team.TeamDTO{{ID: id, Name: q.Name}}}, nil
	}
	return team.SearchTeamQueryResult{}, nil
}

type accessPerms struct {
	accesscontrol.DatasourcePermissionsService
	existing []accesscontrol.ResourcePermission
	set      []accesscontrol.SetResourcePermissionCommand
}

func (a *accessPerms) GetPermissions(context.Context, identity.Requester, string) ([]accesscontrol.ResourcePermission, error) {
	return a.existing, nil
}

func (a *accessPerms) SetPermissions(_ context.Context, _ int64, _ string, cmds ...accesscontrol.SetResourcePermissionCommand) ([]accesscontrol.ResourcePermission, error) {
	a.set = cmds
	return nil, nil
}

func TestDatasourceAccessSetTeamAccess(t *testing.T) {
	t.Run("teams restrict and clear stale grants", func(t *testing.T) {
		perms := &accessPerms{existing: []accesscontrol.ResourcePermission{{TeamID: 9}, {BuiltInRole: "Viewer"}}}
		a := &datasourceAccess{teams: accessTeams{}, permissions: perms, log: log.NewNopLogger()}
		require.NoError(t, a.SetTeamAccess(context.Background(), 0, "ds", []string{"sre", "missing"}))

		require.Contains(t, perms.set, accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Viewer", Permission: ""})
		require.Contains(t, perms.set, accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Editor", Permission: ""})
		require.Contains(t, perms.set, accesscontrol.SetResourcePermissionCommand{TeamID: 1, Permission: "Query"})
		require.Contains(t, perms.set, accesscontrol.SetResourcePermissionCommand{TeamID: 9, Permission: ""})
		require.Len(t, perms.set, 4, "an unknown team must not grant anything")
	})

	t.Run("no teams opens to Viewer and Editor", func(t *testing.T) {
		perms := &accessPerms{existing: []accesscontrol.ResourcePermission{{TeamID: 2}}}
		a := &datasourceAccess{teams: accessTeams{}, permissions: perms, log: log.NewNopLogger()}
		require.NoError(t, a.SetTeamAccess(context.Background(), 1, "ds", nil))

		require.Contains(t, perms.set, accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Viewer", Permission: "Query"})
		require.Contains(t, perms.set, accesscontrol.SetResourcePermissionCommand{BuiltinRole: "Editor", Permission: "Query"})
		require.Contains(t, perms.set, accesscontrol.SetResourcePermissionCommand{TeamID: 2, Permission: ""})
	})
}
