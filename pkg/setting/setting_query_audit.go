package setting

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/gtime"
	"gopkg.in/ini.v1"
)

const defaultQueryAuditFlushInterval = time.Hour

// QueryAuditSettings configures the datasource query audit log. Every value can be overridden with
// the standard GF_QUERY_AUDIT_<KEY> environment variables.
type QueryAuditSettings struct {
	// Enabled is true when S3Bucket is set.
	Enabled  bool
	S3Bucket string
	S3Prefix string
	// S3Region is optional, the AWS SDK default region chain is used when empty.
	S3Region      string
	FlushInterval time.Duration
}

func readQueryAuditSettings(iniFile *ini.File, cfg *Cfg) error {
	section := iniFile.Section("query_audit")

	s := QueryAuditSettings{
		S3Bucket: strings.TrimSpace(section.Key("s3_bucket").String()),
		S3Prefix: strings.Trim(strings.TrimSpace(section.Key("s3_prefix").String()), "/"),
		S3Region: strings.TrimSpace(section.Key("s3_region").String()),
	}
	s.Enabled = s.S3Bucket != ""

	interval, err := gtime.ParseDuration(valueAsString(section, "flush_interval", defaultQueryAuditFlushInterval.String()))
	if err != nil {
		return fmt.Errorf("query_audit: invalid flush_interval: %w", err)
	}
	if interval <= 0 {
		return fmt.Errorf("query_audit: flush_interval must be positive, got %s", interval)
	}
	s.FlushInterval = interval

	cfg.QueryAudit = s
	return nil
}
