package utils

import (
	"path/filepath"
	"strings"
)

func ParseArgs(argList []string) map[string]string {
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

func FilenameWithoutExtension(filename string) string {
	return filename[:len(filename)-len(filepath.Ext(filename))]
}
