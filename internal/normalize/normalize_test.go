package normalize

import (
	"strings"
	"testing"
)

func TestExtractFirstLabel(t *testing.T) {
	cases := map[string]string{
		"Example.com":      "Example",
		"example.com/path": "example",
		"example.":         "example",
		"  Café.Example  ": "Café",
		"singlelabel":      "singlelabel",
	}
	for in, want := range cases {
		if got := ExtractFirstLabel(in); got != want {
			t.Fatalf("ExtractFirstLabel(%q)=%q; want %q", in, got, want)
		}
	}
}

func TestNormalizeInput(t *testing.T) {
	ascii, uni, err := NormalizeInput("Café.example")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ascii != "xn--caf-dma" {
		t.Fatalf("ascii=%q; want %q", ascii, "xn--caf-dma")
	}
	if strings.ToLower(uni) != "café" { // display unicode in lowercase canonical form is acceptable
		t.Fatalf("unicode=%q; want lowercase %q", uni, "café")
	}

	// LDH invalid due to trailing hyphen
	if _, _, err := NormalizeInput("bad-"); err == nil {
		t.Fatalf("expected error for trailing hyphen")
	}
	// LDH invalid due to leading hyphen
	if _, _, err := NormalizeInput("-bad"); err == nil {
		t.Fatalf("expected error for leading hyphen")
	}
	// Slash invalid when not part of trimming (we trim path before); direct slash in label remains invalid
	if _, _, err := NormalizeInput("exa/mple"); err == nil {
		t.Fatalf("expected error for slash in label")
	}
}
