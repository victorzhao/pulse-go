package fsx

import (
	_ "embed"
	"strings"
	"sync"
	"unicode"
)

//go:embed pinyin.bin
var pinyinData string

var pinyinOnce = sync.OnceValues(func() ([]string, []uint16) {
	syllables, table, _ := strings.Cut(pinyinData, "\n")
	words := strings.Fields(syllables)
	idx := make([]uint16, 0, len(table)/2)
	for i := 0; i+1 < len(table); i += 2 {
		idx = append(idx, uint16(table[i])<<8|uint16(table[i+1]))
	}
	return words, idx
})

const pinyinFirst, pinyinEnd = 0x4E00, 0xA000

// Readings returns the primary Mandarin reading of r, without tones, and
// false for characters that have none in the table.
func Readings(r rune) (string, bool) {
	words, idx := pinyinOnce()
	if r < pinyinFirst || r >= pinyinEnd || int(r)-pinyinFirst >= len(idx) {
		return "", false
	}
	n := idx[r-pinyinFirst]
	if n == 0 || int(n) > len(words) {
		return "", false
	}
	return words[n-1], true
}

// Pinyin returns the full pinyin of s, lowercase, and its initials.
func Pinyin(s string) (full, initials string) {
	var f, i strings.Builder
	for _, r := range s {
		if reading, ok := Readings(r); ok {
			f.WriteString(reading)
			i.WriteByte(reading[0])
		} else {
			lr := unicode.ToLower(r)
			if lr >= 'a' && lr <= 'z' || lr >= '0' && lr <= '9' {
				f.WriteRune(lr)
				i.WriteRune(lr)
			}
		}
	}
	return f.String(), i.String()
}

// MatchName reports whether name matches query, directly or by pinyin. The
// query is split into terms by spaces, and every term must match.
func MatchName(name, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	lower := strings.ToLower(name)
	full, initials := Pinyin(name)
	for _, term := range strings.Fields(q) {
		if !strings.Contains(lower, term) && !strings.Contains(full, term) && !strings.Contains(initials, term) {
			return false
		}
	}
	return true
}

// MatchRanges returns the byte ranges of name that match query, for
// highlighting, and whether it matches at all. Pinyin matches highlight the
// characters whose readings match, not the Latin query.
func MatchRanges(name, query string) ([][2]int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, true
	}
	lower := strings.ToLower(name)
	var ranges [][2]int
	ok := true
	for _, term := range strings.Fields(q) {
		if idx := strings.Index(lower, term); idx >= 0 {
			ranges = append(ranges, [2]int{idx, idx + len(term)})
			continue
		}
		// Try pinyin: find the run of characters whose readings contain term.
		if rs, found := pinyinRange(name, term); found {
			ranges = append(ranges, rs)
			continue
		}
		ok = false
	}
	return ranges, ok
}

func pinyinRange(name, term string) ([2]int, bool) {
	type char struct {
		r     rune
		at    int
		full  string
		first byte
	}
	chars := make([]char, 0, len(name))
	full := strings.Builder{}
	for at, r := range name {
		reading, ok := Readings(r)
		if !ok {
			continue
		}
		chars = append(chars, char{r, at, reading, reading[0]})
		full.WriteString(reading)
	}
	idx := strings.Index(full.String(), term)
	if idx < 0 {
		return [2]int{}, false
	}
	// Map the pinyin byte range back to the characters that produced it.
	pos := 0
	start, end := -1, -1
	for _, ch := range chars {
		next := pos + len(ch.full)
		if start < 0 && idx >= pos && idx < next {
			start = ch.at
		}
		if idx+len(term) > pos && idx+len(term) <= next {
			end = ch.at + len(string(ch.r))
			break
		}
		pos = next
	}
	if start < 0 || end < 0 {
		return [2]int{}, false
	}
	return [2]int{start, end}, true
}
