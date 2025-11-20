package signingkeys

import "github.com/capitalrx/grafana/pkg/apimachinery/errutil"

var (
	ErrSigningKeyNotFound      = errutil.NotFound("signingkeys.keyNotFound")
	ErrSigningKeyAlreadyExists = errutil.BadRequest("signingkeys.keyAlreadyExists")
	ErrKeyGenerationFailed     = errutil.Internal("signingkeys.keyGenerationFailed")
)
