package config

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultConfigDir  = "config"
	defaultConfigName = "config"
	defaultConfigType = "yaml"
)

type Config struct {
	AppEnv                    string
	HTTPAddr                  string
	DatabaseURL               string
	PasetoSymmetricKey        []byte
	AccessTokenTTL            time.Duration
	RefreshTokenTTL           time.Duration
	BizTimezone               string
	RealtimeHeartbeatInterval time.Duration
	RealtimeSessionTTL        time.Duration
	WSWriteTimeout            time.Duration
	WSReadTimeout             time.Duration
	WSAllowedOrigins          []string
	Argon2                    Argon2Config
}

type Argon2Config struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func Load() (*Config, error) {
	return loadFromDir(defaultConfigDir, defaultConfigName, defaultConfigType)
}

func loadFromDir(dir, name, configType string) (*Config, error) {
	v, err := newViper()
	if err != nil {
		return nil, err
	}

	v.SetConfigName(name)
	v.SetConfigType(configType)
	v.AddConfigPath(dir)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %q from %q: %w", name+"."+configType, dir, err)
	}

	return buildConfig(v)
}

func loadFromFile(path string) (*Config, error) {
	v, err := newViper()
	if err != nil {
		return nil, err
	}

	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	return buildConfig(v)
}

func newViper() (*viper.Viper, error) {
	v := viper.New()
	setDefaults(v)
	if err := bindEnv(v); err != nil {
		return nil, err
	}
	return v, nil
}

func bindEnv(v *viper.Viper) error {
	bindings := map[string]string{
		"app.env":                    "APP_ENV",
		"server.httpAddr":            "HTTP_ADDR",
		"database.url":               "DATABASE_URL",
		"auth.pasetoSymmetricKey":    "PASETO_SYMMETRIC_KEY",
		"auth.accessTokenTTL":        "ACCESS_TOKEN_TTL",
		"auth.refreshTokenTTL":       "REFRESH_TOKEN_TTL",
		"biz.timezone":               "BIZ_TIMEZONE",
		"realtime.heartbeatInterval": "REALTIME_HEARTBEAT_INTERVAL",
		"realtime.sessionTTL":        "REALTIME_SESSION_TTL",
		"ws.writeTimeout":            "WS_WRITE_TIMEOUT",
		"ws.readTimeout":             "WS_READ_TIMEOUT",
		"ws.allowedOrigins":          "WS_ALLOWED_ORIGINS",
		"argon2.memoryKiB":           "ARGON2_MEMORY_KIB",
		"argon2.iterations":          "ARGON2_ITERATIONS",
		"argon2.parallelism":         "ARGON2_PARALLELISM",
		"argon2.saltLength":          "ARGON2_SALT_LENGTH",
		"argon2.keyLength":           "ARGON2_KEY_LENGTH",
	}

	for key, envName := range bindings {
		if err := v.BindEnv(key, envName); err != nil {
			return fmt.Errorf("bind %s to %s: %w", key, envName, err)
		}
	}

	return nil
}

func buildConfig(v *viper.Viper) (*Config, error) {
	databaseURL, err := requiredValue(v, "database.url", "DATABASE_URL")
	if err != nil {
		return nil, err
	}

	keyText, err := requiredValue(v, "auth.pasetoSymmetricKey", "PASETO_SYMMETRIC_KEY")
	if err != nil {
		return nil, err
	}

	keyBytes, err := decodeKey(keyText)
	if err != nil {
		return nil, fmt.Errorf("decode PASETO_SYMMETRIC_KEY: %w", err)
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("PASETO_SYMMETRIC_KEY must decode to 32 bytes, got %d", len(keyBytes))
	}

	return &Config{
		AppEnv:                    stringOrDefault(v, "app.env", "development"),
		HTTPAddr:                  stringOrDefault(v, "server.httpAddr", ":8080"),
		DatabaseURL:               databaseURL,
		PasetoSymmetricKey:        keyBytes,
		AccessTokenTTL:            durationOrDefault(v, "auth.accessTokenTTL", 15*time.Minute),
		RefreshTokenTTL:           durationOrDefault(v, "auth.refreshTokenTTL", 7*24*time.Hour),
		BizTimezone:               stringOrDefault(v, "biz.timezone", "Asia/Shanghai"),
		RealtimeHeartbeatInterval: durationOrDefault(v, "realtime.heartbeatInterval", 20*time.Second),
		RealtimeSessionTTL:        durationOrDefault(v, "realtime.sessionTTL", 30*time.Second),
		WSWriteTimeout:            durationOrDefault(v, "ws.writeTimeout", 10*time.Second),
		WSReadTimeout:             durationOrDefault(v, "ws.readTimeout", 60*time.Second),
		WSAllowedOrigins:          stringSliceOrDefault(v, "ws.allowedOrigins", []string{"*"}),
		Argon2: Argon2Config{
			MemoryKiB:   uint32OrDefault(v, "argon2.memoryKiB", 64*1024),
			Iterations:  uint32OrDefault(v, "argon2.iterations", 3),
			Parallelism: uint8(uint32OrDefault(v, "argon2.parallelism", 2)),
			SaltLength:  uint32OrDefault(v, "argon2.saltLength", 16),
			KeyLength:   uint32OrDefault(v, "argon2.keyLength", 32),
		},
	}, nil
}

func requiredValue(v *viper.Viper, path, envName string) (string, error) {
	value := stringValue(v, path)
	if value == "" {
		return "", fmt.Errorf("%s is required", envName)
	}
	return value, nil
}

func stringOrDefault(v *viper.Viper, path, fallback string) string {
	value := stringValue(v, path)
	if value == "" {
		return fallback
	}
	return value
}

func durationOrDefault(v *viper.Viper, path string, fallback time.Duration) time.Duration {
	value := stringValue(v, path)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func uint32OrDefault(v *viper.Viper, path string, fallback uint32) uint32 {
	value := stringValue(v, path)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return fallback
	}
	return uint32(parsed)
}

func stringValue(v *viper.Viper, path string) string {
	value := v.Get(path)
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func stringSliceOrDefault(v *viper.Viper, path string, fallback []string) []string {
	copyFallback := append([]string(nil), fallback...)

	value := v.Get(path)
	switch typed := value.(type) {
	case nil:
		return copyFallback
	case []string:
		if cleaned := cleanStringSlice(typed); len(cleaned) > 0 {
			return cleaned
		}
		return copyFallback
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprint(item))
		}
		if cleaned := cleanStringSlice(items); len(cleaned) > 0 {
			return cleaned
		}
		return copyFallback
	case string:
		if cleaned := cleanStringSlice(strings.Split(typed, ",")); len(cleaned) > 0 {
			return cleaned
		}
		return copyFallback
	default:
		return copyFallback
	}
}

func cleanStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func decodeKey(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}

	decoded, rawErr := base64.RawStdEncoding.DecodeString(value)
	if rawErr == nil {
		return decoded, nil
	}

	return nil, err
}
