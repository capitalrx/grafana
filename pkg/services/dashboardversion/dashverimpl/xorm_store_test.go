package dashverimpl

import (
	"testing"

	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/util/testutil"
)

func TestIntegrationXORMGetDashboardVersion(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	testIntegrationGetDashboardVersion(t, func(ss db.DB) store {
		return &sqlStore{
			db:      ss,
			dialect: ss.GetDialect(),
		}
	})
}
