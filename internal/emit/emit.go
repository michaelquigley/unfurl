package emit

import (
	"bytes"
	"fmt"
	"io"

	"github.com/michaelquigley/unfurl/internal/reflow"
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
			_, err = w.Write(reflow.ReflowLines(block.Lines))
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
		if _, err := w.Write(line.Prefix); err != nil {
			return fmt.Errorf("write %s block prefix: %w", block.Kind, err)
		}
		if _, err := w.Write(line.Raw); err != nil {
			return fmt.Errorf("write %s block: %w", block.Kind, err)
		}
	}
	return nil
}
