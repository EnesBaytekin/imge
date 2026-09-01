package main

import (
	"flag"
	"testing"
)

func TestSplitFlags(t *testing.T) {
	flags, positional := splitFlags([]string{"web", "--debug"})
	if len(positional) != 1 || positional[0] != "web" {
		t.Fatalf("positional = %v, want [web]", positional)
	}
	if len(flags) != 1 || flags[0] != "--debug" {
		t.Fatalf("flags = %v, want [--debug]", flags)
	}

	// Flag before the positional works the same.
	flags, positional = splitFlags([]string{"--debug", "web"})
	if len(flags) != 1 || flags[0] != "--debug" {
		t.Fatalf("flags = %v, want [--debug]", flags)
	}
	if len(positional) != 1 || positional[0] != "web" {
		t.Fatalf("positional = %v, want [web]", positional)
	}

	// No positional at all.
	flags, positional = splitFlags([]string{"--debug"})
	if len(flags) != 1 || flags[0] != "--debug" {
		t.Fatalf("flags = %v, want [--debug]", flags)
	}
	if len(positional) != 0 {
		t.Fatalf("positional = %v, want empty", positional)
	}
}

// TestBuildDebugAfterTarget reproduces the `imge build web --debug` bug: the
// stdlib flag package stops parsing at the first positional argument, so a
// --debug that appeared after the `web` target was silently dropped. splitFlags +
// fs.Parse(flags) must honor the flag regardless of argument order.
func TestBuildDebugAfterTarget(t *testing.T) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	debug := fs.Bool("debug", false, "")
	web := fs.Bool("web", false, "")

	flags, positional := splitFlags([]string{"web", "--debug"})
	if err := fs.Parse(flags); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !*debug {
		t.Fatal("--debug after the `web` target should set debug")
	}
	if *web {
		t.Fatal("the --web flag must not be set by the `web` positional")
	}
	if len(positional) != 1 || positional[0] != "web" {
		t.Fatalf("positional = %v, want [web]", positional)
	}
}
