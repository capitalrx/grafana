package provisioning

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/team"
)

type datasourceAccess struct {
	teams       team.Service
	permissions accesscontrol.DatasourcePermissionsService
	log         log.Logger
}

func (a *datasourceAccess) background(orgID int64) identity.Requester {
	return accesscontrol.BackgroundUser("datasource_provisioner", orgID, org.RoleAdmin, []accesscontrol.Permission{
		{Action: accesscontrol.ActionTeamsRead, Scope: accesscontrol.ScopeTeamsAll},
		{Action: accesscontrol.ActionOrgUsersRead, Scope: accesscontrol.ScopeUsersAll},
	})
}

// SetTeamAccess makes the named teams the only non-Admin principals that can query the data source.
// Admins keep access through their org role. Unknown team names are skipped, which fails closed.
func (a *datasourceAccess) SetTeamAccess(ctx context.Context, orgID int64, uid string, teamNames []string) error {
	if orgID == 0 {
		orgID = 1
	}

	want := map[int64]struct{}{}
	for _, name := range teamNames {
		res, err := a.teams.SearchTeams(ctx, &team.SearchTeamsQuery{
			Name:         name,
			OrgID:        orgID,
			Page:         1,
			Limit:        1000,
			SignedInUser: a.background(orgID),
		})
		if err != nil {
			return fmt.Errorf("failed to look up team %q: %w", name, err)
		}
		found := false
		for _, t := range res.Teams {
			if t.Name == name {
				want[t.ID] = struct{}{}
				found = true
			}
		}
		if !found {
			a.log.Error("Team for datasource access does not exist, skipping", "team", name, "datasource", uid)
		}
	}

	restricted := len(teamNames) > 0

	existing, err := a.permissions.GetPermissions(ctx, a.background(orgID), uid)
	if err != nil {
		return fmt.Errorf("failed to read datasource permissions: %w", err)
	}

	builtin := "Query"
	if restricted {
		builtin = ""
	}
	cmds := []accesscontrol.SetResourcePermissionCommand{
		{BuiltinRole: string(org.RoleViewer), Permission: builtin},
		{BuiltinRole: string(org.RoleEditor), Permission: builtin},
	}

	for id := range want {
		cmds = append(cmds, accesscontrol.SetResourcePermissionCommand{TeamID: id, Permission: "Query"})
	}
	for _, p := range existing {
		if p.TeamID == 0 {
			continue
		}
		if _, ok := want[p.TeamID]; !ok {
			cmds = append(cmds, accesscontrol.SetResourcePermissionCommand{TeamID: p.TeamID, Permission: ""})
		}
	}

	if _, err := a.permissions.SetPermissions(ctx, orgID, uid, cmds...); err != nil {
		return fmt.Errorf("failed to set datasource permissions: %w", err)
	}
	return nil
}
