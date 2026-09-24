// Package vdf parses Valve's text KeyValues format, used by .vdf and .acf files.
package vdf

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Node is one key in a KeyValues tree. A node has either a Value or Children;
// Children is non-nil for every block, including an empty one.
type Node struct {
	Key      string
	Value    string
	Children []*Node
}

// Get returns the first child whose key matches, ignoring case, or nil.
// It is nil-safe so lookups can be chained.
func (n *Node) Get(key string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if strings.EqualFold(c.Key, key) {
			return c
		}
	}
	return nil
}

// Path follows keys from n downward and returns the node found, or nil.
func (n *Node) Path(keys ...string) *Node {
	for _, k := range keys {
		n = n.Get(k)
	}
	return n
}

// Parse reads a KeyValues document. The returned root has an empty key and
// the document's top-level entries as children.
func Parse(r io.Reader) (*Node, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("vdf: read: %w", err)
	}
	p := &parser{data: bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF")), line: 1}
	children, err := p.parseBlock(0)
	if err != nil {
		return nil, err
	}
	return &Node{Children: children}, nil
}

// maxDepth guards against stack exhaustion on hostile input; real Steam files
// nest fewer than 10 levels.
const maxDepth = 256

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokString
	tokOpen
	tokClose
)

type token struct {
	kind   tokenKind
	text   string
	quoted bool
}

type parser struct {
	data []byte
	pos  int
	line int
}

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("vdf: line %d: %s", p.line, fmt.Sprintf(format, args...))
}

func (p *parser) parseBlock(depth int) ([]*Node, error) {
	if depth > maxDepth {
		return nil, p.errorf("nesting deeper than %d levels", maxDepth)
	}
	inside := depth > 0
	nodes := []*Node{} // non-nil, so an empty { } block differs from a "" value
	for {
		key, err := p.next()
		if err != nil {
			return nil, err
		}
		switch key.kind {
		case tokEOF:
			if inside {
				return nil, p.errorf("unexpected end of input, missing }")
			}
			return nodes, nil
		case tokClose:
			if !inside {
				return nil, p.errorf("unexpected }")
			}
			return nodes, nil
		case tokOpen:
			return nil, p.errorf("unexpected { without a key")
		}

		// Platform conditionals like [$WIN32] trail a value; they carry no data.
		if isConditional(key) {
			continue
		}

		val, err := p.next()
		if err != nil {
			return nil, err
		}
		switch val.kind {
		case tokString:
			nodes = append(nodes, &Node{Key: key.text, Value: val.text})
		case tokOpen:
			children, err := p.parseBlock(depth + 1)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &Node{Key: key.text, Children: children})
		default:
			return nil, p.errorf("key %q has no value", key.text)
		}
	}
}

func isConditional(t token) bool {
	return !t.quoted && strings.HasPrefix(t.text, "[") && strings.HasSuffix(t.text, "]")
}

func (p *parser) next() (token, error) {
	p.skipSpaceAndComments()
	if p.pos >= len(p.data) {
		return token{kind: tokEOF}, nil
	}
	switch c := p.data[p.pos]; c {
	case '{':
		p.pos++
		return token{kind: tokOpen}, nil
	case '}':
		p.pos++
		return token{kind: tokClose}, nil
	case '"':
		return p.quoted()
	default:
		return p.unquoted(), nil
	}
}

func (p *parser) skipSpaceAndComments() {
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		switch {
		case c == '\n':
			p.line++
			p.pos++
		case c == ' ' || c == '\t' || c == '\r':
			p.pos++
		case c == '/' && p.pos+1 < len(p.data) && p.data[p.pos+1] == '/':
			for p.pos < len(p.data) && p.data[p.pos] != '\n' {
				p.pos++
			}
		default:
			return
		}
	}
}

func (p *parser) quoted() (token, error) {
	startLine := p.line
	p.pos++ // opening quote
	start := p.pos

	// Fast path: most strings have no escapes and can be sliced directly.
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case '"':
			text := string(p.data[start:p.pos])
			p.pos++
			return token{kind: tokString, text: text, quoted: true}, nil
		case '\\':
			return p.quotedWithEscapes(start, startLine)
		case '\n':
			p.line++
		}
		p.pos++
	}
	return token{}, fmt.Errorf("vdf: line %d: unclosed quote", startLine)
}

func (p *parser) quotedWithEscapes(start, startLine int) (token, error) {
	var b strings.Builder
	b.Write(p.data[start:p.pos])
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		switch {
		case c == '"':
			p.pos++
			return token{kind: tokString, text: b.String(), quoted: true}, nil
		case c == '\\' && p.pos+1 < len(p.data):
			p.pos++
			switch e := p.data[p.pos]; e {
			case '"', '\\':
				b.WriteByte(e)
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				// Unknown escapes are kept as written.
				b.WriteByte('\\')
				b.WriteByte(e)
			}
		default:
			if c == '\n' {
				p.line++
			}
			b.WriteByte(c)
		}
		p.pos++
	}
	return token{}, fmt.Errorf("vdf: line %d: unclosed quote", startLine)
}

func (p *parser) unquoted() token {
	start := p.pos
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\r', '\n', '{', '}', '"':
			return token{kind: tokString, text: string(p.data[start:p.pos])}
		}
		p.pos++
	}
	return token{kind: tokString, text: string(p.data[start:p.pos])}
}
