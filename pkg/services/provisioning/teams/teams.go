package teams

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/team"
)

func Provision(ctx context.Context, path string, teamService team.Service) error {
	logger := log.New("provisioning.teams")
	reader := &configReader{
		path: path,
		log:  logger,
	}

	teamsCfg, err := reader.readConfig()
	if err != nil {
		return err
	}

	for _, t := range teamsCfg {
		if err := provisionTeam(ctx, t, teamService, logger); err != nil {
			return err
		}
	}
	return nil
}

func provisionTeam(ctx context.Context, t *TeamConfig, teamService team.Service, logger log.Logger) error {
	orgID := t.OrgID.Value()
	if orgID == 0 {
		orgID = 1
	}

	name := t.Name.Value()
	if name == "" {
		return nil
	}

	provisionerUser := accesscontrol.BackgroundUser(
		"team_provisioner",
		orgID,
		org.RoleAdmin,
		[]accesscontrol.Permission{},
	)

	query := &team.SearchTeamsQuery{
		Name:         name,
		OrgID:        orgID,
		Page:         1,
		Limit:        1,
		SignedInUser: provisionerUser,
	}

	result, err := teamService.SearchTeams(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search for team %s: %w", name, err)
	}

	if result.TotalCount > 0 {
		logger.Debug("Team already exists", "name", name, "orgID", orgID)
		return nil
	}

	cmd := &team.CreateTeamCommand{
		Name:          name,
		Email:         t.Email.Value(),
		OrgID:         orgID,
		IsProvisioned: true,
	}

	if _, err := teamService.CreateTeam(ctx, cmd); err != nil {
		return fmt.Errorf("failed to create team %s: %w", name, err)
	}

	logger.Info("Team provisioned", "name", name, "orgID", orgID)
	return nil
}
