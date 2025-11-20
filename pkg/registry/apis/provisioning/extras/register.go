package extras

import (
	"github.com/capitalrx/grafana/apps/provisioning/pkg/repository"
	"github.com/capitalrx/grafana/apps/provisioning/pkg/repository/github"
	"github.com/capitalrx/grafana/apps/provisioning/pkg/repository/local"
	"github.com/capitalrx/grafana/apps/secret/pkg/decrypt"
	"github.com/capitalrx/grafana/pkg/registry/apis/provisioning"
	"github.com/capitalrx/grafana/pkg/registry/apis/provisioning/webhooks"
	"github.com/capitalrx/grafana/pkg/setting"
)

// HACK: This is a hack so that wire can uniquely identify dependencies
func ProvideProvisioningOSSExtras(webhook *webhooks.WebhookExtraBuilder) []provisioning.ExtraBuilder {
	return []provisioning.ExtraBuilder{
		webhook.ExtraBuilder,
	}
}

func ProvideProvisioningOSSRepositoryExtras(
	cfg *setting.Cfg,
	decryptSvc decrypt.DecryptService,
	ghFactory *github.Factory,
	webhooksBuilder *webhooks.WebhookExtraBuilder,
) []repository.Extra {
	return []repository.Extra{
		local.Extra(
			cfg.HomePath,
			cfg.PermittedProvisioningPaths,
		),
		github.Extra(
			repository.ProvideDecrypter(decryptSvc),
			ghFactory,
			webhooksBuilder,
		),
	}
}
