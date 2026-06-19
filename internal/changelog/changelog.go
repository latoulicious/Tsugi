package changelog

import (
	"regexp"
	"strings"
)

// conventional-commit subject: type(scope)?!?: description
var subjectRE = regexp.MustCompile(`^([a-zA-Z]+)(?:\([^)]*\))?!?: (.+)$`)

// only the three types PLAN.md names map to a section; others are ignored
var sections = []struct {
	typ   string
	title string
}{
	{"feat", "Features"},
	{"fix", "Fixes"},
	{"refactor", "Improvements"},
}

const emptyNotes = "_No notable changes._\n"

// Entry is one parsed conventional-commit subject; scope is dropped (unused in output).
type Entry struct {
	Type        string
	Description string
}

func Parse(subject string) (Entry, bool) {
	m := subjectRE.FindStringSubmatch(strings.TrimSpace(subject))
	if m == nil {
		return Entry{}, false
	}
	return Entry{Type: strings.ToLower(m[1]), Description: strings.TrimSpace(m[2])}, true
}

// Generate groups commit subjects into markdown release notes; non-conventional
// subjects and unmapped types are skipped. Empty input yields a fallback note.
func Generate(subjects []string) string {
	grouped := make(map[string][]string)
	for _, s := range subjects {
		if e, ok := Parse(s); ok {
			grouped[e.Type] = append(grouped[e.Type], e.Description)
		}
	}
	var b strings.Builder
	for _, sec := range sections {
		items := grouped[sec.typ]
		if len(items) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## " + sec.title + "\n\n")
		for _, it := range items {
			b.WriteString("- " + it + "\n")
		}
	}
	if b.Len() == 0 {
		return emptyNotes
	}
	return b.String()
}
