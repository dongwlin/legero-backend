package config

import "github.com/spf13/viper"

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "development")
	v.SetDefault("server.httpAddr", ":8080")
	v.SetDefault("auth.accessTokenTTL", "15m")
	v.SetDefault("auth.refreshTokenTTL", "168h")
	v.SetDefault("biz.timezone", "Asia/Shanghai")
	v.SetDefault("sse.pingInterval", "20s")
	v.SetDefault("argon2.memoryKiB", 64*1024)
	v.SetDefault("argon2.iterations", 3)
	v.SetDefault("argon2.parallelism", 2)
	v.SetDefault("argon2.saltLength", 16)
	v.SetDefault("argon2.keyLength", 32)
}
