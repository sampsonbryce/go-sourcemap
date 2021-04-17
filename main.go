package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Sourcemap struct {
	Version        int      `json:"version"`
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
	StartColumn                  int // Field Index 0
	SourcesIndex                 int // Field Index 1
	OriginalSourceStartingLine   int // Field Index 2
	OriginalSrouceStartingColumn int // Field Index 3
	NameIndex                    int // Field Index 4
}

func main() {
	filepath := os.Args[1]

	sourcemap, err := createSourcemapFromFile(filepath)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse sourcemap from file: %v\n", err)
	}

	fmt.Printf("%s", sourcemap.Mappings)
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

	for i, group := range strings.Split(sourcemap.Mappings, ";") {
		currentGroup := Group{Line: i, Segments: []Segment{}}
		for _, segment := range strings.Split(group, ",") {
			decodedMapping := decodeMapping(segment)
			// fmt.Printf("Decoded Mapping %s = %v\n", segment)
			mappingLength := len(decodedMapping)

			if mappingLength == 0 {
				continue
			}

			currentSegment := Segment{StartColumn: decodedMapping[0]}

			if mappingLength >= 4 {
				currentSegment.SourcesIndex = decodedMapping[1]
				currentSegment.OriginalSourceStartingLine = decodedMapping[2]
				currentSegment.OriginalSrouceStartingColumn = decodedMapping[3]
			}

			if mappingLength == 5 {
				currentSegment.NameIndex = decodedMapping[4]
			}

			currentGroup.Segments = append(currentGroup.Segments, currentSegment)
		}

		sourcemap.Groups = append(sourcemap.Groups, currentGroup)
	}

	return sourcemap, nil
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
