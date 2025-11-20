package receiver

import (
	grafanarest "github.com/capitalrx/grafana/pkg/apiserver/rest"
	"github.com/capitalrx/grafana/pkg/services/apiserver/endpoints/request"
)

func NewStorage(
	legacySvc ReceiverService,
	namespacer request.NamespaceMapper,
	metadata MetadataService,
) grafanarest.Storage {
	return &legacyStorage{
		service:        legacySvc,
		namespacer:     namespacer,
		tableConverter: ResourceInfo.TableConverter(),
		metadata:       metadata,
	}
}
