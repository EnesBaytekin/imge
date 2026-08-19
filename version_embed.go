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

// CurrentFormatVersion is the IMGE project-format version this build understands.
// It pins the schema of game.imge, the scene/object JSON shape, the component-kind
// scheme, and the path-resolution rules. Bump it only when one of those changes
// incompatibly — never for a routine engine release. Projects declare their own
// format_version and are validated against this at build time.
const CurrentFormatVersion = 1
