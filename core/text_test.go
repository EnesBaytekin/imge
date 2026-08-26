package core

import (
	"encoding/json"
	"testing"
)

func TestWrapModeUnmarshalJSON(t *testing.T) {
	cases := []struct {
		in   string
		want WrapMode
		err  bool
	}{
		{`"word"`, WrapWord, false},
		{`"char"`, WrapChar, false},
		{`"clip"`, WrapClip, false},
		{`"WORD"`, WrapWord, false}, // case-insensitive
		{`"Clip"`, WrapClip, false},
		{`1`, 0, true}, // numbers are not accepted
		{`"bogus"`, 0, true},
		{`true`, 0, true},
	}
	for _, c := range cases {
		var got WrapMode
		err := json.Unmarshal([]byte(c.in), &got)
		if c.err {
			if err == nil {
				t.Errorf("Unmarshal(%s): expected error, got %d", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Unmarshal(%s): unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Unmarshal(%s) = %d, want %d", c.in, got, c.want)
		}
	}
}
