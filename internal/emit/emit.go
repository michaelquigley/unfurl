package emit

import (
	"bytes"
	"fmt"
	"io"

	"github.com/michaelquigley/unfurl/internal/tokenize"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func Emit(w io.Writer, doc *tokenize.Document, blocks []tokenize.Block) error {
	if doc != nil && doc.HadBOM {
		if _, err := w.Write(utf8BOM); err != nil {
			return fmt.Errorf("write BOM: %w", err)
		}
	}
	for _, block := range blocks {
		var err error
		if block.Kind == tokenize.BlockParagraph {
			_, err = w.Write(reflowParagraph(block.Lines))
		} else {
			err = writeRawBlock(w, block)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func Bytes(doc *tokenize.Document, blocks []tokenize.Block) ([]byte, error) {
	var out bytes.Buffer
	if err := Emit(&out, doc, blocks); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeRawBlock(w io.Writer, block tokenize.Block) error {
	for _, line := range block.Lines {
		if _, err := w.Write(line.Raw); err != nil {
			return fmt.Errorf("write %s block: %w", block.Kind, err)
		}
	}
	return nil
}

func reflowParagraph(lines []tokenize.Line) []byte {
	if len(lines) == 0 {
		return nil
	}

	var out bytes.Buffer
	for i, line := range lines {
		text := line.Text()
		if i > 0 {
			out.WriteByte(' ')
			text = bytes.TrimLeft(text, " \t")
		}
		if i < len(lines)-1 {
			text = bytes.TrimRight(text, " \t")
		}
		out.Write(text)
	}
	out.Write(lines[len(lines)-1].LineEnding())
	return out.Bytes()
}
