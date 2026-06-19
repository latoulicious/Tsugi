package deploy

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run invokes deploy/bin/deploy.sh for target+env; a non-empty ref deploys that
// specific commit (rollback), else the env's branch HEAD.
func Run(ctx context.Context, binDir, target, env, ref string) error {
	args := []string{"--target", target, "--env", env}
	if ref != "" {
		args = append(args, "--ref", ref)
	}
	cmd := exec.CommandContext(ctx, filepath.Join(binDir, "deploy.sh"), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deploy %s/%s: %w", target, env, err)
	}
	return nil
}

// Script adapts Run to cli.Deployer, bound to the deploy/bin directory.
type Script struct{ BinDir string }

func (s Script) Run(ctx context.Context, target, env, ref string) error {
	return Run(ctx, s.BinDir, target, env, ref)
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
