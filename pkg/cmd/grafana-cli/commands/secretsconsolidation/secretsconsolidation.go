package secretsconsolidation

import (
	"context"

	"github.com/capitalrx/grafana/pkg/cmd/grafana-cli/utils"
	"github.com/capitalrx/grafana/pkg/server"
)

func ConsolidateSecrets(_ utils.CommandLine, runner server.Runner) error {
	err := runner.SecretsConsolidationService.Consolidate(context.Background())
	return err
}
