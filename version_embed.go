package imge

import (
	_ "embed"
	"strings"
)

//go:embed version
var versionRaw string

// EngineVersion is the current IMGE engine version, read from the "version" file.
// Update the version file at repo root when bumping.
var EngineVersion = strings.TrimSpace(versionRaw)
