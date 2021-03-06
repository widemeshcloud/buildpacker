package run

import (
	"context"
	"os"
	"os/exec"
)

// Config runs the pack command
type Config struct {
	Path         string
	Builder      string
	ImageTag     string
	Buildpacks   []string
	Env          []string
	TrustBuilder bool
}

// CnbPack runs pack
type CnbPack struct {
	Config
}

// Run runs pack command
func (pack *CnbPack) Run(ctx context.Context) error {
	args := []string{"build", "--path", pack.Path, "--builder", pack.Builder}
	if pack.TrustBuilder {
		args = append(args, "--trust-builder")
	}
	for _, bp := range pack.Buildpacks {
		args = append(args, "--buildpack", bp)
	}
	args = append(args, pack.ImageTag)
	// log.Printf("pack args: %s", strings.Join(args, " "))
	cmd := exec.Command("pack", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
