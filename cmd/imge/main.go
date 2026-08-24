package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/EnesBaytekin/imge"
	"github.com/EnesBaytekin/imge/build"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "build":
		handleBuild()
	case "run":
		handleRun()
	case "init":
		handleInit()
	case "new":
		handleNew()
	case "version":
		fmt.Printf("imge version %s\n", imge.EngineVersion)
	case "help", "-h", "--help":
		printUsage()
	default:
		log.Printf("Unknown command: %s", command)
		printUsage()
		os.Exit(1)
	}
}

func handleBuild() {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	linux := fs.Bool("linux", false, "build for Linux")
	windows := fs.Bool("windows", false, "build for Windows")
	macos := fs.Bool("macos", false, "build for macOS")
	amd64 := fs.Bool("amd64", false, "build for amd64 architecture")
	arm64 := fs.Bool("arm64", false, "build for arm64 architecture")
	web := fs.Bool("web", false, "build the web (WASM) bundle")
	all := fs.Bool("all", false, "build every supported target")
	fs.Usage = printUsage
	_ = fs.Parse(os.Args[2:])

	projectDir := requireProjectDir()

	// Resolve the requested desktop platforms (OS x arch) plus whether to build web.
	var requested []build.Platform
	var wantWeb bool

	anyOS := *linux || *windows || *macos
	anyArch := *amd64 || *arm64
	posWeb := len(fs.Args()) > 0 && normalizeTarget(fs.Args()[0]) == build.TargetWeb

	switch {
	case *all:
		requested = platformCombos([]string{"linux", "darwin", "windows"}, []string{"amd64", "arm64"})
		wantWeb = true
	case !anyOS && !anyArch && !*web && !posWeb:
		// Default: native desktop for this machine.
		requested = []build.Platform{build.HostPlatform()}
	case !anyOS && !anyArch && (*web || posWeb):
		wantWeb = true
	default:
		var oses, archs []string
		if anyOS {
			if *linux {
				oses = append(oses, "linux")
			}
			if *windows {
				oses = append(oses, "windows")
			}
			if *macos {
				oses = append(oses, "darwin")
			}
		} else {
			oses = []string{"linux", "darwin", "windows"}
		}
		if anyArch {
			if *amd64 {
				archs = append(archs, "amd64")
			}
			if *arm64 {
				archs = append(archs, "arm64")
			}
		} else {
			archs = []string{"amd64", "arm64"}
		}
		requested = platformCombos(oses, archs)
		wantWeb = *web || posWeb
	}

	// Build each requested target, continuing past failures so one broken target
	// (e.g. a native Linux build on a host missing X11 dev headers) doesn't abort
	// the rest of an --all run. Targets this host fundamentally can't produce are
	// "skipped"; targets we attempted but that errored are "failed".
	built, skipped, failed := 0, 0, 0
	for _, p := range requested {
		if err := build.ValidateTarget(p.GOOS, p.GOARCH); err != nil {
			fmt.Printf("Skipping %s/%s: %v\n", p.GOOS, p.GOARCH, err)
			skipped++
			continue
		}
		b := &build.Builder{ProjectDir: projectDir, Target: build.TargetDesktop, GOOS: p.GOOS, GOARCH: p.GOARCH}
		if _, err := b.Build(); err != nil {
			fmt.Printf("Failed to build %s/%s: %v\n", p.GOOS, p.GOARCH, err)
			failed++
			continue
		}
		built++
	}
	if wantWeb {
		b := &build.Builder{ProjectDir: projectDir, Target: build.TargetWeb}
		if _, err := b.Build(); err != nil {
			fmt.Printf("Failed to build web: %v\n", err)
			failed++
		} else {
			built++
		}
	}

	fmt.Printf("\nBuild summary: %d built, %d skipped, %d failed.\n", built, skipped, failed)
	if built == 0 {
		log.Fatal("Nothing was built: none of the requested targets succeeded.")
	}
}

// platformCombos returns the cartesian product of the given OSes and
// architectures as desktop build targets.
func platformCombos(oses, archs []string) []build.Platform {
	var ps []build.Platform
	for _, o := range oses {
		for _, a := range archs {
			ps = append(ps, build.Platform{GOOS: o, GOARCH: a})
		}
	}
	return ps
}

// normalizeTarget maps a user-provided target string to a build target.
func normalizeTarget(s string) string {
	switch s {
	case "web", "wasm":
		return build.TargetWeb
	case "desktop", "ebitengine":
		return build.TargetDesktop
	default:
		log.Fatalf("Invalid target %q. Supported: desktop (default), web", s)
		return ""
	}
}

func handleRun() {
	// `imge run` builds and runs natively. `imge run web` just builds the bundle.
	if len(os.Args) >= 3 && (os.Args[2] == "web" || os.Args[2] == "wasm") {
		handleBuild()
		return
	}

	projectDir := requireProjectDir()

	builder := &build.Builder{ProjectDir: projectDir, Target: build.TargetDesktop}
	outPath, err := builder.Build()
	if err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	fmt.Printf("Running: %s\n", outPath)

	cmd := exec.Command(outPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatalf("Game exited with error: %v", err)
	}
}

// requireProjectDir returns the current directory, failing if it isn't a project.
func requireProjectDir() string {
	projectDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}
	if _, err := build.FindGameFile(projectDir); err != nil {
		log.Fatal("No game.imge found in this directory. Run `imge init` first.")
	}
	return projectDir
}

func handleInit() {
	// An optional positional argument selects the sample template. `imge init`
	// alone scaffolds a blank project; `imge init sample` scaffolds the demo.
	sample := false
	if len(os.Args) >= 3 {
		switch os.Args[2] {
		case "sample", "demo", "example":
			sample = true
		default:
			log.Fatalf("Unknown init template %q. Use `imge init` (blank) or `imge init sample`.", os.Args[2])
		}
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		log.Fatalf("Failed to read current directory: %v", err)
	}
	if len(entries) > 0 {
		fmt.Fprintln(os.Stderr, "Refusing to initialize: this directory is not empty.")
		fmt.Fprintln(os.Stderr, "Run `imge init` in an empty directory so it doesn't overwrite existing files.")
		os.Exit(1)
	}

	if sample {
		fmt.Println("Initializing sample platformer project...")
		if err := imge.ExtractSampleTemplate("."); err != nil {
			log.Fatalf("Failed to create project files: %v", err)
		}
		fmt.Println("\nSample project initialized successfully!")
		fmt.Println("Next steps:")
		fmt.Println("  1. Build and run: imge run")
		fmt.Println("  2. Move with A/D (or ←/→), jump with Space — break crates, collect the gold")
		fmt.Println("  3. Study components/ to learn how custom components work")
		return
	}

	fmt.Println("Initializing new IMGE game project...")
	if err := imge.ExtractBlankTemplate("."); err != nil {
		log.Fatalf("Failed to create project files: %v", err)
	}
	fmt.Println("\nProject initialized successfully!")
	fmt.Println("Next steps:")
	fmt.Println("  1. Build and run: imge run")
	fmt.Println("  2. Add objects to scenes/main.scene")
	fmt.Println("  3. Create a component: imge new component <name>")
}

func printUsage() {
	fmt.Printf("IMGE Minimal Game Engine CLI Tool (version %s)\n", imge.EngineVersion)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  imge init                 Initialize a blank game project (empty directory only)")
	fmt.Println("  imge init sample          Initialize the sample platformer demo")
	fmt.Println("  imge build [flags]        Build the game")
	fmt.Println("  imge run                  Build and run natively (desktop)")
	fmt.Println("  imge new <kind> <name>    Create a blank template file")
	fmt.Println("  imge version              Show engine version")
	fmt.Println("  imge help                 Show this help")
	fmt.Println()
	fmt.Println("  <kind>   object (.obj) | component (.go) | scene (.scene)")
	fmt.Println("  <name>   filename; may include a relative path (e.g. objects/player)")
	fmt.Println()
	fmt.Println("Build flags (combine freely):")
	fmt.Println("  --linux --windows --macos   target OS (omit with --arch to target every OS)")
	fmt.Println("  --amd64 --arm64             target architecture (omit to build both)")
	fmt.Println("  --web                       build the web (WASM) bundle")
	fmt.Println("  --all                       build every supported target")
	fmt.Println()
	fmt.Println("Output goes to imge_build/<name>_<os>-<arch> (web to imge_build/web/).")
}
