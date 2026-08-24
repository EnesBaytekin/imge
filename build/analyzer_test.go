package build

import (
	"testing"

	"github.com/EnesBaytekin/imge"
	corejson "github.com/EnesBaytekin/imge/core/json"
)

func TestValidateFormatVersionDefaultsToCurrent(t *testing.T) {
	c := &corejson.GameConfig{Name: "x"} // FormatVersion 0 = "original format"
	if err := validateFormatVersion(c); err != nil {
		t.Fatal(err)
	}
	if c.FormatVersion != imge.CurrentFormatVersion {
		t.Fatalf("format_version = %d, want %d", c.FormatVersion, imge.CurrentFormatVersion)
	}
}

func TestValidateFormatVersionRejectsNewer(t *testing.T) {
	c := &corejson.GameConfig{Name: "x", FormatVersion: imge.CurrentFormatVersion + 1}
	if err := validateFormatVersion(c); err == nil {
		t.Fatal("expected error for a newer format_version, got nil")
	}
}

// An "older than current" version can only exist once CurrentFormatVersion is
// bumped above 1; until then the only lower value (0) means "absent" and is
// defaulted rather than rejected, so that branch has no testable input yet.
