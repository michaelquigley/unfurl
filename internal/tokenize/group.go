package tokenize

import "bytes"

func Group(doc *Document) []Block {
	if doc == nil || len(doc.Lines) == 0 {
		return nil
	}
	return groupLines(doc.Lines)
}

func groupLines(lines []Line) []Block {
	var blocks []Block
	for i := 0; i < len(lines); {
		line := lines[i]
		switch line.Kind {
		case LineBlank:
			start := i
			for i < len(lines) && lines[i].Kind == LineBlank {
				i++
			}
			blocks = append(blocks, block(BlockBlank, lines[start:i]))
		case LineFrontMatterDelimiter:
			next := consumeFrontMatter(lines, i)
			blocks = append(blocks, block(BlockFrontMatter, lines[i:next]))
			i = next
		case LineATXHeading:
			blocks = append(blocks, block(BlockATXHeading, []Line{line}))
			i++
		case LineHRule:
			blocks = append(blocks, block(BlockHRule, []Line{line}))
			i++
		case LineFence:
			next := consumeFencedCode(lines, i)
			blocks = append(blocks, block(BlockFencedCode, lines[i:next]))
			i = next
		case LineHTMLBlockStart:
			next := consumeHTMLBlock(lines, i)
			html := block(BlockHTML, lines[i:next])
			html.HTMLType = line.HTMLType
			blocks = append(blocks, html)
			i = next
		case LineIndentedCode:
			next := consumeIndentedCode(lines, i)
			blocks = append(blocks, block(BlockIndentedCode, lines[i:next]))
			i = next
		case LineReferenceDefinition:
			start := i
			for i < len(lines) && lines[i].Kind == LineReferenceDefinition {
				i++
			}
			blocks = append(blocks, block(BlockReferenceDefinition, lines[start:i]))
		case LineBlockquote:
			next, containerBlocks := consumeBlockquote(lines, i)
			blocks = append(blocks, containerBlocks...)
			i = next
		case LineListMarker:
			next, containerBlocks := consumeList(lines, i)
			blocks = append(blocks, containerBlocks...)
			i = next
		default:
			next, paragraph := consumeParagraph(lines, i)
			blocks = append(blocks, paragraph)
			i = next
		}
	}
	return blocks
}

func block(kind BlockKind, lines []Line) Block {
	return Block{Kind: kind, Lines: append([]Line(nil), lines...)}
}

func consumeFrontMatter(lines []Line, start int) int {
	opener := lines[start]
	delimiter := opener.FrontMatterDelimiter
	for i := start + 1; i < len(lines); i++ {
		if isFrontMatterClose(lines[i], delimiter) {
			return i + 1
		}
	}
	return len(lines)
}

func isFrontMatterClose(line Line, delimiter byte) bool {
	trimmed := trimSpaceTab(line.Text())
	switch delimiter {
	case '-':
		return bytes.Equal(trimmed, []byte("---"))
	case '+':
		return bytes.Equal(trimmed, []byte("+++"))
	default:
		return false
	}
}

func consumeFencedCode(lines []Line, start int) int {
	opener := lines[start].Fence
	for i := start + 1; i < len(lines); i++ {
		if isClosingFence(lines[i], opener) {
			return i + 1
		}
	}
	return len(lines)
}

func isClosingFence(line Line, opener Fence) bool {
	text := line.Text()
	offset, ok := skipUpToThreeSpaces(text)
	if !ok || offset >= len(text) || text[offset] != opener.Char {
		return false
	}
	length := 0
	for offset+length < len(text) && text[offset+length] == opener.Char {
		length++
	}
	if length < opener.Length {
		return false
	}
	return isBlank(text[offset+length:])
}

func consumeIndentedCode(lines []Line, start int) int {
	i := start
	for i < len(lines) && (lines[i].Kind == LineIndentedCode || lines[i].Kind == LineBlank) {
		i++
	}
	return i
}

// CommonMark HTML block end conditions by type:
// 1. <script>, <pre>, <style>, <textarea>: line containing the matching closing tag.
// 2. <!-- comment: line containing -->.
// 3. <? processing instruction: line containing ?>.
// 4. <! declaration: line containing >.
// 5. <![CDATA[: line containing ]]>.
// 6. Block-level open/close tag: the next blank line.
// 7. Complete open/close tag on a line by itself: the next blank line.
func consumeHTMLBlock(lines []Line, start int) int {
	htmlType := lines[start].HTMLType
	if htmlType == 6 || htmlType == 7 {
		i := start
		for i < len(lines) && lines[i].Kind != LineBlank {
			i++
		}
		return i
	}
	for i := start; i < len(lines); i++ {
		if htmlEndCondition(lines[i], lines[start]) {
			return i + 1
		}
	}
	return len(lines)
}

func htmlEndCondition(line Line, opener Line) bool {
	text := line.Text()
	switch opener.HTMLType {
	case 1:
		tag := htmlType1Tag(opener.Text())
		if tag == "" {
			return false
		}
		return bytes.Contains(asciiLower(text), []byte("</"+tag+">"))
	case 2:
		return bytes.Contains(text, []byte("-->"))
	case 3:
		return bytes.Contains(text, []byte("?>"))
	case 4:
		return bytes.Contains(text, []byte(">"))
	case 5:
		return bytes.Contains(text, []byte("]]>"))
	default:
		return false
	}
}

func htmlType1Tag(text []byte) string {
	offset, ok := skipUpToThreeSpaces(text)
	if !ok {
		return ""
	}
	lower := asciiLower(text[offset:])
	for _, tag := range []string{"script", "pre", "style", "textarea"} {
		prefix := []byte("<" + tag)
		if bytes.HasPrefix(lower, prefix) && htmlTagBoundary(lower[len(prefix):]) {
			return tag
		}
	}
	return ""
}

func consumeBlockquote(lines []Line, start int) (int, []Block) {
	var inner []Line
	i := start
	canLazy := false
	for i < len(lines) {
		line := lines[i]
		if line.Kind == LineBlockquote {
			innerLine := deriveLine(line, line.BlockquotePrefixLen, false)
			inner = append(inner, innerLine)
			canLazy = canLazyAfter(innerLine)
			i++
			continue
		}
		if line.Kind == LineBlank {
			break
		}
		if canLazy && isLazyContinuationLine(line) {
			innerLine := deriveLine(line, 0, true)
			inner = append(inner, innerLine)
			canLazy = canLazyAfter(innerLine)
			i++
			continue
		}
		break
	}
	return i, groupLines(inner)
}

func consumeList(lines []Line, start int) (int, []Block) {
	var blocks []Block
	i := start
	for i < len(lines) && lines[i].Kind == LineListMarker {
		next, itemBlocks := consumeListItem(lines, i)
		blocks = append(blocks, itemBlocks...)
		i = next
		if i >= len(lines) || lines[i].Kind == LineBlank || lines[i].Kind != LineListMarker {
			break
		}
	}
	return i, blocks
}

func consumeListItem(lines []Line, start int) (int, []Block) {
	opener := lines[start]
	contentColumn := columnAtOffset(opener.Text(), opener.ListMarker.ContentOffset)
	inner := []Line{deriveLine(opener, opener.ListMarker.ContentOffset, false)}
	canLazy := canLazyAfter(inner[0])

	i := start + 1
	for i < len(lines) {
		line := lines[i]
		if line.Kind == LineBlank {
			break
		}
		if line.Kind == LineListMarker && line.Indent <= opener.ListMarker.MarkerOffset {
			break
		}
		if line.Indent >= contentColumn {
			innerLine := deriveLine(line, offsetForColumn(line.Text(), contentColumn), false)
			inner = append(inner, innerLine)
			canLazy = canLazyAfter(innerLine)
			i++
			continue
		}
		if canLazy && isLazyContinuationLine(line) {
			innerLine := deriveLine(line, 0, true)
			inner = append(inner, innerLine)
			canLazy = canLazyAfter(innerLine)
			i++
			continue
		}
		break
	}
	return i, groupLines(inner)
}

func deriveLine(line Line, prefixLen int, lazy bool) Line {
	text, ending := splitLineEnding(line.Raw)
	if prefixLen > len(text) {
		prefixLen = len(text)
	}
	raw := make([]byte, 0, len(text)-prefixLen+len(ending))
	raw = append(raw, text[prefixLen:]...)
	raw = append(raw, ending...)

	derived := classifyLine(1, raw)
	derived.Index = line.Index
	derived.Prefix = append(append([]byte(nil), line.Prefix...), text[:prefixLen]...)
	if lazy {
		derived.Kind = LineParagraphText
		derived.SetextCandidate = false
	}
	return derived
}

func canLazyAfter(line Line) bool {
	switch line.Kind {
	case LineParagraphText, LineSetextUnderline, LineTableDelimiter, LineIndentedCode:
		return true
	case LineListMarker:
		return !interruptsParagraph(line)
	default:
		return false
	}
}

func isLazyContinuationLine(line Line) bool {
	if line.Kind == LineBlank {
		return false
	}
	if line.Kind == LineHRule && line.SetextCandidate {
		return true
	}
	return !interruptsParagraph(line)
}

func columnAtOffset(text []byte, offset int) int {
	if offset > len(text) {
		offset = len(text)
	}
	column := 0
	for i := 0; i < offset; i++ {
		if text[i] == '\t' {
			column += 4 - column%4
		} else {
			column++
		}
	}
	return column
}

func offsetForColumn(text []byte, target int) int {
	column := 0
	for i, b := range text {
		if column >= target {
			return i
		}
		if b == '\t' {
			column += 4 - column%4
		} else {
			column++
		}
	}
	return len(text)
}

func consumeParagraph(lines []Line, start int) (int, Block) {
	var paragraph []Line
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if line.Kind == LineBlank {
			break
		}
		if len(paragraph) > 0 {
			if line.SetextCandidate && sameContainerPrefix(paragraph[len(paragraph)-1], line) && !(line.ListMarker.Valid && line.ListMarker.Empty) {
				headingLines := append(append([]Line(nil), paragraph...), line)
				return i + 1, block(BlockSetextHeading, headingLines)
			}
			if line.Kind == LineTableDelimiter {
				if tableLines, ok := tableFromParagraph(lines, paragraph, i); ok {
					return i + len(tableLines) - len(paragraph), block(BlockTable, tableLines)
				}
			}
			if interruptsParagraph(line) {
				break
			}
		} else if !canStartParagraph(line) {
			return i + 1, block(BlockRaw, []Line{line})
		}
		paragraph = append(paragraph, line)
	}
	return start + len(paragraph), block(BlockParagraph, paragraph)
}

func sameContainerPrefix(a Line, b Line) bool {
	return bytes.Equal(a.Prefix, b.Prefix)
}

func tableFromParagraph(lines []Line, paragraph []Line, delimiterIndex int) ([]Line, bool) {
	if len(paragraph) != 1 {
		return nil, false
	}
	header := paragraph[0]
	delimiter := lines[delimiterIndex]
	if header.Kind != LineParagraphText || delimiter.Kind != LineTableDelimiter {
		return nil, false
	}
	if countTableCells(header.Text()) != delimiter.TableDelimiter.CellCount {
		return nil, false
	}

	end := delimiterIndex + 1
	for end < len(lines) && tableBodyContinues(lines[end]) {
		end++
	}

	tableLines := make([]Line, 0, end-delimiterIndex+len(paragraph))
	tableLines = append(tableLines, paragraph...)
	tableLines = append(tableLines, lines[delimiterIndex:end]...)
	return tableLines, true
}

func tableBodyContinues(line Line) bool {
	if line.Kind == LineBlank {
		return false
	}
	if line.Kind == LineTableDelimiter {
		return true
	}
	return !interruptsParagraph(line)
}

func countTableCells(text []byte) int {
	return len(splitTableCells(text))
}

func canStartParagraph(line Line) bool {
	switch line.Kind {
	case LineParagraphText, LineSetextUnderline, LineTableDelimiter:
		return true
	default:
		return false
	}
}

func interruptsParagraph(line Line) bool {
	switch line.Kind {
	case LineBlank, LineATXHeading, LineHRule, LineFence, LineBlockquote, LineReferenceDefinition:
		return true
	case LineFrontMatterDelimiter:
		return line.Index == 0
	case LineHTMLBlockStart:
		return line.HTMLType >= 1 && line.HTMLType <= 6
	case LineIndentedCode, LineTableDelimiter, LineSetextUnderline, LineParagraphText:
		return false
	case LineListMarker:
		if line.ListMarker.Ordered {
			return line.ListMarker.Start == 1 && !line.ListMarker.Empty
		}
		return !line.ListMarker.Empty
	default:
		return false
	}
}
