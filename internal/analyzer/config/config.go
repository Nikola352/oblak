package config

type Config struct {
	// Infrastructure Engines
	OllamaURL    string
	AIModelName  string
	AntivirusURL string
	Minio        struct {
		Endpoint  string
		AccessKey string
		SecretKey string
	}
	DbConnectionString string

	// Local Host Tooling Execution Paths
	AuditBinaryPath string
	SastBinaryPath  string

	// Target Storage Reporting Parameters
	JSONReportOutputPath string
	AMQPURI              string
	SemgrepToken         string
}

var Cfg Config
