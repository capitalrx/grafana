package quotaimpl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/services/quota"
	"github.com/capitalrx/grafana/pkg/tests/testsuite"
	"github.com/capitalrx/grafana/pkg/util/testutil"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

func TestIntegrationQuotaDataAccess(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	ss := db.InitTestDB(t)
	quotaStore := sqlStore{
		db: ss,
	}

	t.Run("quota deleted", func(t *testing.T) {
		ctx := quota.FromContext(context.Background(), &quota.TargetToSrv{})
		err := quotaStore.DeleteByUser(ctx, 1)
		require.NoError(t, err)
	})
}
