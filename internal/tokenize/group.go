package tokenize

import "bytes"

func Group(doc *Document) []Block {
	if doc == nil || len(doc.Lines) == 0 {
		return nil
	}

	var blocks []Block
	for i := 0; i < len(doc.Lines); {
		line := doc.Lines[i]
		switch line.Kind {
		case LineBlank:
			start := i
			for i < len(doc.Lines) && doc.Lines[i].Kind == LineBlank {
				i++
			}
			blocks = append(blocks, block(BlockBlank, doc.Lines[start:i]))
		case LineFrontMatterDelimiter:
			next := consumeFrontMatter(doc.Lines, i)
			blocks = append(blocks, block(BlockFrontMatter, doc.Lines[i:next]))
			i = next
		case LineATXHeading:
			blocks = append(blocks, block(BlockATXHeading, []Line{line}))
			i++
		case LineHRule:
			blocks = append(blocks, block(BlockHRule, []Line{line}))
			i++
		case LineFence:
			next := consumeFencedCode(doc.Lines, i)
			blocks = append(blocks, block(BlockFencedCode, doc.Lines[i:next]))
			i = next
		case LineHTMLBlockStart:
			next := consumeHTMLBlock(doc.Lines, i)
			html := block(BlockHTML, doc.Lines[i:next])
			html.HTMLType = line.HTMLType
			blocks = append(blocks, html)
			i = next
		case LineIndentedCode:
			next := consumeIndentedCode(doc.Lines, i)
			blocks = append(blocks, block(BlockIndentedCode, doc.Lines[i:next]))
			i = next
		case LineReferenceDefinition:
			start := i
			for i < len(doc.Lines) && doc.Lines[i].Kind == LineReferenceDefinition {
				i++
			}
			blocks = append(blocks, block(BlockReferenceDefinition, doc.Lines[start:i]))
		case LineBlockquote, LineListMarker:
			start := i
			for i < len(doc.Lines) && doc.Lines[i].Kind != LineBlank {
				i++
			}
			blocks = append(blocks, block(BlockRaw, doc.Lines[start:i]))
		default:
			next, paragraph := consumeParagraph(doc.Lines, i)
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

func consumeParagraph(lines []Line, start int) (int, Block) {
	var paragraph []Line
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if line.Kind == LineBlank {
			break
		}
		if len(paragraph) > 0 {
			if line.SetextCandidate && !(line.ListMarker.Valid && line.ListMarker.Empty) {
				headingLines := append(append([]Line(nil), paragraph...), line)
				return i + 1, block(BlockSetextHeading, headingLines)
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
