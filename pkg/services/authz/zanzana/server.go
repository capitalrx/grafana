package zanzana

import (
	"net/http"

	openfgaserver "github.com/openfga/openfga/pkg/server"
	openfgastorage "github.com/openfga/openfga/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/capitalrx/grafana/pkg/infra/log"
	"github.com/capitalrx/grafana/pkg/infra/tracing"
	"github.com/capitalrx/grafana/pkg/services/authz/zanzana/server"
	"github.com/capitalrx/grafana/pkg/services/grpcserver"
	"github.com/capitalrx/grafana/pkg/setting"
)

func NewServer(cfg setting.ZanzanaServerSettings, openfga server.OpenFGAServer, logger log.Logger, tracer tracing.Tracer, reg prometheus.Registerer) (*server.Server, error) {
	return server.NewServer(cfg, openfga, logger, tracer, reg)
}

func NewHealthServer(target server.DiagnosticServer) *server.HealthServer {
	return server.NewHealthServer(target)
}

func NewOpenFGAServer(cfg setting.ZanzanaServerSettings, store openfgastorage.OpenFGADatastore) (*openfgaserver.Server, error) {
	return server.NewOpenFGAServer(cfg, store)
}

func NewOpenFGAHttpServer(cfg setting.ZanzanaServerSettings, srv grpcserver.Provider) (*http.Server, error) {
	return server.NewOpenFGAHttpServer(cfg, srv)
}
