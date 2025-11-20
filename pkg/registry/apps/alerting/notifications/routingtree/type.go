package routingtree

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	model "github.com/capitalrx/grafana/apps/alerting/notifications/pkg/apis/alerting/v0alpha1"
	"github.com/capitalrx/grafana/pkg/apimachinery/utils"
)

var kind = model.RoutingTreeKind()
var ResourceInfo = utils.NewResourceInfo(kind.Group(), kind.Version(),
	kind.GroupVersionResource().Resource, strings.ToLower(kind.Kind()), kind.Kind(),
	func() runtime.Object { return kind.ZeroValue() },
	func() runtime.Object { return kind.ZeroListValue() },
	utils.TableColumns{},
)
