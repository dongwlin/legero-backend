package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFromFileUsesNestedYAMLAndDefaults(t *testing.T) {
	clearConfigEnv(t)

	configPath := writeConfigFile(t, fmt.Sprintf(`
app:
  env: "production"
server:
  httpAddr: ":9090"
database:
  url: "postgres://example.com/legero"
auth:
  pasetoSymmetricKey: "%s"
  accessTokenTTL: "30m"
biz:
  timezone: "UTC"
realtime:
  heartbeatInterval: "10s"
ws:
  allowedOrigins:
    - "http://localhost:5173"
    - "capacitor://localhost"
argon2:
  iterations: 5
  parallelism: 4
`, encodedKey(t, "0123456789abcdef0123456789abcdef")))

	cfg, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("loadFromFile() error = %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, "production")
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9090")
	}
	if cfg.DatabaseURL != "postgres://example.com/legero" {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://example.com/legero")
	}
	if cfg.AccessTokenTTL != 30*time.Minute {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 30*time.Minute)
	}
	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Fatalf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 7*24*time.Hour)
	}
	if cfg.BizTimezone != "UTC" {
		t.Fatalf("BizTimezone = %q, want %q", cfg.BizTimezone, "UTC")
	}
	if cfg.RealtimeHeartbeatInterval != 10*time.Second {
		t.Fatalf("RealtimeHeartbeatInterval = %v, want %v", cfg.RealtimeHeartbeatInterval, 10*time.Second)
	}
	if cfg.RealtimeSessionTTL != 30*time.Second {
		t.Fatalf("RealtimeSessionTTL = %v, want %v", cfg.RealtimeSessionTTL, 30*time.Second)
	}
	if cfg.WSWriteTimeout != 10*time.Second {
		t.Fatalf("WSWriteTimeout = %v, want %v", cfg.WSWriteTimeout, 10*time.Second)
	}
	if cfg.WSReadTimeout != 60*time.Second {
		t.Fatalf("WSReadTimeout = %v, want %v", cfg.WSReadTimeout, 60*time.Second)
	}
	if len(cfg.WSAllowedOrigins) != 2 {
		t.Fatalf("WSAllowedOrigins length = %d, want %d", len(cfg.WSAllowedOrigins), 2)
	}
	if cfg.WSAllowedOrigins[0] != "http://localhost:5173" || cfg.WSAllowedOrigins[1] != "capacitor://localhost" {
		t.Fatalf("WSAllowedOrigins = %v, want configured origins", cfg.WSAllowedOrigins)
	}
	if cfg.Argon2.MemoryKiB != 64*1024 {
		t.Fatalf("Argon2.MemoryKiB = %d, want %d", cfg.Argon2.MemoryKiB, 64*1024)
	}
	if cfg.Argon2.Iterations != 5 {
		t.Fatalf("Argon2.Iterations = %d, want %d", cfg.Argon2.Iterations, 5)
	}
	if cfg.Argon2.Parallelism != 4 {
		t.Fatalf("Argon2.Parallelism = %d, want %d", cfg.Argon2.Parallelism, 4)
	}
	if cfg.Argon2.SaltLength != 16 {
		t.Fatalf("Argon2.SaltLength = %d, want %d", cfg.Argon2.SaltLength, 16)
	}
	if cfg.Argon2.KeyLength != 32 {
		t.Fatalf("Argon2.KeyLength = %d, want %d", cfg.Argon2.KeyLength, 32)
	}
}

func TestLoadFromFileEnvOverridesYAML(t *testing.T) {
	clearConfigEnv(t)

	configPath := writeConfigFile(t, fmt.Sprintf(`
app:
  env: "development"
server:
  httpAddr: ":8080"
database:
  url: "postgres://file.example/legero"
auth:
  pasetoSymmetricKey: "%s"
  accessTokenTTL: "15m"
  refreshTokenTTL: "720h"
argon2:
  iterations: 3
`, encodedKey(t, "0123456789abcdef0123456789abcdef")))

	overrideKey := encodedKey(t, "abcdefghijklmnopqrstuvwxzy012345")
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_ADDR", ":9191")
	t.Setenv("DATABASE_URL", "postgres://env.example/legero")
	t.Setenv("PASETO_SYMMETRIC_KEY", overrideKey)
	t.Setenv("ACCESS_TOKEN_TTL", "45m")
	t.Setenv("REALTIME_HEARTBEAT_INTERVAL", "25s")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com,capacitor://localhost")
	t.Setenv("ARGON2_ITERATIONS", "9")

	cfg, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("loadFromFile() error = %v", err)
	}

	if cfg.AppEnv != "staging" {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, "staging")
	}
	if cfg.HTTPAddr != ":9191" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9191")
	}
	if cfg.DatabaseURL != "postgres://env.example/legero" {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://env.example/legero")
	}
	if cfg.AccessTokenTTL != 45*time.Minute {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 45*time.Minute)
	}
	if cfg.RealtimeHeartbeatInterval != 25*time.Second {
		t.Fatalf("RealtimeHeartbeatInterval = %v, want %v", cfg.RealtimeHeartbeatInterval, 25*time.Second)
	}
	if len(cfg.WSAllowedOrigins) != 2 {
		t.Fatalf("WSAllowedOrigins length = %d, want %d", len(cfg.WSAllowedOrigins), 2)
	}
	if cfg.WSAllowedOrigins[0] != "https://app.example.com" || cfg.WSAllowedOrigins[1] != "capacitor://localhost" {
		t.Fatalf("WSAllowedOrigins = %v, want env override", cfg.WSAllowedOrigins)
	}
	if cfg.Argon2.Iterations != 9 {
		t.Fatalf("Argon2.Iterations = %d, want %d", cfg.Argon2.Iterations, 9)
	}
	wantKey, err := base64.StdEncoding.DecodeString(overrideKey)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	if string(cfg.PasetoSymmetricKey) != string(wantKey) {
		t.Fatal("PasetoSymmetricKey did not use env override")
	}
}

func TestLoadFromFileInvalidValuesFallbackToDefaults(t *testing.T) {
	clearConfigEnv(t)

	configPath := writeConfigFile(t, fmt.Sprintf(`
database:
  url: "postgres://example.com/legero"
auth:
  pasetoSymmetricKey: "%s"
  accessTokenTTL: "not-a-duration"
  refreshTokenTTL: "still-not-a-duration"
realtime:
  heartbeatInterval: "bad"
  sessionTTL: "still-bad"
ws:
  writeTimeout: "bad"
  readTimeout: "bad"
  allowedOrigins: ""
argon2:
  memoryKiB: "bad"
  iterations: "bad"
  parallelism: "bad"
  saltLength: "bad"
  keyLength: "bad"
`, encodedKey(t, "0123456789abcdef0123456789abcdef")))

	cfg, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("loadFromFile() error = %v", err)
	}

	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 15*time.Minute)
	}
	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Fatalf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 7*24*time.Hour)
	}
	if cfg.RealtimeHeartbeatInterval != 20*time.Second {
		t.Fatalf("RealtimeHeartbeatInterval = %v, want %v", cfg.RealtimeHeartbeatInterval, 20*time.Second)
	}
	if cfg.RealtimeSessionTTL != 30*time.Second {
		t.Fatalf("RealtimeSessionTTL = %v, want %v", cfg.RealtimeSessionTTL, 30*time.Second)
	}
	if cfg.WSWriteTimeout != 10*time.Second {
		t.Fatalf("WSWriteTimeout = %v, want %v", cfg.WSWriteTimeout, 10*time.Second)
	}
	if cfg.WSReadTimeout != 60*time.Second {
		t.Fatalf("WSReadTimeout = %v, want %v", cfg.WSReadTimeout, 60*time.Second)
	}
	if len(cfg.WSAllowedOrigins) != 1 || cfg.WSAllowedOrigins[0] != "*" {
		t.Fatalf("WSAllowedOrigins = %v, want [*]", cfg.WSAllowedOrigins)
	}
	if cfg.Argon2.MemoryKiB != 64*1024 {
		t.Fatalf("Argon2.MemoryKiB = %d, want %d", cfg.Argon2.MemoryKiB, 64*1024)
	}
	if cfg.Argon2.Iterations != 3 {
		t.Fatalf("Argon2.Iterations = %d, want %d", cfg.Argon2.Iterations, 3)
	}
	if cfg.Argon2.Parallelism != 2 {
		t.Fatalf("Argon2.Parallelism = %d, want %d", cfg.Argon2.Parallelism, 2)
	}
	if cfg.Argon2.SaltLength != 16 {
		t.Fatalf("Argon2.SaltLength = %d, want %d", cfg.Argon2.SaltLength, 16)
	}
	if cfg.Argon2.KeyLength != 32 {
		t.Fatalf("Argon2.KeyLength = %d, want %d", cfg.Argon2.KeyLength, 32)
	}
}

func TestLoadFromFileRequiresDatabaseURL(t *testing.T) {
	clearConfigEnv(t)

	configPath := writeConfigFile(t, fmt.Sprintf(`
auth:
  pasetoSymmetricKey: "%s"
`, encodedKey(t, "0123456789abcdef0123456789abcdef")))

	_, err := loadFromFile(configPath)
	if err == nil {
		t.Fatal("loadFromFile() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("loadFromFile() error = %v, want DATABASE_URL is required", err)
	}
}

func TestLoadFromFileRejectsInvalidPasetoKey(t *testing.T) {
	clearConfigEnv(t)

	configPath := writeConfigFile(t, `
database:
  url: "postgres://example.com/legero"
auth:
  pasetoSymmetricKey: "c2hvcnQ="
`)

	_, err := loadFromFile(configPath)
	if err == nil {
		t.Fatal("loadFromFile() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "must decode to 32 bytes") {
		t.Fatalf("loadFromFile() error = %v, want length validation error", err)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, envName := range []string{
		"APP_ENV",
		"HTTP_ADDR",
		"DATABASE_URL",
		"PASETO_SYMMETRIC_KEY",
		"ACCESS_TOKEN_TTL",
		"REFRESH_TOKEN_TTL",
		"BIZ_TIMEZONE",
		"REALTIME_HEARTBEAT_INTERVAL",
		"REALTIME_SESSION_TTL",
		"WS_WRITE_TIMEOUT",
		"WS_READ_TIMEOUT",
		"WS_ALLOWED_ORIGINS",
		"ARGON2_MEMORY_KIB",
		"ARGON2_ITERATIONS",
		"ARGON2_PARALLELISM",
		"ARGON2_SALT_LENGTH",
		"ARGON2_KEY_LENGTH",
	} {
		t.Setenv(envName, "")
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return configPath
}

func encodedKey(t *testing.T, raw string) string {
	t.Helper()

	if len(raw) != 32 {
		t.Fatalf("raw key length = %d, want 32", len(raw))
	}
	return base64.StdEncoding.EncodeToString([]byte(raw))
}
