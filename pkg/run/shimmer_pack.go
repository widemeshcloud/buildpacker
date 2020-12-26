package run

import (
	"context"

	"github.com/widemeshcloud/pack-shimmer/pkg/shimmer"
)

// ShimmerPack runs pack with shimmed buildpacks
type ShimmerPack struct {
	Config
}

// Run runs pack command
func (pack *ShimmerPack) Run(ctx context.Context) error {
	shimmer := &shimmer.Shimmer{}
	localBuildpacks, err := shimmer.Apply(ctx, pack.Buildpacks)
	if err != nil {
		return err
	}
	config := pack.Config
	config.Buildpacks = localBuildpacks
	runner := &CnbPack{
		Config: config,
	}
	if err := runner.Run(ctx); err != nil {
		return err
	}
	return nil
}