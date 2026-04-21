//go:build sdl
// +build sdl

// Package main registers the SDL platform factory when built with -tags sdl.
package main

import (
	"github.com/EnesBaytekin/imge/core"
	sdlplatform "github.com/EnesBaytekin/imge/platform/sdl"
)

func init() {
	defaultPlatformFactory = func() (core.Platform, error) {
		return sdlplatform.New()
	}
}