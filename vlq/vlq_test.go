package vlq

import "testing"

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestDecode(t *testing.T) {
	toTest := make(map[string][]int)

	toTest["AAAA"] = []int{0, 0, 0, 0}
	toTest["EAAgB"] = []int{2, 0, 0, 16}
	toTest["mBAAD"] = []int{19, 0, 0, -1}
	toTest["SAAa"] = []int{9, 0, 0, 13}

	for mapping, expected := range toTest {

		t.Run(mapping, func(t *testing.T) {
			result := Decode(mapping)

			if !Equal(result, expected) {
				t.Errorf("Mapping %s expected result %v but got %v", mapping, expected, result)
			}
		})
	}
}