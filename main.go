package main

import (
	"bufio"
	"fmt"
	"go-sourcemap/sourcemap"
	"go-sourcemap/stacktrace"
	"log"
	"os"
	"strings"
)

func main() {
	filepath := os.Args[1]
	nodeModulesPath := filepath
	if strings.HasSuffix(filepath, "/") {
		nodeModulesPath += "node_modules/"
	} else {
		nodeModulesPath += "/node_modules/"
	}
	ignore := []string{nodeModulesPath}
	sourcemaps := sourcemap.FindSourcemaps(filepath, ignore)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Stacktrace JSON: ")
	stacktraceRaw, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	stacktraceParsed, err := stacktrace.FromJson(stacktraceRaw)
	if err != nil {
		log.Fatal(err)
	}

	for i, entry := range stacktraceParsed {
		mappedEntry, err := entry.MapToOriginal(&sourcemaps)
		if err != nil {
			log.Fatal(err)
		}

		stacktraceParsed[i] = mappedEntry
	}

	for _, entry := range stacktraceParsed {
		entry.Print()
	}
}
