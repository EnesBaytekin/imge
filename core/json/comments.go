package json

// StripComments removes // line comments and /* */ block comments from JSON
// data, so .obj, .scene, and game.imge files can carry explanatory comments
// (JSONC-style). String literals are respected, so comment markers inside a
// value such as "http://example.com" are left untouched.
func StripComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false

	for i := 0; i < len(data); {
		c := data[i]

		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			i++
			continue
		}

		switch c {
		case '"':
			inString = true
			out = append(out, c)
			i++
		case '/':
			if i+1 < len(data) && data[i+1] == '/' {
				// Line comment: drop through the end of the line.
				i += 2
				for i < len(data) && data[i] != '\n' {
					i++
				}
			} else if i+1 < len(data) && data[i+1] == '*' {
				// Block comment: drop through the closing */.
				i += 2
				for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
					i++
				}
				if i+1 < len(data) {
					i += 2
				} else {
					i = len(data)
				}
			} else {
				out = append(out, c)
				i++
			}
		default:
			out = append(out, c)
			i++
		}
	}

	return out
}
