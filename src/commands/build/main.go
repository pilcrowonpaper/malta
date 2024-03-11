package build

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
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

func BuildCommand() int {
	configJson, err := os.ReadFile("malta.config.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Missing 'malta.config.json'")
			return 1
		}
		panic(err)
	}

	json.Unmarshal(configJson, &config)
	if config.Name == "" {
		fmt.Println("Missing config: name")
		return 1
	}
	if config.Domain == "" {
		fmt.Println("Missing config: domain")
		return 1
	}
	if config.Description == "" {
		fmt.Println("Missing config: description")
		return 1
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

	markdown := goldmark.New(goldmark.WithExtensions(extension.Table))
	markdown.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(&codeBlockLinksAstTransformer{}, 500)), parser.WithAutoHeadingID())
	markdown.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&codeBlockLinksRenderer{}, 100)))

	os.RemoveAll("dist")

	tmpl, _ := template.New("html").Parse(string(htmlTemplate))

	for _, markdownFilePath := range markdownFilePaths {
		var matter struct {
			Title string `yaml:"title"`
		}

		markdownFile, _ := os.Open(markdownFilePath)
		defer markdownFile.Close()
		pageMarkdown, _ := frontmatter.MustParse(markdownFile, &matter)
		if matter.Title == "" {
			fmt.Printf("Page %s missing attribute: title\n", markdownFilePath)
			return 1
		}

		var markdownHtmlBuf bytes.Buffer

		if err := markdown.Convert(pageMarkdown, &markdownHtmlBuf, parser.WithContext(parser.NewContext())); err != nil {
			panic(err)
		}

		markdownHtml := markdownHtmlBuf.String()
		markdownHtml = strings.ReplaceAll(markdownHtml, "<table>", "<div class=\"table-wrapper\"><table>")
		markdownHtml = strings.ReplaceAll(markdownHtml, "</table>", "</table></div>")

		markdownContent := markdownHtmlBuf.Bytes()
		headings := extractHeadings(markdownContent)
		for _, heading := range headings {
			anchorTag := fmt.Sprintf(`<a href="#%s">%s</a>`, heading.Anchor, heading.Text)

			markdownHtml = strings.ReplaceAll(markdownHtml, heading.OriginalText, fmt.Sprintf("<h%s id=\"%s\">%s</h%s>", heading.Level, heading.Anchor, anchorTag, heading.Level))
		}

		dstPath := strings.Replace(strings.Replace(markdownFilePath, "pages/", "dist/", 1), ".md", ".html", 1)

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
		if urlPathname == "" {
			urlPathname = "/"
		}
		url := config.Domain + urlPathname

		var currentNavPageHref string

		for _, navSection := range navSections {
			for _, sectionPage := range navSection.Pages {
				if urlPathname == sectionPage.Href || strings.HasPrefix(urlPathname, sectionPage.Href+"/") {
					currentNavPageHref = sectionPage.Href
					break
				}
			}
		}

		var convertedHeadings []Heading
		for _, h := range headings {
			convertedHeadings = append(convertedHeadings, Heading{Anchor: h.Anchor, Text: h.Text, OriginalText: h.OriginalText})
		}

		err = tmpl.Execute(dstHtmlFile, Data{
			Markdown:           template.HTML(markdownHtml),
			Name:               config.Name,
			Description:        config.Description,
			Url:                url,
			Twitter:            config.Twitter,
			Title:              matter.Title,
			NavSections:        navSections,
			CurrentNavPageHref: currentNavPageHref,
			Headings:           convertedHeadings,
		})
		if err != nil {
			panic(err)
		}
	}

	notFoundDstHtmlFile, err := os.Create("dist/404.html")
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(notFoundDstHtmlFile, Data{
		Markdown:           template.HTML("<h1>404 - Not found</h1><p>The page you were looking for does not exist.</p>"),
		Name:               config.Name,
		Description:        config.Description,
		Url:                config.Domain,
		Twitter:            config.Twitter,
		Title:              "Not found",
		NavSections:        navSections,
		CurrentNavPageHref: "",
	})
	if err != nil {
		panic(err)
	}

	os.WriteFile("dist/main.css", mainCss, os.ModePerm)
	os.WriteFile("dist/markdown.css", markdownCss, os.ModePerm)
	return 0
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

func extractHeadings(markdownContent []byte) []Heading {
	headingRegex := regexp.MustCompile(`(?i)<h([2-6])\s+id="([^"]+)">([^<]+)<\/h[2-6]>`)

	headings := []Heading{}
	matches := headingRegex.FindAllSubmatch(markdownContent, -1)

	for _, match := range matches {
		level := string(match[1])
		text := strings.TrimSpace(html.UnescapeString(string(match[3])))
		anchor := strings.ToLower(strings.ReplaceAll(string(match[2]), " ", "-"))
		originalText := string(match[0])
		headings = append(headings, Heading{Level: level, Anchor: anchor, Text: text, OriginalText: originalText})
	}

	return headings
}

type Data struct {
	Markdown           template.HTML
	Title              string
	Description        string
	Twitter            string
	Url                string
	Headings           []Heading
	Name               string
	NavSections        []NavSection
	CurrentNavPageHref string
}

type Heading struct {
	Level        string
	Anchor       string
	Text         string
	OriginalText string
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
