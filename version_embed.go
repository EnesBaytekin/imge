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

// buildTag is an optional per-build suffix, empty by default. It is injected at link
// time for a local development build (see build.sh), e.g.
//
//	go build -ldflags "-X github.com/EnesBaytekin/imge.buildTag=<suffix>" ./cmd/imge
//
// so every rebuild carries a fresh short marker even though EngineVersion is unchanged.
// A release built at a tagged commit passes no tag and keeps the plain "vX.Y.Z" form.
var buildTag string

// BuildVersion is the version reported at runtime: EngineVersion plus "-<buildTag>"
// when a development suffix was injected, otherwise the plain EngineVersion.
var BuildVersion = computeBuildVersion()

func computeBuildVersion() string {
	if buildTag != "" {
		return EngineVersion + "-" + buildTag
	}
	return EngineVersion
}
