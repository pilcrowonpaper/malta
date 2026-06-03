package build

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/pilcrowonpaper/malta/assets"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var markdown goldmark.Markdown
var tmpl *template.Template

func init() {
	markdown = goldmark.New(goldmark.WithExtensions(extension.Table))
	markdown.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(&codeBlockLinksAstTransformer{}, 500)), parser.WithAutoHeadingID())
	markdown.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&codeBlockLinksRenderer{}, 100)))

	htmlTemplate, err := assets.Read("template.html")
	if err != nil {
		log.Fatal("template.html does not exist")
	}
	tmpl, _ = template.New("html").Parse(string(htmlTemplate))
}

type HTMLBuilder struct {
	siteName          string
	siteDescription   string
	siteDomain        string
	siteTwitterHandle string
	faviconHref       string
	logoImageSrc      string
	ogImageURL        string
	navSections       []NavSection
	styleSheetSrc     []string
	alert             *Alert
}

type Alert struct {
	Message  string
	LinkText string
	LinkURL  string
}

type NavSection struct {
	Title string
	Pages []NavPage
}

type NavPage struct {
	Title string
	Href  string
}

func NewBuilder(siteName string, siteDescription string, siteDomain string, navSections []NavSection, styleSheetNames []string) *HTMLBuilder {
	builder := HTMLBuilder{
		siteName:        siteName,
		siteDescription: siteDescription,
		siteDomain:      siteDomain,
		navSections:     navSections,
	}
	for _, name := range styleSheetNames {
		builder.styleSheetSrc = append(builder.styleSheetSrc, "/"+name)
	}
	return &builder
}

func (builder *HTMLBuilder) SetSiteTwitterHandle(handle string) {
	builder.siteTwitterHandle = handle
}

func (builder *HTMLBuilder) IncludeFavicon() {
	builder.faviconHref = "/favicon.ico"
}

func (builder *HTMLBuilder) SetLogoFile(filename string) {
	builder.logoImageSrc = "/" + filename
}

func (builder *HTMLBuilder) SetOGImage(filename string) {
	builder.ogImageURL = builder.siteDomain + "/" + filename
}

func (builder *HTMLBuilder) SetAlert(alert Alert) {
	builder.alert = &alert
}

func (builder *HTMLBuilder) GenerateHTML(urlPath string, src io.Reader, dst io.Writer) error {
	var matter struct {
		Title string `yaml:"title"`
	}

	pageMarkdown, _ := frontmatter.MustParse(src, &matter)
	if matter.Title == "" {
		return &MissingAttributeError{"title"}
	}

	var markdownHtmlBuf bytes.Buffer

	if err := markdown.Convert(pageMarkdown, &markdownHtmlBuf, parser.WithContext(parser.NewContext())); err != nil {
		panic(err)
	}

	markdownHtml := markdownHtmlBuf.String()
	markdownHtml = strings.ReplaceAll(markdownHtml, "<table>", "<div class=\"table-wrapper\"><table>")
	markdownHtml = strings.ReplaceAll(markdownHtml, "</table>", "</table></div>")

	currentNavPageHref, _ := matchClosestPage(builder.navSections, urlPath)
	err := tmpl.Execute(dst, pageData{
		Markdown:           template.HTML(markdownHtml),
		Name:               builder.siteName,
		Description:        builder.siteDescription,
		Url:                builder.siteDomain + urlPath,
		Twitter:            builder.siteTwitterHandle,
		Title:              matter.Title,
		NavSections:        builder.navSections,
		CurrentNavPageHref: currentNavPageHref,
		LogoImageSrc:       builder.logoImageSrc,
		OGImageURL:         builder.ogImageURL,
		FaviconHref:        builder.faviconHref,
		Stylesheets:        builder.styleSheetSrc,
		Alert:              builder.alert,
	})
	return err
}

type MissingAttributeError struct {
	Attribute string
}

func (e *MissingAttributeError) Error() string {
	return fmt.Sprintf("missing attributes: %s", e.Attribute)
}

func (builder *HTMLBuilder) Generate404HTML(dst io.Writer) error {
	err := tmpl.Execute(dst, pageData{
		Markdown:     template.HTML("<h1>404 - Not found</h1><p>The page you were looking for does not exist.</p>"),
		Name:         builder.siteName,
		Description:  builder.siteDescription,
		Url:          builder.siteDomain + "/404",
		Twitter:      builder.siteTwitterHandle,
		Title:        "Not found",
		NavSections:  builder.navSections,
		LogoImageSrc: builder.logoImageSrc,
		OGImageURL:   builder.ogImageURL,
		FaviconHref:  builder.faviconHref,
		Stylesheets:  builder.styleSheetSrc,
	})
	return err
}

type pageData struct {
	Markdown           template.HTML
	Title              string
	Description        string
	Twitter            string
	Url                string
	Name               string
	NavSections        []NavSection
	CurrentNavPageHref string
	LogoImageSrc       string
	OGImageURL         string
	Stylesheets        []string
	FaviconHref        string
	Alert              *Alert
}

func ParseURLPath(p string) []string {
	if len(p) < 1 {
		panic("invalid path")
	}
	chars := []rune(p)
	var parts []string
	var part string
	for i := 1; i < len(chars); i++ {
		if chars[i] == '/' {
			parts = append(parts, part)
		} else {
			part += string(chars[i])
		}
	}
	if part != "" {
		parts = append(parts, part)
	}
	return parts
}

func matchClosestPage(sections []NavSection, target string) (string, bool) {
	var matched string
	var depth int
	var sameDepthDuplicate int
	targetParts := ParseURLPath(target)
	for _, section := range sections {
		for _, page := range section.Pages {
			if target == page.Href {
				return page.Href, true
			}
			if !strings.HasPrefix(page.Href, "/") {
				continue
			}
			parts := ParseURLPath(page.Href)
			var currentDepth int
			for i := 0; i < len(targetParts) && i < len(parts); i++ {
				if targetParts[i] != parts[i] {
					break
				}
				currentDepth++
			}
			if currentDepth > depth {
				matched = page.Href
				depth = currentDepth
				sameDepthDuplicate = 0
			} else if currentDepth == depth {
				sameDepthDuplicate++
			}
		}
	}
	if depth < 1 || sameDepthDuplicate > 0 {
		return "", false
	}
	return matched, true
}
