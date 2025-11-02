package util

import (
	"runtime/debug"
)

func GetVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	// get version from build info
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	// fallback to VCS info
	var revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value[:7]
		case "vcs.modified":
			if setting.Value == "true" {
				modified = "-dirty"
			}
		}
	}

	if revision != "" {
		return revision + modified
	}

	return "dev"
}
