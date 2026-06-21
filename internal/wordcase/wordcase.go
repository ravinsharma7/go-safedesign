package wordcase

import (
	"strings"
	"unicode"
)

func SplitWords(value string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		words = append(words, strings.ToLower(string(current)))
		current = nil
	}
	runes := []rune(value)
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(current) > 0 {
			prev := current[len(current)-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsUpper(r) && (unicode.IsLower(prev) || (unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next))) {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return words
}
