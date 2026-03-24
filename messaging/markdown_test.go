package messaging

import "testing"

func TestMarkdownToPlainText_CodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	want := "fmt.Println(\"hello\")"
	got := MarkdownToPlainText(input)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarkdownToPlainText_InlineCode(t *testing.T) {
	input := "use `fmt.Println` to print"
	want := "use fmt.Println to print"
	got := MarkdownToPlainText(input)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarkdownToPlainText_Image(t *testing.T) {
	input := "see ![alt](http://example.com/img.png) here"
	want := "see  here"
	got := MarkdownToPlainText(input)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarkdownToPlainText_Link(t *testing.T) {
	input := "visit [Google](https://google.com) now"
	want := "visit Google now"
	got := MarkdownToPlainText(input)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarkdownToPlainText_Table(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |"
	got := MarkdownToPlainText(input)
	if got == "" {
		t.Error("expected non-empty output for table")
	}
	// Table separator should be removed
	if contains(got, "---") {
		t.Errorf("table separator not removed: %q", got)
	}
}

func TestMarkdownToPlainText_Header(t *testing.T) {
	input := "## Hello World"
	want := "Hello World"
	got := MarkdownToPlainText(input)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarkdownToPlainText_Bold(t *testing.T) {
	input := "this is **bold** text"
	want := "this is bold text"
	got := MarkdownToPlainText(input)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarkdownToPlainText_Mixed(t *testing.T) {
	input := "# Title\n\n**bold** and `code` with [link](http://x.com)\n\n> quote"
	got := MarkdownToPlainText(input)
	if got == "" {
		t.Error("expected non-empty output")
	}
	if contains(got, "**") || contains(got, "`") || contains(got, "](") {
		t.Errorf("markdown artifacts remain: %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && containsStr(s, sub)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
