package build

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/alecthomas/chroma"
	htmlFormatter "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type codeBlockLinksAstTransformer struct{}

func (a codeBlockLinksAstTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	walker := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindFencedCodeBlock {
			return ast.WalkContinue, nil
		}
		lineCount := n.Lines().Len()
		defCount := 0
		for i := 0; i < lineCount; i++ {
			lineValue := string(reader.Value(n.Lines().At(i)))
			if strings.HasPrefix(lineValue, "//$") {
				defCount += 1
				keyValue := strings.Split(strings.TrimSpace(strings.Replace(lineValue, "//$", "", 1)), "=")
				if len(keyValue) != 2 {
					continue
				}
				n.SetAttribute([]byte("link:"+keyValue[0]), keyValue[1])
			}
		}
		n.Lines().SetSliced(defCount, n.Lines().Len())
		return ast.WalkContinue, nil
	}
	ast.Walk(node, walker)
}

type codeBlockLinksRenderer struct{}

func (r codeBlockLinksRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderCustomCodeBlockLinks)
}

func (r codeBlockLinksRenderer) renderCustomCodeBlockLinks(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		return ast.WalkContinue, nil
	}
	codeBlock := node.(*ast.FencedCodeBlock)

	var content string
	for i := 0; i < codeBlock.Lines().Len(); i++ {
		line := codeBlock.Lines().At(i)
		content += string(line.Value(source))
	}
	for _, attribute := range node.Attributes() {
		attributeName := string(attribute.Name)
		if !strings.HasPrefix(attributeName, "link:") {
			continue
		}
		target := strings.Replace(attributeName, "link:", "", 1)
		content = strings.ReplaceAll(content, "$$"+target, "__MALTA_CODEBLOCK_LINK_"+target)
		content = strings.ReplaceAll(content, "$\\$"+target, "$$"+target)
	}
	lexer := lexers.Get(string(codeBlock.Language(source)))
	if lexer == nil {
		w.WriteString("<pre class=\"codeblock\"><code>")
		w.WriteString(html.EscapeString(content))
		w.WriteString("</code></pre>")
		return ast.WalkContinue, nil
	}
	lexer = chroma.Coalesce(lexer)

	formatter := htmlFormatter.New(htmlFormatter.WithClasses(true), htmlFormatter.PreventSurroundingPre(true))

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return ast.WalkStop, err
	}
	buf := new(bytes.Buffer)
	// random style
	formatter.Format(buf, styles.GitHub, iterator)

	html := buf.String()
	for _, attribute := range node.Attributes() {
		attributeName := string(attribute.Name)
		if !strings.HasPrefix(attributeName, "link:") {
			continue
		}
		target := strings.Replace(attributeName, "link:", "", 1)
		dest := attribute.Value.(string)
		html = strings.ReplaceAll(html, "__MALTA_CODEBLOCK_LINK_"+target, fmt.Sprintf("<a href=\"%s\">%s</a>", dest, target))
	}

	w.WriteString(fmt.Sprintf("<pre class=\"codeblock\"><code class=\"%s\">", string(codeBlock.Language(source))))
	w.WriteString(html)
	w.WriteString("</code class=%s></pre>")

	return ast.WalkContinue, nil
}
