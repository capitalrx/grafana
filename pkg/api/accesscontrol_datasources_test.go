package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatasourceReaderGrantsFollowEnforcement(t *testing.T) {
	grantsFor := func(enforced bool) []string {
		for _, r := range FixedRoleRegistrations(enforced) {
			if r.Role.Name == datasourcesReaderRoleName {
				return r.Grants
			}
		}
		t.Fatal("datasources reader role not registered")
		return nil
	}

	require.Equal(t, []string{"Viewer"}, grantsFor(false))
	require.Equal(t, []string{"Admin"}, grantsFor(true))
}
