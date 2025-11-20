package prefimpl

import (
	"testing"

	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/util/testutil"
)

func TestIntegrationXORMPreferencesDataAccess(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	testIntegrationPreferencesDataAccess(t, func(ss db.DB) store {
		return &sqlStore{db: ss}
	})
}
