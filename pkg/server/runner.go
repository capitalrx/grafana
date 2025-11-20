package server

import (
	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/services/encryption"
	"github.com/capitalrx/grafana/pkg/services/featuremgmt"
	"github.com/capitalrx/grafana/pkg/services/secrets"
	"github.com/capitalrx/grafana/pkg/services/secrets/manager"
	"github.com/capitalrx/grafana/pkg/services/user"
	"github.com/capitalrx/grafana/pkg/setting"

	"github.com/capitalrx/grafana/pkg/registry/apis/secret/contracts"
)

type Runner struct {
	Cfg                         *setting.Cfg
	SQLStore                    db.DB
	SettingsProvider            setting.Provider
	Features                    featuremgmt.FeatureToggles
	EncryptionService           encryption.Internal
	SecretsService              *manager.SecretsService
	SecretsMigrator             secrets.Migrator
	UserService                 user.Service
	SecretsConsolidationService contracts.ConsolidationService
}

func NewRunner(cfg *setting.Cfg, sqlStore db.DB, settingsProvider setting.Provider,
	encryptionService encryption.Internal, features featuremgmt.FeatureToggles,
	secretsService *manager.SecretsService, secretsMigrator secrets.Migrator,
	userService user.Service, secretsConsolidationService contracts.ConsolidationService,
) Runner {
	return Runner{
		Cfg:                         cfg,
		SQLStore:                    sqlStore,
		SettingsProvider:            settingsProvider,
		EncryptionService:           encryptionService,
		SecretsService:              secretsService,
		SecretsMigrator:             secretsMigrator,
		Features:                    features,
		UserService:                 userService,
		SecretsConsolidationService: secretsConsolidationService,
	}
}
