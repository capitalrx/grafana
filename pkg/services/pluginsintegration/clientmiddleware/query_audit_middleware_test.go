package clientmiddleware

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/queryaudit"
)

func TestQueryAuditMiddleware(t *testing.T) {
	t.Run("disabled auditor adds no middleware", func(t *testing.T) {
		next := &handlertest.Handler{}
		h := NewQueryAuditMiddleware(&queryaudit.Service{}).CreateHandlerMiddleware(next)
		require.Same(t, next, h)
	})
}
