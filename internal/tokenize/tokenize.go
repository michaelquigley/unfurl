package tokenize

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func Tokenize(r io.Reader) (*Document, error) {
	br := bufio.NewReader(r)
	doc := &Document{}

	for index := 0; ; index++ {
		raw, err := br.ReadBytes('\n')
		if len(raw) > 0 {
			if index == 0 && bytes.HasPrefix(raw, utf8BOM) {
				doc.HadBOM = true
				raw = raw[len(utf8BOM):]
			}
			if len(raw) > 0 {
				line := classifyLine(len(doc.Lines), raw)
				if len(line.LineEnding()) > 0 {
					doc.EndedWithNewline = true
				} else {
					doc.EndedWithNewline = false
				}
				if doc.LineEnding == nil && !isBlank(line.Text()) {
					doc.LineEnding = detectedLineEnding(line)
				}
				doc.Lines = append(doc.Lines, line)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read markdown line: %w", err)
		}
	}
	if doc.LineEnding == nil {
		doc.LineEnding = []byte{'\n'}
	}

	return doc, nil
}

func detectedLineEnding(line Line) []byte {
	if bytes.Equal(line.LineEnding(), []byte{'\r', '\n'}) {
		return []byte{'\r', '\n'}
	}
	return []byte{'\n'}
}

func TokenizeBytes(src []byte) (*Document, error) {
	return Tokenize(bytes.NewReader(src))
}

func classifyLine(index int, raw []byte) Line {
	line := Line{
		Index: index,
		Raw:   append([]byte(nil), raw...),
	}

	text := line.Text()
	line.Indent, _ = leadingIndent(text)
	line.SetextCandidate = isSetextCandidate(text)
	if marker, ok := matchListMarker(text); ok {
		line.ListMarker = marker
	}
	if marker, ok := detectHardBreak(raw); ok {
		line.HardBreak = true
		line.HardBreakMarker = marker
	}

	if isBlank(text) {
		line.Kind = LineBlank
		line.HardBreak = false
		line.HardBreakMarker = nil
		return line
	}
	if delimiter, ok := matchFrontMatterDelimiter(index, text); ok {
		line.Kind = LineFrontMatterDelimiter
		line.FrontMatterDelimiter = delimiter
		return line
	}
	if matchATXHeading(text) {
		line.Kind = LineATXHeading
		return line
	}
	if matchHRule(text) {
		line.Kind = LineHRule
		return line
	}
	if matchSetextUnderline(text) {
		line.Kind = LineSetextUnderline
		return line
	}
	if fence, ok := matchFence(text); ok {
		line.Kind = LineFence
		line.Fence = fence
		return line
	}
	if htmlType := matchHTMLBlockStart(text); htmlType != 0 {
		line.Kind = LineHTMLBlockStart
		line.HTMLType = htmlType
		return line
	}
	if prefixLen, ok := matchBlockquote(text); ok {
		line.Kind = LineBlockquote
		line.BlockquotePrefixLen = prefixLen
		return line
	}
	if line.ListMarker.Valid {
		line.Kind = LineListMarker
		return line
	}
	if table, ok := matchTableDelimiter(text); ok {
		line.Kind = LineTableDelimiter
		line.TableDelimiter = table
		return line
	}
	if matchIndentedCode(text) {
		line.Kind = LineIndentedCode
		return line
	}
	if matchReferenceDefinition(text) {
		line.Kind = LineReferenceDefinition
		return line
	}

	line.Kind = LineParagraphText
	return line
}

func splitLineEnding(raw []byte) ([]byte, []byte) {
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		return raw, nil
	}
	if len(raw) >= 2 && raw[len(raw)-2] == '\r' {
		return raw[:len(raw)-2], raw[len(raw)-2:]
	}
	return raw[:len(raw)-1], raw[len(raw)-1:]
}

func isBlank(text []byte) bool {
	for _, b := range text {
		if b != ' ' && b != '\t' {
			return false
		}
	}
	return true
}

func leadingIndent(text []byte) (int, int) {
	column := 0
	offset := 0
	for offset < len(text) {
		switch text[offset] {
		case ' ':
			column++
			offset++
		case '\t':
			column += 4 - column%4
			offset++
		default:
			return column, offset
		}
	}
	return column, offset
}

func skipUpToThreeSpaces(text []byte) (int, bool) {
	offset := 0
	for offset < len(text) && text[offset] == ' ' && offset < 4 {
		offset++
	}
	return offset, offset <= 3
}

func trimSpaceTab(text []byte) []byte {
	return bytes.Trim(text, " \t")
}

func matchFrontMatterDelimiter(index int, text []byte) (byte, bool) {
	if index != 0 {
		return 0, false
	}
	trimmed := trimSpaceTab(text)
	if bytes.Equal(trimmed, []byte("---")) {
		return '-', true
	}
	if bytes.Equal(trimmed, []byte("+++")) {
		return '+', true
	}
	return 0, false
}

func matchATXHeading(text []byte) bool {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) || text[offset] != '#' {
		return false
	}
	count := 0
	for offset+count < len(text) && text[offset+count] == '#' {
		count++
	}
	if count == 0 || count > 6 {
		return false
	}
	next := offset + count
	return next == len(text) || text[next] == ' ' || text[next] == '\t'
}

func matchHRule(text []byte) bool {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) {
		return false
	}
	marker := text[offset]
	if marker != '-' && marker != '_' && marker != '*' {
		return false
	}
	count := 0
	for i := offset; i < len(text); i++ {
		switch text[i] {
		case marker:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
}

func isSetextCandidate(text []byte) bool {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) {
		return false
	}
	marker := text[offset]
	if marker != '-' && marker != '=' {
		return false
	}
	for i := offset; i < len(text); i++ {
		switch text[i] {
		case marker, ' ', '\t':
		default:
			return false
		}
	}
	return true
}

func matchSetextUnderline(text []byte) bool {
	if matchHRule(text) {
		return false
	}
	return isSetextCandidate(text)
}

func matchFence(text []byte) (Fence, bool) {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) {
		return Fence{}, false
	}
	marker := text[offset]
	if marker != '`' && marker != '~' {
		return Fence{}, false
	}
	length := 0
	for offset+length < len(text) && text[offset+length] == marker {
		length++
	}
	if length < 3 {
		return Fence{}, false
	}
	info := bytes.Trim(text[offset+length:], " \t")
	if marker == '`' && bytes.Contains(info, []byte{'`'}) {
		return Fence{}, false
	}
	return Fence{Char: marker, Length: length, Info: append([]byte(nil), info...)}, true
}

func matchHTMLBlockStart(text []byte) int {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok {
		return 0
	}
	rest := text[offset:]
	lower := asciiLower(rest)

	for _, tag := range []string{"script", "pre", "style", "textarea"} {
		prefix := append([]byte{'<'}, []byte(tag)...)
		if bytes.HasPrefix(lower, prefix) && htmlTagBoundary(lower[len(prefix):]) {
			return 1
		}
	}
	if bytes.HasPrefix(rest, []byte("<!--")) {
		return 2
	}
	if bytes.HasPrefix(rest, []byte("<?")) {
		return 3
	}
	if len(rest) >= 3 && rest[0] == '<' && rest[1] == '!' && isASCIILetter(rest[2]) {
		return 4
	}
	if bytes.HasPrefix(rest, []byte("<![CDATA[")) {
		return 5
	}
	if tag, ok := parseHTMLTagName(rest); ok && isBlockHTMLTag(tag) {
		return 6
	}
	if completeHTMLTagLine(rest) {
		return 7
	}
	return 0
}

func asciiLower(src []byte) []byte {
	out := make([]byte, len(src))
	for i, b := range src {
		if b >= 'A' && b <= 'Z' {
			out[i] = b + ('a' - 'A')
		} else {
			out[i] = b
		}
	}
	return out
}

func htmlTagBoundary(rest []byte) bool {
	return len(rest) == 0 || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '>' || rest[0] == '/'
}

func parseHTMLTagName(rest []byte) (string, bool) {
	if len(rest) < 2 || rest[0] != '<' {
		return "", false
	}
	i := 1
	if i < len(rest) && rest[i] == '/' {
		i++
	}
	if i >= len(rest) || !isASCIILetter(rest[i]) {
		return "", false
	}
	start := i
	i++
	for i < len(rest) && (isASCIILetter(rest[i]) || isASCIIDigit(rest[i]) || rest[i] == '-') {
		i++
	}
	if i < len(rest) && !htmlTagBoundary(rest[i:]) {
		return "", false
	}
	return string(asciiLower(rest[start:i])), true
}

func completeHTMLTagLine(rest []byte) bool {
	trimmed := trimSpaceTab(rest)
	if len(trimmed) < 3 || trimmed[0] != '<' || trimmed[len(trimmed)-1] != '>' {
		return false
	}
	if bytes.Contains(trimmed[1:len(trimmed)-1], []byte{'<'}) {
		return false
	}
	if _, ok := parseHTMLTagName(trimmed); !ok {
		return false
	}
	return true
}

func matchBlockquote(text []byte) (int, bool) {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) || text[offset] != '>' {
		return 0, false
	}
	prefixLen := offset + 1
	if prefixLen < len(text) && (text[prefixLen] == ' ' || text[prefixLen] == '\t') {
		prefixLen++
	}
	return prefixLen, true
}

func matchListMarker(text []byte) (ListMarker, bool) {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) {
		return ListMarker{}, false
	}
	if text[offset] == '-' || text[offset] == '+' || text[offset] == '*' {
		end := offset + 1
		if end < len(text) && text[end] != ' ' && text[end] != '\t' {
			return ListMarker{}, false
		}
		return buildBulletMarker(text, offset, end), true
	}
	if !isASCIIDigit(text[offset]) {
		return ListMarker{}, false
	}
	end := offset
	for end < len(text) && isASCIIDigit(text[end]) && end-offset < 9 {
		end++
	}
	if end == offset || end >= len(text) || (text[end] != '.' && text[end] != ')') {
		return ListMarker{}, false
	}
	delimiter := text[end]
	end++
	if end < len(text) && text[end] != ' ' && text[end] != '\t' {
		return ListMarker{}, false
	}
	start, err := strconv.Atoi(string(text[offset : end-1]))
	if err != nil {
		return ListMarker{}, false
	}
	return buildOrderedMarker(text, offset, end, delimiter, start), true
}

func buildBulletMarker(text []byte, offset int, markerEnd int) ListMarker {
	contentOffset := consumeMarkerPadding(text, markerEnd)
	return ListMarker{
		Valid:         true,
		Bullet:        text[offset],
		MarkerOffset:  offset,
		MarkerWidth:   markerEnd - offset,
		ContentOffset: contentOffset,
		Empty:         isBlank(text[contentOffset:]),
	}
}

func buildOrderedMarker(text []byte, offset int, markerEnd int, delimiter byte, start int) ListMarker {
	contentOffset := consumeMarkerPadding(text, markerEnd)
	return ListMarker{
		Valid:         true,
		Ordered:       true,
		Delimiter:     delimiter,
		Start:         start,
		MarkerOffset:  offset,
		MarkerWidth:   markerEnd - offset,
		ContentOffset: contentOffset,
		Empty:         isBlank(text[contentOffset:]),
	}
}

func consumeMarkerPadding(text []byte, markerEnd int) int {
	offset := markerEnd
	for offset < len(text) && (text[offset] == ' ' || text[offset] == '\t') {
		offset++
	}
	return offset
}

func matchTableDelimiter(text []byte) (TableDelimiter, bool) {
	trimmed := trimSpaceTab(text)
	if !bytes.Contains(trimmed, []byte{'|'}) {
		return TableDelimiter{}, false
	}
	cells := splitTableCells(trimmed)
	if len(cells) == 0 {
		return TableDelimiter{}, false
	}
	for _, cell := range cells {
		if !matchTableDelimiterCell(trimSpaceTab(cell)) {
			return TableDelimiter{}, false
		}
	}
	return TableDelimiter{Valid: true, CellCount: len(cells)}, true
}

func splitTableCells(row []byte) [][]byte {
	if len(row) > 0 && row[0] == '|' {
		row = row[1:]
	}
	if len(row) > 0 && row[len(row)-1] == '|' {
		row = row[:len(row)-1]
	}
	var cells [][]byte
	start := 0
	escaped := false
	for i, b := range row {
		if escaped {
			escaped = false
			continue
		}
		if b == '\\' {
			escaped = true
			continue
		}
		if b == '|' {
			cells = append(cells, row[start:i])
			start = i + 1
		}
	}
	cells = append(cells, row[start:])
	return cells
}

func matchTableDelimiterCell(cell []byte) bool {
	if len(cell) == 0 {
		return false
	}
	i := 0
	if cell[i] == ':' {
		i++
	}
	hyphens := 0
	for i < len(cell) && cell[i] == '-' {
		hyphens++
		i++
	}
	if hyphens == 0 {
		return false
	}
	if i < len(cell) && cell[i] == ':' {
		i++
	}
	return i == len(cell)
}

func matchIndentedCode(text []byte) bool {
	indent, _ := leadingIndent(text)
	return indent >= 4
}

func matchReferenceDefinition(text []byte) bool {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) || text[offset] != '[' {
		return false
	}
	close := bytes.IndexByte(text[offset+1:], ']')
	if close < 0 {
		return false
	}
	close += offset + 1
	return close+1 < len(text) && text[close+1] == ':'
}

func detectHardBreak(raw []byte) ([]byte, bool) {
	text, ending := splitLineEnding(raw)
	if len(ending) == 0 || len(text) == 0 {
		return nil, false
	}
	if text[len(text)-1] == '\\' {
		return []byte{'\\'}, true
	}
	count := 0
	for i := len(text) - 1; i >= 0 && text[i] == ' '; i-- {
		count++
	}
	if count >= 2 {
		return append([]byte(nil), text[len(text)-count:]...), true
	}
	return nil, false
}

func isBlockHTMLTag(tag string) bool {
	_, ok := blockHTMLTags[tag]
	return ok
}

var blockHTMLTags = map[string]struct{}{
	"address":    {},
	"article":    {},
	"aside":      {},
	"base":       {},
	"basefont":   {},
	"blockquote": {},
	"body":       {},
	"caption":    {},
	"center":     {},
	"col":        {},
	"colgroup":   {},
	"dd":         {},
	"details":    {},
	"dialog":     {},
	"dir":        {},
	"div":        {},
	"dl":         {},
	"dt":         {},
	"fieldset":   {},
	"figcaption": {},
	"figure":     {},
	"footer":     {},
	"form":       {},
	"frame":      {},
	"frameset":   {},
	"h1":         {},
	"h2":         {},
	"h3":         {},
	"h4":         {},
	"h5":         {},
	"h6":         {},
	"head":       {},
	"header":     {},
	"hr":         {},
	"html":       {},
	"iframe":     {},
	"legend":     {},
	"li":         {},
	"link":       {},
	"main":       {},
	"menu":       {},
	"menuitem":   {},
	"nav":        {},
	"noframes":   {},
	"ol":         {},
	"optgroup":   {},
	"option":     {},
	"p":          {},
	"param":      {},
	"search":     {},
	"section":    {},
	"summary":    {},
	"table":      {},
	"tbody":      {},
	"td":         {},
	"tfoot":      {},
	"th":         {},
	"thead":      {},
	"title":      {},
	"tr":         {},
	"track":      {},
	"ul":         {},
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
