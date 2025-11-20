package kinds

import (
	"github.com/capitalrx/grafana/apps/iam/kinds/v0alpha1"
)

teambindingKind: {
	kind:       "TeamBinding"
	pluralName: "TeamBindings"
	codegen: {
		ts: { enabled: false }
		go: { enabled: true }
	}
}

teambindingv0alpha1: teambindingKind & {
	schema: {
		spec: v0alpha1.TeamBindingSpec
	}
}
