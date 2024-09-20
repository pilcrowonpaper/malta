package build

import (
	"crypto/sha1"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pilcrowOnPaper/malta/build"
	"github.com/pilcrowOnPaper/malta/utils"
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
	configJson, err := os.ReadFile("malta.config.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Missing 'malta.config.json'")
			return 1
		}
		fmt.Println(err)
		return 1
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
		fmt.Println(err)
		return 1
	}

	var logoFilename, ogLogoFilename string

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		filename := dirEntry.Name()
		filenameWithoutExtension := utils.FilenameWithoutExtension(filename)
		if filenameWithoutExtension == "logo" {
			logoFilename = filename
		}
		if filenameWithoutExtension == "og-logo" {
			ogLogoFilename = filename
		}
	}

	var logoFile, ogLogoFile []byte

	if logoFilename != "" {
		logoFile, err = os.ReadFile(logoFilename)
		if err != nil {
			fmt.Println(err)
			return 1
		}
	}
	if ogLogoFilename != "" {
		ogLogoFile, err = os.ReadFile(ogLogoFilename)
		if err != nil {
			fmt.Println(err)
			return 1
		}
	}

	cssAssets := []Asset{}
	assetFilenames, _ := build.GetAssetFilenames()
	for _, assetFilename := range assetFilenames {
		if filepath.Ext(assetFilename) != ".css" {
			continue
		}
		asset := Asset{
			Filename: assetFilename,
		}
		if config.AssetHashing {
			file, err := embedded.Open(filepath.Join("assets", asset.Filename))
			if err != nil {
				fmt.Println(err)
				return 1
			}
			defer file.Close()
			css, _ := io.ReadAll(file)
			asset.OutputFilename = getHashedFilename(css, asset.Filename)
		} else {
			asset.OutputFilename = asset.Filename
		}
		cssAssets = append(cssAssets, asset)
	}

	if config.AssetHashing && logoFilename != "" {
		logoFilename = getHashedFilename(logoFile, logoFilename)
	}
	if config.AssetHashing && ogLogoFilename != "" {
		ogLogoFilename = getHashedFilename(ogLogoFile, ogLogoFilename)
	}

	navSections := []build.NavSection{}
	for _, sidebarSection := range config.Sidebar {
		navSection := build.NavSection{Title: sidebarSection.Title, Pages: []build.NavPage{}}
		for _, sidebarSectionPage := range sidebarSection.Pages {
			navPage := build.NavPage{Title: sidebarSectionPage[0], Href: sidebarSectionPage[1]}
			navSection.Pages = append(navSection.Pages, navPage)
		}
		navSections = append(navSections, navSection)
	}

	if err := filepath.Walk("pages", walkPagesDir); err != nil {
		fmt.Println(err)
		return 1
	}

	markdown := goldmark.New(goldmark.WithExtensions(extension.Table))
	markdown.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(&codeBlockLinksAstTransformer{}, 500)), parser.WithAutoHeadingID())
	markdown.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&codeBlockLinksRenderer{}, 100)))

	os.RemoveAll("dist")

	var favicon bool
	if _, err := os.Stat("favicon.ico"); err == nil {
		favicon = true
	}

	styleSheetFilenames := []string{}
	for _, asset := range cssAssets {
		styleSheetFilenames = append(styleSheetFilenames, asset.OutputFilename)
	}

	builder := build.NewBuilder(config.Name, config.Description, config.Domain, navSections, styleSheetFilenames)
	if config.Twitter != "" {
		builder.SetSiteTwitterHandle(config.Twitter)
	}
	if favicon {
		builder.IncludeFavicon()
	}
	if ogLogoFilename != "" {
		builder.SetOGImage(ogLogoFilename)
	}
	if logoFilename != "" {
		builder.SetLogoFile(logoFilename)
	}

	for _, markdownFilePath := range markdownFilePaths {
		markdownFile, _ := os.Open(markdownFilePath)
		defer markdownFile.Close()

		dstPath := strings.Replace(strings.Replace(markdownFilePath, "pages/", "dist/", 1), ".md", ".html", 1)

		if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
			fmt.Println(err)
			return 1
		}

		dstHtmlFile, err := os.Create(dstPath)
		if err != nil {
			fmt.Println(err)
			return 1
		}

		defer dstHtmlFile.Close()

		urlPathname := strings.Replace(strings.Replace(dstPath, "dist", "", 1), ".html", "", 1)
		urlPathname = strings.Replace(urlPathname, "/index", "", 1)
		if urlPathname == "" {
			urlPathname = "/"
		}

		err = builder.GenerateHTML(urlPathname, markdownFile, dstHtmlFile)
		if err != nil {
			fmt.Println(err)
			return 1
		}
	}

	notFoundDstHtmlFile, err := os.Create("dist/404.html")
	if err != nil {
		fmt.Println(err)
		return 1
	}
	err = builder.Generate404HTML(notFoundDstHtmlFile)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	for _, asset := range cssAssets {
		src, err := embedded.Open(filepath.Join("assets", asset.Filename))
		if err != nil {
			fmt.Println(err)
			return 1
		}
		defer src.Close()
		dst, err := os.Create(filepath.Join("dist", asset.OutputFilename))
		if err != nil {
			fmt.Println(err)
			return 1
		}
		defer dst.Close()
		io.Copy(dst, src)
	}

	if logoFilename != "" {
		os.WriteFile(filepath.Join("dist", logoFilename), logoFile, os.ModePerm)
	}
	if ogLogoFilename != "" {
		os.WriteFile(filepath.Join("dist", ogLogoFilename), ogLogoFile, os.ModePerm)
	}

	if favicon {
		faviconICO, err := os.ReadFile("favicon.ico")
		if err != nil {
			fmt.Println(err)
			return 1
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

func getHashedFilename(data []byte, filename string) string {
	fileHash := sha1.Sum(data)
	hashString := hex.EncodeToString(fileHash[:])
	return hashString + filepath.Ext(filename)
}

type SidebarSectionConfig struct {
	Title string     `json:"title"`
	Pages [][]string `json:"pages"`
}

type Asset struct {
	Filename       string
	OutputFilename string
}
