package login

import (
	"github.com/capitalrx/grafana/pkg/services/user"
)

type UserProtectionService interface {
	AllowUserMapping(user *user.User, authModule string) error
}
