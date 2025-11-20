package clients

import "github.com/capitalrx/grafana/pkg/apimachinery/errutil"

const (
	basicPrefix             = "Basic "
	bearerPrefix            = "Bearer "
	authorizationHeaderName = "Authorization"
)

var (
	errIdentityNotFound = errutil.NotFound("identity.not-found")
)
