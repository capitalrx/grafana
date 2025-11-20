package service

import (
	"encoding/json"

	"github.com/capitalrx/grafana/pkg/components/simplejson"
	"github.com/capitalrx/grafana/pkg/services/publicdashboards/models"
)

func UnmarshalDashboardAnnotations(sj *simplejson.Json) (*models.AnnotationsDto, error) {
	bytes, err := sj.MarshalJSON()
	if err != nil {
		return nil, err
	}
	dto := &models.AnnotationsDto{}
	err = json.Unmarshal(bytes, dto)
	if err != nil {
		return nil, err
	}

	return dto, err
}
