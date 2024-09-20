package dev

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pilcrowOnPaper/malta/build"
)

func DevCommand() int {
	port := 3000
	args := parseArgs(os.Args[2:])
	portArg, ok := args["p"]
	if !ok {
		portArg, ok = args["port"]
	}
	if ok {
		parsedPort, err := strconv.Atoi(portArg)
		if err != nil {
			fmt.Println("Invalid argument: 'port' must be a number")
			return 1
		}
		port = parsedPort
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		config, err := build.ParseConfigFile()
		if err != nil {
			log.Fatal(err)
		}

		assetFilenames, err := build.GetAssetFilenames()
		if err != nil {
			log.Fatal(err)
		}

		var cssAssetFilenames []string
		for _, assetFilename := range assetFilenames {
			if filepath.Ext(assetFilename) == ".css" {
				cssAssetFilenames = append(cssAssetFilenames, assetFilename)
			}
		}

		builder := build.NewBuilder(config.Name, config.Description, fmt.Sprintf("http://localhost:%d", port), config.NavSections, cssAssetFilenames)
		if config.TwitterHandle != "" {
			builder.SetSiteTwitterHandle(config.TwitterHandle)
		}

		ogFilename, err := build.GetOGImageFilename()
		if err == nil {
			builder.SetOGImage(ogFilename)
		} else if !errors.Is(err, fs.ErrNotExist) {
			log.Fatal(err)
		}

		logoFilename, err := build.GetLogoFilename()
		if err == nil {
			builder.SetLogoFile(logoFilename)
		} else if !errors.Is(err, fs.ErrNotExist) {
			log.Fatal(err)
		}

		favicon, err := build.GetFaviconFile()
		if err != nil {
			builder.IncludeFavicon()
		} else if !errors.Is(err, fs.ErrNotExist) {
			log.Fatal(err)
		}

		if ogFilename != "" && req.URL.Path == "/"+ogFilename {
			image, err := os.ReadFile(ogFilename)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("Failed to read %s: %v", path.Join("pages", req.URL.Path+".md"), err)))
				return
			}
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(ogFilename)))
			w.Write(image)
			return
		}
		if logoFilename != "" && req.URL.Path == "/"+logoFilename {
			image, err := os.ReadFile(logoFilename)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("Failed to read %s: %v", path.Join("pages", req.URL.Path+".md"), err)))
				return
			}
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(logoFilename)))
			w.Write(image)
			return
		}
		if len(favicon) > 0 && req.URL.Path == "/favicon.ico" {
			w.Header().Set("Content-Type", "image/x-icon")
			w.Write(favicon)
			return
		}

		fileExtension := filepath.Ext(req.URL.Path)
		if fileExtension == ".css" {
			if strings.Count(req.URL.Path, "/") != 1 {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(404)
				builder.Generate404HTML(w)
				return
			}
			asset, err := build.GetAsset(filepath.Base(req.URL.Path))
			if errors.Is(err, fs.ErrNotExist) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(404)
				builder.Generate404HTML(w)
				return
			}
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("Failed to read %s: %v", filepath.Base(req.URL.Path), err)))
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.Copy(w, asset)
			return
		}

		if fileExtension == "" {
			file, err := ResolveMarkdownFileFromHTTPRequestPath(req.URL.Path)
			if errors.Is(err, fs.ErrNotExist) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(404)
				builder.Generate404HTML(w)
				return
			} else if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("Failed to read %s: %v", path.Join("pages", req.URL.Path+".md"), err)))
				return
			}

			var html bytes.Buffer
			err = builder.GenerateHTML(req.URL.Path, file, &html)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("Failed to build %s: %v", file.Name(), err)))
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.Copy(w, &html)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(404)
		builder.Generate404HTML(w)
	})
	fmt.Printf("Starting server on port %v...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	fmt.Println(err)
	return 1
}

func ResolveMarkdownFileFromHTTPRequestPath(reqPath string) (*os.File, error) {
	file, err := os.Open(path.Join("pages", reqPath+".md"))
	if err == nil {
		return file, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	file, err = os.Open(path.Join("pages", reqPath, "index.md"))
	return file, err

}

func parseArgs(argList []string) map[string]string {
	args := make(map[string]string)
	for i := 0; i < len(argList); i++ {
		item := argList[i]
		if item == "" {
			continue
		}
		if !strings.HasPrefix(item, "-") {
			continue
		}
		item = strings.TrimLeft(item, "-")
		if strings.Contains(item, "=") {
			keyValue := strings.Split(item, "=")
			key := keyValue[0]
			args[key] = ""
			if len(keyValue) > 1 {
				args[key] = keyValue[1]
			}
			continue
		}
		if i+1 == len(argList) || strings.HasPrefix(argList[i+1], "-") {
			args[item] = ""
			continue
		}
		args[item] = argList[i+1]
	}
	return args
}
