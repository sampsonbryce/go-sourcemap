package sourcemap

import (
	"encoding/json"
	"fmt"
	"go-sourcemap/vlq"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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

func (s *Sourcemap) GetFullPath(segment *Segment) string {
	sourcePath := s.Sources[segment.SourcesIndex]

	if s.SourceRoot == "" {
		return s.SourceRoot + sourcePath
	}

	return sourcePath
}

func (s *Sourcemap) FindSegmentFromPosition(line int, column int) (Segment, error) {
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

func (s *Sourcemap) Print() {
	for _, group := range s.Groups {
		output := fmt.Sprintf("Line #%d: ", group.Line)

		for _, segment := range group.Segments {
			output += fmt.Sprintf(" | %d => (#%d)[%d, %d]", segment.StartColumn, segment.SourcesIndex, segment.OriginalSourceStartLine, segment.OriginalSourceStartColumn)
		}

		fmt.Println(output)
	}
}

func CreateSourcemapFromFile(filepath string) (Sourcemap, error) {

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
			decodedMapping := vlq.Decode(segment)
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

func FindSourcemaps(dir string, ignore []string) map[string]Sourcemap {
	sourcemaps := make(map[string]Sourcemap)

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(info.Name(), ".map") {
				// Skip paths we want to ignore
				for _, pathToIgnore := range ignore {
					if strings.HasPrefix(path, pathToIgnore) {
						return nil
					}
				}

				fmt.Printf("Found map at %s\n", path)
				_sourcemap, err := CreateSourcemapFromFile(path)
				if err != nil {
					return err
				}

				sourcemaps[path] = _sourcemap
			}

			return nil
		})

	if err != nil {
		log.Println(err)
	}

	return sourcemaps
}

func valueWithPrevious(value int, previous int) int {
	if previous == -1 {
		return value
	}

	return value + previous
}
