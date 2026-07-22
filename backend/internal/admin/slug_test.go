package admin

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Python Classic Tee":     "python-classic-tee",
		"  Next.js Edge Hoodie ": "next-js-edge-hoodie",
		"C++ Tee":                "c-tee",
		"Tailwind   CSS":         "tailwind-css",
		"---weird---":            "weird",
		"Go/Rust":                "go-rust",
		"":                       "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeStatus(t *testing.T) {
	for _, s := range []string{"active", "draft", "archived"} {
		if normalizeStatus(s) != s {
			t.Errorf("normalizeStatus(%q) changed a valid value", s)
		}
	}
	if normalizeStatus("bogus") != "draft" {
		t.Errorf("normalizeStatus should default unknown values to draft")
	}
}
