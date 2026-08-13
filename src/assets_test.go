package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// localAssetRef matches a first-party stylesheet or script referenced from a
// template, in either form: a raw path, or one already wrapped in the asset
// helper. Absolute URLs are left alone; they belong to a CDN and are versioned
// by whoever publishes them.
var localAssetRef = regexp.MustCompile("(?:href|src)=\"(?:\\{\\{asset `)?(/[a-zA-Z0-9._-]+\\.(?:css|js))")

// assetHelperCall matches a path that is wrapped in the asset helper.
var assetHelperCall = regexp.MustCompile("\\{\\{asset `(/[a-zA-Z0-9._-]+\\.(?:css|js))`\\}\\}")

// versionedURL matches the shape the helper emits: the path, then the short
// content hash it appends.
var versionedURL = regexp.MustCompile(`^(/[a-zA-Z0-9._-]+)\?v=[0-9a-f]{10}$`)

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
	require.NoError(t, err)
	require.NotEmpty(t, files, "no templates found; the walk root is probably wrong")
	return files
}

// assetDir populates a throwaway directory with every versioned asset, so an
// index built on it behaves as it does against public/ without depending on
// what happens to be committed there.
func assetDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, path := range versionedAssets {
		require.NoError(t, os.WriteFile(filepath.Join(dir, path), []byte("/* "+path+" */"), 0o600))
	}
	return dir
}

// A stylesheet or script served from a fixed path lets a cache pair one deploy's
// asset with the next deploy's markup, which renders the page unstyled. Every
// first-party asset must therefore go through the asset helper, which appends a
// content hash. This test exists because adding a new script and forgetting to
// version it reintroduces that failure silently: nothing breaks locally, and the
// damage only appears at a CDN edge after a deploy.
func TestAssetReferences(t *testing.T) {
	t.Run("every local asset reference goes through the helper", func(t *testing.T) {
		for _, path := range templateFiles(t) {
			body, err := os.ReadFile(path)
			require.NoError(t, err)
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
	})

	// Every asset the templates ask the helper to version has to be in the list
	// the startup hash pass walks, or it ships unversioned in production.
	t.Run("templates only version assets the index knows about", func(t *testing.T) {
		known := map[string]bool{}
		for _, path := range versionedAssets {
			known[path] = true
		}
		for _, path := range templateFiles(t) {
			body, err := os.ReadFile(path)
			require.NoError(t, err)
			for _, m := range assetHelperCall.FindAllStringSubmatch(string(body), -1) {
				assert.Truef(t, known[m[1]],
					"%s versions %s but it is missing from versionedAssets, so no hash is computed for it at startup",
					path, m[1])
			}
		}
	})

	// The helper falls back to an unversioned path when it cannot read a file,
	// so a typo or a rename would silently reintroduce the fixed-URL problem
	// rather than failing loudly.
	t.Run("every versioned asset exists on disk", func(t *testing.T) {
		for _, path := range versionedAssets {
			_, err := os.Stat("../public" + path)
			assert.NoErrorf(t, err, "versionedAssets lists %s but no such file exists under public/", path)
		}
	})
}

func TestAssetIndex(t *testing.T) {
	t.Run("a known asset gets its content hash appended", func(t *testing.T) {
		index := newAssetIndex(assetDir(t), false)

		for _, path := range versionedAssets {
			assert.Regexpf(t, versionedURL, index.url(path), "%s should be served versioned", path)
		}
	})

	t.Run("assets with different contents get different versions", func(t *testing.T) {
		index := newAssetIndex(assetDir(t), false)

		assert.NotEqual(t, index.url("/app.css"), index.url("/index.js"))
	})

	t.Run("identical contents anywhere produce the same version", func(t *testing.T) {
		first, second := assetDir(t), t.TempDir()
		body, err := os.ReadFile(filepath.Join(first, "/app.css"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(second, "/app.css"), body, 0o600))

		assert.Equal(t,
			newAssetIndex(first, false).url("/app.css"),
			newAssetIndex(second, false).url("/app.css"))
	})

	// A missing asset must still render a usable page: an unversioned URL is a
	// caching risk, a broken reference is a broken page.
	t.Run("an unreadable asset falls back to its plain path", func(t *testing.T) {
		index := newAssetIndex(filepath.Join(t.TempDir(), "does-not-exist"), false)

		assert.Equal(t, "/app.css", index.url("/app.css"))
	})

	t.Run("a path the index never hashed is left alone", func(t *testing.T) {
		index := newAssetIndex(assetDir(t), false)

		assert.Equal(t, "/logo.png", index.url("/logo.png"))
	})

	t.Run("reload picks up an edited asset without a restart", func(t *testing.T) {
		dir := assetDir(t)
		index := newAssetIndex(dir, true)
		before := index.url("/app.css")

		rewriteAsset(t, dir, "/app.css", "/* edited */")

		assert.NotEqual(t, before, index.url("/app.css"))
	})

	// Production hashes once at startup: the files cannot change under a running
	// process, and re-stating on every render would tax every page.
	t.Run("without reload an edited asset keeps its startup version", func(t *testing.T) {
		dir := assetDir(t)
		index := newAssetIndex(dir, false)
		before := index.url("/app.css")

		rewriteAsset(t, dir, "/app.css", "/* edited */")

		assert.Equal(t, before, index.url("/app.css"))
	})

	t.Run("an unchanged asset keeps one stable URL", func(t *testing.T) {
		index := newAssetIndex(assetDir(t), true)

		assert.Equal(t, index.url("/app.css"), index.url("/app.css"))
	})
}

// rewriteAsset replaces an asset's contents and moves its modification time,
// which is what the reload check keys on. Filesystem timestamp resolution is
// coarse enough that a same-instant rewrite would otherwise look unchanged.
func rewriteAsset(t *testing.T, dir, path, contents string) {
	t.Helper()
	full := filepath.Join(dir, path)
	require.NoError(t, os.WriteFile(full, []byte(contents), 0o600))
	later := time.Now().Add(time.Second)
	require.NoError(t, os.Chtimes(full, later, later))
}
