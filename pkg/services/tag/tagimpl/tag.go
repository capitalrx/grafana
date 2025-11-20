package tagimpl

import (
	"context"

	"github.com/capitalrx/grafana/pkg/infra/db"
	"github.com/capitalrx/grafana/pkg/services/tag"
)

type Service struct {
	store store
}

func ProvideService(db db.DB) *Service {
	return &Service{
		store: &sqlStore{
			db: db,
		},
	}
}

func (s *Service) EnsureTagsExist(ctx context.Context, tags []*tag.Tag) ([]*tag.Tag, error) {
	return s.store.EnsureTagsExist(ctx, tags)
}
