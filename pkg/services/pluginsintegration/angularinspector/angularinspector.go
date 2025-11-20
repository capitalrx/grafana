package angularinspector

import (
	"github.com/capitalrx/grafana/pkg/plugins/manager/loader/angular/angulardetector"
	"github.com/capitalrx/grafana/pkg/plugins/manager/loader/angular/angularinspector"
	"github.com/capitalrx/grafana/pkg/services/pluginsintegration/angulardetectorsprovider"
)

type Service struct {
	angularinspector.Inspector
}

func ProvideService(dynamic *angulardetectorsprovider.Dynamic) (*Service, error) {
	return &Service{
		Inspector: angularinspector.NewPatternListInspector(
			angulardetector.SequenceDetectorsProvider{
				dynamic,
				angularinspector.NewDefaultStaticDetectorsProvider(),
			},
		),
	}, nil
}
