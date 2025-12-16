package scanner

import (
	"bufio"
	"embed"
	"log"
	"strings"
)

//go:embed reserved_words/*.txt
var reservedWordsData embed.FS

var reservedWords map[Dialect]map[string]bool

func isReservedWord(dialect Dialect, word string) bool {
	m, ok := reservedWords[dialect]
	if !ok {
		return false
	}
	return m[strings.ToLower(word)]
}

func init() {
	reservedWords = make(map[Dialect]map[string]bool)
	reservedWords[DialectPostgreSQL] = loadReservedWords("postgresql.txt")
	reservedWords[DialectSQLite] = loadReservedWords("sqlite.txt")
	reservedWords[DialectSQLServer] = loadReservedWords("sqlserver.txt")
	reservedWords[DialectOracle] = loadReservedWords("oracle.txt")
	reservedWords[DialectMySQL] = loadReservedWords("mysql.txt")
}

func loadReservedWords(file string) map[string]bool {
	r := make(map[string]bool)
	f, err := reservedWordsData.Open("reserved_words/" + file)
	if err != nil {
		log.Fatalf("sqlb: failed to open embedded file %s: %v", file, err)
	}
	// f.Close() is not necessary for embed.FS files, but it's good practice
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r[strings.ToLower(line)] = true
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("sqlb: error scanning embedded file %s: %v", file, err)
	}
	return r
}
