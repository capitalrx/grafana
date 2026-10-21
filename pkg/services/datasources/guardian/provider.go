package guardian

import (
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/setting"
)

type DatasourceGuardianProvider interface {
	New(orgID int64, user identity.Requester, dataSources ...datasources.DataSource) DatasourceGuardian
}

type DatasourceGuardian interface {
	CanQuery(datasourceID int64) (bool, error)
	FilterDatasourcesByReadPermissions([]*datasources.DataSource) ([]*datasources.DataSource, error)
	FilterDatasourcesByQueryPermissions([]*datasources.DataSource) ([]*datasources.DataSource, error)
}

func ProvideGuardian(cfg *setting.Cfg, ac accesscontrol.AccessControl) *OSSProvider {
	return &OSSProvider{
		enforced: cfg != nil && cfg.RBAC.DatasourcePermissionsEnforced,
		ac:       ac,
		log:      log.New("datasources.guardian"),
	}
}

type OSSProvider struct {
	enforced bool
	ac       accesscontrol.AccessControl
	log      log.Logger
}

func (p *OSSProvider) New(orgID int64, user identity.Requester, dataSources ...datasources.DataSource) DatasourceGuardian {
	if !p.enforced || p.ac == nil {
		return &AllowGuardian{}
	}
	uidByID := make(map[int64]string, len(dataSources))
	for _, ds := range dataSources {
		uidByID[ds.ID] = ds.UID
	}
	return &rbacGuardian{ac: p.ac, user: user, uidByID: uidByID, log: p.log}
}
