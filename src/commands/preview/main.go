package preview

import (
	"fmt"
	"net/http"
	"os"
	"path"
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
		html, err := os.ReadFile(path.Join("dist", req.URL.Path+".html"))
		if err != nil {
			html, err = os.ReadFile(path.Join("dist", req.URL.Path, "index.html"))
			if err != nil {
				data, err := os.ReadFile(path.Join("dist", req.URL.Path))
				if err != nil {
					w.WriteHeader(404)
					w.Write([]byte("404 - Not found"))
					return
				}
				if strings.HasSuffix(req.URL.Path, ".css") {
					w.Header().Set("Content-Type", "text/css")
				}
				w.WriteHeader(200)
				w.Write(data)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write(html)
	})
	fmt.Printf("Starting server on port %v...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	fmt.Println(err)
	return 1
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
