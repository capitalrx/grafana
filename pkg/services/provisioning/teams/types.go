package teams

import (
	"github.com/grafana/grafana/pkg/services/provisioning/values"
)

type TeamsConfig struct {
	APIVersion int64         `yaml:"apiVersion"`
	Teams      []*TeamConfig `yaml:"teams"`
}

type TeamConfig struct {
	Name  values.StringValue `yaml:"name"`
	Email values.StringValue `yaml:"email"`
	OrgID values.Int64Value  `yaml:"orgId"`
	// Members are emails (or logins). Users that do not exist yet are created
	// without an org role and are adopted by the auth provider on first login.
	Members []values.StringValue `yaml:"members"`
}
