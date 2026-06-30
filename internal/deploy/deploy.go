package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/latoulicious/Tsugi/internal/deployflow"
)

// Run invokes deploy/bin/deploy.sh for target+env, streaming stdout/stderr to sink
// line by line; a non-empty ref deploys that specific commit (rollback), else the
// env's branch HEAD.
func Run(ctx context.Context, binDir, target, env, ref string, sink deployflow.LogSink) error {
	args := []string{"--target", target, "--env", env}
	if ref != "" {
		args = append(args, "--ref", ref)
	}
	cmd := exec.CommandContext(ctx, filepath.Join(binDir, "deploy.sh"), args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("deploy %s/%s: stdout pipe: %w", target, env, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("deploy %s/%s: stderr pipe: %w", target, env, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("deploy %s/%s: %w", target, env, err)
	}
	// Drain both pipes before Wait — Wait closes them, so a still-reading scanner
	// would race the close.
	var wg sync.WaitGroup
	wg.Add(2)
	go streamLines(&wg, stdout, "stdout", sink)
	go streamLines(&wg, stderr, "stderr", sink)
	wg.Wait()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("deploy %s/%s: %w", target, env, err)
	}
	return nil
}

// streamLines forwards each line from r to sink, tagged with stream.
func streamLines(wg *sync.WaitGroup, r io.Reader, stream string, sink deployflow.LogSink) {
	defer wg.Done()
	sc := bufio.NewScanner(r)
	// ponytail: default 64KB line cap; bump the scanner buffer if compose ever
	// emits a longer single line.
	for sc.Scan() {
		sink.Line(stream, sc.Text())
	}
}

// Script adapts Run to deployflow.Deployer, bound to the deploy/bin directory.
type Script struct{ BinDir string }

func (s Script) Run(ctx context.Context, target, env, ref string, sink deployflow.LogSink) error {
	return Run(ctx, s.BinDir, target, env, ref, sink)
}

// StagingCheckout reads CHECKOUT_STAGING from a target's target.env (the dev
// checkout the CLI reads git history from when creating a release).
func StagingCheckout(deployDir, target string) (string, error) {
	path := filepath.Join(deployDir, "targets", target, "target.env")
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open target config: %w", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if ok && k == "CHECKOUT_STAGING" {
			return strings.Trim(v, `"'`), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("read target config: %w", err)
	}
	return "", fmt.Errorf("CHECKOUT_STAGING unset in %s", path)
}
