package app

import (
	"runtime/debug"
	"strings"
)

const (
	Name    = "EDIForge"
	Command = "edi-json"
)

var (
	Version = "0.1.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if Version == "0.1.0-dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = strings.TrimPrefix(info.Main.Version, "v")
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if Commit == "unknown" && setting.Value != "" {
				Commit = shortCommit(setting.Value)
			}
		case "vcs.time":
			if Date == "unknown" && setting.Value != "" {
				Date = setting.Value
			}
		}
	}
}

func shortCommit(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
