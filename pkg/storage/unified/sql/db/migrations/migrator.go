package migrations

import (
	"context"

	"github.com/capitalrx/grafana/pkg/util/xorm"

	"github.com/capitalrx/grafana/pkg/services/sqlstore/migrator"
	"github.com/capitalrx/grafana/pkg/setting"
)

func MigrateResourceStore(ctx context.Context, engine *xorm.Engine, cfg *setting.Cfg) error {
	mg := migrator.NewScopedMigrator(engine, cfg, "resource")
	mg.AddCreateMigration()

	initResourceTables(mg)

	sec := cfg.Raw.Section("database")
	return mg.RunMigrations(
		ctx,
		sec.Key("migration_locking").MustBool(true),
		sec.Key("locking_attempt_timeout_sec").MustInt(),
	)
}
