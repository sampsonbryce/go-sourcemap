package main

import (
	"encoding/json"
	"fmt"
	"go-sourcemap/sourcemap"
	"go-sourcemap/stacktrace"
	"log"
	"net/http"
	"os"
	"strings"
)

type ExceptionData struct {
	Name    string                       `json:"name"`
	Message string                       `json:"message"`
	Trace   []stacktrace.StacktraceEntry `json:"trace"`
}

func handleException(sourcemaps *map[string]sourcemap.Sourcemap) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var exceptionData ExceptionData
		err := json.NewDecoder(r.Body).Decode(&exceptionData)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		for i, entry := range exceptionData.Trace {
			mappedEntry, err := entry.MapToOriginal(sourcemaps)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}

			exceptionData.Trace[i] = mappedEntry
		}

		fmt.Fprintf(w, "%s: %s\n", exceptionData.Name, exceptionData.Message)
		for _, entry := range exceptionData.Trace {
			fmt.Fprintf(w, "     %s\n", entry.GetTraceText())
		}
	}
}

func main() {

	filepath := os.Args[1]

	// Always ignore node_modules
	nodeModulesPath := filepath
	if strings.HasSuffix(filepath, "/") {
		nodeModulesPath += "node_modules/"
	} else {
		nodeModulesPath += "/node_modules/"
	}

	ignore := []string{nodeModulesPath}

	sourcemaps := sourcemap.FindSourcemaps(filepath, ignore)

	fmt.Println("\n\tReady to collect exceptions. Server listening on 8080..")

	http.HandleFunc("/exception", handleException(&sourcemaps))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
