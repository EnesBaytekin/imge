package imge

import "embed"

// EngineSource embeds the pure-Go engine source that the imge CLI extracts when
// building a game. Only the CGO-free parts are embedded (core, engine, and the
// Ebitengine platform), so game builds never require SDL or a C compiler.
//
// The imge tool writes this out to a build directory and points the generated
// go.mod at it with a replace directive, which is what makes `imge build` work
// without a published engine release on GitHub.
//
//go:embed core engine platform/ebitengine
var EngineSource embed.FS
