package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bedrock-mc/geyser-go/data"
)

func main() {
	cloudburstDir := flag.String("cloudburst-dir", "", "path to a pinned CloudburstMC/Data checkout")
	output := flag.String("out", "generated_catalog.go", "generated Go source path")
	packageName := flag.String("package", "data", "generated Go package name")
	version := flag.String("version", "cloudburst-current", "catalog version label")
	flag.Parse()
	if *cloudburstDir == "" {
		fatal("-cloudburst-dir is required; payloads are inputs and are not vendored")
	}
	catalog, err := data.LoadCloudburst(*cloudburstDir, *version)
	if err != nil {
		fatal(err.Error())
	}
	file, err := os.Create(*output)
	if err != nil {
		fatal(fmt.Sprintf("create %s: %v", *output, err))
	}
	if err := data.WriteGo(file, *packageName, catalog); err != nil {
		_ = file.Close()
		fatal(err.Error())
	}
	if err := file.Close(); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("generated %d items and %d blocks from %s\n", len(catalog.Items), len(catalog.Blocks), *cloudburstDir)
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
