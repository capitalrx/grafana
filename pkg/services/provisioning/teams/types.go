package teams

import (
	"github.com/capitalrx/grafana/pkg/services/provisioning/values"
)

type TeamsConfig struct {
	APIVersion int64         `yaml:"apiVersion"`
	Teams      []*TeamConfig `yaml:"teams"`
}

type TeamConfig struct {
	Name  values.StringValue `yaml:"name"`
	Email values.StringValue `yaml:"email"`
	OrgID values.Int64Value  `yaml:"orgId"`
}
