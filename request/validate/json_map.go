package validate

import "strings"

// jsonToMap parses a flat JSON object (string, bool, and number values
// only - no nesting, no arrays) into a map[string]string. It exists to
// avoid pulling gjson and reflect (via encoding/json) into the WASM build.
// Booleans and numbers are stored as their literal text ("true", "2").
func jsonToMap(json string) map[string]string {
	m := make(map[string]string, 32)
	p := jsonParser{s: json}
	p.skipSpace()
	if p.pos < len(p.s) && p.s[p.pos] == '{' {
		p.pos++
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.s) || p.s[p.pos] == '}' {
			break
		}
		if p.s[p.pos] == ',' {
			p.pos++
			p.skipSpace()
		}
		if p.pos >= len(p.s) || p.s[p.pos] != '"' {
			break
		}
		key := p.parseString()
		p.skipSpace()
		if p.pos < len(p.s) && p.s[p.pos] == ':' {
			p.pos++
		}
		p.skipSpace()
		m[key] = p.parseValue()
	}
	return m
}

type jsonParser struct {
	s   string
	pos int
}

func (p *jsonParser) skipSpace() {
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// parseValue reads a string, true, false, null, or bare number token,
// returning its text form.
func (p *jsonParser) parseValue() string {
	if p.pos >= len(p.s) {
		return ""
	}
	switch {
	case p.s[p.pos] == '"':
		return p.parseString()
	case strings.HasPrefix(p.s[p.pos:], "true"):
		p.pos += 4
		return "true"
	case strings.HasPrefix(p.s[p.pos:], "false"):
		p.pos += 5
		return "false"
	case strings.HasPrefix(p.s[p.pos:], "null"):
		p.pos += 4
		return ""
	default:
		start := p.pos
		for p.pos < len(p.s) && p.s[p.pos] != ',' && p.s[p.pos] != '}' && p.s[p.pos] != ' ' {
			p.pos++
		}
		return p.s[start:p.pos]
	}
}

// parseString reads a JSON string starting at the opening quote and
// returns its unescaped contents, handling the common escapes JSON.stringify
// produces (\", \\, \/, \n, \t, \r); \uXXXX escapes are not expected from
// simple form field values and are passed through unchanged.
func (p *jsonParser) parseString() string {
	p.pos++ // skip opening quote
	var sb strings.Builder
	start := p.pos
	for p.pos < len(p.s) && p.s[p.pos] != '"' {
		if p.s[p.pos] == '\\' && p.pos+1 < len(p.s) {
			sb.WriteString(p.s[start:p.pos])
			p.pos++
			switch p.s[p.pos] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			case '/':
				sb.WriteByte('/')
			default:
				sb.WriteByte(p.s[p.pos])
			}
			p.pos++
			start = p.pos
			continue
		}
		p.pos++
	}
	sb.WriteString(p.s[start:p.pos])
	if p.pos < len(p.s) {
		p.pos++ // skip closing quote
	}
	return sb.String()
}
