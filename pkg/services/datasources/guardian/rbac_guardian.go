package guardian

import (
	"context"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/datasources"
)

var _ DatasourceGuardian = new(rbacGuardian)

// rbacGuardian checks the requester's datasources:read and datasources:query
// permissions against each data source's UID, so per data source permissions
// are enforced on every query path that goes through the guardian.
type rbacGuardian struct {
	ac      accesscontrol.AccessControl
	user    identity.Requester
	uidByID map[int64]string
	log     log.Logger
}

func (g *rbacGuardian) can(action, uid string) (bool, error) {
	if g.user == nil || g.user.IsNil() {
		return false, nil
	}
	return g.ac.Evaluate(context.Background(), g.user,
		accesscontrol.EvalPermission(action, datasources.ScopeProvider.GetResourceScopeUID(uid)))
}

func (g *rbacGuardian) CanQuery(datasourceID int64) (bool, error) {
	uid, ok := g.uidByID[datasourceID]
	if !ok {
		g.log.Warn("Data source not known to guardian, denying query", "id", datasourceID)
		return false, nil
	}
	return g.can(datasources.ActionQuery, uid)
}

func (g *rbacGuardian) filter(action string, dss []*datasources.DataSource) ([]*datasources.DataSource, error) {
	out := make([]*datasources.DataSource, 0, len(dss))
	for _, ds := range dss {
		ok, err := g.can(action, ds.UID)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, ds)
		}
	}
	return out, nil
}

func (g *rbacGuardian) FilterDatasourcesByReadPermissions(dss []*datasources.DataSource) ([]*datasources.DataSource, error) {
	return g.filter(datasources.ActionRead, dss)
}

func (g *rbacGuardian) FilterDatasourcesByQueryPermissions(dss []*datasources.DataSource) ([]*datasources.DataSource, error) {
	return g.filter(datasources.ActionQuery, dss)
}
