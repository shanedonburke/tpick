package text

import "unicode/utf8"

func Width(s string) int {
	return utf8.RuneCountInString(s)
}
