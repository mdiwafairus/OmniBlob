import os

with open('internal/config/config.go', 'r') as f:
    text = f.read()

text = text.replace(
    '\tDatabase  DatabaseConfig  `yaml:"database" mapstructure:"database"`',
    '\tReconciliation ReconciliationConfig `yaml:"reconciliation" mapstructure:"reconciliation"`\n\tDatabase  DatabaseConfig  `yaml:"database" mapstructure:"database"`'
)

text = text.replace(
    '\tAllowedExtensions []string     `yaml:"allowed_extensions" mapstructure:"allowed_extensions"`\n\tClients         []ClientConfig `yaml:"clients" mapstructure:"clients"`',
    '\tAllowedExtensions []string       `yaml:"allowed_extensions" mapstructure:"allowed_extensions"`\n\tRateLimitRPM      int            `yaml:"rate_limit_rpm" mapstructure:"rate_limit_rpm"`\n\tClients           []ClientConfig `yaml:"clients" mapstructure:"clients"`'
)

text = text.replace(
    'type DatabaseConfig struct {',
    'type ReconciliationConfig struct {\n\tEnabled     bool `yaml:"enabled" mapstructure:"enabled"`\n\tBatchSize   int  `yaml:"batch_size" mapstructure:"batch_size"`\n\tIntervalSec int  `yaml:"interval_sec" mapstructure:"interval_sec"`\n}\n\ntype DatabaseConfig struct {'
)

text = text.replace(
    '\t\t\tcfg.Migration.Enabled = b\n\t\t}\n\t}',
    '\t\t\tcfg.Migration.Enabled = b\n\t\t}\n\t}\n\tif reconciliationEnabled := os.Getenv("RECONCILIATION_ENABLED"); reconciliationEnabled != "" {\n\t\tif b, err := strconv.ParseBool(reconciliationEnabled); err == nil {\n\t\t\tcfg.Reconciliation.Enabled = b\n\t\t}\n\t}'
)

with open('internal/config/config.go', 'w') as f:
    f.write(text)

with open('internal/api/router.go', 'r') as f:
    text = f.read()

text = text.replace(
    '\thandler = auditMiddleware(handler, secCfg, auditLogger)\n\thandler = corsMiddleware(handler)',
    '\thandler = auditMiddleware(handler, secCfg, auditLogger)\n\thandler = rateLimitMiddleware(cfg)(handler)\n\thandler = corsMiddleware(handler)'
)

with open('internal/api/router.go', 'w') as f:
    f.write(text)

with open('internal/api/handler.go', 'r') as f:
    text = f.read()

text = text.replace(
    '\t\tmaxUploadMB = 100\n\t}\n\treturn &Handler{',
    '\t\tmaxUploadMB = 100\n\t}\n\n\texts := make([]string, 0, len(allowedExtensions))\n\tfor _, ext := range allowedExtensions {\n\t\texts = append(exts, strings.ToLower(strings.TrimSpace(ext)))\n\t}\n\n\treturn &Handler{'
)

text = text.replace(
    '\t\tallowedExtensions: allowedExtensions,',
    '\t\tallowedExtensions: exts,'
)

with open('internal/api/handler.go', 'w') as f:
    f.write(text)
