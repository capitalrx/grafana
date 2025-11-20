package starimpl

import (
	"testing"

	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/util/testutil"
)

func TestIntegrationXormUserStarsDataAccess(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	testIntegrationUserStarsDataAccess(t, func(ss db.DB) store {
		return &sqlStore{db: ss}
	})
}
