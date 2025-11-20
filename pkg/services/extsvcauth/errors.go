package extsvcauth

import "github.com/capitalrx/grafana/pkg/apimachinery/errutil"

var (
	ErrUnknownProvider = errutil.BadRequest("extsvcauth.unknown-provider")
)
