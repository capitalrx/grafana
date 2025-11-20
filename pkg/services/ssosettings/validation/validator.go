package validation

import (
	"github.com/capitalrx/grafana/pkg/apimachinery/identity"
	"github.com/capitalrx/grafana/pkg/login/social"
	"github.com/capitalrx/grafana/pkg/services/ssosettings"
)

func Validate(info *social.OAuthInfo, requester identity.Requester, validators ...ssosettings.ValidateFunc[social.OAuthInfo]) error {
	for _, validatorFunc := range validators {
		if err := validatorFunc(info, requester); err != nil {
			return err
		}
	}
	return nil
}
