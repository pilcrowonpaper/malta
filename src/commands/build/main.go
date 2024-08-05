package build

import (
	"bytes"
	"crypto/sha1"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var config struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Domain       string                 `json:"domain"`
	Twitter      string                 `json:"twitter"`
	Sidebar      []SidebarSectionConfig `json:"sidebar"`
	AssetHashing bool                   `json:"asset_hashing"`
}

var markdownFilePaths []string

//go:embed assets/*
var embedded embed.FS

func BuildCommand() int {
	htmlTemplate, err := embedded.ReadFile("assets/template.html")
	if err != nil {
		log.Fatal("template.html does not exist")
	}
	mainCSS, err := embedded.ReadFile("assets/main.css")
	if err != nil {
		log.Fatal("main.css does not exist")
	}
	markdownCSS, err := embedded.ReadFile("assets/markdown.css")
	if err != nil {
		log.Fatal("markdown.css does not exist")
	}

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

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic(err)
	}

	var logoFileName, ogLogoFileName string

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		fileName := dirEntry.Name()
		fileNameWithoutExtension := fileNameWithoutExtension(fileName)
		if fileNameWithoutExtension == "logo" {
			logoFileName = fileName
		}
		if fileNameWithoutExtension == "og-logo" {
			ogLogoFileName = fileName
		}
	}

	var logoFile, ogLogoFile []byte

	if logoFileName != "" {
		logoFile, err = os.ReadFile(logoFileName)
		if err != nil {
			panic(err)
		}
	}
	if ogLogoFileName != "" {
		ogLogoFile, err = os.ReadFile(ogLogoFileName)
		if err != nil {
			panic(err)
		}
	}

	cssAssets := []Asset{}
	assetsEntries, _ := embedded.ReadDir("assets")
	for _, entry := range assetsEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".css" {
			continue
		}
		asset := Asset{}
		asset.Filename = entry.Name()
		if config.AssetHashing {
			file, err := embedded.Open(filepath.Join("assets", asset.Filename))
			if err != nil {
				panic(err)
			}
			defer file.Close()
			css, _ := io.ReadAll(file)
			asset.OutputFileName = getHashedFileName(css, asset.Filename)
		} else {
			asset.OutputFileName = asset.Filename
		}
		cssAssets = append(cssAssets, asset)
	}

	mainCSSFileName, markdownCSSFileName := "main.css", "markdown.css"

	if config.AssetHashing && logoFileName != "" {
		logoFileName = getHashedFileName(logoFile, logoFileName)
	}

	if config.AssetHashing && ogLogoFileName != "" {
		ogLogoFileName = getHashedFileName(ogLogoFile, ogLogoFileName)
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

	var ogImageURL, logoImageSrc string

	if ogLogoFileName != "" {
		ogImageURL = config.Domain + "/" + ogLogoFileName
	}
	if logoFileName != "" {
		logoImageSrc = "/" + logoFileName
	}

	var favicon bool
	if _, err := os.Stat("favicon.ico"); err == nil {
		favicon = true
	}

	styleSheets := []string{}
	for _, asset := range cssAssets {
		styleSheets = append(styleSheets, "/"+asset.OutputFileName)
	}

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

		var currentNavPageHref string

		for _, navSection := range navSections {
			for _, sectionPage := range navSection.Pages {
				if urlPathname == sectionPage.Href || strings.HasPrefix(urlPathname, sectionPage.Href+"/") {
					currentNavPageHref = sectionPage.Href
					break
				}
			}
		}

		err = tmpl.Execute(dstHtmlFile, Data{
			Markdown:           template.HTML(markdownHtml),
			Name:               config.Name,
			Description:        config.Description,
			Url:                config.Domain + urlPathname,
			Twitter:            config.Twitter,
			Title:              matter.Title,
			NavSections:        navSections,
			CurrentNavPageHref: currentNavPageHref,
			LogoImageSrc:       logoImageSrc,
			OGImageURL:         ogImageURL,
			Favicon:            favicon,
			Stylesheets:        styleSheets,
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
		Markdown:     template.HTML("<h1>404 - Not found</h1><p>The page you were looking for does not exist.</p>"),
		Name:         config.Name,
		Description:  config.Description,
		Url:          config.Domain,
		Twitter:      config.Twitter,
		Title:        "Not found",
		NavSections:  navSections,
		LogoImageSrc: logoImageSrc,
		OGImageURL:   ogImageURL,
		Favicon:      favicon,
		Stylesheets:  styleSheets,
	})
	if err != nil {
		panic(err)
	}

	for _, asset := range cssAssets {
		src, err := embedded.Open(filepath.Join("assets", asset.Filename))
		if err != nil {
			panic(err)
		}
		defer src.Close()
		dst, err := os.Create(filepath.Join("dist", asset.OutputFileName))
		if err != nil {
			panic(err)
		}
		defer dst.Close()
		io.Copy(dst, src)
	}
	os.WriteFile(filepath.Join("dist", mainCSSFileName), mainCSS, os.ModePerm)
	os.WriteFile(filepath.Join("dist", markdownCSSFileName), markdownCSS, os.ModePerm)
	if logoFileName != "" {
		os.WriteFile(filepath.Join("dist", logoFileName), logoFile, os.ModePerm)
	}
	if ogLogoFileName != "" {
		os.WriteFile(filepath.Join("dist", ogLogoFileName), ogLogoFile, os.ModePerm)
	}

	if favicon {
		faviconICO, err := os.ReadFile("favicon.ico")
		if err != nil {
			panic(err)
		}
		os.WriteFile("dist/favicon.ico", faviconICO, os.ModePerm)
	}
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

func fileNameWithoutExtension(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}

func getHashedFileName(data []byte, fileName string) string {
	fileHash := sha1.Sum(data)
	hashString := hex.EncodeToString(fileHash[:])
	return hashString + filepath.Ext(fileName)
}

type Data struct {
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
	Favicon            bool
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

type Asset struct {
	Filename       string
	OutputFileName string
}
