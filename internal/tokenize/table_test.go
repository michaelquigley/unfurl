package tokenize

import "testing"

func TestGroupGFMTable(t *testing.T) {
	blocks := groupFixture(t, "| a | b |\n| --- | --- |\n| x | y |\n")
	assertBlockKinds(t, blocks, BlockTable)
	assertBlockLineCount(t, blocks[0], 3)
}

func TestGroupGFMTableWithPipelessBodyRow(t *testing.T) {
	blocks := groupFixture(t, "| a | b |\n| --- | --- |\n| x | y |\nbar\n")
	assertBlockKinds(t, blocks, BlockTable)
	assertBlockLineCount(t, blocks[0], 4)
}

func TestGroupGFMTableWithVaryingBodyCellCounts(t *testing.T) {
	blocks := groupFixture(t, "| a | b | c |\n| --- | --- | --- |\n| x | y |\n| one | two | three | four |\n")
	assertBlockKinds(t, blocks, BlockTable)
	assertBlockLineCount(t, blocks[0], 4)
}

func TestGroupGFMTableStopsAtBlank(t *testing.T) {
	blocks := groupFixture(t, "| a | b |\n| --- | --- |\nbar\n\nnext\n")
	assertBlockKinds(t, blocks, BlockTable, BlockBlank, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 3)
}

func TestGroupGFMTableStopsAtBlockStart(t *testing.T) {
	blocks := groupFixture(t, "| a | b |\n| --- | --- |\nbar\n# next\n")
	assertBlockKinds(t, blocks, BlockTable, BlockATXHeading)
	assertBlockLineCount(t, blocks[0], 3)
}

func TestGroupDoesNotCreateTableOnHeaderDelimiterArityMismatch(t *testing.T) {
	blocks := groupFixture(t, "a | b\n| --- | --- | --- |\ncontinues\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 3)
}

func TestGroupDoesNotCreateTableOnInvalidDelimiterShape(t *testing.T) {
	blocks := groupFixture(t, "a | b\n| --- | text |\ncontinues\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 3)
}

func TestGroupTableHeaderCountsEscapedPipes(t *testing.T) {
	blocks := groupFixture(t, `a \| still a | b
| --- | --- |
body
`)
	assertBlockKinds(t, blocks, BlockTable)
	assertBlockLineCount(t, blocks[0], 3)
}
