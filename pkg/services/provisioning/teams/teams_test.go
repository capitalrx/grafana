package teams

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/team"
	"github.com/grafana/grafana/pkg/services/user"
)

type fakeTeams struct {
	team.Service
	nextID  int64
	teams   map[int64]*team.TeamDTO
	members map[int64]map[int64]bool
}

func (f *fakeTeams) CreateTeam(_ context.Context, cmd *team.CreateTeamCommand) (team.Team, error) {
	f.nextID++
	f.teams[f.nextID] = &team.TeamDTO{ID: f.nextID, Name: cmd.Name, Email: cmd.Email, OrgID: cmd.OrgID, IsProvisioned: cmd.IsProvisioned}
	return team.Team{ID: f.nextID, Name: cmd.Name}, nil
}

func (f *fakeTeams) UpdateTeam(_ context.Context, cmd *team.UpdateTeamCommand) error {
	f.teams[cmd.ID].Email = cmd.Email
	return nil
}

func (f *fakeTeams) DeleteTeam(_ context.Context, cmd *team.DeleteTeamCommand) error {
	delete(f.teams, cmd.ID)
	return nil
}

func (f *fakeTeams) SearchTeams(_ context.Context, q *team.SearchTeamsQuery) (team.SearchTeamQueryResult, error) {
	var res team.SearchTeamQueryResult
	for _, t := range f.teams {
		if t.OrgID == q.OrgID && (q.Name == "" || t.Name == q.Name) {
			res.Teams = append(res.Teams, t)
		}
	}
	return res, nil
}

func (f *fakeTeams) GetTeamMembers(_ context.Context, q *team.GetTeamMembersQuery) ([]*team.TeamMemberDTO, error) {
	var out []*team.TeamMemberDTO
	for id := range f.members[q.TeamID] {
		out = append(out, &team.TeamMemberDTO{UserID: id})
	}
	return out, nil
}

type fakePerms struct {
	accesscontrol.TeamPermissionsService
	teams *fakeTeams
}

func (f *fakePerms) SetUserPermission(_ context.Context, _ int64, u accesscontrol.User, teamID, permission string) (*accesscontrol.ResourcePermission, error) {
	var id int64
	for k := range f.teams.teams {
		if teamID == itoa(k) {
			id = k
		}
	}
	if f.teams.members[id] == nil {
		f.teams.members[id] = map[int64]bool{}
	}
	if permission == "" {
		delete(f.teams.members[id], u.ID)
	} else {
		f.teams.members[id][u.ID] = true
	}
	return nil, nil
}

func itoa(i int64) string {
	return string(rune('0' + i))
}

type fakeUsers struct {
	user.Service
	nextID  int64
	byLogin map[string]*user.User
	created []*user.CreateUserCommand
}

func (f *fakeUsers) GetByLogin(_ context.Context, q *user.GetUserByLoginQuery) (*user.User, error) {
	if u, ok := f.byLogin[q.LoginOrEmail]; ok {
		return u, nil
	}
	return nil, user.ErrUserNotFound
}

func (f *fakeUsers) Create(_ context.Context, cmd *user.CreateUserCommand) (*user.User, error) {
	f.nextID++
	u := &user.User{ID: f.nextID, Login: cmd.Login, Email: cmd.Email, IsProvisioned: cmd.IsProvisioned}
	f.byLogin[cmd.Login] = u
	f.created = append(f.created, cmd)
	return u, nil
}

type fakeOrgs struct {
	org.Service
	roles map[int64]org.RoleType
}

func (f *fakeOrgs) GetUserOrgList(_ context.Context, q *org.GetUserOrgListQuery) ([]*org.UserOrgDTO, error) {
	if _, ok := f.roles[q.UserID]; ok {
		return []*org.UserOrgDTO{{OrgID: 1}}, nil
	}
	return nil, nil
}

func (f *fakeOrgs) AddOrgUser(_ context.Context, cmd *org.AddOrgUserCommand) error {
	f.roles[cmd.UserID] = cmd.Role
	return nil
}

func newFakes() (Services, *fakeTeams, *fakeUsers, *fakeOrgs) {
	ft := &fakeTeams{teams: map[int64]*team.TeamDTO{}, members: map[int64]map[int64]bool{}}
	fu := &fakeUsers{byLogin: map[string]*user.User{}}
	fo := &fakeOrgs{roles: map[int64]org.RoleType{}}
	return Services{Teams: ft, Users: fu, Orgs: fo, TeamPermissions: &fakePerms{teams: ft}}, ft, fu, fo
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "teams.yaml"), []byte(content), 0o600))
	return dir
}

func TestProvisionCreatesTeamsAndPlaceholderUsers(t *testing.T) {
	svc, ft, fu, fo := newFakes()
	existing, err := fu.Create(context.Background(), &user.CreateUserCommand{Login: "a@x.com", Email: "a@x.com"})
	require.NoError(t, err)
	fu.created = nil
	fo.roles[existing.ID] = org.RoleEditor

	dir := writeConfig(t, `
apiVersion: 1
teams:
  - name: SRE
    email: sre@x.com
    members:
      - A@x.com
      - new@x.com
`)
	require.NoError(t, Provision(context.Background(), dir, svc))

	require.Len(t, ft.teams, 1)
	require.Len(t, fu.created, 1)
	require.Equal(t, "new@x.com", fu.created[0].Login)
	require.True(t, fu.created[0].SkipOrgSetup)
	require.False(t, fu.created[0].IsProvisioned)

	placeholder := fu.byLogin["new@x.com"]
	require.Equal(t, org.RoleNone, fo.roles[placeholder.ID])
	require.Equal(t, org.RoleEditor, fo.roles[existing.ID], "existing org role must be preserved")
	require.Len(t, ft.members[1], 2)

	// Idempotent.
	require.NoError(t, Provision(context.Background(), dir, svc))
	require.Len(t, ft.teams, 1)
	require.Len(t, fu.created, 1)
}

func TestProvisionRemovesMembersAndDeletesDroppedTeams(t *testing.T) {
	svc, ft, _, _ := newFakes()
	dir := writeConfig(t, `
teams:
  - name: SRE
    members: [a@x.com, b@x.com]
  - name: Old
`)
	require.NoError(t, Provision(context.Background(), dir, svc))
	require.Len(t, ft.teams, 2)

	// A hand-made team must survive.
	ft.teams[9] = &team.TeamDTO{ID: 9, Name: "Manual", OrgID: 1}

	dir = writeConfig(t, `
teams:
  - name: SRE
    members: [a@x.com]
`)
	require.NoError(t, Provision(context.Background(), dir, svc))
	require.Len(t, ft.members[1], 1)
	require.Len(t, ft.teams, 2)
	require.Contains(t, ft.teams, int64(9))
}

func TestProvisionEmptyConfigDeletesNothing(t *testing.T) {
	svc, ft, _, _ := newFakes()
	require.NoError(t, Provision(context.Background(), writeConfig(t, "teams: [{name: SRE}]"), svc))
	require.NoError(t, Provision(context.Background(), t.TempDir(), svc))
	require.Len(t, ft.teams, 1)
}
