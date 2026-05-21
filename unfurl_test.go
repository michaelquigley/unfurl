package unfurl

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnfurlCopiesInputToOutput(t *testing.T) {
	src := "one\nwrapped paragraph\n\n```go\nfmt.Println(\"preserved\")\n```\n"
	want := "one wrapped paragraph\n\n```go\nfmt.Println(\"preserved\")\n```\n"
	var out bytes.Buffer

	if err := Unfurl(strings.NewReader(src), &out); err != nil {
		t.Fatalf("Unfurl returned error: %v", err)
	}
	if got := out.String(); got != want {
		t.Fatalf("output mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestUnfurlBytesReturnsFreshCopy(t *testing.T) {
	src := []byte("alpha beta\n")

	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("output mismatch:\nwant %q\n got %q", src, got)
	}

	got[0] = 'A'
	if src[0] != 'a' {
		t.Fatal("UnfurlBytes returned a slice aliasing the input")
	}
}

func TestScenarioAlreadyUnwrapped(t *testing.T) {
	src := []byte("# Title\n\nA paragraph on one physical line.\n\n```mermaid\ngraph TD\n  A --> B\n```\n")
	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("already-unwrapped document changed:\nwant %q\n got %q", src, got)
	}
}

func TestScenarioHardBreakPreserved(t *testing.T) {
	src := []byte("This sentence keeps its pause.   \nThe surrounding paragraph\nstill reflows.\n")
	want := []byte("This sentence keeps its pause.   \nThe surrounding paragraph still reflows.\n")
	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("hard-break output mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestBackslashHardBreakPreserved(t *testing.T) {
	src := []byte("This sentence keeps its pause.\\\nThe surrounding paragraph\nstill reflows.\n")
	want := []byte("This sentence keeps its pause.\\\nThe surrounding paragraph still reflows.\n")
	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("hard-break output mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestScenarioTable(t *testing.T) {
	src := []byte("| name | value |\n| --- | ---: |\n| alpha | 1 |\n| beta | 2 |\n")
	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("table changed:\nwant %q\n got %q", src, got)
	}
}

func TestTableWithPipelessBodyRowPreserved(t *testing.T) {
	src := []byte("| a | b |\n| --- | --- |\n| x | y |\nbar\n")
	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("table changed:\nwant %q\n got %q", src, got)
	}
}

func TestInvalidTableCandidateReflowsAsParagraph(t *testing.T) {
	src := []byte("a | b\n| --- | --- | --- |\ncontinues\n")
	want := []byte("a | b | --- | --- | --- | continues\n")
	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("invalid table output mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestConstructPreservation(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "atx heading", src: "# Heading\n"},
		{name: "setext heading", src: "Heading\n---\n"},
		{name: "hrule", src: "***\n"},
		{name: "fenced go code", src: "```go\nfmt.Println(\"x\")\n```\n"},
		{name: "fenced mermaid", src: "```mermaid\ngraph TD\n  A --> B\n```\n"},
		{name: "indented code", src: "    code\n    more\n"},
		{name: "gfm table", src: "| a | b |\n| --- | --- |\n| x | y |\n"},
		{name: "html type 1", src: "<script>\nconst x = 1;\n</script>\n"},
		{name: "html type 2", src: "<!--\ncomment\n-->\n"},
		{name: "html type 3", src: "<?pi\n?>\n"},
		{name: "html type 4", src: "<!DOCTYPE html>\n"},
		{name: "html type 5", src: "<![CDATA[\ntext\n]]>\n"},
		{name: "html type 6", src: "<div>\ntext\n\n"},
		{name: "html type 7", src: "<custom-tag>\ntext\n\n"},
		{name: "yaml frontmatter", src: "---\ntitle: x\n---\n"},
		{name: "toml frontmatter", src: "+++\ntitle = \"x\"\n+++\n"},
		{name: "reference definitions", src: "[a]: https://example.com/a\n[b]: https://example.com/b\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnfurlBytes([]byte(tt.src))
			if err != nil {
				t.Fatalf("UnfurlBytes returned error: %v", err)
			}
			if string(got) != tt.src {
				t.Fatalf("construct changed:\nwant %q\n got %q", tt.src, string(got))
			}
		})
	}
}
