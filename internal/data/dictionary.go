package data

import (
	_ "embed"
	"sort"
	"strings"
	"sync"
)

//go:embed dictionary/entries.tsv
var dictionary string

//go:embed dictionary/LICENSE
var DictionaryLicense string

type Entry struct{ Word, Phonetic, POS, Translation string }

var indexWords = sync.OnceValue(func() []string { return strings.Split(strings.TrimSuffix(dictionary, "\n"), "\n") })

func Lookup(word string) (Entry, bool) {
	key := strings.ToLower(strings.TrimSpace(word)) + "\t"
	words := indexWords()
	i := sort.SearchStrings(words, key)
	if i == len(words) || !strings.HasPrefix(words[i], key) {
		return Entry{}, false
	}
	fields := strings.SplitN(words[i], "\t", 5)
	return Entry{Word: fields[1], Phonetic: fields[2], POS: fields[3], Translation: strings.ReplaceAll(fields[4], `\n`, "\n")}, true
}
