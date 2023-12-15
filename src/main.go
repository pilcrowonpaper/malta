package main

import (
	"fmt"
	"os"

	"github.com/pilcrowOnPaper/malta/commands/build"
	"github.com/pilcrowOnPaper/malta/commands/preview"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Print(`
Usage:

malta build   - build and generate HTML files
malta preview - preview build

`)
		os.Exit(0)
	}
	if os.Args[1] == "build" {
		os.Exit(build.BuildCommand())
	}
	if os.Args[1] == "preview" {
		os.Exit(preview.PreviewCommand())
	}
	fmt.Printf("Unknown command: %s\n", os.Args[1])
	os.Exit(1)
}
