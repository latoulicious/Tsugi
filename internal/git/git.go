package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// HeadSHA returns the full commit SHA of HEAD in dir.
func HeadSHA(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Subjects returns commit subjects in prev..head (or all up to head if prev is
// empty), newest first — the input for changelog generation.
func Subjects(ctx context.Context, dir, prev, head string) ([]string, error) {
	rng := head
	if prev != "" {
		rng = prev + ".." + head
	}
	out, err := run(ctx, dir, "log", "--format=%s", rng)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// Default adapts the package functions to cli.GitReader.
type Default struct{}

func (Default) HeadSHA(ctx context.Context, dir string) (string, error) { return HeadSHA(ctx, dir) }

func (Default) Subjects(ctx context.Context, dir, prev, head string) ([]string, error) {
	return Subjects(ctx, dir, prev, head)
}
