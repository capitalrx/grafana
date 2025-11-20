package service

import (
	"github.com/capitalrx/grafana/pkg/services/dashboards"
	"github.com/capitalrx/grafana/pkg/services/featuremgmt"
)

func ProvideDashboardService(
	features featuremgmt.FeatureToggles,
	orig *DashboardServiceImpl,
) dashboards.DashboardService {
	return orig
}

func ProvideDashboardProvisioningService(
	features featuremgmt.FeatureToggles, orig *DashboardServiceImpl,
) dashboards.DashboardProvisioningService {
	return orig
}

func ProvideDashboardPluginService(
	features featuremgmt.FeatureToggles, orig *DashboardServiceImpl,
) dashboards.PluginService {
	return orig
}
