package config

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

func Load() {
	// 1. Core AI / Inspection Services Engines Configuration
	flag.StringVar(&Cfg.OllamaURL, "ollama-url",
		GetEnv("OLLAMA_URL", "http://localhost:11434"),
		"Ollama API Engine URL",
	)
	flag.StringVar(&Cfg.AIModelName, "ai-model-name",
		GetEnv("AI_MODEL_NAME", "qwen2.5:3b"),
		"Target Ollama Large Language Model Variant identifier tag",
	)
	flag.StringVar(&Cfg.AntivirusURL, "antivirus-url",
		GetEnv("ANTIVIRUS_URL", "tcp://localhost:3310"),
		"ClamAV Engine Daemon TCP Endpoint",
	)

	// 2. Object Storage Configuration (MinIO)
	flag.StringVar(&Cfg.Minio.Endpoint, "minio-endpoint",
		GetEnv("MINIO_ENDPOINT", "localhost:9000"),
		"MinIO Cluster Endpoint URL Address",
	)
	flag.StringVar(&Cfg.Minio.AccessKey, "minio-access-key",
		GetEnv("MINIO_ACCESS_KEY", "minioadmin"),
		"MinIO Root/Service Identification Access Key User",
	)
	flag.StringVar(&Cfg.Minio.SecretKey, "minio-secret-key",
		GetEnv("MINIO_SECRET_KEY", "minioadmin"),
		"MinIO Access Verification Secret Key Token",
	)

	// 3. Relational Persistence Storage Configuration
	flag.StringVar(&Cfg.DbConnectionString, "database_url",
		GetEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/oblak"),
		"Target Data Warehouse Connection Access String Parameter URI",
	)

	// 4. Host System Security Application Binaries Scopes
	flag.StringVar(&Cfg.AuditBinaryPath, "auditor-path",
		GetEnv("AUDITOR_PATH", "/home/nikola-velemir/faks/rbs/oblak/.venv/bin/pip-audit"),
		"Absolute host execution system path pointing toward the pip-audit binary executable",
	)
	flag.StringVar(&Cfg.SastBinaryPath, "sast-path",
		GetEnv("SAST_PATH", "/home/nikola-velemir/faks/rbs/oblak/.venv/bin/semgrep"),
		"Absolute host execution system path pointing toward the Semgrep scanning core engine",
	)

	// 5. Audit Results Target Delivery Specs
	flag.StringVar(&Cfg.JSONReportOutputPath, "json-report-output-path",
		GetEnv("JSON_REPORT_OUTPUT_PATH", "/tmp/reports/"),
		"Host File System Directory Route destination folder target where security scan summaries drop",
	)

	flag.StringVar(&Cfg.SemgrepToken, "semgrep-token",
		GetEnv("SEMGREP_APP_TOKEN", "prazno"),
		"Semgrep secret key",
	)

	flag.Parse()

	validate()
}

func validate() {
	// Guard against unconfigured database runtimes
	if Cfg.DbConnectionString == "" {
		log.Fatal("[CONFIG CRITICAL] A non-empty db-connection-string / DATABASE_URL parameter is strictly required to bootstrap execution context structures.")
	}

	// Verify local dependency binaries exist before launching listeners
	verifyBinaryExists(Cfg.AuditBinaryPath, "pip-audit-path / PIP_AUDIT_PATH")
	verifyBinaryExists(Cfg.SastBinaryPath, "semgrep-path / SEMGREP_PATH")

	// Pre-emptively fix path quirks to clean trailing slashes out
	Cfg.JSONReportOutputPath = filepath.Clean(Cfg.JSONReportOutputPath)
}

func verifyBinaryExists(path string, parameterName string) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Fatalf("[CONFIG CRITICAL] System application validation error: The path specified for '%s' inside target execution matrix (%s) does not exist on this host system file architecture tree.", parameterName, path)
	}
	if info.IsDir() {
		log.Fatalf("[CONFIG CRITICAL] System application validation error: Configuration parameter '%s' points to a directory path location target instead of an explicit executable file target blueprint link.", parameterName)
	}
}
