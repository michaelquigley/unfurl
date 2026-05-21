package tokenize

import "testing"

func TestGroupPlainParagraphs(t *testing.T) {
	blocks := groupFixture(t, "alpha\nbeta\n\ncharlie\n")
	assertBlockKinds(t, blocks, BlockParagraph, BlockBlank, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
	assertBlockLineCount(t, blocks[1], 1)
	assertBlockLineCount(t, blocks[2], 1)
}

func TestGroupSetextHeading(t *testing.T) {
	blocks := groupFixture(t, "Title\n---\n\nbody\n")
	assertBlockKinds(t, blocks, BlockSetextHeading, BlockBlank, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
}

func TestGroupFencedCodeContainingBlockShapedLines(t *testing.T) {
	blocks := groupFixture(t, "```go\n# not a heading\n---\n```\nafter\n")
	assertBlockKinds(t, blocks, BlockFencedCode, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 4)
}

func TestGroupIndentedCode(t *testing.T) {
	blocks := groupFixture(t, "    code\n\tmore\nnext\n")
	assertBlockKinds(t, blocks, BlockIndentedCode, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
}

func TestGroupReferenceDefinitions(t *testing.T) {
	blocks := groupFixture(t, "[a]: https://example.com/a\n[b]: https://example.com/b\n\nbody\n")
	assertBlockKinds(t, blocks, BlockReferenceDefinition, BlockBlank, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 2)
}

func TestGroupFrontMatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "yaml", input: "---\ntitle: x\n---\nbody\n"},
		{name: "toml", input: "+++\ntitle = \"x\"\n+++\nbody\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := groupFixture(t, tt.input)
			assertBlockKinds(t, blocks, BlockFrontMatter, BlockParagraph)
			assertBlockLineCount(t, blocks[0], 3)
		})
	}
}

func TestGroupFrontMatterShapedDelimiterMidDocument(t *testing.T) {
	blocks := groupFixture(t, "body\n\n---\nafter\n")
	assertBlockKinds(t, blocks, BlockParagraph, BlockBlank, BlockHRule, BlockParagraph)
}

func TestGroupHTMLBlockEndConditions(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		htmlType  int
		lineCount int
	}{
		{name: "type 1 script", input: "<script>\n# not heading\n</script>\nafter\n", htmlType: 1, lineCount: 3},
		{name: "type 2 comment", input: "<!--\ncomment\n-->\nafter\n", htmlType: 2, lineCount: 3},
		{name: "type 3 pi", input: "<?pi\n?>\nafter\n", htmlType: 3, lineCount: 2},
		{name: "type 4 declaration", input: "<!DOCTYPE html>\nafter\n", htmlType: 4, lineCount: 1},
		{name: "type 5 cdata", input: "<![CDATA[\ntext\n]]>\nafter\n", htmlType: 5, lineCount: 3},
		{name: "type 6 block tag", input: "<div>\ntext\n\nafter\n", htmlType: 6, lineCount: 2},
		{name: "type 7 complete tag", input: "<custom-tag>\ntext\n\nafter\n", htmlType: 7, lineCount: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := groupFixture(t, tt.input)
			assertBlockKinds(t, blocks[:1], BlockHTML)
			if blocks[0].HTMLType != tt.htmlType {
				t.Fatalf("HTMLType mismatch: want %d got %d", tt.htmlType, blocks[0].HTMLType)
			}
			assertBlockLineCount(t, blocks[0], tt.lineCount)
		})
	}
}

func TestGroupHTMLType7DoesNotInterruptParagraph(t *testing.T) {
	blocks := groupFixture(t, "alpha\n<custom-tag>\nbeta\n")
	assertBlockKinds(t, blocks, BlockParagraph)
	assertBlockLineCount(t, blocks[0], 3)
}

func TestGroupParagraphNonInterruptions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "indented code continuation", input: "alpha\n    beta\n"},
		{name: "ordered list start not one", input: "alpha\n2. beta\n"},
		{name: "empty ordered list marker", input: "alpha\n1.\nbeta\n"},
		{name: "empty bullet list marker", input: "alpha\n-\nbeta\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := groupFixture(t, tt.input)
			assertBlockKinds(t, blocks, BlockParagraph)
		})
	}
}

func groupFixture(t *testing.T, input string) []Block {
	t.Helper()
	doc, err := TokenizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("TokenizeBytes returned error: %v", err)
	}
	return Group(doc)
}

func assertBlockKinds(t *testing.T, blocks []Block, want ...BlockKind) {
	t.Helper()
	if len(blocks) != len(want) {
		t.Fatalf("block count mismatch: want %d got %d (%v)", len(want), len(blocks), blockKinds(blocks))
	}
	for i := range want {
		if blocks[i].Kind != want[i] {
			t.Fatalf("block %d kind mismatch: want %s got %s", i, want[i], blocks[i].Kind)
		}
	}
}

func assertBlockLineCount(t *testing.T, block Block, want int) {
	t.Helper()
	if len(block.Lines) != want {
		t.Fatalf("%s line count mismatch: want %d got %d", block.Kind, want, len(block.Lines))
	}
}

func blockKinds(blocks []Block) []BlockKind {
	kinds := make([]BlockKind, len(blocks))
	for i, block := range blocks {
		kinds[i] = block.Kind
	}
	return kinds
}
