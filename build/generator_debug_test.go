package build

import (
	"os"
	"strings"
	"testing"
)

// mainTemplateData is the data shape both main.go templates render against. It
// mirrors the anonymous structs in generateDesktopMainGo / generateWebMainGo, plus
// the Debug field that selects the debug-overlay line in loadScenes.
type mainTemplateData struct {
	ModuleName         string
	WindowTitle        string
	WindowWidth        int
	WindowHeight       int
	WindowFullscreen   bool
	WindowResizable    bool
	WindowPixelPerUnit int
	WindowScale        int
	WindowSmoothShapes bool
	TargetFPS          int
	InitialScene       string
	EmbedDirective     string
	HasData            bool
	Debug              bool
}

// TestMainTemplateDebugFlag verifies both entrypoint templates render without a
// template error and that the debug overlay line appears exactly when Debug is set.
func TestMainTemplateDebugFlag(t *testing.T) {
	data := mainTemplateData{
		ModuleName:     "demo_build",
		WindowTitle:    "demo",
		WindowWidth:    320,
		WindowHeight:   180,
		TargetFPS:      60,
		InitialScene:   "main",
		EmbedDirective: "//go:embed all:project\n",
		HasData:        true,
	}

	for _, tc := range []struct {
		tmpl    string
		name    string
		hasData bool
	}{
		{mainTemplateDesktop, "desktop", true},
		{mainTemplateWeb, "web", true},
	} {
		g := &Generator{BuildDir: t.TempDir()}

		// Debug off: the overlay line must be absent.
		data.Debug = false
		if err := g.renderTemplate(tc.tmpl, data); err != nil {
			t.Fatalf("%s template render (debug off): %v", tc.name, err)
		}
		out, _ := os.ReadFile(g.BuildDir + "/main.go")
		if strings.Contains(string(out), "SetDebugDraw") {
			t.Fatalf("%s: debug-off output should not enable the overlay", tc.name)
		}

		// Debug on: the overlay line must be present.
		data.Debug = true
		if err := g.renderTemplate(tc.tmpl, data); err != nil {
			t.Fatalf("%s template render (debug on): %v", tc.name, err)
		}
		out, _ = os.ReadFile(g.BuildDir + "/main.go")
		if !strings.Contains(string(out), "scene.SetDebugDraw(true)") {
			t.Fatalf("%s: debug-on output missing scene.SetDebugDraw(true):\n%s", tc.name, string(out))
		}
	}
}
