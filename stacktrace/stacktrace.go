package stacktrace

import (
	"encoding/json"
	"fmt"
	"go-sourcemap/sourcemap"
	"strings"
)

type StacktraceEntry struct {
	File       string   `json:"file"`
	MethodName string   `json:"methodName"`
	Arguments  []string `json:"arguments"`
	LineNumber int      `json:"lineNumber"`
	Column     int      `json:"column"`
}

func (s StacktraceEntry) MapToOriginal(sourcemaps *map[string]sourcemap.Sourcemap) (StacktraceEntry, error) {
	for filepath, _sourcemap := range *sourcemaps {

		// Use hasPrefix to see if sourcemap matches file
		// because filepath will have a `.map` while s.File will not
		if strings.HasPrefix(filepath, s.File) {
			segment, err := _sourcemap.FindSegmentFromPosition(s.LineNumber, s.Column)
			fmt.Printf("Found segment for position %d %d | %v\n", s.LineNumber, s.Column, segment)
			if err != nil {
				return StacktraceEntry{}, err
			}

			file := _sourcemap.GetFullPath(&segment)
			line := segment.OriginalSourceStartLine
			column := segment.OriginalSourceStartColumn
			methodName := s.MethodName
			if len(_sourcemap.Names) > 0 {
				methodName = _sourcemap.Names[segment.NameIndex]
			}

			return StacktraceEntry{File: file, LineNumber: line, Column: column, MethodName: methodName, Arguments: s.Arguments}, nil
		}
	}

	return s, nil
}

func (s *StacktraceEntry) GetTraceText() string {
	return fmt.Sprintf("at %s (%s:%d:%d)", s.MethodName, s.File, s.LineNumber, s.Column)
}

func (s *StacktraceEntry) Print() {
	fmt.Printf("     at %s (%s:%d:%d)\n", s.MethodName, s.File, s.LineNumber, s.Column)
}

func FromString(raw string) ([]StacktraceEntry, error) {
	var stacktrace []StacktraceEntry
	err := json.Unmarshal([]byte(raw), &stacktrace)

	return stacktrace, err
}
