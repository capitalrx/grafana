package dashboards

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/grafana/grafana/pkg/util/testutil"
)

const v2ProvisionedDashboardTemplate = `{
  "apiVersion": "dashboard.grafana.app/v2beta1",
  "kind": "Dashboard",
  "metadata": { "name": %q },
  "spec": {
    "title": %q,
    "annotations": [],
    "cursorSync": "Off",
    "preload": false,
    "links": [],
    "tags": [],
    "variables": [],
    "timeSettings": { "from": "now-6h", "to": "now", "autoRefresh": "", "autoRefreshIntervals": [], "hideTimepicker": false, "fiscalYearStartMonth": 0 },
    "elements": {},
    "layout": {
      "kind": "TabsLayout",
      "spec": { "tabs": [ { "kind": "TabsLayoutTab", "spec": { "title": "Overview", "layout": { "kind": "GridLayout", "spec": { "items": [] } } } } ] }
    }
    %s
  }
}`

// Classic file provisioning must accept v2 dashboard resources next to legacy JSON
// dashboards in the same folder, and keep them stored as v2.
func TestIntegrationProvisioningV2DashboardsWithClassicDashboards(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	unifiedConfig := map[string]setting.UnifiedStorageConfig{}
	for _, resource := range []string{"folders.folder.grafana.app", "dashboards.dashboard.grafana.app"} {
		unifiedConfig[resource] = setting.UnifiedStorageConfig{EnableMigration: true}
	}
	dir, path := testinfra.CreateGrafDir(t, testinfra.GrafanaOpts{
		DisableAnonymous:     true,
		UnifiedStorageConfig: unifiedConfig,
	})

	provDir := filepath.Join(dir, "conf", "provisioning", "dashboards")
	filesDir := filepath.Join(provDir, "files")
	teamDir := filepath.Join(filesDir, "team")
	require.NoError(t, os.MkdirAll(teamDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(provDir, "dev.yaml"), []byte(fmt.Sprintf(`
apiVersion: 1
providers:
- name: 'mixed'
  type: file
  updateIntervalSeconds: 1
  options:
    path: %s
    foldersFromFilesStructure: true`, filesDir)), 0o600))

	writeFile := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(teamDir, name), []byte(content), 0o600))
	}
	writeFile("classic.json", `{"uid": "classic-uid", "title": "Classic dashboard", "schemaVersion": 41, "panels": []}`)
	writeFile("v2.json", fmt.Sprintf(v2ProvisionedDashboardTemplate, "v2-uid", "V2 dashboard", ""))

	addr, _ := testinfra.StartGrafanaEnv(t, dir, path)

	get := func(version, uid string) (int, map[string]any) {
		u := fmt.Sprintf("http://admin:admin@%s/apis/dashboard.grafana.app/%s/namespaces/default/dashboards/%s", addr, version, uid)
		// nolint:gosec
		resp, err := http.Get(u)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		out := map[string]any{}
		if resp.StatusCode == http.StatusOK {
			require.NoError(t, json.Unmarshal(body, &out))
		}
		return resp.StatusCode, out
	}
	nested := func(obj map[string]any, keys ...string) any {
		var cur any = obj
		for _, k := range keys {
			m, ok := cur.(map[string]any)
			if !ok {
				return nil
			}
			cur = m[k]
		}
		return cur
	}

	var classic, v2 map[string]any
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var status int
		status, classic = get("v1beta1", "classic-uid")
		assert.Equal(c, http.StatusOK, status)
		status, v2 = get("v2beta1", "v2-uid")
		assert.Equal(c, http.StatusOK, status)
	}, 20*time.Second, 100*time.Millisecond)

	t.Run("v2 dashboard is stored as v2 with its v2-only layout", func(t *testing.T) {
		assert.Equal(t, "V2 dashboard", nested(v2, "spec", "title"))
		assert.Equal(t, "TabsLayout", nested(v2, "spec", "layout", "kind"))
		assert.Nil(t, nested(v2, "spec", "id"))
		assert.Nil(t, nested(v2, "spec", "uid"))
		assert.Nil(t, nested(v2, "spec", "version"))

		_, viaV0 := get("v0alpha1", "v2-uid")
		assert.Contains(t, nested(viaV0, "status", "conversion", "storedVersion"), "v2beta1")
	})

	t.Run("both dashboards are managed by the provisioner and share the folder", func(t *testing.T) {
		for _, obj := range []map[string]any{classic, v2} {
			assert.Equal(t, string(utils.ManagerKindClassicFP), nested(obj, "metadata", "annotations", utils.AnnoKeyManagerKind)) // nolint:staticcheck
			assert.Equal(t, "mixed", nested(obj, "metadata", "annotations", utils.AnnoKeyManagerIdentity))
		}
		folderUID := nested(classic, "metadata", "annotations", utils.AnnoKeyFolder)
		assert.NotEmpty(t, folderUID)
		assert.Equal(t, folderUID, nested(v2, "metadata", "annotations", utils.AnnoKeyFolder))
	})

	t.Run("changes to the v2 file are applied and invalid v2 files are rejected", func(t *testing.T) {
		writeFile("invalid.json", fmt.Sprintf(v2ProvisionedDashboardTemplate, "invalid-uid", "Invalid", `, "notAV2Field": true`))
		writeFile("v2.json", fmt.Sprintf(v2ProvisionedDashboardTemplate, "v2-uid", "V2 dashboard updated", ""))

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			_, obj := get("v2beta1", "v2-uid")
			assert.Equal(c, "V2 dashboard updated", nested(obj, "spec", "title"))
		}, 20*time.Second, 100*time.Millisecond)

		// both files are handled in the same provisioning pass as the update above
		status, _ := get("v2beta1", "invalid-uid")
		assert.Equal(t, http.StatusNotFound, status)
	})

	t.Run("removing the v2 file deletes the dashboard", func(t *testing.T) {
		require.NoError(t, os.Remove(filepath.Join(teamDir, "v2.json")))

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			status, _ := get("v2beta1", "v2-uid")
			assert.Equal(c, http.StatusNotFound, status)
		}, 20*time.Second, 100*time.Millisecond)

		status, _ := get("v1beta1", "classic-uid")
		assert.Equal(t, http.StatusOK, status)
	})
}
