package changelog

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		wantType string
		wantDesc string
		wantOK   bool
	}{
		{"feat", "feat: add thing", "feat", "add thing", true},
		{"with scope", "feat(manga): support chapter prefetch", "feat", "support chapter prefetch", true},
		{"breaking marker", "feat!: drop v1 api", "feat", "drop v1 api", true},
		{"scope and breaking", "fix(image)!: prevent duplicate upload", "fix", "prevent duplicate upload", true},
		{"uppercase type", "FEAT: shout", "feat", "shout", true},
		{"trims", "  refactor(cache): simplify  ", "refactor", "simplify", true},
		{"no colon", "just a message", "", "", false},
		{"empty", "", "", "", false},
		{"colon no space", "feat:nospace", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e, ok := Parse(tc.subject)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && (e.Type != tc.wantType || e.Description != tc.wantDesc) {
				t.Fatalf("got {%q, %q}, want {%q, %q}", e.Type, e.Description, tc.wantType, tc.wantDesc)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name     string
		subjects []string
		want     string
	}{
		{
			name: "all three sections in order",
			subjects: []string{
				"feat(manga): support chapter prefetch",
				"fix(image): prevent duplicate upload",
				"refactor(cache): simplify invalidation",
			},
			want: "## Features\n\n- support chapter prefetch\n\n## Fixes\n\n- prevent duplicate upload\n\n## Improvements\n\n- simplify invalidation\n",
		},
		{
			name:     "unknown types and noise dropped",
			subjects: []string{"chore: deps", "docs: readme", "not a commit", "feat: real"},
			want:     "## Features\n\n- real\n",
		},
		{
			name:     "multiple entries per section",
			subjects: []string{"fix: a", "fix: b"},
			want:     "## Fixes\n\n- a\n- b\n",
		},
		{"empty input", nil, emptyNotes},
		{"no conventional commits", []string{"merge branch", "wip"}, emptyNotes},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Generate(tc.subjects); got != tc.want {
				t.Fatalf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}
