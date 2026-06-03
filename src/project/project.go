package project

import (
	"io/fs"
	"os"

	"github.com/pilcrowonpaper/malta/utils"
)

func GetOGImageFilename() (string, error) {
	dirEntries, err := os.ReadDir(".")
	if err != nil {
		return "", err
	}

	for _, entry := range dirEntries {
		filename := entry.Name()
		filenameWithoutExtension := utils.FilenameWithoutExtension(filename)
		if filenameWithoutExtension == "og-logo" {
			return filename, nil
		}
	}
	return "", fs.ErrNotExist
}

func GetLogoFilename() (string, error) {
	dirEntries, err := os.ReadDir(".")
	if err != nil {
		return "", err
	}

	for _, entry := range dirEntries {
		filename := entry.Name()
		filenameWithoutExtension := utils.FilenameWithoutExtension(filename)
		if filenameWithoutExtension == "logo" {
			return filename, nil
		}
	}
	return "", fs.ErrNotExist
}

func GetFaviconFile() ([]byte, error) {
	return os.ReadFile("favicon.ico")
}
