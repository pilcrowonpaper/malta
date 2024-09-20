package preview

import (
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func PreviewCommand() int {
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
		extension := filepath.Ext(req.URL.Path)
		if extension != "" {
			data, err := os.ReadFile(path.Join("dist", req.URL.Path))
			if err != nil {
				w.WriteHeader(404)
				w.Write([]byte("404 - Not found"))
				return
			}
			w.Header().Set("Content-Type", mime.TypeByExtension(extension))
			w.Write(data)
			return
		}
		html, err := resolveHTMLRequest(req.URL.Path)
		if errors.Is(err, fs.ErrNotExist) {
			html, _ = os.ReadFile("dist/404.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(404)
			w.Write(html)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(html)
	})

	fmt.Printf("Starting server on port %v...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	fmt.Println(err)
	return 1
}

func resolveHTMLRequest(requestPath string) ([]byte, error) {
	html, err := os.ReadFile(path.Join("dist", requestPath+".html"))
	if err == nil {
		return html, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	html, err = os.ReadFile(path.Join("dist", requestPath, "index.html"))
	return html, err
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
