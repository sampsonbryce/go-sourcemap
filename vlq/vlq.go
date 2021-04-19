package vlq

import "strings"

func Decode(mapping string) []int {
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
