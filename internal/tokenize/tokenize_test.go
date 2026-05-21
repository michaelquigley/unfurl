package tokenize

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestTokenizeClassifiesLineKinds(t *testing.T) {
	tests := []struct {
		name  string
		input string
		line  int
		kind  LineKind
		check func(t *testing.T, line Line, doc *Document)
	}{
		{
			name:  "blank spaces and tab",
			input: "  \t\n",
			kind:  LineBlank,
		},
		{
			name:  "paragraph text",
			input: "ordinary paragraph text\n",
			kind:  LineParagraphText,
		},
		{
			name:  "yaml frontmatter delimiter at first line",
			input: "---\ntitle: x\n---\n",
			kind:  LineFrontMatterDelimiter,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.FrontMatterDelimiter != '-' {
					t.Fatalf("frontmatter delimiter mismatch: %q", line.FrontMatterDelimiter)
				}
			},
		},
		{
			name:  "toml frontmatter delimiter at first line",
			input: "+++\ntitle = \"x\"\n+++\n",
			kind:  LineFrontMatterDelimiter,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.FrontMatterDelimiter != '+' {
					t.Fatalf("frontmatter delimiter mismatch: %q", line.FrontMatterDelimiter)
				}
			},
		},
		{
			name:  "atx heading level one",
			input: "# title\n",
			kind:  LineATXHeading,
		},
		{
			name:  "atx heading level two",
			input: "## title\n",
			kind:  LineATXHeading,
		},
		{
			name:  "atx heading level three",
			input: "### title\n",
			kind:  LineATXHeading,
		},
		{
			name:  "atx heading level four",
			input: "#### title\n",
			kind:  LineATXHeading,
		},
		{
			name:  "atx heading level five",
			input: "##### title\n",
			kind:  LineATXHeading,
		},
		{
			name:  "atx heading level six with indent",
			input: "   ###### title\n",
			kind:  LineATXHeading,
		},
		{
			name:  "hrule dash after first line",
			input: "paragraph\n---\n",
			line:  1,
			kind:  LineHRule,
			check: func(t *testing.T, line Line, doc *Document) {
				if !line.SetextCandidate {
					t.Fatal("dash hrule should remain a setext candidate")
				}
			},
		},
		{
			name:  "hrule asterisk",
			input: "***\n",
			kind:  LineHRule,
		},
		{
			name:  "hrule underscore",
			input: "___\n",
			kind:  LineHRule,
		},
		{
			name:  "setext equals underline",
			input: "===\n",
			kind:  LineSetextUnderline,
			check: func(t *testing.T, line Line, doc *Document) {
				if !line.SetextCandidate {
					t.Fatal("setext underline should be a setext candidate")
				}
			},
		},
		{
			name:  "backtick fence with info",
			input: "```go\n",
			kind:  LineFence,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.Fence.Char != '`' || line.Fence.Length != 3 || string(line.Fence.Info) != "go" {
					t.Fatalf("fence mismatch: %+v", line.Fence)
				}
			},
		},
		{
			name:  "backtick fence without info",
			input: "```\n",
			kind:  LineFence,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.Fence.Char != '`' || line.Fence.Length != 3 || len(line.Fence.Info) != 0 {
					t.Fatalf("fence mismatch: %+v", line.Fence)
				}
			},
		},
		{
			name:  "tilde fence with info",
			input: "~~~~ mermaid\n",
			kind:  LineFence,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.Fence.Char != '~' || line.Fence.Length != 4 || string(line.Fence.Info) != "mermaid" {
					t.Fatalf("fence mismatch: %+v", line.Fence)
				}
			},
		},
		{
			name:  "tilde fence without info",
			input: "~~~\n",
			kind:  LineFence,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.Fence.Char != '~' || line.Fence.Length != 3 || len(line.Fence.Info) != 0 {
					t.Fatalf("fence mismatch: %+v", line.Fence)
				}
			},
		},
		{
			name:  "html type 1 script",
			input: "<script>\n",
			kind:  LineHTMLBlockStart,
			check: expectHTMLType(1),
		},
		{
			name:  "html type 2 comment",
			input: "<!-- comment\n",
			kind:  LineHTMLBlockStart,
			check: expectHTMLType(2),
		},
		{
			name:  "html type 3 processing instruction",
			input: "<?pi\n",
			kind:  LineHTMLBlockStart,
			check: expectHTMLType(3),
		},
		{
			name:  "html type 4 declaration",
			input: "<!DOCTYPE html>\n",
			kind:  LineHTMLBlockStart,
			check: expectHTMLType(4),
		},
		{
			name:  "html type 5 cdata",
			input: "<![CDATA[\n",
			kind:  LineHTMLBlockStart,
			check: expectHTMLType(5),
		},
		{
			name:  "html type 6 block tag",
			input: "<div>\n",
			kind:  LineHTMLBlockStart,
			check: expectHTMLType(6),
		},
		{
			name:  "html type 7 complete custom tag",
			input: "<custom-tag>\n",
			kind:  LineHTMLBlockStart,
			check: expectHTMLType(7),
		},
		{
			name:  "blockquote marker with following space",
			input: "> quote\n",
			kind:  LineBlockquote,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.BlockquotePrefixLen != 2 {
					t.Fatalf("blockquote prefix len mismatch: %d", line.BlockquotePrefixLen)
				}
			},
		},
		{
			name:  "blockquote marker without following space",
			input: ">quote\n",
			kind:  LineBlockquote,
			check: func(t *testing.T, line Line, doc *Document) {
				if line.BlockquotePrefixLen != 1 {
					t.Fatalf("blockquote prefix len mismatch: %d", line.BlockquotePrefixLen)
				}
			},
		},
		{
			name:  "dash bullet list marker",
			input: "- item\n",
			kind:  LineListMarker,
			check: expectBullet('-', false),
		},
		{
			name:  "plus bullet list marker",
			input: "+ item\n",
			kind:  LineListMarker,
			check: expectBullet('+', false),
		},
		{
			name:  "asterisk bullet list marker",
			input: "* item\n",
			kind:  LineListMarker,
			check: expectBullet('*', false),
		},
		{
			name:  "empty bullet list marker remains tracked",
			input: "- \n",
			kind:  LineSetextUnderline,
			check: func(t *testing.T, line Line, doc *Document) {
				expectBullet('-', true)(t, line, doc)
				if !line.SetextCandidate {
					t.Fatal("empty dash marker should remain a setext candidate")
				}
			},
		},
		{
			name:  "ordered dot list marker",
			input: "1. item\n",
			kind:  LineListMarker,
			check: expectOrdered('.', 1, false),
		},
		{
			name:  "ordered paren list marker",
			input: "23) item\n",
			kind:  LineListMarker,
			check: expectOrdered(')', 23, false),
		},
		{
			name:  "empty ordered list marker",
			input: "1. \n",
			kind:  LineListMarker,
			check: expectOrdered('.', 1, true),
		},
		{
			name:  "table delimiter with alignments",
			input: "| :--- | ---: | :---: |\n",
			kind:  LineTableDelimiter,
			check: func(t *testing.T, line Line, doc *Document) {
				if !line.TableDelimiter.Valid || line.TableDelimiter.CellCount != 3 {
					t.Fatalf("table delimiter mismatch: %+v", line.TableDelimiter)
				}
			},
		},
		{
			name:  "indented code with spaces",
			input: "    code\n",
			kind:  LineIndentedCode,
		},
		{
			name:  "indented code with tab",
			input: "\tcode\n",
			kind:  LineIndentedCode,
		},
		{
			name:  "reference definition",
			input: "[label]: https://example.com\n",
			kind:  LineReferenceDefinition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := TokenizeBytes([]byte(tt.input))
			if err != nil {
				t.Fatalf("TokenizeBytes returned error: %v", err)
			}
			if tt.line >= len(doc.Lines) {
				t.Fatalf("line %d out of range for %d lines", tt.line, len(doc.Lines))
			}
			line := doc.Lines[tt.line]
			if line.Kind != tt.kind {
				t.Fatalf("kind mismatch: want %s got %s", tt.kind, line.Kind)
			}
			if tt.check != nil {
				tt.check(t, line, doc)
			}
		})
	}
}

func TestTokenizePreservesRawLinesAndLineEndings(t *testing.T) {
	doc, err := TokenizeBytes([]byte("alpha\r\nbeta\ncharlie"))
	if err != nil {
		t.Fatalf("TokenizeBytes returned error: %v", err)
	}
	if len(doc.Lines) != 3 {
		t.Fatalf("line count mismatch: %d", len(doc.Lines))
	}
	assertBytes(t, "raw line 0", doc.Lines[0].Raw, []byte("alpha\r\n"))
	assertBytes(t, "text line 0", doc.Lines[0].Text(), []byte("alpha"))
	assertBytes(t, "ending line 0", doc.Lines[0].LineEnding(), []byte("\r\n"))
	assertBytes(t, "raw line 1", doc.Lines[1].Raw, []byte("beta\n"))
	assertBytes(t, "ending line 1", doc.Lines[1].LineEnding(), []byte("\n"))
	assertBytes(t, "raw line 2", doc.Lines[2].Raw, []byte("charlie"))
	assertBytes(t, "ending line 2", doc.Lines[2].LineEnding(), nil)
}

func TestTokenizeStripsLeadingBOMBeforeClassification(t *testing.T) {
	doc, err := TokenizeBytes(append([]byte{0xEF, 0xBB, 0xBF}, []byte("# title\n")...))
	if err != nil {
		t.Fatalf("TokenizeBytes returned error: %v", err)
	}
	if !doc.HadBOM {
		t.Fatal("HadBOM false")
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("line count mismatch: %d", len(doc.Lines))
	}
	if doc.Lines[0].Kind != LineATXHeading {
		t.Fatalf("kind mismatch: want %s got %s", LineATXHeading, doc.Lines[0].Kind)
	}
	assertBytes(t, "raw line", doc.Lines[0].Raw, []byte("# title\n"))
}

func TestTokenizeHardBreakDetection(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		hard   bool
		marker string
	}{
		{name: "two trailing spaces", input: "foo  \n", hard: true, marker: "  "},
		{name: "three trailing spaces", input: "foo   \n", hard: true, marker: "   "},
		{name: "backslash", input: "foo\\\n", hard: true, marker: "\\"},
		{name: "single trailing space", input: "foo \n"},
		{name: "two trailing spaces without line ending", input: "foo  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := TokenizeBytes([]byte(tt.input))
			if err != nil {
				t.Fatalf("TokenizeBytes returned error: %v", err)
			}
			line := doc.Lines[0]
			if line.HardBreak != tt.hard {
				t.Fatalf("HardBreak mismatch: want %v got %v", tt.hard, line.HardBreak)
			}
			assertBytes(t, "hard-break marker", line.HardBreakMarker, []byte(tt.marker))
		})
	}
}

func TestTokenizeReturnsReadError(t *testing.T) {
	want := errors.New("boom")
	_, err := Tokenize(errorReader{err: want})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, want) {
		t.Fatalf("error mismatch: %v", err)
	}
}

func expectHTMLType(want int) func(t *testing.T, line Line, doc *Document) {
	return func(t *testing.T, line Line, doc *Document) {
		t.Helper()
		if line.HTMLType != want {
			t.Fatalf("HTML type mismatch: want %d got %d", want, line.HTMLType)
		}
	}
}

func expectBullet(want byte, empty bool) func(t *testing.T, line Line, doc *Document) {
	return func(t *testing.T, line Line, doc *Document) {
		t.Helper()
		if !line.ListMarker.Valid {
			t.Fatal("ListMarker.Valid false")
		}
		if line.ListMarker.Ordered {
			t.Fatal("ListMarker.Ordered true")
		}
		if line.ListMarker.Bullet != want {
			t.Fatalf("bullet mismatch: want %q got %q", want, line.ListMarker.Bullet)
		}
		if line.ListMarker.Empty != empty {
			t.Fatalf("empty mismatch: want %v got %v", empty, line.ListMarker.Empty)
		}
	}
}

func expectOrdered(delimiter byte, start int, empty bool) func(t *testing.T, line Line, doc *Document) {
	return func(t *testing.T, line Line, doc *Document) {
		t.Helper()
		if !line.ListMarker.Valid {
			t.Fatal("ListMarker.Valid false")
		}
		if !line.ListMarker.Ordered {
			t.Fatal("ListMarker.Ordered false")
		}
		if line.ListMarker.Delimiter != delimiter {
			t.Fatalf("delimiter mismatch: want %q got %q", delimiter, line.ListMarker.Delimiter)
		}
		if line.ListMarker.Start != start {
			t.Fatalf("start mismatch: want %d got %d", start, line.ListMarker.Start)
		}
		if line.ListMarker.Empty != empty {
			t.Fatalf("empty mismatch: want %v got %v", empty, line.ListMarker.Empty)
		}
	}
}

func assertBytes(t *testing.T, name string, got []byte, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("%s mismatch: want %q got %q", name, want, got)
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

var _ io.Reader = errorReader{}
