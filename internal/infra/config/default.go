package config

import "github.com/spf13/viper"

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "development")
	v.SetDefault("server.httpAddr", ":8080")
	v.SetDefault("auth.accessTokenTTL", "15m")
	v.SetDefault("auth.refreshTokenTTL", "168h")
	v.SetDefault("biz.timezone", "Asia/Shanghai")
	v.SetDefault("realtime.heartbeatInterval", "20s")
	v.SetDefault("realtime.sessionTTL", "30s")
	v.SetDefault("ws.writeTimeout", "10s")
	v.SetDefault("ws.readTimeout", "60s")
	v.SetDefault("ws.allowedOrigins", []string{"*"})
}
