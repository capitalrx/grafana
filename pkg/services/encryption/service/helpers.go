package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/capitalrx/grafana/pkg/infra/tracing"
	"github.com/capitalrx/grafana/pkg/infra/usagestats"
	encryptionprovider "github.com/capitalrx/grafana/pkg/services/encryption/provider"
	"github.com/capitalrx/grafana/pkg/setting"
)

func SetupTestService(tb testing.TB) *Service {
	tb.Helper()

	usMock := &usagestats.UsageStatsMock{T: tb}
	provider := encryptionprovider.ProvideEncryptionProvider()
	settings := setting.NewCfg()

	service, err := ProvideEncryptionService(tracing.InitializeTracerForTest(), provider, usMock, settings)
	require.NoError(tb, err)

	return service
}
