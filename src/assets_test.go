package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// localAssetRef matches a first-party stylesheet or script referenced from a
// template, in either form: a raw path, or one already wrapped in the asset
// helper. Absolute URLs are left alone; they belong to a CDN and are versioned
// by whoever publishes them.
var localAssetRef = regexp.MustCompile("(?:href|src)=\"(?:\\{\\{asset `)?(/[a-zA-Z0-9._-]+\\.(?:css|js))")

// assetHelperCall matches a path that is wrapped in the asset helper.
var assetHelperCall = regexp.MustCompile("\\{\\{asset `(/[a-zA-Z0-9._-]+\\.(?:css|js))`\\}\\}")

func templateFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.Walk("./views", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			files = append(files, path)
		}
		return nil
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, files, "no templates found; the walk root is probably wrong")
	return files
}

// A stylesheet or script served from a fixed path lets a cache pair one deploy's
// asset with the next deploy's markup, which renders the page unstyled. Every
// first-party asset must therefore go through the asset helper, which appends a
// content hash. This test exists because adding a new script and forgetting to
// version it reintroduces that failure silently: nothing breaks locally, and the
// damage only appears at a CDN edge after a deploy.
func TestEveryLocalAssetReferenceIsVersioned(t *testing.T) {
	for _, path := range templateFiles(t) {
		body, err := os.ReadFile(path)
		assert.NoError(t, err)
		source := string(body)

		versioned := map[string]bool{}
		for _, m := range assetHelperCall.FindAllStringSubmatch(source, -1) {
			versioned[m[1]] = true
		}

		for _, m := range localAssetRef.FindAllStringSubmatch(source, -1) {
			assert.Truef(t, versioned[m[1]],
				"%s references %s without the asset helper; wrap it as {{asset `%s`}} so a cache cannot serve a stale copy against new markup",
				path, m[1], m[1])
		}
	}
}

// The helper falls back to an unversioned path when it cannot read a file, so a
// typo or a rename would silently reintroduce the fixed-URL problem rather than
// failing loudly.
func TestVersionedAssetsExistOnDisk(t *testing.T) {
	for _, path := range versionedAssets {
		_, err := os.Stat("../public" + path)
		assert.NoErrorf(t, err, "versionedAssets lists %s but no such file exists under public/", path)
	}
}

// Every asset the templates ask the helper to version has to be in the list the
// startup hash pass walks, or it ships unversioned in production.
func TestTemplatesOnlyVersionKnownAssets(t *testing.T) {
	known := map[string]bool{}
	for _, path := range versionedAssets {
		known[path] = true
	}
	for _, path := range templateFiles(t) {
		body, err := os.ReadFile(path)
		assert.NoError(t, err)
		for _, m := range assetHelperCall.FindAllStringSubmatch(string(body), -1) {
			assert.Truef(t, known[m[1]],
				"%s versions %s but it is missing from versionedAssets, so no hash is computed for it at startup",
				path, m[1])
		}
	}
}
