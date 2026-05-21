package reflow

import (
	"bytes"
	"testing"

	"github.com/michaelquigley/unfurl/internal/tokenize"
)

func TestReflowSingleLineParagraph(t *testing.T) {
	got := reflowFixture(t, "alpha beta\n")
	assertBytes(t, got, []byte("alpha beta\n"))
}

func TestReflowTwoLineParagraph(t *testing.T) {
	got := reflowFixture(t, "alpha\nbeta\n")
	assertBytes(t, got, []byte("alpha beta\n"))
}

func TestReflowDropsSingleTrailingSpaceAtSoftWrap(t *testing.T) {
	got := reflowFixture(t, "alpha \nbeta\n")
	assertBytes(t, got, []byte("alpha beta\n"))
}

func TestReflowPreservesFinalLineTrailingSpace(t *testing.T) {
	got := reflowFixture(t, "alpha \n")
	assertBytes(t, got, []byte("alpha \n"))
}

func TestReflowHardBreakSegments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "two spaces", input: "alpha  \nbeta\n", want: "alpha  \nbeta\n"},
		{name: "three spaces", input: "alpha   \nbeta\n", want: "alpha   \nbeta\n"},
		{name: "four spaces", input: "alpha    \nbeta\n", want: "alpha    \nbeta\n"},
		{name: "backslash", input: "alpha\\\nbeta\n", want: "alpha\\\nbeta\n"},
		{name: "backslash with prior space", input: "alpha \\\nbeta\n", want: "alpha \\\nbeta\n"},
		{name: "reflow after hard break", input: "alpha  \nbeta\ngamma\n", want: "alpha  \nbeta gamma\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reflowFixture(t, tt.input)
			assertBytes(t, got, []byte(tt.want))
		})
	}
}

func TestReflowPreservesLineEndingAndTrailingNewlinePresence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "lf", input: "alpha\nbeta\n", want: "alpha beta\n"},
		{name: "crlf", input: "alpha\r\nbeta\r\n", want: "alpha beta\r\n"},
		{name: "no trailing newline", input: "alpha\nbeta", want: "alpha beta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reflowFixture(t, tt.input)
			assertBytes(t, got, []byte(tt.want))
		})
	}
}

func TestReflowSegmentPrefix(t *testing.T) {
	doc, err := tokenize.TokenizeBytes([]byte("alpha\nbeta  \ngamma\ndelta\n"))
	if err != nil {
		t.Fatalf("TokenizeBytes returned error: %v", err)
	}
	paragraph := Paragraph{Segments: []Segment{
		{
			Prefix:          []byte("> "),
			Lines:           doc.Lines[:2],
			HardBreakMarker: []byte("  "),
		},
		{
			Prefix: []byte("> "),
			Lines:  doc.Lines[2:],
		},
	}}

	got := Reflow(paragraph)
	assertBytes(t, got, []byte("> alpha beta  \n> gamma delta\n"))
}

func TestReflowCapturesLinePrefixForSegments(t *testing.T) {
	doc, err := tokenize.TokenizeBytes([]byte("alpha  \nbeta\ngamma\n"))
	if err != nil {
		t.Fatalf("TokenizeBytes returned error: %v", err)
	}
	doc.Lines[0].Prefix = []byte("- ")
	doc.Lines[1].Prefix = []byte("  ")
	doc.Lines[2].Prefix = []byte("  ")

	got := ReflowLines(doc.Lines)
	assertBytes(t, got, []byte("- alpha  \n  beta gamma\n"))
}

func TestReflowIdempotentAcrossTrailingSpaceHardBreakCounts(t *testing.T) {
	for _, input := range []string{
		"alpha  \nbeta\n",
		"alpha   \nbeta\n",
		"alpha    \nbeta\n",
	} {
		t.Run(input, func(t *testing.T) {
			once := reflowFixture(t, input)
			twice := reflowFixture(t, string(once))
			assertBytes(t, twice, once)
		})
	}
}

func reflowFixture(t *testing.T, input string) []byte {
	t.Helper()
	doc, err := tokenize.TokenizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("TokenizeBytes returned error: %v", err)
	}
	return ReflowLines(doc.Lines)
}

func assertBytes(t *testing.T, got []byte, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes mismatch:\nwant %q\n got %q", want, got)
	}
}
