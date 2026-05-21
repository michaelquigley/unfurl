package tokenize

import (
	"bytes"
	"testing"
)

func TestGroupBlockquoteWithLazyContinuation(t *testing.T) {
	blocks := groupFixture(t, "> alpha\nbeta\n> gamma\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 3)
	assertLinePrefixRaw(t, blocks[0].Lines[0], "> ", "alpha\n")
	assertLinePrefixRaw(t, blocks[0].Lines[1], "", "beta\n")
	assertLinePrefixRaw(t, blocks[0].Lines[2], "> ", "gamma\n")
}

func TestGroupListItemWithLazyContinuation(t *testing.T) {
	blocks := groupFixture(t, "- alpha\nbeta\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
	assertLinePrefixRaw(t, blocks[0].Lines[0], "- ", "alpha\n")
	assertLinePrefixRaw(t, blocks[0].Lines[1], "", "beta\n")
}

func TestGroupNestedBlockquote(t *testing.T) {
	blocks := groupFixture(t, "> > alpha\n> > beta\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
	assertLinePrefixRaw(t, blocks[0].Lines[0], "> > ", "alpha\n")
	assertLinePrefixRaw(t, blocks[0].Lines[1], "> > ", "beta\n")
}

func TestGroupNestedListInsideBlockquote(t *testing.T) {
	blocks := groupFixture(t, "> - alpha\n>   beta\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
	assertLinePrefixRaw(t, blocks[0].Lines[0], "> - ", "alpha\n")
	assertLinePrefixRaw(t, blocks[0].Lines[1], ">   ", "beta\n")
}

func TestGroupBlockquoteInsideList(t *testing.T) {
	blocks := groupFixture(t, "- > alpha\n  > beta\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
	assertLinePrefixRaw(t, blocks[0].Lines[0], "- > ", "alpha\n")
	assertLinePrefixRaw(t, blocks[0].Lines[1], "  > ", "beta\n")
}

func TestGroupBlockquoteLazySetextShapeDoesNotReclassify(t *testing.T) {
	blocks := groupFixture(t, "> alpha\n---\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
	assertLinePrefixRaw(t, blocks[0].Lines[0], "> ", "alpha\n")
	assertLinePrefixRaw(t, blocks[0].Lines[1], "", "---\n")
}

func assertLinePrefixRaw(t *testing.T, line Line, prefix string, raw string) {
	t.Helper()
	if !bytes.Equal(line.Prefix, []byte(prefix)) {
		t.Fatalf("prefix mismatch: want %q got %q", prefix, line.Prefix)
	}
	if !bytes.Equal(line.Raw, []byte(raw)) {
		t.Fatalf("raw mismatch: want %q got %q", raw, line.Raw)
	}
}
