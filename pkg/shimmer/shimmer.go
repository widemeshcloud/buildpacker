package shimmer

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mholt/archiver/v3"
	"github.com/widemeshio/buildpacker/pkg/dl"
	"github.com/widemeshio/buildpacker/pkg/shimmer/sources"
)

// DefaultBuildpackAPIVersion the default API version to write to buildpack.toml
const DefaultBuildpackAPIVersion = "0.4"

// DefaultCnbShimVersion the default cnb-shim version to use
const DefaultCnbShimVersion = "0.2"

// DefaultBuildpackStacks the stacks to write to buildpack.toml
func DefaultBuildpackStacks() []string {
	return []string{"heroku-18", "heroku-20"}
}

// Shimmer shims all the specified buildpacks
type Shimmer struct {
	Sources     []sources.Source
	APIVersion  string
	Stacks      []string
	ShimVersion string
}

// BuildpackAPIVersion the BuildpackAPIVersion to use in buildpack.toml
func (shimmer *Shimmer) BuildpackAPIVersion() string {
	if v := shimmer.APIVersion; v != "" {
		return v
	}
	return DefaultBuildpackAPIVersion
}

// BuildpackStacks the stacks to use in buildpack.toml
func (shimmer *Shimmer) BuildpackStacks() []string {
	if v := shimmer.Stacks; len(v) > 0 {
		return shimmer.Stacks
	}
	return DefaultBuildpackStacks()
}

// CnbShimVersion returns the cnb-shim version to use
func (shimmer *Shimmer) CnbShimVersion() string {
	if v := shimmer.ShimVersion; v != "" {
		return shimmer.ShimVersion
	}
	return DefaultCnbShimVersion
}

// Apply prepares all the specified buildpacks with a shim and returns path to local directories with shim applied
func (shimmer *Shimmer) Apply(ctx context.Context, buildpacks []string) (Buildpacks, IDDictionary, error) {
	shimSupportFile, err := ioutil.TempFile("", "cnd-shim-*.tgz")
	if err != nil {
		return nil, nil, err
	}
	shimSupportFilepath := shimSupportFile.Name()
	defer os.Remove(shimSupportFilepath)
	shimSupportURL := fmt.Sprintf("https://github.com/heroku/cnb-shim/releases/download/v%s/cnb-shim-v%s.tgz", shimmer.CnbShimVersion(), shimmer.CnbShimVersion())
	if err := dl.DownloadFile(shimSupportURL, shimSupportFilepath); err != nil {
		return nil, nil, fmt.Errorf("failed to unpack cnb-shim files, %w", err)
	}
	prepared, err := shimmer.prepare(ctx, buildpacks)
	if err != nil {
		return nil, nil, err
	}
	ids := IDDictionary{}
	for ix, buildpack := range prepared {
		unpacked, ok := buildpack.(UnpackedBuildpack)
		if !ok {
			continue
		}
		shimmedBuildpack := ShimmedBuildpack{
			UnpackedBuildpack: unpacked,
		}
		tomlFile := shimmedBuildpack.ShimBuildpackToml()
		tomlContent := &bytes.Buffer{}
		version := unpacked.Unpacker.RequestedVersion()
		if version == "" {
			version = "0.1"
		}
		originalID := shimmedBuildpack.Unpacker.CanonicalBuildpack()
		id := originalID
		if urlIndex := strings.Index(id, "://"); urlIndex != -1 {
			id = id[urlIndex+3:]
		}
		id = strings.ReplaceAll(id, ":", "_")
		id = strings.ReplaceAll(id, "//", "_")
		err := buildpackTomlTemplate.Execute(tomlContent, &buildpackTomlTemplateParams{
			APIID:   shimmer.BuildpackAPIVersion(),
			ID:      id,
			Name:    shimmedBuildpack.Unpacker.OriginalBuildpack(),
			Version: version,
			Stacks:  shimmer.BuildpackStacks(),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create buildpack.toml content, %w", err)
		}
		if err := ioutil.WriteFile(tomlFile, tomlContent.Bytes(), os.ModePerm); err != nil {
			return nil, nil, fmt.Errorf("failed to create buildpack.toml, %w", err)
		}
		if err := archiver.Unarchive(shimSupportFilepath, unpacked.LocalDir); err != nil {
			return nil, nil, fmt.Errorf("failed to unpack cnb-shim files, %w", err)
		}
		prepared[ix] = shimmedBuildpack
		ids[shimmedBuildpack.Unpacker.OriginalBuildpack()] = id
	}
	return prepared, ids, nil
}

var buildpackToml = `
api = "{{.APIID}}"

[buildpack]
id = "{{.ID}}"
version = "{{.Version}}"
name = "{{.Name}}"

{{range .Stacks}}
[[stacks]]
id = "{{.}}"
{{end}}
`

var buildpackTomlTemplate = template.Must(template.New("toml").Parse(buildpackToml))

type buildpackTomlTemplateParams struct {
	APIID   string
	ID      string
	Name    string
	Version string
	Stacks  []string
}

// ShimmedBuildpack holds information about a shimmed buildpack
type ShimmedBuildpack struct {
	UnpackedBuildpack
}

// ShimBuildpackToml returns the path to the buildpack.toml of the shim
func (shimmed ShimmedBuildpack) ShimBuildpackToml() string {
	return filepath.Join(shimmed.LocalDir, "buildpack.toml")
}
