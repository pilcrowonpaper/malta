package malta

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var config struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Domain      string                 `json:"domain"`
	Twitter     string                 `json:"twitter"`
	Sidebar     []SidebarSectionConfig `json:"sidebar"`
}
var markdownFilePaths []string

//go:embed assets/template.html
var htmlTemplate []byte

//go:embed assets/main.css
var mainCss []byte

//go:embed assets/markdown.css
var markdownCss []byte

func main() {
	configJson, err := os.ReadFile("malta.config.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Missing 'malta.config.json'")
			return
		}
		panic(err)
	}

	json.Unmarshal(configJson, &config)
	if config.Name == "" {
		fmt.Println("Missing config: name")
		return
	}
	if config.Domain == "" {
		fmt.Println("Missing config: domain")
		return
	}
	if config.Description == "" {
		fmt.Println("Missing config: description")
		return
	}

	navSections := []NavSection{}
	for _, sidebarSection := range config.Sidebar {
		navSection := NavSection{sidebarSection.Title, []NavPage{}}
		for _, sidebarSectionPage := range sidebarSection.Pages {
			navPage := NavPage{Title: sidebarSectionPage[0], Href: sidebarSectionPage[1]}
			navSection.Pages = append(navSection.Pages, navPage)
		}
		navSections = append(navSections, navSection)
	}

	if err := filepath.Walk("pages", walkPagesDir); err != nil {
		panic(err)
	}

	markdown := goldmark.New()
	markdown.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(&astTransformer{}, 500)))
	markdown.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&codeBlockLinkRenderer{}, 100)))

	os.RemoveAll("dist")

	for _, markdownFilePath := range markdownFilePaths {
		fmt.Println(markdownFilePath)
		var matter struct {
			Title string `yaml:"title"`
		}

		markdownFile, _ := os.Open(markdownFilePath)
		defer markdownFile.Close()
		pageMarkdown, _ := frontmatter.MustParse(markdownFile, &matter)
		if matter.Title == "" {
			fmt.Printf("Page %s missing attribute: title\n", markdownFilePath)
			return
		}

		var markdownHtmlBuf bytes.Buffer

		if err := markdown.Convert(pageMarkdown, &markdownHtmlBuf, parser.WithContext(parser.NewContext())); err != nil {
			panic(err)
		}

		tmpl, _ := template.New("html").Parse(string(htmlTemplate))

		dstPath := strings.Replace(strings.Replace(markdownFilePath, "pages", "dist", 1), ".md", ".html", 1)

		if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
			panic(err)
		}
		dstHtmlFile, err := os.Create(dstPath)
		if err != nil {
			panic(err)
		}
		defer dstHtmlFile.Close()

		urlPathname := strings.Replace(strings.Replace(dstPath, "dist", "", 1), ".html", "", 1)
		urlPathname = strings.Replace(urlPathname, "/index", "", 1)

		err = tmpl.Execute(dstHtmlFile, Data{
			Markdown:    template.HTML(markdownHtmlBuf.String()),
			Name:        config.Name,
			Description: config.Description,
			Url:         config.Domain + urlPathname,
			Twitter:     config.Twitter,
			Title:       matter.Title,
			NavSections: navSections,
		})
		if err != nil {
			panic(err)
		}
	}

	os.WriteFile("dist/main.css", mainCss, os.ModePerm)
	os.WriteFile("dist/markdown.css", markdownCss, os.ModePerm)
}

func walkPagesDir(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	markdownFilePaths = append(markdownFilePaths, path)
	return nil
}

type astTransformer struct{}

func (a *astTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
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
			if !strings.HasPrefix(lineValue, "//$") {
				break
			}
			defCount += 1
			keyValue := strings.Split(strings.TrimSpace(strings.Replace(lineValue, "//$", "", 1)), "=")
			if len(keyValue) != 2 {
				continue
			}
			n.SetAttribute([]byte("link:"+keyValue[0]), keyValue[1])
		}
		n.Lines().SetSliced(defCount, n.Lines().Len())
		return ast.WalkContinue, nil
	}
	ast.Walk(node, walker)
}

type codeBlockLinkRenderer struct{}

func (r codeBlockLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderCustomCodeBlockLinks)
}

func (r codeBlockLinkRenderer) renderCustomCodeBlockLinks(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		return ast.WalkContinue, nil
	}
	codeBlock := node.(*ast.FencedCodeBlock)

	var content string
	for i := 0; i < codeBlock.Lines().Len(); i++ {
		line := codeBlock.Lines().At(i)
		content += string(line.Value(source))
	}
	lexer := lexers.Get(string(codeBlock.Language(source)))
	if lexer == nil {
		w.WriteString("<pre class=\"codeblock\"><code>")
		w.WriteString(content)
		w.WriteString("</code></pre>")
		return ast.WalkContinue, nil
	}
	lexer = chroma.Coalesce(lexer)

	formatter := html.New(html.WithClasses(true), html.PreventSurroundingPre(true))

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
		html = strings.ReplaceAll(html, "$$"+target, fmt.Sprintf("<a href=\"%s\">%s</a>", dest, target))
	}

	w.WriteString("<pre class=\"codeblock\"><code>")
	w.WriteString(html)
	w.WriteString("</code></pre>")

	return ast.WalkContinue, nil
}

type Data struct {
	Markdown    template.HTML
	Title       string
	Description string
	Twitter     string
	Url         string
	Name        string
	NavSections []NavSection
}

type NavSection struct {
	Title string
	Pages []NavPage
}

type NavPage struct {
	Title string
	Href  string
}

type SidebarSectionConfig struct {
	Title string     `json:"title"`
	Pages [][]string `json:"pages"`
}
