package unfurl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
)

func TestPropertyASTEquivalence(t *testing.T) {
	for _, fixture := range propertyFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			src := readFixture(t, fixture.path)
			out, err := UnfurlBytes(src)
			if err != nil {
				t.Fatalf("UnfurlBytes returned error: %v", err)
			}

			diffs, err := compareMarkdown(src, out)
			if err != nil {
				t.Fatalf("compare markdown: %v", err)
			}
			if len(diffs) > 0 {
				t.Fatalf("AST divergence:\n%s", formatDiffs(diffs))
			}
		})
	}
}

func TestPropertyIdempotence(t *testing.T) {
	for _, fixture := range propertyFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			src := readFixture(t, fixture.path)
			once, err := UnfurlBytes(src)
			if err != nil {
				t.Fatalf("first UnfurlBytes returned error: %v", err)
			}
			twice, err := UnfurlBytes(once)
			if err != nil {
				t.Fatalf("second UnfurlBytes returned error: %v", err)
			}
			if !bytes.Equal(twice, once) {
				t.Fatalf("not idempotent:\nonce  %q\ntwice %q", once, twice)
			}
		})
	}
}

func TestScenarioClaudeOutput(t *testing.T) {
	src := readFixture(t, "testdata/property/scenario-claude-output.md")
	out, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if bytes.Count(out, []byte("\n")) >= bytes.Count(src, []byte("\n")) {
		t.Fatalf("expected fewer physical lines after unfurl")
	}
	diffs, err := compareMarkdown(src, out)
	if err != nil {
		t.Fatalf("compare markdown: %v", err)
	}
	if len(diffs) > 0 {
		t.Fatalf("AST divergence:\n%s", formatDiffs(diffs))
	}
}

type propertyFixture struct {
	name string
	path string
}

func propertyFixtures(t *testing.T) []propertyFixture {
	t.Helper()
	var fixtures []propertyFixture
	entries, err := os.ReadDir("testdata/property")
	if err != nil {
		t.Fatalf("read property fixtures: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		fixtures = append(fixtures, propertyFixture{
			name: strings.TrimSuffix(entry.Name(), ".md"),
			path: filepath.Join("testdata/property", entry.Name()),
		})
	}
	fixtures = append(fixtures, propertyFixture{
		name: "unfurl-spec",
		path: "docs/future/unfurl-spec.md",
	})
	sort.Slice(fixtures, func(i, j int) bool {
		return fixtures[i].name < fixtures[j].name
	})
	return fixtures
}

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return src
}

type parsedMarkdown struct {
	source      []byte
	root        gast.Node
	frontMatter map[string]any
}

func parseMarkdown(src []byte) (parsedMarkdown, error) {
	// Pure CommonMark mode would misread front matter as a setext heading and fail
	// the comparison for reasons unrelated to anything the tool is doing.
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&frontmatter.Extender{},
		),
	)
	ctx := parser.NewContext()
	root := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))
	meta, err := frontMatterMetadata(ctx)
	if err != nil {
		return parsedMarkdown{}, err
	}
	return parsedMarkdown{source: src, root: root, frontMatter: meta}, nil
}

func frontMatterMetadata(ctx parser.Context) (map[string]any, error) {
	data := frontmatter.Get(ctx)
	if data == nil {
		return nil, nil
	}
	var meta map[string]any
	if err := data.Decode(&meta); err != nil {
		return nil, fmt.Errorf("decode front matter: %w", err)
	}
	return meta, nil
}

func compareMarkdown(before []byte, after []byte) ([]propertyDiff, error) {
	expected, err := parseMarkdown(before)
	if err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}
	actual, err := parseMarkdown(after)
	if err != nil {
		return nil, fmt.Errorf("parse output: %w", err)
	}

	var diffs []propertyDiff
	if !reflect.DeepEqual(expected.frontMatter, actual.frontMatter) {
		diffs = append(diffs, propertyDiff{
			path:     "/frontmatter",
			expected: fmt.Sprintf("%#v", expected.frontMatter),
			actual:   fmt.Sprintf("%#v", actual.frontMatter),
		})
	}

	expectedNode := canonicalize(expected.root, expected.source)
	actualNode := canonicalize(actual.root, actual.source)
	diffs = append(diffs, compareCanonical("/", expectedNode, actualNode)...)
	return diffs, nil
}

type canonicalNode struct {
	Kind     string
	Attrs    []string
	Value    string
	Children []canonicalNode
}

func canonicalize(node gast.Node, source []byte) canonicalNode {
	if textValue, ok := canonicalTextNode(node, source); ok {
		return canonicalNode{Kind: "#text", Value: textValue}
	}

	c := canonicalNode{
		Kind:  node.Kind().String(),
		Attrs: nodeAttrs(node, source),
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		c.appendChild(canonicalize(child, source))
	}
	return c
}

func (c *canonicalNode) appendChild(child canonicalNode) {
	if child.Kind == "#text" && child.Value == "" {
		return
	}
	if child.Kind == "#text" && len(c.Children) > 0 && c.Children[len(c.Children)-1].Kind == "#text" {
		c.Children[len(c.Children)-1].Value += child.Value
		return
	}
	c.Children = append(c.Children, child)
}

func canonicalTextNode(node gast.Node, source []byte) (string, bool) {
	switch n := node.(type) {
	case *gast.Text:
		value := append([]byte(nil), n.Value(source)...)
		if n.SoftLineBreak() {
			value = trimSingleTrailingSpace(value)
			value = append(value, ' ')
		}
		if n.HardLineBreak() {
			value = append(value, '\n')
		}
		return string(value), true
	case *gast.String:
		return string(n.Value), true
	default:
		return "", false
	}
}

func trimSingleTrailingSpace(value []byte) []byte {
	if len(value) == 0 || value[len(value)-1] != ' ' {
		return value
	}
	if len(value) >= 2 && value[len(value)-2] == ' ' {
		return value
	}
	return value[:len(value)-1]
}

func nodeAttrs(node gast.Node, source []byte) []string {
	var attrs []string
	switch n := node.(type) {
	case *gast.Heading:
		attrs = append(attrs, fmt.Sprintf("level=%d", n.Level))
	case *gast.List:
		attrs = append(attrs,
			fmt.Sprintf("marker=%q", n.Marker),
			fmt.Sprintf("ordered=%t", n.IsOrdered()),
			fmt.Sprintf("start=%d", n.Start),
			fmt.Sprintf("tight=%t", n.IsTight),
		)
	case *gast.ListItem:
		attrs = append(attrs, fmt.Sprintf("offset=%d", n.Offset))
	case *gast.FencedCodeBlock:
		attrs = append(attrs,
			fmt.Sprintf("info=%q", fencedInfo(n, source)),
			fmt.Sprintf("text=%q", n.Text(source)),
		)
	case *gast.CodeBlock:
		attrs = append(attrs, fmt.Sprintf("text=%q", n.Text(source)))
	case *gast.HTMLBlock:
		attrs = append(attrs,
			fmt.Sprintf("type=%d", n.HTMLBlockType),
			fmt.Sprintf("text=%q", n.Text(source)),
		)
	case *gast.LinkReferenceDefinition:
		attrs = append(attrs,
			fmt.Sprintf("label=%q", n.Label),
			fmt.Sprintf("destination=%q", n.Destination),
			fmt.Sprintf("title=%q", n.Title),
		)
	case *gast.Link:
		attrs = append(attrs, linkAttrs(&n.BaseInline, n.Destination, n.Title, n.Reference)...)
	case *gast.Image:
		attrs = append(attrs, linkAttrs(&n.BaseInline, n.Destination, n.Title, n.Reference)...)
	case *gast.AutoLink:
		attrs = append(attrs,
			fmt.Sprintf("type=%d", n.AutoLinkType),
			fmt.Sprintf("protocol=%q", n.Protocol),
			fmt.Sprintf("url=%q", n.URL(source)),
			fmt.Sprintf("label=%q", n.Label(source)),
		)
	case *gast.RawHTML:
		attrs = append(attrs, fmt.Sprintf("text=%q", n.Text(source)))
	case *gast.CodeSpan:
		attrs = append(attrs, fmt.Sprintf("text=%q", n.Text(source)))
	case *gast.Emphasis:
		attrs = append(attrs, fmt.Sprintf("level=%d", n.Level))
	case *east.Table:
		attrs = append(attrs, fmt.Sprintf("alignments=%v", n.Alignments))
	case *east.TableRow:
		attrs = append(attrs, fmt.Sprintf("alignments=%v", n.Alignments))
	case *east.TableHeader:
		attrs = append(attrs, fmt.Sprintf("alignments=%v", n.Alignments))
	case *east.TableCell:
		attrs = append(attrs, fmt.Sprintf("alignment=%v", n.Alignment))
	case *east.TaskCheckBox:
		attrs = append(attrs, fmt.Sprintf("checked=%t", n.IsChecked))
	}
	sort.Strings(attrs)
	return attrs
}

func fencedInfo(n *gast.FencedCodeBlock, source []byte) []byte {
	if n.Info == nil {
		return nil
	}
	return n.Info.Text(source)
}

func linkAttrs(_ *gast.BaseInline, destination []byte, title []byte, ref *gast.ReferenceLink) []string {
	attrs := []string{
		fmt.Sprintf("destination=%q", destination),
		fmt.Sprintf("title=%q", title),
	}
	if ref != nil {
		attrs = append(attrs,
			fmt.Sprintf("reference_type=%s", ref.Type.String()),
			fmt.Sprintf("reference_value=%q", ref.Value),
		)
	}
	return attrs
}

type propertyDiff struct {
	path     string
	expected string
	actual   string
}

func compareCanonical(path string, expected canonicalNode, actual canonicalNode) []propertyDiff {
	var diffs []propertyDiff
	if expected.Kind != actual.Kind {
		return append(diffs, propertyDiff{path: path, expected: expected.Kind, actual: actual.Kind})
	}
	if !reflect.DeepEqual(expected.Attrs, actual.Attrs) {
		diffs = append(diffs, propertyDiff{
			path:     path + "/@" + expected.Kind,
			expected: fmt.Sprintf("%v", expected.Attrs),
			actual:   fmt.Sprintf("%v", actual.Attrs),
		})
	}
	if expected.Value != actual.Value {
		diffs = append(diffs, propertyDiff{
			path:     path + "/text()",
			expected: expected.Value,
			actual:   actual.Value,
		})
	}
	if len(expected.Children) != len(actual.Children) {
		diffs = append(diffs, propertyDiff{
			path:     path + "/children",
			expected: fmt.Sprintf("%d", len(expected.Children)),
			actual:   fmt.Sprintf("%d", len(actual.Children)),
		})
	}
	n := min(len(expected.Children), len(actual.Children))
	for i := 0; i < n; i++ {
		childPath := fmt.Sprintf("%s/%s[%d]", strings.TrimRight(path, "/"), expected.Children[i].Kind, i)
		diffs = append(diffs, compareCanonical(childPath, expected.Children[i], actual.Children[i])...)
	}
	return diffs
}

func formatDiffs(diffs []propertyDiff) string {
	var b strings.Builder
	for _, diff := range diffs {
		fmt.Fprintf(&b, "%s\n  expected: %s\n  actual:   %s\n", diff.path, diff.expected, diff.actual)
	}
	return b.String()
}
