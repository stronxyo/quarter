package vdf

import (
	"strings"
	"testing"
)

// render flattens a tree to a compact string: key=value;key{...}
func render(n *Node) string {
	var b strings.Builder
	for i, c := range n.Children {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(c.Key)
		if c.Children != nil {
			b.WriteByte('{')
			b.WriteString(render(c))
			b.WriteByte('}')
		} else {
			b.WriteByte('=')
			b.WriteString(c.Value)
		}
	}
	return b.String()
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"pair", `"a" "1"`, "a=1"},
		{"nested", `"root" { "a" "1" "b" { "c" "2" } }`, "root{a=1;b{c=2}}"},
		{"comments", "// header\n\"a\" \"1\" // trailing\n", "a=1"},
		{"escapes", `"p" "C:\\Games\\Steam" "q" "say \"hi\"" "n" "x\ny\tz"`, "p=C:\\Games\\Steam;q=say \"hi\";n=x\ny\tz"},
		{"unknown escape kept", `"a" "\d"`, `a=\d`},
		{"unquoted tokens", "root { a 1 b 2 }", "root{a=1;b=2}"},
		{"duplicate keys kept in order", `"a" "1" "a" "2"`, "a=1;a=2"},
		{"crlf", "\"root\"\r\n{\r\n\t\"a\"\t\t\"1\"\r\n}\r\n", "root{a=1}"},
		{"bom", "\xEF\xBB\xBF\"a\" \"1\"", "a=1"},
		{"conditional skipped", `"a" "1" [$WIN32] "b" "2"`, "a=1;b=2"},
		{"empty value", `"a" ""`, "a="},
		{"empty block", `"a" { }`, "a{}"},
		{"multiline quoted", "\"a\" \"one\ntwo\"", "a=one\ntwo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := Parse(strings.NewReader(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := render(root); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"unclosed brace", `"root" { "a" "1"`},
		{"unclosed quote", `"a" "1`},
		{"unclosed quote with escape", `"a" "x\"`},
		{"stray close", `"a" "1" }`},
		{"open without key", `{ "a" "1" }`},
		{"key without value", `"root" { "a" }`},
		{"key at end of input", `"a"`},
		{"too deep", strings.Repeat(`"k" {`, maxDepth+2)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(tt.in)); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestErrorHasLineNumber(t *testing.T) {
	_, err := Parse(strings.NewReader("\"a\" \"1\"\n\"b\" \"2\"\n}"))
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("want error mentioning line 3, got %v", err)
	}
}

func TestGetAndPath(t *testing.T) {
	root, err := Parse(strings.NewReader(`"Store" { "Software" { "Valve" { "Apps" { "620" { "Playtime" "90" } } } } }`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := root.Path("store", "SOFTWARE", "valve", "apps", "620", "playtime"); got == nil || got.Value != "90" {
		t.Errorf("case-insensitive Path: got %v, want Value 90", got)
	}
	if got := root.Path("Store", "Missing", "Anything"); got != nil {
		t.Errorf("missing Path: got %v, want nil", got)
	}

	if got := root.Path("Store", "Software", "Valve", "Apps", "620").ValueOf("PLAYTIME"); got != "90" {
		t.Errorf("ValueOf: got %q, want 90", got)
	}

	var nilNode *Node
	if got := nilNode.Get("x"); got != nil {
		t.Errorf("Get on nil: got %v, want nil", got)
	}
	if got := nilNode.Entries(); got != nil {
		t.Errorf("Entries on nil: got %v, want nil", got)
	}
	if got := nilNode.ValueOf("x"); got != "" {
		t.Errorf("ValueOf on nil: got %q, want empty", got)
	}
}
