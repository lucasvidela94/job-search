package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"table", FormatTable},
		{"plain", FormatPlain},
		{"", FormatJSON},
		{"unknown", FormatJSON},
	}
	for _, tt := range tests {
		got := ParseFormat(tt.input)
		if got != tt.want {
			t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]int{"a": 1}
	err := WriteJSON(&buf, data)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, `"a"`) || !strings.Contains(got, `1`) {
		t.Errorf("WriteJSON output missing expected content: %s", got)
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer
	data := [][]string{
		{"ID", "TITLE", "COMPANY"},
		{"1", "Engineer", "Acme"},
		{"2", "Designer", "Beta"},
	}
	err := WriteTable(&buf, data)
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "ID") || !strings.Contains(got, "Engineer") {
		t.Errorf("WriteTable missing content:\n%s", got)
	}
}

func TestWriteTableNonSlice(t *testing.T) {
	var buf bytes.Buffer
	err := WriteTable(&buf, "not a slice")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "not a slice") {
		t.Errorf("WriteTable fallback failed: %s", got)
	}
}

func TestWriteError(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, fmt.Errorf("something broke"), "TEST_ERR")
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, "TEST_ERR") || !strings.Contains(got, "something broke") {
		t.Errorf("WriteError output: %s", got)
	}
}
