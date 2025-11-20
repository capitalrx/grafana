package sandbox

import (
	"context"

	"github.com/capitalrx/grafana/pkg/setting"
)

type Sandbox interface {
	Plugins(ctx context.Context) ([]string, error)
}

type Service struct {
	cfg *setting.Cfg
}

func ProvideService(cfg *setting.Cfg) *Service {
	return &Service{
		cfg: cfg,
	}
}

func (s *Service) Plugins(ctx context.Context) ([]string, error) {
	return s.cfg.EnableFrontendSandboxForPlugins, nil
}
