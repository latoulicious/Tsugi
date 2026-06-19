package version

import "runtime/debug"

// defaults cover a plain `go build` with no -ldflags injection
const (
	devVersion  = "dev"
	noCommit    = "none"
	unknownDate = "unknown"
)

// set at link time via -X (see Makefile / Dockerfile)
var (
	Version = devVersion
	Commit  = noCommit
	Date    = unknownDate
)

type Info struct {
	Version    string `json:"version"`
	Commit     string `json:"commit"`
	DeployedAt string `json:"deployed_at"`
}

func Get() Info {
	commit, date := Commit, Date
	if commit == noCommit || date == unknownDate {
		c, d := vcsStamp()
		if commit == noCommit && c != "" {
			commit = c
		}
		if date == unknownDate && d != "" {
			date = d
		}
	}
	return Info{Version: Version, Commit: commit, DeployedAt: date}
}

// vcsStamp reads the commit and build time Go embeds from version control.
func vcsStamp() (commit, date string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.time":
			date = s.Value
		}
	}
	return commit, date
}
