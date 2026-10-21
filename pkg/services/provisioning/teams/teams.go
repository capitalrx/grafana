package teams

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/team"
	"github.com/grafana/grafana/pkg/services/user"
)

const defaultOrgID = int64(1)

// Services are the dependencies needed to reconcile teams and their members.
type Services struct {
	Teams           team.Service
	Users           user.Service
	Orgs            org.Service
	TeamPermissions accesscontrol.TeamPermissionsService
}

// Provision reconciles the teams described under path with the database:
// missing teams are created, members are added and removed to match the file,
// and provisioned teams that are no longer in the file are deleted.
func Provision(ctx context.Context, path string, svc Services) error {
	logger := log.New("provisioning.teams")
	reader := &configReader{
		path: path,
		log:  logger,
	}

	teamsCfg, err := reader.readConfig()
	if err != nil {
		return err
	}

	desired := map[int64]map[string]struct{}{}
	for _, t := range teamsCfg {
		orgID := teamOrgID(t)
		name := t.Name.Value()
		if name == "" {
			continue
		}
		if desired[orgID] == nil {
			desired[orgID] = map[string]struct{}{}
		}
		desired[orgID][name] = struct{}{}

		if err := provisionTeam(ctx, t, svc, logger); err != nil {
			return err
		}
	}

	// An empty or missing config must never wipe every provisioned team.
	if len(desired) == 0 {
		return nil
	}

	for orgID, names := range desired {
		if err := deleteRemovedTeams(ctx, orgID, names, svc, logger); err != nil {
			return err
		}
	}
	return nil
}

func teamOrgID(t *TeamConfig) int64 {
	if orgID := t.OrgID.Value(); orgID != 0 {
		return orgID
	}
	return defaultOrgID
}

func provisionerUser(orgID int64) identity.Requester {
	return accesscontrol.BackgroundUser(
		"team_provisioner",
		orgID,
		org.RoleAdmin,
		[]accesscontrol.Permission{
			{Action: accesscontrol.ActionTeamsRead, Scope: accesscontrol.ScopeTeamsAll},
			{Action: accesscontrol.ActionOrgUsersRead, Scope: accesscontrol.ScopeUsersAll},
		},
	)
}

func findTeam(ctx context.Context, name string, orgID int64, svc Services) (*team.TeamDTO, error) {
	result, err := svc.Teams.SearchTeams(ctx, &team.SearchTeamsQuery{
		Name:         name,
		OrgID:        orgID,
		Page:         1,
		Limit:        1000,
		SignedInUser: provisionerUser(orgID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search for team %s: %w", name, err)
	}
	for _, t := range result.Teams {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, nil
}

func provisionTeam(ctx context.Context, t *TeamConfig, svc Services, logger log.Logger) error {
	orgID := teamOrgID(t)
	name := t.Name.Value()
	if name == "" {
		return nil
	}

	existing, err := findTeam(ctx, name, orgID, svc)
	if err != nil {
		return err
	}

	var teamID int64
	switch {
	case existing == nil:
		created, err := svc.Teams.CreateTeam(ctx, &team.CreateTeamCommand{
			Name:          name,
			Email:         t.Email.Value(),
			OrgID:         orgID,
			IsProvisioned: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create team %s: %w", name, err)
		}
		teamID = created.ID
		logger.Info("Team provisioned", "name", name, "orgID", orgID)
	case !existing.IsProvisioned:
		// A team created by hand with the same name is not ours to manage.
		logger.Warn("Team exists but is not provisioned, skipping", "name", name, "orgID", orgID)
		return nil
	default:
		teamID = existing.ID
		if existing.Email != t.Email.Value() {
			if err := svc.Teams.UpdateTeam(ctx, &team.UpdateTeamCommand{
				ID:    teamID,
				Name:  name,
				Email: t.Email.Value(),
				OrgID: orgID,
			}); err != nil {
				return fmt.Errorf("failed to update team %s: %w", name, err)
			}
		}
	}

	return reconcileMembers(ctx, t, teamID, orgID, svc, logger)
}

func reconcileMembers(ctx context.Context, t *TeamConfig, teamID, orgID int64, svc Services, logger log.Logger) error {
	name := t.Name.Value()

	want := map[int64]string{}
	for _, m := range t.Members {
		id := strings.ToLower(strings.TrimSpace(m.Value()))
		if id == "" {
			continue
		}
		userID, err := ensureUser(ctx, id, orgID, svc, logger)
		if err != nil {
			return fmt.Errorf("team %s: %w", name, err)
		}
		want[userID] = id
	}

	members, err := svc.Teams.GetTeamMembers(ctx, &team.GetTeamMembersQuery{
		OrgID:        orgID,
		TeamID:       teamID,
		SignedInUser: provisionerUser(orgID),
	})
	if err != nil {
		return fmt.Errorf("failed to list members of team %s: %w", name, err)
	}

	teamIDStr := strconv.FormatInt(teamID, 10)
	have := map[int64]struct{}{}
	for _, m := range members {
		have[m.UserID] = struct{}{}
		if _, ok := want[m.UserID]; ok {
			continue
		}
		// Members synced from an external provider are not managed here.
		if m.External {
			continue
		}
		if _, err := svc.TeamPermissions.SetUserPermission(ctx, orgID, accesscontrol.User{ID: m.UserID}, teamIDStr, ""); err != nil {
			return fmt.Errorf("failed to remove %s from team %s: %w", m.Login, name, err)
		}
		logger.Info("Removed team member", "team", name, "user", m.Login)
	}

	for userID, ident := range want {
		if _, ok := have[userID]; ok {
			continue
		}
		if _, err := svc.TeamPermissions.SetUserPermission(ctx, orgID, accesscontrol.User{ID: userID}, teamIDStr, team.PermissionTypeMember.String()); err != nil {
			return fmt.Errorf("failed to add %s to team %s: %w", ident, name, err)
		}
		logger.Info("Added team member", "team", name, "user", ident)
	}
	return nil
}

// ensureUser returns the ID of the user matching ident (email or login),
// creating a placeholder if none exists. The placeholder is deliberately not
// marked as provisioned so the auth provider can adopt it and fill in its
// attributes on first login, and it joins the org with no role so it grants
// no access until the login-time org mapping assigns one.
func ensureUser(ctx context.Context, ident string, orgID int64, svc Services, logger log.Logger) (int64, error) {
	u, err := svc.Users.GetByLogin(ctx, &user.GetUserByLoginQuery{LoginOrEmail: ident})
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return 0, fmt.Errorf("failed to look up user %s: %w", ident, err)
	}

	if u == nil || err != nil {
		u, err = svc.Users.Create(ctx, &user.CreateUserCommand{
			Login:        ident,
			Email:        ident,
			Name:         ident,
			SkipOrgSetup: true,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to create user %s: %w", ident, err)
		}
		logger.Info("Created placeholder user", "user", ident)
	}

	if err := ensureOrgMembership(ctx, u.ID, ident, orgID, svc); err != nil {
		return 0, err
	}
	return u.ID, nil
}

func ensureOrgMembership(ctx context.Context, userID int64, ident string, orgID int64, svc Services) error {
	orgs, err := svc.Orgs.GetUserOrgList(ctx, &org.GetUserOrgListQuery{UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to list orgs of user %s: %w", ident, err)
	}
	for _, o := range orgs {
		if o.OrgID == orgID {
			return nil
		}
	}

	err = svc.Orgs.AddOrgUser(ctx, &org.AddOrgUserCommand{
		OrgID:  orgID,
		UserID: userID,
		Role:   org.RoleNone,
	})
	if err != nil && !errors.Is(err, org.ErrOrgUserAlreadyAdded) {
		return fmt.Errorf("failed to add user %s to org %d: %w", ident, orgID, err)
	}
	return nil
}

func deleteRemovedTeams(ctx context.Context, orgID int64, keep map[string]struct{}, svc Services, logger log.Logger) error {
	result, err := svc.Teams.SearchTeams(ctx, &team.SearchTeamsQuery{
		OrgID:        orgID,
		Page:         1,
		Limit:        10000,
		SignedInUser: provisionerUser(orgID),
	})
	if err != nil {
		return fmt.Errorf("failed to list teams in org %d: %w", orgID, err)
	}

	for _, t := range result.Teams {
		if !t.IsProvisioned {
			continue
		}
		if _, ok := keep[t.Name]; ok {
			continue
		}
		if err := svc.Teams.DeleteTeam(ctx, &team.DeleteTeamCommand{OrgID: orgID, ID: t.ID}); err != nil {
			return fmt.Errorf("failed to delete team %s: %w", t.Name, err)
		}
		logger.Info("Deleted provisioned team no longer in config", "name", t.Name, "orgID", orgID)
	}
	return nil
}
