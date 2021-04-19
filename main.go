package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Sourcemap struct {
	Version        int      `json:"version"`
	SourceRoot     string   `json:"sourceRoot"`
	Sources        []string `json:"sources"`
	Names          []string `json:"names"`
	File           string   `json:"file"`
	SourcesContent []string `json:"sourcesContent"`
	Mappings       string   `json:"mappings"`
	Groups         []Group
}

type Group struct {
	Line     int
	Segments []Segment
}

type Segment struct {
	StartColumn               int // Field Index 0
	SourcesIndex              int // Field Index 1
	OriginalSourceStartLine   int // Field Index 2
	OriginalSourceStartColumn int // Field Index 3
	NameIndex                 int // Field Index 4
}

func (s *Sourcemap) getFullPath(segment *Segment) string {
	sourcePath := s.Sources[segment.SourcesIndex]

	if s.SourceRoot == "" {
		return s.SourceRoot + sourcePath
	}

	return sourcePath
}

func (s *Sourcemap) findSegmentFromPosition(line int, column int) (Segment, error) {
	idxLine := -1
	for i, group := range s.Groups {
		if group.Line == line {
			idxLine = i
			break
		}
	}

	if idxLine == -1 {
		return Segment{}, fmt.Errorf("could not find a mapping for line %d", line)
	}

	group := s.Groups[idxLine]

	idxColumn := -1

	for i, segment := range group.Segments {
		if segment.StartColumn > column {
			break
		}

		idxColumn = i
	}

	if idxColumn == -1 {
		return Segment{}, fmt.Errorf("could not find a mapping for column %d", column)
	}

	return group.Segments[idxColumn], nil
}

func (s *Sourcemap) print() {
	for _, group := range s.Groups {
		output := fmt.Sprintf("Line #%d: ", group.Line)

		for _, segment := range group.Segments {
			output += fmt.Sprintf(" | %d => (#%d)[%d, %d]", segment.StartColumn, segment.SourcesIndex, segment.OriginalSourceStartLine, segment.OriginalSourceStartColumn)
		}

		fmt.Println(output)
	}
}

type StacktraceEntry struct {
	File       string   `json:"file"`
	MethodName string   `json:"methodName"`
	Arguments  []string `json:"arguments"`
	LineNumber int      `json:"lineNumber"`
	Column     int      `json:"column"`
}

func main() {
	filepath := os.Args[1]

	sourcemap, err := createSourcemapFromFile(filepath)

	if err != nil {
		log.Fatalf("Failed to parse sourcemap from file: %v\n", err)
	}

	// fmt.Printf("%s\n", sourcemap.Mappings)
	sourcemap.print()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Stacktrace JSON: ")
	stacktraceRaw, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	var stacktrace []StacktraceEntry
	err = json.Unmarshal([]byte(stacktraceRaw), &stacktrace)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Got stacktrace %v\n", stacktrace)

	for _, entry := range stacktrace {
		methodName := entry.MethodName
		file := entry.File
		line := entry.LineNumber
		column := entry.Column

		if strings.HasSuffix(entry.File, "main-compiled.js") {
			segment, err := sourcemap.findSegmentFromPosition(entry.LineNumber, entry.Column)
			if err != nil {
				log.Fatal(err)
			}

			file = sourcemap.getFullPath(&segment)
			line = segment.OriginalSourceStartLine
			column = segment.OriginalSourceStartColumn

		}

		fmt.Printf("     at %s (%s:%d:%d)\n", methodName, file, line, column)
	}
}

func createSourcemapFromFile(filepath string) (Sourcemap, error) {

	var sourcemap Sourcemap

	fmt.Printf("Reading sourcemap file at `%s`\n", filepath)
	file, err := os.Open(filepath)

	if err != nil {
		return sourcemap, err
	}

	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)

	if err != nil {
		return sourcemap, err
	}

	json.Unmarshal(byteValue, &sourcemap)

	// These previous values do NOT reset. but the previous StartColumn value resets
	// every line
	previousSourcesIndex := -1

	// Start previous line and column at 1 so that 1 gets
	// added initally to the line and column because the VLQ is 0 indexed
	// but lines and columns are 1 index so we need to bump them up by 1
	previousOriginalSourceStartLine := 1
	previousOriginalSourceStartColumn := 1

	previousNameIndex := -1

	for i, group := range strings.Split(sourcemap.Mappings, ";") {
		currentGroup := Group{Line: i + 1, Segments: []Segment{}}
		for j, segment := range strings.Split(group, ",") {
			decodedMapping := decodeMapping(segment)
			// fmt.Printf("Decoded Mapping %s = %v\n", segment)
			mappingLength := len(decodedMapping)

			if mappingLength == 0 {
				continue
			}

			// If this is the first field of the first segment, or the first segment following a new generated line (“;”),
			// then this field holds the whole base 64 VLQ. Otherwise, this field contains a base 64 VLQ that is relative to the previous occurrence of this field.
			// Resets on each line
			startColumn := decodedMapping[0]
			if j > 0 {
				startColumn = startColumn + currentGroup.Segments[j-1].StartColumn
			}

			currentSegment := Segment{StartColumn: startColumn}

			if mappingLength >= 4 {
				// Get Sources Index
				sourcesIndex := valueWithPrevious(decodedMapping[1], previousSourcesIndex)
				previousSourcesIndex = sourcesIndex
				currentSegment.SourcesIndex = sourcesIndex

				// Get original source start line
				originalSourceStartLine := valueWithPrevious(decodedMapping[2], previousOriginalSourceStartLine)
				previousOriginalSourceStartLine = originalSourceStartLine
				currentSegment.OriginalSourceStartLine = originalSourceStartLine

				// Get original source start column
				originalSourceStartColumn := valueWithPrevious(decodedMapping[3], previousOriginalSourceStartColumn)
				previousOriginalSourceStartColumn = originalSourceStartColumn
				currentSegment.OriginalSourceStartColumn = originalSourceStartColumn
			}

			if mappingLength == 5 {
				nameIndex := valueWithPrevious(decodedMapping[4], previousNameIndex)
				previousNameIndex = nameIndex
				currentSegment.NameIndex = nameIndex
			}

			currentGroup.Segments = append(currentGroup.Segments, currentSegment)
		}

		sourcemap.Groups = append(sourcemap.Groups, currentGroup)
	}

	return sourcemap, nil
}

func valueWithPrevious(value int, previous int) int {
	if previous == -1 {
		return value
	}

	return value + previous
}

func decodeMapping(mapping string) []int {
	// fmt.Printf("Decoding Mapping %s\n", mapping)

	// binary: 100000
	var VLQ_BASE byte = 1 << 5

	// binary: 011111
	var VLQ_BASE_MASK byte = VLQ_BASE - 1

	// binary: 100000
	var VLQ_CONTINUATION_MASK byte = VLQ_BASE

	// binary: 000001
	var VLQ_SIGN_MASK byte = 1

	BASE64 := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	values := []int{}

	for i := 0; i < len(mapping); i++ {
		digit := byte(strings.Index(BASE64, string(mapping[i])))

		// fmt.Printf("Decoding Value %s, %08b\n", string(mapping[i]), digit)

		// Get value bytes, drop the sign bit
		var valueBytes byte = (digit & VLQ_BASE_MASK) >> 1

		var sign byte = (digit & VLQ_SIGN_MASK)
		// fmt.Printf("Sign %08b\n", sign)

		continues := digit & VLQ_CONTINUATION_MASK
		continuedCount := 0
		for continues > 0 {
			continuedCount += 1
			i += 1
			digit = mapping[i]

			// Get value bytes, minus the sign bit
			continuedValueBytes := (digit & VLQ_BASE_MASK) >> 1

			// Append continued value bits onto value bits
			valueBytes = valueBytes | (continuedValueBytes << (4 * continuedCount))

			// Get continuation bit from value
			continues = digit & VLQ_CONTINUATION_MASK
		}

		number := int(valueBytes)
		if sign > 0 {
			number = -number
		}

		// fmt.Printf("Found Bytes: %08b %08b\n", valueBytes, byte(number))
		// fmt.Printf("Found Number: %d\n", number)

		values = append(values, number)
	}

	return values
}
