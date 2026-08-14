package build

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My Game", "my-game"},
		{"IMGE Feature Demo", "imge-feature-demo"},
		{"  Hello   World  ", "hello-world"},
		{"Çocuk Oyunu", "ocuk-oyunu"},
		{"123", "123"},
		{"", "game"},
		{"!!!", "game"},
		{"Game!", "game"},
	}
	for _, c := range cases {
		if got := slugify(c.in); got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestCanBuildDesktop locks in the cross-compilation capability matrix,
// independent of the host this test runs on.
func TestCanBuildDesktop(t *testing.T) {
	cases := []struct {
		hostOS, hostArch, os, arch string
		want                       bool
	}{
		// native always works
		{"linux", "amd64", "linux", "amd64", true},
		{"linux", "arm64", "linux", "arm64", true},
		{"darwin", "amd64", "darwin", "amd64", true},
		{"darwin", "arm64", "darwin", "arm64", true},
		{"windows", "amd64", "windows", "amd64", true},

		// windows is pure Go: cross-compiles from anywhere
		{"linux", "amd64", "windows", "amd64", true},
		{"linux", "amd64", "windows", "arm64", true},
		{"darwin", "amd64", "windows", "amd64", true},
		{"darwin", "arm64", "windows", "arm64", true},
		{"windows", "amd64", "windows", "arm64", true},

		// macOS: both archs buildable from a mac host
		{"darwin", "amd64", "darwin", "arm64", true},
		{"darwin", "arm64", "darwin", "amd64", true},
		// but not from non-mac hosts
		{"linux", "amd64", "darwin", "arm64", false},
		{"linux", "arm64", "darwin", "amd64", false},
		{"windows", "amd64", "darwin", "amd64", false},

		// linux cross-arch: not without a cross C toolchain
		{"linux", "amd64", "linux", "arm64", false},
		{"linux", "arm64", "linux", "amd64", false},
		// linux from non-linux: not without a cross C toolchain
		{"darwin", "amd64", "linux", "amd64", false},
		{"windows", "amd64", "linux", "amd64", false},

		// windows only supports amd64/arm64
		{"linux", "amd64", "windows", "386", false},
	}
	for _, c := range cases {
		if got := canBuildDesktop(c.hostOS, c.hostArch, c.os, c.arch); got != c.want {
			t.Errorf("canBuildDesktop(%s/%s, %s/%s) = %v, want %v",
				c.hostOS, c.hostArch, c.os, c.arch, got, c.want)
		}
	}
}
