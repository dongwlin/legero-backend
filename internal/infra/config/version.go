package config

import "runtime"

// Build information, overridable at build time via -ldflags, e.g.:
//
//	go build -ldflags "-X github.com/dongwlin/legero-backend/internal/infra/config.Version=v1.0.0 //	-X github.com/dongwlin/legero-backend/internal/infra/config.Commit=abc123 //	-X github.com/dongwlin/legero-backend/internal/infra/config.BuildTime=2026-01-01T00:00:00Z"
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)
