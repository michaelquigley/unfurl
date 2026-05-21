package emit

import (
	"bytes"
	"testing"

	"github.com/michaelquigley/unfurl/internal/tokenize"
)

func TestEmitReflowsParagraph(t *testing.T) {
	got := emitFixture(t, "alpha\nbeta\n")
	assertBytes(t, got, []byte("alpha beta\n"))
}

func TestEmitDropsSingleTrailingSpaceAtSoftWrap(t *testing.T) {
	got := emitFixture(t, "alpha \nbeta\n")
	assertBytes(t, got, []byte("alpha beta\n"))
}

func TestEmitPreservesParagraphHardBreak(t *testing.T) {
	got := emitFixture(t, "alpha   \nbeta\ngamma\n")
	assertBytes(t, got, []byte("alpha   \nbeta gamma\n"))
}

func TestEmitPreservesNonParagraphBlocks(t *testing.T) {
	src := []byte("# title\n\n```go\nfmt.Println(\"x\")\n```\n\nTitle\n---\n")
	got := emitFixture(t, string(src))
	assertBytes(t, got, src)
}

func TestEmitReattachesBOM(t *testing.T) {
	got := emitFixture(t, string(append([]byte{0xEF, 0xBB, 0xBF}, []byte("# title\n")...)))
	assertBytes(t, got, []byte{0xEF, 0xBB, 0xBF, '#', ' ', 't', 'i', 't', 'l', 'e', '\n'})
}

func TestEmitUsesDocumentLineEndingForReflow(t *testing.T) {
	got := emitFixture(t, "alpha\r\nbeta\n")
	assertBytes(t, got, []byte("alpha beta\r\n"))
}

func emitFixture(t *testing.T, input string) []byte {
	t.Helper()
	doc, err := tokenize.TokenizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("TokenizeBytes returned error: %v", err)
	}
	var out bytes.Buffer
	if err := Emit(&out, doc, tokenize.Group(doc)); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	return out.Bytes()
}

func assertBytes(t *testing.T, got []byte, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes mismatch:\nwant %q\n got %q", want, got)
	}
}
