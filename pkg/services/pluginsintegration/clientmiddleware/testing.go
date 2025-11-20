package clientmiddleware

import (
	"net/http"
	"net/http/httptest"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/capitalrx/grafana/pkg/services/contexthandler/ctxkey"
	contextmodel "github.com/capitalrx/grafana/pkg/services/contexthandler/model"
	"github.com/capitalrx/grafana/pkg/services/user"
	"github.com/capitalrx/grafana/pkg/web"
)

func WithReqContext(req *http.Request, user *user.SignedInUser) handlertest.HandlerMiddlewareTestOption {
	return handlertest.HandlerMiddlewareTestOption(func(cdt *handlertest.HandlerMiddlewareTest) {
		reqContext := &contextmodel.ReqContext{
			Context: &web.Context{
				Req:  req,
				Resp: web.NewResponseWriter(req.Method, httptest.NewRecorder()),
			},
			SignedInUser: user,
		}

		ctx := ctxkey.Set(req.Context(), reqContext)
		*req = *req.WithContext(ctx)
	})
}

var nopCallResourceSender = backend.CallResourceResponseSenderFunc(func(res *backend.CallResourceResponse) error {
	return nil
})
