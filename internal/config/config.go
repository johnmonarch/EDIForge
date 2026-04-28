package config

type Config struct {
	Server      ServerConfig      `json:"server"`
	Translation TranslationConfig `json:"translation"`
	Schemas     SchemaConfig      `json:"schemas"`
	Privacy     PrivacyConfig     `json:"privacy"`
	Limits      LimitsConfig      `json:"limits"`
}

type ServerConfig struct {
	Host                         string `json:"host"`
	Port                         int    `json:"port"`
	Token                        string `json:"-"`
	RequireToken                 bool   `json:"requireToken"`
	RequireTokenOutsideLocalhost bool   `json:"requireTokenOutsideLocalhost"`
	MaxBodyMB                    int64  `json:"maxBodyMb"`
	CORSOrigin                   string `json:"corsOrigin,omitempty"`
}

type TranslationConfig struct {
	DefaultMode        string `json:"defaultMode"`
	IncludeEnvelope    bool   `json:"includeEnvelope"`
	IncludeRawSegments bool   `json:"includeRawSegments"`
}

type SchemaConfig struct {
	Paths []string `json:"paths"`
}

type PrivacyConfig struct {
	StoreHistory bool `json:"storeHistory"`
	Telemetry    bool `json:"telemetry"`
}

type LimitsConfig struct {
	MaxFileSizeMB int64 `json:"maxFileSizeMb"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Host:                         "127.0.0.1",
			Port:                         8765,
			RequireTokenOutsideLocalhost: true,
			MaxBodyMB:                    50,
		},
		Translation: TranslationConfig{
			DefaultMode:        "structural",
			IncludeEnvelope:    true,
			IncludeRawSegments: false,
		},
		Privacy: PrivacyConfig{
			StoreHistory: false,
			Telemetry:    false,
		},
		Limits: LimitsConfig{MaxFileSizeMB: 50},
	}
}

func IsLocalHost(host string) bool {
	return host == "" || host == "127.0.0.1" || host == "localhost" || host == "::1"
}
