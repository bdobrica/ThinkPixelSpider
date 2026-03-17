package markdown

import "testing"

func TestNormalize_CollapsesBlankLines(t *testing.T) {
	input := "Line one\n\n\n\n\nLine two"
	got := Normalize(input)
	want := "Line one\n\nLine two"
	if got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
}

func TestNormalize_TrimsWhitespace(t *testing.T) {
	input := "  \n\n content \n\n  "
	got := Normalize(input)
	want := "content"
	if got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
}

func TestNormalize_NormalizesLineEndings(t *testing.T) {
	input := "line1\r\nline2\rline3\nline4"
	got := Normalize(input)
	want := "line1\nline2\nline3\nline4"
	if got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
}

func TestNormalize_PreservesSingleBlankLine(t *testing.T) {
	input := "para one\n\npara two"
	got := Normalize(input)
	want := "para one\n\npara two"
	if got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
}

func TestNormalize_EmptyString(t *testing.T) {
	got := Normalize("")
	if got != "" {
		t.Errorf("Normalize(\"\") = %q, want \"\"", got)
	}
}
