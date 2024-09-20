package main

import (
	"fmt"
	"os"

	"github.com/pilcrowOnPaper/malta/commands/build"
	"github.com/pilcrowOnPaper/malta/commands/dev"
	"github.com/pilcrowOnPaper/malta/commands/preview"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Print(`
Usage:

malta build   - build and generate HTML files
malta preview - preview build
malta dev     - start dev server

`)
		os.Exit(0)
	}
	if os.Args[1] == "build" {
		os.Exit(build.BuildCommand())
	}
	if os.Args[1] == "preview" {
		os.Exit(preview.PreviewCommand())
	}
	if os.Args[1] == "dev" {
		os.Exit(dev.DevCommand())
	}
	fmt.Printf("Unknown command: %s\n", os.Args[1])
	os.Exit(1)
}
