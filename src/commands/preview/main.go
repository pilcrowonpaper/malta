package preview

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
)

func PreviewCommand() int {
	port := 3000
	args := parseArgs(os.Args[2:]) // Change to start from index 1
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
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte("404 - Not found"))
					return
				}
				if strings.HasSuffix(req.URL.Path, ".css") {
					w.Header().Set("Content-Type", "text/css")
				}
				w.WriteHeader(http.StatusOK)
				w.Write(data)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(html)
	})

	go func() {
		if portInUse(port) {
			killPort(port)
		}

		fmt.Printf("Starting server on port %v...\n", port)
		err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	go openBrowser(fmt.Sprintf("http://localhost:%d", port))

	select {}
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

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func portInUse(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

func killPort(port int) {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/C", fmt.Sprintf("netstat -ano | findstr :%d", port))
		out, _ := cmd.Output()
		pid := string(out)
		cmd = exec.Command("taskkill", "/F", "/PID", pid)
		cmd.Run()
	default:
		cmd := exec.Command("sh", "-c", fmt.Sprintf("lsof -t -i:%d", port))
		out, _ := cmd.Output()
		pid := string(out)
		fmt.Println(pid)
		cmd = exec.Command("kill", "-9", pid)
		cmd.Run()
	}
}
