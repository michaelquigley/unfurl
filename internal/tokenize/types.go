package tokenize

// LineKind is the primary classification assigned to one physical source line.
type LineKind int

const (
	LineBlank LineKind = iota
	LineParagraphText
	LineFrontMatterDelimiter
	LineATXHeading
	LineHRule
	LineSetextUnderline
	LineFence
	LineHTMLBlockStart
	LineBlockquote
	LineListMarker
	LineTableDelimiter
	LineIndentedCode
	LineReferenceDefinition
)

func (k LineKind) String() string {
	switch k {
	case LineBlank:
		return "blank"
	case LineParagraphText:
		return "paragraph_text"
	case LineFrontMatterDelimiter:
		return "frontmatter_delimiter"
	case LineATXHeading:
		return "atx_heading"
	case LineHRule:
		return "hrule"
	case LineSetextUnderline:
		return "setext_underline"
	case LineFence:
		return "fence"
	case LineHTMLBlockStart:
		return "html_block_start"
	case LineBlockquote:
		return "blockquote"
	case LineListMarker:
		return "list_marker"
	case LineTableDelimiter:
		return "table_delimiter"
	case LineIndentedCode:
		return "indented_code"
	case LineReferenceDefinition:
		return "reference_definition"
	default:
		return "unknown"
	}
}

// Document is the pass-1 result. Lines are BOM-free; HadBOM records whether a
// leading UTF-8 BOM was stripped before classification.
type Document struct {
	HadBOM           bool
	LineEnding       []byte
	EndedWithNewline bool
	Lines            []Line
}

type Line struct {
	Index int
	// Prefix is emitted before Raw for lines that have had container markers
	// stripped for recursive grouping.
	Prefix []byte
	Raw    []byte
	Kind   LineKind

	Indent int

	HardBreak       bool
	HardBreakMarker []byte

	SetextCandidate bool

	FrontMatterDelimiter byte
	Fence                Fence
	HTMLType             int
	BlockquotePrefixLen  int
	ListMarker           ListMarker
	TableDelimiter       TableDelimiter
}

type Fence struct {
	Char   byte
	Length int
	Info   []byte
}

type ListMarker struct {
	Valid         bool
	Ordered       bool
	Bullet        byte
	Delimiter     byte
	Start         int
	MarkerOffset  int
	MarkerWidth   int
	ContentOffset int
	Empty         bool
}

type TableDelimiter struct {
	Valid     bool
	CellCount int
}

type BlockKind int

const (
	BlockParagraph BlockKind = iota
	BlockBlank
	BlockFrontMatter
	BlockATXHeading
	BlockSetextHeading
	BlockHRule
	BlockTable
	BlockFencedCode
	BlockIndentedCode
	BlockHTML
	BlockReferenceDefinition
	BlockRaw
)

func (k BlockKind) String() string {
	switch k {
	case BlockParagraph:
		return "paragraph"
	case BlockBlank:
		return "blank"
	case BlockFrontMatter:
		return "frontmatter"
	case BlockATXHeading:
		return "atx_heading"
	case BlockSetextHeading:
		return "setext_heading"
	case BlockHRule:
		return "hrule"
	case BlockTable:
		return "table"
	case BlockFencedCode:
		return "fenced_code"
	case BlockIndentedCode:
		return "indented_code"
	case BlockHTML:
		return "html"
	case BlockReferenceDefinition:
		return "reference_definition"
	case BlockRaw:
		return "raw"
	default:
		return "unknown"
	}
}

type Block struct {
	Kind     BlockKind
	Lines    []Line
	HTMLType int
}

func (l Line) Text() []byte {
	text, _ := splitLineEnding(l.Raw)
	return text
}

func (l Line) LineEnding() []byte {
	_, ending := splitLineEnding(l.Raw)
	return ending
}

func (l Line) IsBlank() bool {
	return l.Kind == LineBlank
}

func (l Line) IsParagraphText() bool {
	return l.Kind == LineParagraphText
}

func (l Line) IsBlockStart() bool {
	switch l.Kind {
	case LineATXHeading, LineHRule, LineSetextUnderline, LineFence, LineHTMLBlockStart,
		LineBlockquote, LineListMarker, LineTableDelimiter, LineIndentedCode,
		LineReferenceDefinition, LineFrontMatterDelimiter:
		return true
	default:
		return false
	}
}
