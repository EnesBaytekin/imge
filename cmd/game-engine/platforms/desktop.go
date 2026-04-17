//go:build !sdl && !web && !mock
// +build !sdl,!web,!mock

// Package main registers the default platform factory (currently mock).
// In the future, this will register a proper desktop platform (GLFW/raylib/etc).
package main

import (
	"github.com/EnesBaytekin/imge/internal/core"
	mockplatform "github.com/EnesBaytekin/imge/platform/mock"
)

func init() {
	// For now, use mock platform as default until proper desktop platform is implemented
	defaultPlatformFactory = func() (core.Platform, error) {
		return mockplatform.New(), nil
	}
}