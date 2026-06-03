package assets

import (
	"embed"
	"io/fs"
	"path/filepath"
)

//go:embed files/*
var embedded embed.FS

func GetFile(filename string) (fs.File, error) {
	return embedded.Open(filepath.Join("files", filename))
}

func Read(filename string) ([]byte, error) {
	return embedded.ReadFile(filepath.Join("files", filename))
}

func GetFilenames() ([]string, error) {
	entries, err := embedded.ReadDir("files")
	if err != nil {
		return nil, err
	}
	var filenames []string
	for _, entry := range entries {
		filenames = append(filenames, entry.Name())
	}
	return filenames, nil
}
