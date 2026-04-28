package config

type Config struct {
	Server ServerConfig `json:"server"`
	Limits LimitsConfig `json:"limits"`
}

type ServerConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Token        string `json:"-"`
	RequireToken bool   `json:"requireToken"`
	MaxBodyMB    int64  `json:"maxBodyMb"`
	CORSOrigin   string `json:"corsOrigin,omitempty"`
}

type LimitsConfig struct {
	MaxFileSizeMB int64 `json:"maxFileSizeMb"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Host:      "127.0.0.1",
			Port:      8765,
			MaxBodyMB: 50,
		},
		Limits: LimitsConfig{MaxFileSizeMB: 50},
	}
}

func IsLocalHost(host string) bool {
	return host == "" || host == "127.0.0.1" || host == "localhost" || host == "::1"
}
