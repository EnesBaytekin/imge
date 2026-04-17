//go:build !sdl && !web && !mock
// +build !sdl,!web,!mock

// Package main provides a generic entry point for the IMGE game engine.
// Platform selection is done at build time using Go build tags:
//   - Default: go build ./cmd/game-engine
//   - SDL: go build -tags sdl ./cmd/game-engine
//   - Web: GOOS=js GOARCH=wasm go build -tags web ./cmd/game-engine
//   - Mock: go build -tags mock ./cmd/game-engine
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/json"
)

// platformFactory is the function signature for platform creation.
type platformFactory func() (core.Platform, error)

// defaultPlatformFactory is registered by platform-specific files via build tags.
var defaultPlatformFactory platformFactory

func main() {
	// Check if a platform factory was registered via build tags
	if defaultPlatformFactory == nil {
		log.Fatal("No platform factory registered. Build with appropriate platform tag:\n" +
			"  - SDL: go build -tags sdl\n" +
			"  - Web: GOOS=js GOARCH=wasm go build -tags web\n" +
			"  - Mock: go build -tags mock\n" +
			"  - Desktop (future): go build (default)")
	}

	// Load game configuration
	config, err := loadGameConfig()
	if err != nil {
		log.Fatalf("Failed to load game configuration: %v", err)
	}

	// Create platform implementation
	platform, err := defaultPlatformFactory()
	if err != nil {
		log.Fatalf("Failed to create platform: %v", err)
	}

	// Create and configure game
	game := core.NewGameWithConfig(core.Config{
		WindowWidth:  config.Window.Width,
		WindowHeight: config.Window.Height,
		WindowTitle:  config.Window.Title,
		TargetFPS:    config.Game.TargetFPS,
		InitialScene: config.Game.InitialScene,
	})

	game.SetPlatform(platform)

	// Initialize and run game
	if err := game.Init(); err != nil {
		log.Fatalf("Failed to initialize game: %v", err)
	}

	defer game.Shutdown()

	if err := game.Run(); err != nil {
		log.Fatalf("Game error: %v", err)
	}
}

// loadGameConfig searches for game.json in current or parent directory.
func loadGameConfig() (*json.GameConfig, error) {
	configPath := "game.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try parent directory (common for game projects)
		configPath = filepath.Join("..", "game.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("game.json not found in current or parent directory")
		}
	}

	return json.LoadGameConfig(configPath)
}