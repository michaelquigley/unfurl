package reflow

import (
	"bytes"

	"github.com/michaelquigley/unfurl/internal/tokenize"
)

type Segment struct {
	Prefix          []byte
	Lines           []tokenize.Line
	HardBreakMarker []byte
}

type Paragraph struct {
	Segments []Segment
}

func NewParagraph(lines []tokenize.Line) Paragraph {
	if len(lines) == 0 {
		return Paragraph{}
	}

	var paragraph Paragraph
	var current Segment
	for _, line := range lines {
		if len(current.Lines) == 0 {
			current.Prefix = nil
		}
		current.Lines = append(current.Lines, line)
		if line.HardBreak {
			current.HardBreakMarker = append([]byte(nil), line.HardBreakMarker...)
			paragraph.Segments = append(paragraph.Segments, current)
			current = Segment{}
		}
	}
	if len(current.Lines) > 0 {
		paragraph.Segments = append(paragraph.Segments, current)
	}
	return paragraph
}

func ReflowLines(lines []tokenize.Line) []byte {
	return Reflow(NewParagraph(lines))
}

func Reflow(paragraph Paragraph) []byte {
	var out bytes.Buffer
	for _, segment := range paragraph.Segments {
		if len(segment.Lines) == 0 {
			continue
		}
		out.Write(segment.Prefix)
		out.Write(joinSegment(segment))
		if segment.HardBreakMarker != nil {
			out.Write(segment.HardBreakMarker)
		}
		out.Write(segment.Lines[len(segment.Lines)-1].LineEnding())
	}
	return out.Bytes()
}

func joinSegment(segment Segment) []byte {
	var out bytes.Buffer
	for i := range segment.Lines {
		text := segmentLineText(segment, i)
		if i > 0 {
			out.WriteByte(' ')
			text = bytes.TrimLeft(text, " \t")
		}
		if i < len(segment.Lines)-1 {
			text = bytes.TrimRight(text, " \t")
		}
		out.Write(text)
	}
	return out.Bytes()
}

func segmentLineText(segment Segment, index int) []byte {
	line := segment.Lines[index]
	text := line.Text()
	if index == len(segment.Lines)-1 && segment.HardBreakMarker != nil {
		return trimHardBreakMarker(text, segment.HardBreakMarker)
	}
	return text
}

func trimHardBreakMarker(text []byte, marker []byte) []byte {
	if len(marker) == 0 || len(marker) > len(text) {
		return text
	}
	if bytes.Equal(text[len(text)-len(marker):], marker) {
		return text[:len(text)-len(marker)]
	}
	return text
}
