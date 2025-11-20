package routingtree

import (
	"k8s.io/apiserver/pkg/registry/rest"

	grafanarest "github.com/capitalrx/grafana/pkg/apiserver/rest"
	"github.com/capitalrx/grafana/pkg/services/apiserver/endpoints/request"
)

func NewStorage(legacySvc RouteService, namespacer request.NamespaceMapper) grafanarest.Storage {
	return &legacyStorage{
		service:        legacySvc,
		namespacer:     namespacer,
		tableConverter: rest.NewDefaultTableConvertor(ResourceInfo.GroupResource()),
	}
}
