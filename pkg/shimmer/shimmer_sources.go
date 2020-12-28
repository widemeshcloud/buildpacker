package shimmer

import (
	"github.com/widemeshcloud/buildpacker/pkg/shimmer/sources"
)

// InstallSources installs a list of sources
func (shimmer *Shimmer) InstallSources(sources []sources.Source) {
	shimmer.Sources = append(shimmer.Sources, sources...)
}
