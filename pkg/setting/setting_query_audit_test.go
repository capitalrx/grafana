package setting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"
)

func TestReadQueryAuditSettings(t *testing.T) {
	t.Run("disabled without a bucket", func(t *testing.T) {
		cfg := NewCfg()
		require.NoError(t, readQueryAuditSettings(ini.Empty(), cfg))
		require.False(t, cfg.QueryAudit.Enabled)
		require.Equal(t, time.Hour, cfg.QueryAudit.FlushInterval)
	})

	t.Run("enabled with a bucket", func(t *testing.T) {
		f, err := ini.Load([]byte(`
[query_audit]
s3_bucket = observability-logs-cap-rx-com
s3_prefix = /services/grafana_audit/
flush_interval = 15m
`))
		require.NoError(t, err)
		cfg := NewCfg()
		require.NoError(t, readQueryAuditSettings(f, cfg))
		require.Equal(t, QueryAuditSettings{
			Enabled:       true,
			S3Bucket:      "observability-logs-cap-rx-com",
			S3Prefix:      "services/grafana_audit",
			FlushInterval: 15 * time.Minute,
		}, cfg.QueryAudit)
	})

	t.Run("rejects a non-positive interval", func(t *testing.T) {
		f, err := ini.Load([]byte("[query_audit]\nflush_interval = 0s\n"))
		require.NoError(t, err)
		require.Error(t, readQueryAuditSettings(f, NewCfg()))
	})

	t.Run("environment variables override defaults", func(t *testing.T) {
		t.Setenv("GF_QUERY_AUDIT_S3_BUCKET", "env-bucket")
		t.Setenv("GF_QUERY_AUDIT_FLUSH_INTERVAL", "30m")
		cfg := NewCfg()
		require.NoError(t, cfg.Load(CommandLineArgs{HomePath: "../../"}))
		require.True(t, cfg.QueryAudit.Enabled)
		require.Equal(t, "env-bucket", cfg.QueryAudit.S3Bucket)
		require.Equal(t, 30*time.Minute, cfg.QueryAudit.FlushInterval)
	})
}
