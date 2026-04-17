//go:build mock
// +build mock

// Package main registers the mock platform factory when built with -tags mock.
package main

import (
	"github.com/EnesBaytekin/imge/core"
	mockplatform "github.com/EnesBaytekin/imge/platform/mock"
)

func init() {
	defaultPlatformFactory = func() (core.Platform, error) {
		return mockplatform.New(), nil
	}
}